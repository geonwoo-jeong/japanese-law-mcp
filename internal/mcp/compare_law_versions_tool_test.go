package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/comparelawversions"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawversioncompare"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCompareLawVersionsToolReturnsNormalizedComparison(t *testing.T) {
	t.Parallel()

	port := &recordingCompareLawVersionsPort{
		result: mustMCPCompareLawVersionsResult(t),
	}
	result, err := callCompareLawVersions(
		context.Background(),
		port,
		json.RawMessage(`{
			"lawId":" law-1 ",
			"before":{"revisionId":"rev-1"},
			"after":{"asOf":"2024-04-01"}
		}`),
	)
	if err != nil {
		t.Fatalf("callCompareLawVersions() のエラー = %v", err)
	}
	if result.IsError || port.calls != 1 || port.request.LawID() != "law-1" {
		t.Fatalf("MCP 結果又は呼出し = %#v / %#v", result, port)
	}
	if revisionID, exists := port.request.Before().RevisionID(); !exists || revisionID != "rev-1" {
		t.Fatalf("before.revisionId = %q, %t", revisionID, exists)
	}
	if asOf, exists := port.request.After().AsOf(); !exists || asOf.String() != "2024-04-01" {
		t.Fatalf("after.asOf = %v, %t", asOf, exists)
	}

	content := result.Content[0].(*sdk.TextContent).Text
	var payload map[string]any
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("公開 JSON を解析できません: %v", err)
	}
	if payload["lawId"] != "law-1" ||
		payload["scope"] != string(model.LawVersionComparisonScopeMainAndOriginalSupplementaryArticles) ||
		payload["totalCount"] != float64(1) {
		t.Fatalf("公開 payload = %#v", payload)
	}
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("差分一覧 = %#v", payload["items"])
	}
	item := items[0].(map[string]any)
	if item["changeKind"] != "modified" {
		t.Fatalf("差分 item = %#v", item)
	}
	beforeItem := item["before"].(map[string]any)
	location := beforeItem["location"].(map[string]any)
	if location["articleNumber"] != "1" {
		t.Fatalf("差分 location = %#v", location)
	}
	if reasons, exists := item["changeReasons"].([]any); !exists || len(reasons) != 1 || reasons[0] != "text" {
		t.Fatalf("changeReasons = %#v", item["changeReasons"])
	}
	structured, ok := result.StructuredContent.(json.RawMessage)
	if !ok || string(structured) != content {
		t.Fatalf("structuredContent と text が一致しません: %#v", result.StructuredContent)
	}
}

func TestCompareLawVersionsToolRejectsInvalidInputWithoutCallingPort(t *testing.T) {
	t.Parallel()

	tests := map[string]json.RawMessage{
		"欠落":          nil,
		"object 空":    json.RawMessage(`{}`),
		"lawId 欠落":    json.RawMessage(`{"before":{"revisionId":"rev-1"},"after":{"revisionId":"rev-2"}}`),
		"selector 欠落": json.RawMessage(`{"lawId":"law-1","before":{"revisionId":"rev-1"}}`),
		"未知項目":        json.RawMessage(`{"lawId":"law-1","before":{"revisionId":"rev-1"},"after":{"revisionId":"rev-2"},"other":true}`),
		"selector 空":  json.RawMessage(`{"lawId":"law-1","before":{},"after":{"revisionId":"rev-2"}}`),
		"配列":          json.RawMessage(`[]`),
	}
	for name, arguments := range tests {
		name, arguments := name, arguments
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			port := &recordingCompareLawVersionsPort{}
			result, err := callCompareLawVersions(context.Background(), port, arguments)
			if err != nil {
				t.Fatalf("callCompareLawVersions() のエラー = %v", err)
			}
			assertCompareLawVersionsErrorCode(t, result, "invalid_argument")
			if port.calls != 0 {
				t.Fatalf("不正入力で port を %d 回呼び出しました", port.calls)
			}
		})
	}
}

func TestCompareLawVersionsToolMapsApplicationErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err  error
		code string
	}{
		"not found": {
			err:  lawversioncompare.ErrNotFound,
			code: "not_found",
		},
		"invalid source response": {
			err:  comparelawversions.ErrInvalidSourceResponse,
			code: "invalid_source_response",
		},
		"source unavailable": {
			err: mustSourceError(model.SourceErrorValues{
				Code:       model.SourceErrorCodeSourceUnavailable,
				Provider:   mustSearchLawsTestProviderDescriptor(),
				Capability: mustSearchLawsTestProviderDescriptor().Capabilities()[0],
				Operation:  testOperation("compare_law_versions"),
			}),
			code: "source_unavailable",
		},
		"unknown": {
			err:  errors.New("unknown"),
			code: "internal_error",
		},
	}
	for name, fixture := range tests {
		name, fixture := name, fixture
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := callCompareLawVersions(
				context.Background(),
				&recordingCompareLawVersionsPort{err: fixture.err},
				json.RawMessage(`{"lawId":"law-1","before":{"revisionId":"rev-1"},"after":{"revisionId":"rev-2"}}`),
			)
			if err != nil {
				t.Fatalf("callCompareLawVersions() のエラー = %v", err)
			}
			assertCompareLawVersionsErrorCode(t, result, fixture.code)
		})
	}
}

func TestCompareLawVersionsToolFailsClosedForNilPortVariants(t *testing.T) {
	t.Parallel()

	var typedNil *recordingCompareLawVersionsPort
	for name, port := range map[string]comparelawversions.Port{
		"nil":       nil,
		"typed nil": typedNil,
	} {
		name, port := name, port
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := callCompareLawVersions(
				context.Background(),
				port,
				json.RawMessage(`{"lawId":"law-1","before":{"revisionId":"rev-1"},"after":{"revisionId":"rev-2"}}`),
			)
			if err != nil {
				t.Fatalf("callCompareLawVersions() のエラー = %v", err)
			}
			assertCompareLawVersionsErrorCode(t, result, "internal_error")
		})
	}
}

type recordingCompareLawVersionsPort struct {
	result  model.LawVersionComparison
	err     error
	calls   int
	request comparelawversions.Request
}

func (p *recordingCompareLawVersionsPort) Compare(
	_ context.Context,
	request comparelawversions.Request,
) (model.LawVersionComparison, error) {
	p.calls++
	p.request = request
	return p.result, p.err
}

func mustMCPCompareLawVersionsResult(t *testing.T) model.LawVersionComparison {
	t.Helper()

	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API Version 2",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/redoc/",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できません: %v", err)
	}
	source, err := model.NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("LegalSource を作成できません: %v", err)
	}
	beforeLaw, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-1",
		RevisionID: "rev-1",
		Title:      "試験法",
		Source:     source,
	})
	if err != nil {
		t.Fatalf("beforeLaw を作成できません: %v", err)
	}
	afterLaw, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-1",
		RevisionID: "rev-2",
		Title:      "試験法",
		Source:     source,
	})
	if err != nil {
		t.Fatalf("afterLaw を作成できません: %v", err)
	}
	beforeCitation, err := model.NewCitation(model.CitationValues{
		Source:     source,
		LawID:      "law-1",
		RevisionID: "rev-1",
		Location:   "第1条",
		URL:        "https://laws.e-gov.go.jp/document?lawid=law-1&rev=rev-1",
	})
	if err != nil {
		t.Fatalf("beforeCitation を作成できません: %v", err)
	}
	afterCitation, err := model.NewCitation(model.CitationValues{
		Source:     source,
		LawID:      "law-1",
		RevisionID: "rev-2",
		Location:   "第1条",
		URL:        "https://laws.e-gov.go.jp/document?lawid=law-1&rev=rev-2",
	})
	if err != nil {
		t.Fatalf("afterCitation を作成できません: %v", err)
	}
	change, err := model.NewLawVersionChange(model.LawVersionChangeValues{
		ChangeKind:    model.LawVersionChangeKindModified,
		ChangeReasons: []model.LawVersionChangeReason{model.LawVersionChangeReasonText},
		Before: lawVersionArticlePointer(t, model.LawVersionArticleValues{
			Location: mustLawVersionArticleLocation(t, model.LawVersionArticleLocationValues{
				Provision:     model.LawArticleProvisionMain,
				ArticleNumber: "1",
			}),
			Text:                 "旧条文",
			Citation:             beforeCitation,
			DocumentOrder:        1,
			StructureFingerprint: "article-structure",
		}),
		After: lawVersionArticlePointer(t, model.LawVersionArticleValues{
			Location: mustLawVersionArticleLocation(t, model.LawVersionArticleLocationValues{
				Provision:     model.LawArticleProvisionMain,
				ArticleNumber: "1",
			}),
			Text:                 "新条文",
			Citation:             afterCitation,
			DocumentOrder:        1,
			StructureFingerprint: "article-structure",
		}),
	})
	if err != nil {
		t.Fatalf("LawVersionChange を作成できません: %v", err)
	}
	beforeSnapshot, err := model.NewLawVersionSnapshot(model.LawVersionSnapshotValues{
		Law:      beforeLaw,
		Citation: beforeCitation,
	})
	if err != nil {
		t.Fatalf("before snapshot を作成できません: %v", err)
	}
	afterAsOf, err := model.NewDate("2024-04-01")
	if err != nil {
		t.Fatalf("after asOf を作成できません: %v", err)
	}
	afterSnapshot, err := model.NewLawVersionSnapshot(model.LawVersionSnapshotValues{
		Law:      afterLaw,
		AsOf:     &afterAsOf,
		Citation: afterCitation,
	})
	if err != nil {
		t.Fatalf("after snapshot を作成できません: %v", err)
	}
	result, err := model.NewLawVersionComparison(model.LawVersionComparisonValues{
		LawID:              "law-1",
		Scope:              model.LawVersionComparisonScopeMainAndOriginalSupplementaryArticles,
		Before:             beforeSnapshot,
		After:              afterSnapshot,
		BeforeArticleCount: 1,
		AfterArticleCount:  1,
		AddedCount:         0,
		RemovedCount:       0,
		ModifiedCount:      1,
		UnchangedCount:     0,
		TotalCount:         1,
		Items:              []model.LawVersionChange{change},
	})
	if err != nil {
		t.Fatalf("LawVersionComparison を作成できません: %v", err)
	}
	return result
}

func mustLawVersionArticleLocation(
	t *testing.T,
	values model.LawVersionArticleLocationValues,
) model.LawVersionArticleLocation {
	t.Helper()

	location, err := model.NewLawVersionArticleLocation(values)
	if err != nil {
		t.Fatalf("LawVersionArticleLocation を作成できません: %v", err)
	}
	return location
}

func lawVersionArticlePointer(
	t *testing.T,
	values model.LawVersionArticleValues,
) *model.LawVersionArticle {
	t.Helper()

	article, err := model.NewLawVersionArticle(values)
	if err != nil {
		t.Fatalf("LawVersionArticle を作成できません: %v", err)
	}
	return &article
}

func assertCompareLawVersionsErrorCode(
	t *testing.T,
	result *sdk.CallToolResult,
	want string,
) {
	t.Helper()

	if result == nil || !result.IsError || len(result.Content) != 1 {
		t.Fatalf("error 結果が不正です: %#v", result)
	}
	content, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("error content の型 = %T", result.Content[0])
	}
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(content.Text), &payload); err != nil {
		t.Fatalf("ErrorResult を解析できません: %v", err)
	}
	if payload.Code != want {
		t.Fatalf("error code = %q, want %q", payload.Code, want)
	}
}
