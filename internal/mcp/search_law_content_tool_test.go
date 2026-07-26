package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchlawcontent"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSearchLawContentToolReturnsStructuredSuccess(t *testing.T) {
	t.Parallel()

	result, err := callSearchLawContent(
		context.Background(),
		stubSearchLawContentPort{},
		json.RawMessage(`{"query":"情報 公開"}`),
	)
	if err != nil {
		t.Fatalf("callSearchLawContent() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, want false: %#v", result)
	}
	var payload searchLawContentOutput
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &payload); err != nil {
		t.Fatalf("content JSON error = %v", err)
	}
	if payload.TotalCount != 1 || len(payload.Items) != 1 {
		t.Fatalf("payload = %#v", payload)
	}
	if payload.Items[0].Law.LawID != "law-1" ||
		payload.Items[0].Location != "第一条" ||
		payload.Items[0].Text != "情報公開に関する規定" ||
		payload.Items[0].Citation.URL != "https://example.test/laws/law-1#第一条" {
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
}

func TestSearchLawContentToolReturnsErrorResult(t *testing.T) {
	t.Parallel()

	result, err := callSearchLawContent(
		context.Background(),
		errorSearchLawContentPort{},
		json.RawMessage(`{"query":"情報 公開"}`),
	)
	if err != nil {
		t.Fatalf("callSearchLawContent() error = %v", err)
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

func TestSearchLawContentToolRejectsInvalidInputAsToolError(t *testing.T) {
	t.Parallel()

	tests := map[string]json.RawMessage{
		"query の欠落":     json.RawMessage(`{}`),
		"query の null":  json.RawMessage(`{"query":null}`),
		"asOf の null":   json.RawMessage(`{"query":"情報 公開","asOf":null}`),
		"limit の null":  json.RawMessage(`{"query":"情報 公開","limit":null}`),
		"offset の null": json.RawMessage(`{"query":"情報 公開","offset":null}`),
		"未定義の入力項目":      json.RawMessage(`{"query":"情報 公開","unknown":true}`),
		"配列形式":          json.RawMessage(`[]`),
	}
	for name, arguments := range tests {
		name := name
		arguments := arguments
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := callSearchLawContent(
				context.Background(),
				stubSearchLawContentPort{},
				arguments,
			)
			if err != nil {
				t.Fatalf("callSearchLawContent() のエラー = %v", err)
			}
			if !result.IsError || result.StructuredContent != nil {
				t.Fatalf("入力エラー結果 = %#v", result)
			}
			var payload struct {
				Code      string `json:"code"`
				Retryable bool   `json:"retryable"`
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

type stubSearchLawContentPort struct{}

func (stubSearchLawContentPort) Search(
	context.Context,
	searchlawcontent.Request,
) (model.LawContentSearchResult, error) {
	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API Version 2",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/redoc/",
	})
	if err != nil {
		return model.LawContentSearchResult{}, err
	}
	source, err := model.NewLegalSource(informationSource)
	if err != nil {
		return model.LawContentSearchResult{}, err
	}
	law, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-1",
		RevisionID: "rev-1",
		Title:      "情報公開法",
		Source:     source,
	})
	if err != nil {
		return model.LawContentSearchResult{}, err
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     source,
		LawID:      "law-1",
		RevisionID: "rev-1",
		Location:   "第一条",
		URL:        "https://example.test/laws/law-1#第一条",
	})
	if err != nil {
		return model.LawContentSearchResult{}, err
	}
	match, err := model.NewLawContentMatch(model.LawContentMatchValues{
		Law:      law,
		Location: "第一条",
		Text:     "情報公開に関する規定",
		Citation: citation,
	})
	if err != nil {
		return model.LawContentSearchResult{}, err
	}
	return model.NewLawContentSearchResult(model.LawContentSearchResultValues{
		TotalCount: 1,
		Items:      []model.LawContentMatch{match},
	})
}

type errorSearchLawContentPort struct{}

func (errorSearchLawContentPort) Search(
	context.Context,
	searchlawcontent.Request,
) (model.LawContentSearchResult, error) {
	provider := mustSearchLawsTestProviderDescriptor()
	return model.LawContentSearchResult{}, mustSourceError(
		model.SourceErrorValues{
			Code:       model.SourceErrorCodeSourceBusy,
			Provider:   provider,
			Capability: provider.Capabilities()[0],
			Operation:  testOperation("search_law_content"),
		},
	)
}
