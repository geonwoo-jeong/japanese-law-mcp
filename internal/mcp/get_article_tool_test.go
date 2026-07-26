package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/getarticle"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGetArticleToolReturnsOnlyPublicStructuredSuccess(t *testing.T) {
	t.Parallel()

	port := &recordingGetArticlePort{}
	result, err := callGetArticle(
		context.Background(),
		port,
		json.RawMessage(
			`{"lawId":" law-1 ","provision":"supplementary","article":"38_3","paragraph":2,"asOf":"2026-07-26"}`,
		),
	)
	if err != nil {
		t.Fatalf("callGetArticle() のエラー = %v", err)
	}
	if result.IsError {
		t.Fatalf("get_article がエラーを返しました: %#v", result)
	}
	if port.request.LawID() != "law-1" {
		t.Fatalf("lawId = %q", port.request.LawID())
	}
	location := port.request.Location()
	if location.Provision() != model.LawArticleProvisionSupplementary ||
		location.ArticleNumber() != "38_3" {
		t.Fatalf("location = %#v", location)
	}
	text := result.Content[0].(*sdk.TextContent).Text
	structured, ok := result.StructuredContent.(json.RawMessage)
	if !ok {
		t.Fatalf("structuredContent の型 = %T", result.StructuredContent)
	}
	if text != string(structured) {
		t.Fatal("content と structuredContent が一致しません")
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &object); err != nil {
		t.Fatalf("成功結果を解析できません: %v", err)
	}
	if len(object) != 3 ||
		object["format"] == nil ||
		object["content"] == nil ||
		object["citation"] == nil {
		t.Fatalf("公開項目 = %#v", object)
	}
	for _, forbidden := range []string{"law", "location", "ref", "provenance"} {
		if _, exists := object[forbidden]; exists {
			t.Fatalf("公開してはならない %s があります", forbidden)
		}
	}
}

func TestGetArticleToolDefaultsProvisionAndParagraph(t *testing.T) {
	t.Parallel()

	port := &recordingGetArticlePort{}
	result, err := callGetArticle(
		context.Background(),
		port,
		json.RawMessage(`{"lawId":"law-1","article":"1"}`),
	)
	if err != nil {
		t.Fatalf("callGetArticle() のエラー = %v", err)
	}
	if result.IsError {
		t.Fatalf("get_article がエラーを返しました: %#v", result)
	}
	if port.request.Location().Provision() != model.LawArticleProvisionMain {
		t.Fatalf("provision = %q", port.request.Location().Provision())
	}
	if _, exists := port.request.Location().ParagraphNumber(); exists {
		t.Fatal("省略した paragraph が設定されています")
	}
}

func TestGetArticleToolRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := map[string]json.RawMessage{
		"lawId の欠落":        json.RawMessage(`{"article":"1"}`),
		"lawId の null":     json.RawMessage(`{"lawId":null,"article":"1"}`),
		"article の欠落":      json.RawMessage(`{"lawId":"law-1"}`),
		"article の null":   json.RawMessage(`{"lawId":"law-1","article":null}`),
		"provision の null": json.RawMessage(`{"lawId":"law-1","article":"1","provision":null}`),
		"paragraph の null": json.RawMessage(`{"lawId":"law-1","article":"1","paragraph":null}`),
		"asOf の null":      json.RawMessage(`{"lawId":"law-1","article":"1","asOf":null}`),
		"未定義項目":            json.RawMessage(`{"lawId":"law-1","article":"1","other":true}`),
		"配列":               json.RawMessage(`[]`),
	}
	for name, arguments := range tests {
		name := name
		arguments := arguments
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := callGetArticle(
				context.Background(),
				&recordingGetArticlePort{},
				arguments,
			)
			if err != nil {
				t.Fatalf("callGetArticle() のエラー = %v", err)
			}
			assertGetArticleErrorCode(t, result, "invalid_argument")
		})
	}
}

func TestGetArticleToolMapsDomainAndSourceErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err  error
		code string
	}{
		"not found": {
			err:  lawarticleread.ErrNotFound,
			code: "not_found",
		},
		"ambiguous": {
			err:  lawarticleread.ErrAmbiguousLocation,
			code: "ambiguous_location",
		},
		"source": {
			err:  mustGetArticleSourceError(),
			code: "source_timeout",
		},
		"internal": {
			err:  errors.New("分類できない失敗"),
			code: "internal_error",
		},
	}
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := callGetArticle(
				context.Background(),
				&recordingGetArticlePort{err: test.err},
				json.RawMessage(`{"lawId":"law-1","article":"1"}`),
			)
			if err != nil {
				t.Fatalf("callGetArticle() のエラー = %v", err)
			}
			assertGetArticleErrorCode(t, result, test.code)
		})
	}
}

func assertGetArticleErrorCode(
	t *testing.T,
	result *sdk.CallToolResult,
	want string,
) {
	t.Helper()

	if !result.IsError {
		t.Fatal("IsError = false, want true")
	}
	if result.StructuredContent != nil {
		t.Fatal("エラー結果に structuredContent があります")
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

type recordingGetArticlePort struct {
	request getarticle.Request
	err     error
}

func (p *recordingGetArticlePort) Get(
	_ context.Context,
	request getarticle.Request,
) (model.LawArticleFragment, error) {
	p.request = request
	if p.err != nil {
		return model.LawArticleFragment{}, p.err
	}
	return mustMCPArticleFragment(request.Location()), nil
}

func mustMCPArticleFragment(
	location model.LawArticleLocation,
) model.LawArticleFragment {
	descriptor := mustGetArticleProviderDescriptor()
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
		Location:   "main:article=1",
		URL:        "https://example.com/law/law-1/rev-1",
	})
	if err != nil {
		panic(err)
	}
	fragment, err := model.NewLawArticleFragment(model.LawArticleFragmentValues{
		Law:      summary,
		Location: location,
		Format:   model.LawArticleFormatXML,
		Content:  "<Article Num=\"1\">本文</Article>",
		Citation: citation,
	})
	if err != nil {
		panic(err)
	}
	return fragment
}

func mustGetArticleSourceError() error {
	provider := mustGetArticleProviderDescriptor()
	return mustSourceError(model.SourceErrorValues{
		Code:       model.SourceErrorCodeSourceTimeout,
		Provider:   provider,
		Capability: provider.Capabilities()[0],
		Operation:  testOperation("get_article"),
	})
}

func mustGetArticleProviderDescriptor() model.ProviderDescriptor {
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
		ID:           lawarticleread.CapabilityID,
		MajorVersion: lawarticleread.MajorVersion,
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
