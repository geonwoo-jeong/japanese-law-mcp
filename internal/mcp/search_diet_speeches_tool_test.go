package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSearchDietSpeechesToolReturnsStructuredSuccess(t *testing.T) {
	t.Parallel()

	server := sdk.NewServer(&sdk.Implementation{Name: "test", Version: "test"}, nil)
	addSearchDietSpeechesTool(server, &recordingDietSpeechSearchPort{
		page: mustDietSpeechPage(t),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverResult := make(chan error, 1)
	go func() { serverResult <- server.Run(ctx, serverTransport) }()

	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-client", Version: "test"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("MCP セッションを初期化できません: %v", err)
	}
	result, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name: "search_diet_speeches",
		Arguments: map[string]any{
			"query": "永住許可",
			"limit": 1,
		},
	})
	if err != nil {
		t.Fatalf("search_diet_speeches を呼び出せません: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatal("search_diet_speeches が成功結果を返していません")
	}
	text, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("content type = %T", result.Content[0])
	}
	var payload searchDietSpeechesOutput
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Fatalf("result JSON を解析できません: %v", err)
	}
	if payload.UsageNotice != dietSpeechesUsageNotice ||
		len(payload.Items) != 1 ||
		payload.Items[0].Ref.ProviderID != "ndl-diet-speech-api" ||
		payload.Items[0].Data.SpeechID != "speech-1" ||
		payload.Page.ReturnedCount != 1 {
		t.Fatal("SOT-IF-066: 公開結果が国会発言契約と一致しません")
	}
	if err := session.Close(); err != nil {
		t.Fatalf("MCP セッションを終了できません: %v", err)
	}
	if runErr := <-serverResult; runErr != nil {
		t.Fatalf("MCP サーバーが正常終了しませんでした: %v", runErr)
	}
}

func TestSearchDietSpeechesToolRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	server := sdk.NewServer(&sdk.Implementation{Name: "test", Version: "test"}, nil)
	addSearchDietSpeechesTool(server, stubSearchDietSpeechesPort{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverResult := make(chan error, 1)
	go func() { serverResult <- server.Run(ctx, serverTransport) }()

	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-client", Version: "test"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("MCP セッションを初期化できません: %v", err)
	}
	result, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name: "search_diet_speeches",
		Arguments: map[string]any{
			"limit": 1,
		},
	})
	if err != nil {
		t.Fatalf("tool call error = %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("SOT-IF-066: 不正入力を tool error にしていません")
	}
	if err := session.Close(); err != nil {
		t.Fatalf("MCP セッションを終了できません: %v", err)
	}
	if runErr := <-serverResult; runErr != nil {
		t.Fatalf("MCP サーバーが正常終了しませんでした: %v", runErr)
	}
}

func TestDecodeSearchDietSpeechesInputRejectsInvalidUTF8(t *testing.T) {
	t.Parallel()

	input := append([]byte(`{"query":"`), 0xff)
	input = append(input, []byte(`"}`)...)
	if _, err := decodeSearchDietSpeechesInput(input); err == nil {
		t.Fatal("SOT-IF-062/066: 不正な UTF-8 を受理しました")
	}
}

func TestDecodeSearchDietSpeechesInputMapsEverySearchCondition(t *testing.T) {
	t.Parallel()

	request, err := decodeSearchDietSpeechesInput(json.RawMessage(`{
		"speaker":" 山田太郎 ",
		"meetingName":"法務委員会",
		"house":"house_of_councillors",
		"fromDate":"2024-01-01",
		"untilDate":"2024-01-31",
		"limit":30
	}`))
	if err != nil {
		t.Fatalf("SOT-IF-066: 有効な全検索条件を変換できません: %v", err)
	}
	speaker, speakerExists := request.Speaker()
	meetingName, meetingExists := request.MeetingName()
	house, houseExists := request.House()
	fromDate, fromDateExists := request.FromDate()
	untilDate, untilDateExists := request.UntilDate()
	if !speakerExists || speaker != "山田太郎" ||
		!meetingExists || meetingName != "法務委員会" ||
		!houseExists || house != parliamentspeechsearch.HouseOfCouncillors ||
		!fromDateExists || fromDate.String() != "2024-01-01" ||
		!untilDateExists || untilDate.String() != "2024-01-31" ||
		request.Limit() != 30 {
		t.Fatal("SOT-IF-062/066: 検索条件の変換結果が一致しません")
	}
}

func TestDecodeSearchDietSpeechesInputRejectsBoundaryViolations(t *testing.T) {
	t.Parallel()

	tests := map[string]json.RawMessage{
		"JSON object 以外":    json.RawMessage(`[]`),
		"null":              json.RawMessage(`{"query":null}`),
		"未知項目":              json.RawMessage(`{"query":"民法","unknown":true}`),
		"非整数 limit":         json.RawMessage(`{"query":"民法","limit":1.5}`),
		"実在しない日付":           json.RawMessage(`{"fromDate":"2024-02-30"}`),
		"逆転した日付":            json.RawMessage(`{"fromDate":"2024-02-02","untilDate":"2024-02-01"}`),
		"未定義の院":             json.RawMessage(`{"house":"diet"}`),
		"空でない continuation": json.RawMessage(`{"query":"民法","continuationToken":"next"}`),
	}
	for name, input := range tests {
		name, input := name, input
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := decodeSearchDietSpeechesInput(input); err == nil {
				t.Fatal("SOT-IF-066: 不正な入力境界を受理しました")
			}
		})
	}
}

func TestSearchDietSpeechesToolFailsClosedForUnknownPortError(t *testing.T) {
	t.Parallel()

	result, err := callSearchDietSpeeches(
		context.Background(),
		&recordingDietSpeechSearchPort{err: errors.New("秘密の内部原因")},
		json.RawMessage(`{"query":"民法"}`),
	)
	if err != nil {
		t.Fatalf("tool error = %v", err)
	}
	if result == nil || !result.IsError || len(result.Content) != 1 {
		t.Fatal("SOT-IF-007/066: tool error result を返していません")
	}
	text, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatal("SOT-IF-007: error content が TextContent ではありません")
	}
	if strings.Contains(text.Text, "秘密の内部原因") {
		t.Fatal("SOT-IF-027/066: 内部原因を公開しました")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Fatalf("error result JSON を解析できません: %v", err)
	}
	if payload["code"] != string(model.ErrorCodeInternalError) {
		t.Fatal("SOT-IF-066: 未分類の失敗を internal_error にしていません")
	}
}

type recordingDietSpeechSearchPort struct {
	page parliamentspeechsearch.Page
	err  error
}

func (p *recordingDietSpeechSearchPort) Search(
	_ context.Context,
	_ parliamentspeechsearch.Request,
) (parliamentspeechsearch.Page, error) {
	if p.err != nil {
		return parliamentspeechsearch.Page{}, p.err
	}
	return p.page, nil
}

type stubSearchDietSpeechesPort struct{}

func (stubSearchDietSpeechesPort) Search(
	context.Context,
	parliamentspeechsearch.Request,
) (parliamentspeechsearch.Page, error) {
	return mustDietSpeechPageForStub(), nil
}

func mustDietSpeechPage(t *testing.T) parliamentspeechsearch.Page {
	t.Helper()
	return mustDietSpeechPageForStub()
}

func mustDietSpeechPageForStub() parliamentspeechsearch.Page {
	source := mustDietSpeechSource()
	speaker, _ := model.NewParliamentSpeaker(model.ParliamentSpeakerValues{
		Name: "山田太郎",
	})
	date, _ := model.NewDate("2024-01-01")
	meeting, _ := model.NewParliamentMeetingReference(model.ParliamentMeetingReferenceValues{
		MeetingRecordID: "100",
		ImageKind:       "会議録",
		Session:         1,
		HouseName:       "参議院",
		MeetingName:     "法務委員会",
		Issue:           "1",
		MeetingDate:     date,
		MeetingURL:      "https://kokkai.ndl.go.jp/txt/100",
	})
	speech, _ := model.NewParliamentSpeech(model.ParliamentSpeechValues{
		SpeechID:    "speech-1",
		SpeechOrder: 1,
		Speaker:     speaker,
		SpeechText:  "発言本文",
		SpeechURL:   "https://kokkai.ndl.go.jp/txt/1",
		Meeting:     meeting,
		Source:      source,
	})
	key, _ := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     source.ID(),
		ResourceType: "parliament-speech",
		ResourceID:   speech.SpeechID(),
	})
	ref, _ := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "ndl-diet-speech-api",
		Key:        key,
	})
	provenance, _ := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            "https://kokkai.ndl.go.jp/api/speech?any=%E6%B0%B8%E4%BD%8F%E8%A8%B1%E5%8F%AF&startRecord=1&maximumRecords=1&recordPacking=json",
		RetrievedAt:    time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC),
		MediaType:      "application/json",
		Location:       "speechRecord[0]",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-064",
	})
	item, _ := model.NewSourcedResource(model.SourcedResourceValues[model.ParliamentSpeech]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       speech,
	})
	totalCount := 1
	page, _ := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: 1,
		TotalCount:    &totalCount,
		TotalRelation: model.TotalRelationExact,
	})
	result, _ := parliamentspeechsearch.NewPage(parliamentspeechsearch.PageValues{
		Items: []model.SourcedResource[model.ParliamentSpeech]{item},
		Page:  page,
	})
	return result
}

func mustDietSpeechSource() model.InformationSource {
	source, _ := model.NewInformationSource(model.InformationSourceValues{
		ID:         "ndl-diet-records",
		Name:       "国会会議録検索システム",
		Publisher:  "国立国会図書館",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://kokkai.ndl.go.jp/",
	})
	return source
}
