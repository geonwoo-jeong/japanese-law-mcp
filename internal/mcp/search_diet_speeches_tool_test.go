package mcp

import (
	"context"
	"encoding/json"
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
		t.Fatalf("tool result = %#v", result)
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
		t.Fatalf("payload = %#v", payload)
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
		t.Fatalf("tool result = %#v, want error", result)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("MCP セッションを終了できません: %v", err)
	}
	if runErr := <-serverResult; runErr != nil {
		t.Fatalf("MCP サーバーが正常終了しませんでした: %v", runErr)
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
