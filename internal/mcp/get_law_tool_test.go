package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/getlaw"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGetLawToolReturnsStructuredSuccess(t *testing.T) {
	t.Parallel()

	result, err := callGetLaw(
		context.Background(),
		stubGetLawPort{},
		json.RawMessage(`{"lawId":" law-1 ","asOf":"2026-07-26"}`),
	)
	if err != nil {
		t.Fatalf("callGetLaw() のエラー = %v", err)
	}
	if result.IsError {
		t.Fatalf("get_law がエラーを返しました: %#v", result)
	}
	text := result.Content[0].(*sdk.TextContent).Text
	structured, ok := result.StructuredContent.(json.RawMessage)
	if !ok {
		t.Fatalf("structuredContent の型 = %T", result.StructuredContent)
	}
	if text != string(structured) {
		t.Fatalf("content と structuredContent が一致しません")
	}
	var payload struct {
		Law struct {
			LawID string `json:"lawId"`
		} `json:"law"`
		AsOf    string `json:"asOf"`
		Format  string `json:"format"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("成功結果を解析できません: %v", err)
	}
	if payload.Law.LawID != "law-1" ||
		payload.AsOf != "2026-07-26" ||
		payload.Format != "xml" ||
		payload.Content != "<Law>本文</Law>" {
		t.Fatalf("成功結果 = %#v", payload)
	}
}

func TestGetLawToolRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := map[string]json.RawMessage{
		"lawId の欠落":    json.RawMessage(`{}`),
		"lawId の null": json.RawMessage(`{"lawId":null}`),
		"asOf の null":  json.RawMessage(`{"lawId":"law-1","asOf":null}`),
		"未定義項目":        json.RawMessage(`{"lawId":"law-1","other":true}`),
		"配列":           json.RawMessage(`[]`),
	}
	for name, arguments := range tests {
		name := name
		arguments := arguments
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := callGetLaw(context.Background(), stubGetLawPort{}, arguments)
			if err != nil {
				t.Fatalf("callGetLaw() のエラー = %v", err)
			}
			assertGetLawErrorCode(t, result, "invalid_argument")
		})
	}
}

func TestGetLawToolReturnsNotFound(t *testing.T) {
	t.Parallel()

	result, err := callGetLaw(
		context.Background(),
		notFoundGetLawPort{},
		json.RawMessage(`{"lawId":"law-1"}`),
	)
	if err != nil {
		t.Fatalf("callGetLaw() のエラー = %v", err)
	}
	assertGetLawErrorCode(t, result, "not_found")
}

func TestGetLawToolPreservesSourceErrorCode(t *testing.T) {
	t.Parallel()

	result, err := callGetLaw(
		context.Background(),
		sourceErrorGetLawPort{},
		json.RawMessage(`{"lawId":"law-1"}`),
	)
	if err != nil {
		t.Fatalf("callGetLaw() のエラー = %v", err)
	}
	assertGetLawErrorCode(t, result, "source_timeout")
}

func assertGetLawErrorCode(
	t *testing.T,
	result *sdk.CallToolResult,
	want string,
) {
	t.Helper()

	if !result.IsError {
		t.Fatalf("IsError = false, want true")
	}
	if result.StructuredContent != nil {
		t.Fatalf("エラー結果に structuredContent があります")
	}
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(
		[]byte(result.Content[0].(*sdk.TextContent).Text),
		&payload,
	); err != nil {
		t.Fatalf("ErrorResult を解析できません: %v", err)
	}
	if payload.Code != want {
		t.Fatalf("code = %q, want %q", payload.Code, want)
	}
}

type stubGetLawPort struct{}

func (stubGetLawPort) Get(
	_ context.Context,
	request getlaw.Request,
) (model.LawDocument, error) {
	asOf, exists := request.AsOf()
	var asOfPointer *model.Date
	if exists {
		asOfPointer = &asOf
	}
	return mustMCPGetLawDocument(asOfPointer), nil
}

type notFoundGetLawPort struct{}

func (notFoundGetLawPort) Get(
	context.Context,
	getlaw.Request,
) (model.LawDocument, error) {
	return model.LawDocument{}, lawdocumentread.ErrNotFound
}

type sourceErrorGetLawPort struct{}

func (sourceErrorGetLawPort) Get(
	context.Context,
	getlaw.Request,
) (model.LawDocument, error) {
	provider := mustGetLawProviderDescriptor()
	return model.LawDocument{}, mustSourceError(model.SourceErrorValues{
		Code:       model.SourceErrorCodeSourceTimeout,
		Provider:   provider,
		Capability: provider.Capabilities()[0],
		Operation:  testOperation("get_law"),
	})
}

func mustMCPGetLawDocument(asOf *model.Date) model.LawDocument {
	descriptor := mustGetLawProviderDescriptor()
	source, err := model.NewLegalSource(descriptor.Source())
	if err != nil {
		panic(err)
	}
	summary, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-1",
		RevisionID: "law-1_rev-1",
		Title:      "テスト法",
		Source:     source,
	})
	if err != nil {
		panic(err)
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     source,
		LawID:      "law-1",
		RevisionID: "law-1_rev-1",
		URL:        "https://example.com/law/law-1/rev-1",
	})
	if err != nil {
		panic(err)
	}
	document, err := model.NewLawDocument(model.LawDocumentValues{
		Law:      summary,
		AsOf:     asOf,
		Content:  "<Law>本文</Law>",
		Citation: citation,
	})
	if err != nil {
		panic(err)
	}
	return document
}

func mustGetLawProviderDescriptor() model.ProviderDescriptor {
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API Version 2",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/redoc/",
	})
	if err != nil {
		panic(err)
	}
	verifiedAt, err := model.NewDate("2026-07-26")
	if err != nil {
		panic(err)
	}
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           lawdocumentread.CapabilityID,
		MajorVersion: lawdocumentread.MajorVersion,
		Level:        model.CapabilityLevelCore,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		panic(err)
	}
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             "e-gov-law-api-v2",
		Source:                 source,
		AdapterContractVersion: "1.0.0",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeAPI,
		Capabilities:           []model.ProviderCapability{capability},
	})
	if err != nil {
		panic(err)
	}
	return descriptor
}
