package kokkai

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestParseSpeechSearchResponseAndMapPage(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"numberOfRecords": 2,
		"numberOfReturn": 1,
		"startRecord": 1,
		"nextRecordPosition": 2,
		"speechRecord": [
			{
				"speechID": "0001",
				"speechOrder": 3,
				"speaker": " 山田 太郎 ",
				"speakerYomi": " やまだ たろう ",
				"speakerGroup": " 自由民主党 ",
				"speakerPosition": "",
				"speakerRole": null,
				"speech": "\n  第一行\n第二行  \n",
				"startPage": 0,
				"speechURL": "https://kokkai.ndl.go.jp/txt/0001",
				"issueID": "100",
				"imageKind": "会議録",
				"session": 213,
				"nameOfHouse": "参議院",
				"nameOfMeeting": "法務委員会",
				"issue": "5",
				"date": "2024-03-01",
				"closing": false,
				"meetingURL": "https://kokkai.ndl.go.jp/txt/kaigiroku/100",
				"pdfURL": "https://kokkai.ndl.go.jp/pdf/100.pdf"
			}
		]
	}`)

	response, err := parseSpeechSearchResponseForTest(context.Background(), body, 30)
	if err != nil {
		t.Fatalf("SOT-IF-064: parseSpeechSearchResponse() のエラー = %v", err)
	}
	if response.totalCount != 2 || response.returnedCount != 1 || len(response.records) != 1 {
		t.Fatal("SOT-IF-064: response 件数が一致しません")
	}
	if response.nextRecordPosition == nil || *response.nextRecordPosition != 2 {
		t.Fatalf("SOT-IF-064: nextRecordPosition = %v", response.nextRecordPosition)
	}

	retrievedAt := time.Date(2026, 8, 23, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	page, err := mapSpeechSearchPage(
		context.Background(),
		response,
		"https://kokkai.ndl.go.jp/api/speech?any=%E6%B0%91%E6%B3%95&startRecord=1&maximumRecords=1&recordPacking=json",
		retrievedAt,
	)
	if err != nil {
		t.Fatalf("SOT-IF-064: mapSpeechSearchPage() のエラー = %v", err)
	}
	items := page.Items()
	if len(items) != 1 {
		t.Fatalf("items の件数 = %d", len(items))
	}
	item := items[0]
	if item.Ref().ProviderID() != providerID ||
		item.Ref().Key().SourceID() != sourceID ||
		item.Ref().Key().ResourceType() != "parliament-speech" ||
		item.Ref().Key().ResourceID() != "0001" {
		t.Fatal("ref の固定値が一致しません")
	}
	provenance := item.Provenance()
	location, exists := provenance[0].Location()
	methodID, methodExists := provenance[0].MethodID()
	if len(provenance) != 1 ||
		!exists || location != "speechRecord[0]" ||
		!methodExists || methodID != "SOT-IF-064" ||
		provenance[0].URL() != "https://kokkai.ndl.go.jp/api/speech?any=%E6%B0%91%E6%B3%95&startRecord=1&maximumRecords=1&recordPacking=json" ||
		provenance[0].MediaType() != "application/json" ||
		provenance[0].Transformation() != model.ProvenanceTransformationNormalized ||
		!provenance[0].RetrievedAt().Equal(retrievedAt) {
		t.Fatal("provenance の固定値が一致しません")
	}

	data := item.Data()
	startPage, hasStartPage := data.StartPage()
	reading, hasReading := data.Speaker().Reading()
	group, hasGroup := data.Speaker().Group()
	_, hasPosition := data.Speaker().Position()
	_, hasRole := data.Speaker().Role()
	pdfURL, hasPDFURL := data.Meeting().PDFURL()
	closing, hasClosing := data.Meeting().Closing()
	if data.SpeechID() != "0001" ||
		data.SpeechOrder() != 3 ||
		data.Speaker().Name() != "山田 太郎" ||
		!hasReading || reading != "やまだ たろう" ||
		!hasGroup || group != "自由民主党" ||
		hasPosition || hasRole ||
		!hasStartPage || startPage != 0 ||
		data.SpeechText() != "第一行\n第二行" ||
		data.SpeechURL() != "https://kokkai.ndl.go.jp/txt/0001" ||
		data.Meeting().MeetingRecordID() != "100" ||
		data.Meeting().ImageKind() != "会議録" ||
		data.Meeting().Session() != 213 ||
		data.Meeting().HouseName() != "参議院" ||
		data.Meeting().MeetingName() != "法務委員会" ||
		data.Meeting().Issue() != "5" ||
		data.Meeting().MeetingDate().String() != "2024-03-01" ||
		!hasClosing || closing ||
		data.Meeting().MeetingURL() != "https://kokkai.ndl.go.jp/txt/kaigiroku/100" ||
		!hasPDFURL || pdfURL != "https://kokkai.ndl.go.jp/pdf/100.pdf" {
		t.Fatal("ParliamentSpeech の項目対応が一致しません")
	}

	totalCount, totalExists := page.Page().TotalCount()
	totalRelation, relationExists := page.Page().TotalRelation()
	if page.Page().ReturnedCount() != 1 ||
		!totalExists || totalCount != 2 ||
		!relationExists || totalRelation != model.TotalRelationExact {
		t.Fatal("SourcePage の件数が一致しません")
	}
}

func TestParseSpeechSearchResponseRejectsNonContractJSONShapes(t *testing.T) {
	t.Parallel()

	record := `{"speechID":"1","speechOrder":0,"speaker":"a","speech":"x",` +
		`"speechURL":"https://kokkai.ndl.go.jp/txt/1","issueID":"100",` +
		`"imageKind":"会議録","session":1,"nameOfHouse":"衆議院",` +
		`"nameOfMeeting":"本会議","issue":"1","date":"2024-01-01",` +
		`"meetingURL":"https://kokkai.ndl.go.jp/txt/100"}`
	tests := map[string]string{
		"speechRecord null": `{"numberOfRecords":0,"numberOfReturn":0,` +
			`"startRecord":1,"speechRecord":null}`,
		"top-level key の大文字小文字違い": `{"NumberOfRecords":0,` +
			`"numberOfReturn":0,"startRecord":1}`,
		"record key の大文字小文字違い": `{"numberOfRecords":1,` +
			`"numberOfReturn":1,"startRecord":1,"speechRecord":[` +
			strings.Replace(record, `"speechID"`, `"SpeechID"`, 1) + `]}`,
	}
	for name, body := range tests {
		name, body := name, body
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := parseSpeechSearchResponseForTest(context.Background(), []byte(body), 20)
			assertSpeechSearchSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
		})
	}
}

func TestParseSpeechSearchResponseAcceptsUnknownFields(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"numberOfRecords": 1,
		"numberOfReturn": 1,
		"startRecord": 1,
		"unknownTopLevel": "ignored",
		"speechRecord": [
			{
				"speechID": "1",
				"speechOrder": 0,
				"speaker": "a",
				"speech": "x",
				"speechURL": "https://kokkai.ndl.go.jp/txt/1",
				"issueID": "100",
				"imageKind": "会議録",
				"session": 1,
				"nameOfHouse": "衆議院",
				"nameOfMeeting": "本会議",
				"issue": "1",
				"date": "2024-01-01",
				"meetingURL": "https://kokkai.ndl.go.jp/txt/100",
				"unknownField": {"nested":"kept in budget only"}
			}
		]
	}`)

	response, err := parseSpeechSearchResponseForTest(context.Background(), body, 20)
	if err != nil {
		t.Fatalf("未知 field を含む response を拒否しました: %v", err)
	}
	if response.totalCount != 1 || response.returnedCount != 1 || len(response.records) != 1 {
		t.Fatal("未知 field を無視した後の件数が一致しません")
	}
}

func TestParseSpeechSearchResponseClassifiesBusyAPIError(t *testing.T) {
	t.Parallel()

	_, err := parseSpeechSearchResponseForTest(
		context.Background(),
		[]byte(`{"message":"現在、混み合っております。もうしばらくしてから再度アクセスしてください。","details":[]}`),
		20,
	)
	assertSpeechSearchSourceError(t, err, model.SourceErrorCodeSourceUnavailable)
}

func TestMapSpeechSearchPageRejectsNonOfficialRequestURL(t *testing.T) {
	t.Parallel()

	response, err := parseSpeechSearchResponseForTest(
		context.Background(),
		[]byte(`{"numberOfRecords":1,"numberOfReturn":1,"startRecord":1,`+
			`"speechRecord":[{"speechID":"1","speechOrder":0,"speaker":"a",`+
			`"speech":"x","speechURL":"https://kokkai.ndl.go.jp/txt/1",`+
			`"issueID":"100","imageKind":"会議録","session":1,`+
			`"nameOfHouse":"衆議院","nameOfMeeting":"本会議","issue":"1",`+
			`"date":"2024-01-01","meetingURL":"https://kokkai.ndl.go.jp/txt/100"}]}`),
		20,
	)
	if err != nil {
		t.Fatalf("検証用 response を作成できません: %v", err)
	}
	for name, requestURL := range map[string]string{
		"別 origin":     "https://example.invalid/api/speech",
		"別 path":       "https://kokkai.ndl.go.jp/api/meeting",
		"encoded path": "https://kokkai.ndl.go.jp/api/%73peech",
	} {
		name, requestURL := name, requestURL
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, mapErr := mapSpeechSearchPage(
				context.Background(),
				response,
				requestURL,
				time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC),
			)
			assertSpeechSearchSourceError(
				t,
				mapErr,
				model.SourceErrorCodeInvalidSourceResponse,
			)
		})
	}
}

func TestMapSpeechSearchPagePropagatesCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := mapSpeechSearchPage(
		ctx,
		speechSearchResponse{},
		"https://kokkai.ndl.go.jp/api/speech?startRecord=1",
		time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel の伝播 = %v", err)
	}
}

func TestParseSpeechSearchResponseAcceptsEmptyResult(t *testing.T) {
	t.Parallel()

	response, err := parseSpeechSearchResponseForTest(
		context.Background(),
		[]byte(`{"numberOfRecords":0,"numberOfReturn":0,"startRecord":1}`),
		20,
	)
	if err != nil {
		t.Fatalf("空結果のエラー = %v", err)
	}
	if response.totalCount != 0 || response.returnedCount != 0 || len(response.records) != 0 {
		t.Fatal("空結果の件数が一致しません")
	}
}

func TestParseSpeechSearchResponseRejectsStructuralErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"残件があるのに next がない":   `{"numberOfRecords":2,"numberOfReturn":1,"startRecord":1,"speechRecord":[{"speechID":"1","speechOrder":0,"speaker":"a","speech":"x","speechURL":"https://kokkai.ndl.go.jp/txt/1","issueID":"100","imageKind":"会議録","session":1,"nameOfHouse":"衆議院","nameOfMeeting":"本会議","issue":"1","date":"2024-01-01","meetingURL":"https://kokkai.ndl.go.jp/txt/100"}]}`,
		"count と配列長が不一致":     `{"numberOfRecords":1,"numberOfReturn":1,"startRecord":1,"speechRecord":[]}`,
		"required field 空文字": `{"numberOfRecords":1,"numberOfReturn":1,"startRecord":1,"speechRecord":[{"speechID":"1","speechOrder":0,"speaker":"a","speech":"x","speechURL":"https://kokkai.ndl.go.jp/txt/1","issueID":"100","imageKind":"会議録","session":1,"nameOfHouse":"衆議院","nameOfMeeting":"本会議","issue":"1","date":"","meetingURL":"https://kokkai.ndl.go.jp/txt/100"}]}`,
		"duplicate key":      `{"numberOfRecords":0,"numberOfRecords":0,"numberOfReturn":0,"startRecord":1}`,
		"trailing data":      `{"numberOfRecords":0,"numberOfReturn":0,"startRecord":1} []`,
		"root type":          `[]`,
	}
	for name, body := range tests {
		name, body := name, body
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := parseSpeechSearchResponseForTest(context.Background(), []byte(body), 20)
			assertSpeechSearchSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
		})
	}
}

func TestParseSpeechSearchResponseRejectsResourceBudgetOverruns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		body              string
		maximumInputBytes int
		maximumValues     int
		maximumDepth      int
		want              model.SourceErrorCode
	}{
		{
			name:              "byte 上限",
			body:              "12345",
			maximumInputBytes: 4,
			maximumValues:     10,
			maximumDepth:      10,
			want:              model.SourceErrorCodeSourceResponseTooLarge,
		},
		{
			name:              "JSON value 上限",
			body:              `{"unknown":1}`,
			maximumInputBytes: 64,
			maximumValues:     1,
			maximumDepth:      10,
			want:              model.SourceErrorCodeSourceResponseTooLarge,
		},
		{
			name:              "JSON depth 上限",
			body:              `{"unknown":[]}`,
			maximumInputBytes: 64,
			maximumValues:     10,
			maximumDepth:      1,
			want:              model.SourceErrorCodeUnsafeSourceContent,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			parseContext, cancel := context.WithTimeout(
				context.Background(),
				speechSearchParseTimeout,
			)
			defer cancel()
			_, err := parseSpeechSearchResponseWithBudget(
				context.Background(),
				parseContext,
				[]byte(test.body),
				20,
				test.maximumInputBytes,
				test.maximumValues,
				test.maximumDepth,
			)
			assertSpeechSearchSourceError(t, err, test.want)
		})
	}
}

func TestParseSpeechSearchResponseRejectsUnsafeInput(t *testing.T) {
	t.Parallel()

	_, err := parseSpeechSearchResponseForTest(
		context.Background(),
		[]byte{0xff, 0xfe, 0xfd},
		20,
	)
	assertSpeechSearchSourceError(t, err, model.SourceErrorCodeUnsafeSourceContent)
}

func TestParseSpeechSearchResponsePropagatesCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := parseSpeechSearchResponseForTest(ctx, []byte(`{}`), 20)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel の伝播 = %v", err)
	}
}

func assertSpeechSearchSourceError(
	t *testing.T,
	err error,
	want model.SourceErrorCode,
) {
	t.Helper()
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("SourceError ではありません: %T %v", err, err)
	}
	if sourceError.Code() != want {
		t.Fatalf("SourceError.Code() = %q, want %q", sourceError.Code(), want)
	}
}

func parseSpeechSearchResponseForTest(
	parent context.Context,
	body []byte,
	limit int,
) (speechSearchResponse, error) {
	parseContext, cancel := context.WithTimeout(parent, speechSearchParseTimeout)
	defer cancel()
	return parseSpeechSearchResponse(parent, parseContext, body, limit)
}
