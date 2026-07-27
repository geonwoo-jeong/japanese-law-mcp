package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSearchJudicialCasesToolReturnsStructuredSuccess(t *testing.T) {
	t.Parallel()

	result, err := callSearchJudicialCases(
		context.Background(),
		stubSearchJudicialCasesPort{},
		json.RawMessage(`{"query":"民法"}`),
	)
	if err != nil {
		t.Fatalf("callSearchJudicialCases() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, want false: %#v", result)
	}
	var payload searchJudicialCasesOutput
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &payload); err != nil {
		t.Fatalf("content JSON error = %v", err)
	}
	if payload.CoverageNotice != judicialCasesCoverageNotice {
		t.Fatalf("coverageNotice = %q", payload.CoverageNotice)
	}
	if len(payload.Items) != 1 || payload.Page.ReturnedCount != 1 {
		t.Fatalf("payload = %#v", payload)
	}
	if payload.Items[0].Ref.ProviderID != "courts-hanrei-html" ||
		payload.Items[0].Ref.Key.ResourceType != "judicial-decision" ||
		payload.Items[0].Data.DecisionID != "95570" ||
		payload.Items[0].Data.DetailURL != "https://www.courts.go.jp/app/hanrei_jp/detail2?id=95570" {
		t.Fatalf("payload = %#v", payload)
	}
	if len(payload.Items[0].Provenance) != 1 ||
		payload.Items[0].Provenance[0].ResourceKey.ResourceID != payload.Items[0].Ref.Key.ResourceID {
		t.Fatalf("payload = %#v", payload)
	}
	contentObject := map[string]any{}
	if err := json.Unmarshal(
		[]byte(result.Content[0].(*sdk.TextContent).Text),
		&contentObject,
	); err != nil {
		t.Fatalf("content を比較用に解析できません: %v", err)
	}
	structuredJSON, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("structuredContent を JSON に変換できません: %v", err)
	}
	structuredObject := map[string]any{}
	if err := json.Unmarshal(structuredJSON, &structuredObject); err != nil {
		t.Fatalf("structuredContent を比較用に解析できません: %v", err)
	}
	if !mapsEqual(contentObject, structuredObject) {
		t.Fatalf(
			"content と structuredContent が一致しません: %#v != %#v",
			contentObject,
			structuredObject,
		)
	}

	var envelope struct {
		Items json.RawMessage `json:"items"`
		Page  json.RawMessage `json:"page"`
	}
	if err := json.Unmarshal(
		[]byte(result.Content[0].(*sdk.TextContent).Text),
		&envelope,
	); err != nil {
		t.Fatalf("SOT-IF-047: 公開結果を解析できません: %v", err)
	}
	wantPage := mustJudicialSearchPage()
	assertSameJSONValue(t, envelope.Items, wantPage.Items())
	assertSameJSONValue(t, envelope.Page, wantPage.Page())
}

func TestSearchJudicialCasesToolReturnsEmptySuccess(t *testing.T) {
	t.Parallel()

	result, err := callSearchJudicialCases(
		context.Background(),
		emptySearchJudicialCasesPort{},
		json.RawMessage(`{"query":"不存在"}`),
	)
	if err != nil {
		t.Fatalf("callSearchJudicialCases() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, want false: %#v", result)
	}
	var payload searchJudicialCasesOutput
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &payload); err != nil {
		t.Fatalf("content JSON error = %v", err)
	}
	if payload.Items == nil || len(payload.Items) != 0 || payload.Page.ReturnedCount != 0 {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestSearchJudicialCasesToolReturnsErrorResult(t *testing.T) {
	t.Parallel()

	result, err := callSearchJudicialCases(
		context.Background(),
		errorSearchJudicialCasesPort{},
		json.RawMessage(`{"query":"民法"}`),
	)
	if err != nil {
		t.Fatalf("callSearchJudicialCases() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, want true: %#v", result)
	}
	var payload struct {
		Code      string `json:"code"`
		Retryable bool   `json:"retryable"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &payload); err != nil {
		t.Fatalf("error JSON error = %v", err)
	}
	if payload.Code != "source_busy" || !payload.Retryable {
		t.Fatalf("payload = %#v", payload)
	}
	if result.StructuredContent != nil {
		t.Fatalf("エラー結果に structuredContent があります: %#v", result.StructuredContent)
	}
}

func TestSearchJudicialCasesToolRejectsInvalidInputAsToolError(t *testing.T) {
	t.Parallel()

	tests := map[string]json.RawMessage{
		"query の欠落":                json.RawMessage(`{}`),
		"query の null":             json.RawMessage(`{"query":null}`),
		"limit の null":             json.RawMessage(`{"query":"民法","limit":null}`),
		"continuationToken の null": json.RawMessage(`{"query":"民法","continuationToken":null}`),
		"未定義の入力項目":                 json.RawMessage(`{"query":"民法","unknown":true}`),
		"配列形式":                     json.RawMessage(`[]`),
	}
	for name, arguments := range tests {
		name := name
		arguments := arguments
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := callSearchJudicialCases(
				context.Background(),
				stubSearchJudicialCasesPort{},
				arguments,
			)
			if err != nil {
				t.Fatalf("callSearchJudicialCases() のエラー = %v", err)
			}
			if !result.IsError || result.StructuredContent != nil {
				t.Fatalf("入力エラー結果 = %#v", result)
			}
			var payload struct {
				Code      string         `json:"code"`
				Retryable bool           `json:"retryable"`
				Details   map[string]any `json:"details"`
			}
			if err := json.Unmarshal(
				[]byte(result.Content[0].(*sdk.TextContent).Text),
				&payload,
			); err != nil {
				t.Fatalf("ErrorResult を解析できません: %v", err)
			}
			if payload.Code != "invalid_argument" || payload.Retryable {
				t.Fatalf("ErrorResult = %#v", payload)
			}
		})
	}
}

type stubSearchJudicialCasesPort struct{}

func (stubSearchJudicialCasesPort) Search(
	context.Context,
	judicialdecisionsearch.Request,
) (judicialdecisionsearch.Page, error) {
	return mustJudicialSearchPage(), nil
}

type emptySearchJudicialCasesPort struct{}

func (emptySearchJudicialCasesPort) Search(
	context.Context,
	judicialdecisionsearch.Request,
) (judicialdecisionsearch.Page, error) {
	page, err := judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{
			Page: mustJudicialSourcePage(0, nil, "", ""),
		},
	)
	if err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	return page, nil
}

type errorSearchJudicialCasesPort struct{}

func (errorSearchJudicialCasesPort) Search(
	context.Context,
	judicialdecisionsearch.Request,
) (judicialdecisionsearch.Page, error) {
	provider := hanreiTestProviderDescriptor()
	return judicialdecisionsearch.Page{}, mustSourceError(
		model.SourceErrorValues{
			Code:       model.SourceErrorCodeSourceBusy,
			Provider:   provider,
			Capability: provider.Capabilities()[1],
			Operation:  hanreiTestOperation("search_html"),
		},
	)
}

type hanreiTestOperation string

func (o hanreiTestOperation) SourceOperationProviderID() string { return "courts-hanrei-html" }
func (o hanreiTestOperation) SourceOperationName() string       { return string(o) }
func (o hanreiTestOperation) ValidateSourceOperation() error    { return nil }

func mustJudicialSearchPage() judicialdecisionsearch.Page {
	item := mustJudicialSearchItem()
	totalCount := 1
	page, err := judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{
			Items: []model.SourcedResource[model.JudicialDecisionSummary]{item},
			Page: mustJudicialSourcePage(
				1,
				&totalCount,
				"",
				model.TotalRelationExact,
			),
		},
	)
	if err != nil {
		panic(err)
	}
	return page
}

func mustJudicialSearchItem() model.SourcedResource[model.JudicialDecisionSummary] {
	source := mustCourtsInformationSource()
	key := mustJudicialResourceKey()
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		panic(err)
	}
	document, err := model.NewJudicialDocumentLink(model.JudicialDocumentLinkValues{
		Kind:      model.JudicialDocumentKindFullText,
		Label:     "全文",
		MediaType: model.JudicialDocumentMediaTypePDF,
		URL:       "https://www.courts.go.jp/app/files/hanrei_jp/570/095570_hanrei.pdf",
	})
	if err != nil {
		panic(err)
	}
	date, err := model.NewDate("2024-01-31")
	if err != nil {
		panic(err)
	}
	caseName := "損害賠償請求事件"
	branchName := "本庁"
	divisionName := "第一小法廷"
	decisionType := "判決"
	outcome := "棄却"
	summary, err := model.NewJudicialDecisionSummary(model.JudicialDecisionSummaryValues{
		DecisionID:          "95570",
		PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
		SourceCategoryLabel: "最高裁判例",
		CaseNumber:          "令和5年(行ヒ)第123号",
		CaseName:            &caseName,
		DecisionDate:        date,
		CourtName:           "最高裁判所第一小法廷",
		BranchName:          &branchName,
		DivisionName:        &divisionName,
		DecisionType:        &decisionType,
		Outcome:             &outcome,
		DetailURL:           "https://www.courts.go.jp/app/hanrei_jp/detail2?id=95570",
		Documents:           []model.JudicialDocumentLink{document},
		Source:              source,
	})
	if err != nil {
		panic(err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:          source,
		ResourceKey:     key,
		URL:             "https://www.courts.go.jp/app/hanrei_jp/detail2?id=95570",
		RetrievedAt:     time.Date(2026, time.July, 27, 10, 30, 0, 123456789, time.UTC),
		SourceUpdatedAt: "2026-07-26",
		MediaType:       "text/html",
		Location:        "検索結果 1 件目",
		Transformation:  model.ProvenanceTransformationNormalized,
		MethodID:        "SOT-IF-044",
		InputKeys:       []model.SourceResourceKey{},
		ContentDigest:   "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	})
	if err != nil {
		panic(err)
	}
	resource, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.JudicialDecisionSummary]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       summary,
		},
	)
	if err != nil {
		panic(err)
	}
	return resource
}

func mustJudicialResourceKey() model.SourceResourceKey {
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   "detail2?id=95570",
	})
	if err != nil {
		panic(err)
	}
	return key
}

func mustJudicialSourcePage(
	returnedCount int,
	totalCount *int,
	nextToken string,
	totalRelation model.TotalRelation,
) model.SourcePage {
	page, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: returnedCount,
		TotalCount:    totalCount,
		NextToken:     nextToken,
		TotalRelation: totalRelation,
	})
	if err != nil {
		panic(err)
	}
	return page
}

func mustCourtsInformationSource() model.InformationSource {
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "courts-hanrei",
		Name:       "裁判所 裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/",
	})
	if err != nil {
		panic(err)
	}
	return source
}

func hanreiTestProviderDescriptor() model.ProviderDescriptor {
	source := mustCourtsInformationSource()
	verifiedAt, err := model.NewDate("2026-07-27")
	if err != nil {
		panic(err)
	}
	readCapability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           "judicial-decision.read",
		MajorVersion: 1,
		Level:        model.CapabilityLevelExtended,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		panic(err)
	}
	searchCapability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           "judicial-decision.search",
		MajorVersion: 1,
		Level:        model.CapabilityLevelExtended,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		panic(err)
	}
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             "courts-hanrei-html",
		Source:                 source,
		AdapterContractVersion: "1.0.0",
		UpstreamSpecVersion:    "2026-07-27",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeHTML,
		CredentialRequired:     false,
		Capabilities: []model.ProviderCapability{
			readCapability,
			searchCapability,
		},
	})
	if err != nil {
		panic(err)
	}
	return descriptor
}

func assertSameJSONValue(t *testing.T, gotJSON json.RawMessage, wantValue any) {
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
		t.Fatalf("SOT-IF-047: 共通モデルを変更して公開した: got %#v, want %#v", got, want)
	}
}
