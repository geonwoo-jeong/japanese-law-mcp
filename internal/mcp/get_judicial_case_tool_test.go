package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGetJudicialCaseToolReturnsStructuredSuccess(t *testing.T) {
	t.Parallel()

	item := mustJudicialDecisionDetailsResource()
	port := &recordingGetJudicialCasePort{item: item}
	result, err := callGetJudicialCase(
		context.Background(),
		port,
		canonicalJudicialCaseInput(),
	)
	if err != nil {
		t.Fatalf("callGetJudicialCase() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, want false: %#v", result)
	}
	if port.calls != 1 ||
		port.request.Ref().Key().ResourceID() != "95570/detail2" {
		t.Fatalf("SOT-IF-048: read request = %#v, calls = %d", port.request, port.calls)
	}

	var payload getJudicialCaseOutput
	content := []byte(result.Content[0].(*sdk.TextContent).Text)
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("content JSON error = %v", err)
	}
	if payload.CoverageNotice != judicialCasesCoverageNotice {
		t.Fatalf("coverageNotice = %q", payload.CoverageNotice)
	}
	if payload.Item.Ref.ProviderID != "courts-hanrei-html" ||
		payload.Item.Ref.Key.SourceID != "courts-hanrei" ||
		payload.Item.Ref.Key.ResourceType != "judicial-decision" ||
		payload.Item.Ref.Key.ResourceID != "95570/detail2" ||
		payload.Item.Data.Summary.DecisionID != "95570" {
		t.Fatalf("SOT-IF-048: payload = %#v", payload)
	}

	var contentValue any
	if err := json.Unmarshal(content, &contentValue); err != nil {
		t.Fatalf("content を比較用に解析できません: %v", err)
	}
	structuredJSON, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("structuredContent を JSON に変換できません: %v", err)
	}
	var structuredValue any
	if err := json.Unmarshal(structuredJSON, &structuredValue); err != nil {
		t.Fatalf("structuredContent を比較用に解析できません: %v", err)
	}
	if !reflect.DeepEqual(contentValue, structuredValue) {
		t.Fatalf("SOT-IF-007: content と structuredContent が一致しません")
	}

	var envelope struct {
		Item json.RawMessage `json:"item"`
	}
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatalf("SOT-IF-048: 公開結果を解析できません: %v", err)
	}
	assertSameJudicialJSONValue(t, envelope.Item, item)
}

func TestGetJudicialCaseToolOmitsAbsentOptionalDetails(t *testing.T) {
	t.Parallel()

	searchItem := mustJudicialSearchItem()
	details, err := model.NewJudicialDecisionDetails(
		model.JudicialDecisionDetailsValues{Summary: searchItem.Data()},
	)
	if err != nil {
		t.Fatalf("省略可能値のない裁判例詳細を作成できません: %v", err)
	}
	item, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.JudicialDecisionDetails]{
			Ref:        searchItem.Ref(),
			Provenance: searchItem.Provenance(),
			Data:       details,
		},
	)
	if err != nil {
		t.Fatalf("省略可能値のない情報源結果を作成できません: %v", err)
	}
	result, err := callGetJudicialCase(
		context.Background(),
		&recordingGetJudicialCasePort{item: item},
		canonicalJudicialCaseInput(),
	)
	if err != nil || result.IsError {
		t.Fatalf("get result = %#v, err = %v", result, err)
	}
	var payload struct {
		Item struct {
			Data map[string]json.RawMessage `json:"data"`
		} `json:"item"`
	}
	if err := json.Unmarshal(
		[]byte(result.Content[0].(*sdk.TextContent).Text),
		&payload,
	); err != nil {
		t.Fatalf("公開結果を解析できません: %v", err)
	}
	if len(payload.Item.Data) != 1 || payload.Item.Data["summary"] == nil {
		t.Fatalf("SOT-MODEL-009/021: data = %#v", payload.Item.Data)
	}
}

func canonicalJudicialCaseInput() json.RawMessage {
	return json.RawMessage(
		`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`,
	)
}

type recordingGetJudicialCasePort struct {
	item    model.SourcedResource[model.JudicialDecisionDetails]
	err     error
	request judicialdecisionread.Request
	calls   int
}

func (p *recordingGetJudicialCasePort) Read(
	_ context.Context,
	request judicialdecisionread.Request,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	p.calls++
	p.request = request
	return p.item, p.err
}

func mustJudicialDecisionDetailsResource() model.SourcedResource[model.JudicialDecisionDetails] {
	searchItem := mustJudicialSearchItem()
	reporterCitation := "民集78巻1号1頁"
	lowerCourtName := "東京高等裁判所"
	lowerCourtCaseNumber := "令和4年(行コ)第45号"
	holdingText := "主文のとおり判決する。"
	summaryText := "本件請求を棄却した事例。"
	referencedProvisionsText := "民法709条"
	lowerCourtDecisionDate, err := model.NewDate("2023-06-15")
	if err != nil {
		panic(err)
	}
	details, err := model.NewJudicialDecisionDetails(
		model.JudicialDecisionDetailsValues{
			Summary:                  searchItem.Data(),
			ReporterCitation:         &reporterCitation,
			LowerCourtName:           &lowerCourtName,
			LowerCourtCaseNumber:     &lowerCourtCaseNumber,
			LowerCourtDecisionDate:   &lowerCourtDecisionDate,
			HoldingText:              &holdingText,
			SummaryText:              &summaryText,
			ReferencedProvisionsText: &referencedProvisionsText,
		},
	)
	if err != nil {
		panic(err)
	}
	item, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.JudicialDecisionDetails]{
			Ref:        searchItem.Ref(),
			Provenance: searchItem.Provenance(),
			Data:       details,
		},
	)
	if err != nil {
		panic(err)
	}
	return item
}

func assertSameJudicialJSONValue(
	t *testing.T,
	gotJSON json.RawMessage,
	wantValue any,
) {
	t.Helper()

	wantJSON, err := json.Marshal(wantValue)
	if err != nil {
		t.Fatalf("比較対象を JSON に変換できません: %v", err)
	}
	var got, want any
	if err := json.Unmarshal(gotJSON, &got); err != nil {
		t.Fatalf("公開 JSON を解析できません: %v", err)
	}
	if err := json.Unmarshal(wantJSON, &want); err != nil {
		t.Fatalf("期待 JSON を解析できません: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"SOT-IF-048: 共通モデルを変更して公開した: got %#v, want %#v",
			got,
			want,
		)
	}
}
