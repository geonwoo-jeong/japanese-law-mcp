package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/getarticle"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type getArticleInputSchema struct {
	LawID     string `json:"lawId"`
	Provision string `json:"provision,omitempty"`
	Article   string `json:"article"`
	Paragraph int    `json:"paragraph,omitempty"`
	AsOf      string `json:"asOf,omitempty"`
}

type getArticleOutput struct {
	Format   string               `json:"format"`
	Content  string               `json:"content"`
	Citation getLawOutputCitation `json:"citation"`
}

func addGetArticleTool(server *sdk.Server, getter getarticle.Port) {
	server.AddTool(&sdk.Tool{
		Name:         "get_article",
		Description:  "法令識別子と条文位置を使い、e-Gov で確認できる条文を XML で取得します。",
		InputSchema:  mustSchemaFor[getArticleInputSchema](),
		OutputSchema: mustSchemaFor[getArticleOutput](),
	}, func(ctx context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return callGetArticle(ctx, getter, request.Params.Arguments)
	})
}

func callGetArticle(
	ctx context.Context,
	getter getarticle.Port,
	arguments json.RawMessage,
) (*sdk.CallToolResult, error) {
	if getter == nil {
		return errorToolResult(newInternalErrorResult())
	}
	request, err := decodeGetArticleInput(arguments)
	if err != nil {
		return errorToolResult(newInvalidArgumentResult(err))
	}
	fragment, err := getter.Get(ctx, request)
	if err != nil {
		return errorToolResult(mapGetArticleError(err))
	}
	return successGetArticleToolResult(mapGetArticleOutput(fragment))
}

func decodeGetArticleInput(arguments json.RawMessage) (getarticle.Request, error) {
	if len(arguments) == 0 {
		return getarticle.Request{}, fmt.Errorf("lawId と article は必須です")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(arguments, &fields); err != nil {
		return getarticle.Request{}, fmt.Errorf("入力は JSON object でなければなりません")
	}
	for key := range fields {
		switch key {
		case "lawId", "provision", "article", "paragraph", "asOf":
		default:
			return getarticle.Request{}, fmt.Errorf("定義していない入力項目は使用できません")
		}
	}

	var values getarticle.RequestValues
	lawIDRaw, exists := fields["lawId"]
	if !exists || isJSONNull(lawIDRaw) {
		return getarticle.Request{}, fmt.Errorf("lawId は必須です")
	}
	if err := json.Unmarshal(lawIDRaw, &values.LawID); err != nil {
		return getarticle.Request{}, fmt.Errorf("lawId は文字列でなければなりません")
	}
	articleRaw, exists := fields["article"]
	if !exists || isJSONNull(articleRaw) {
		return getarticle.Request{}, fmt.Errorf("article は必須です")
	}
	if err := json.Unmarshal(articleRaw, &values.Article); err != nil {
		return getarticle.Request{}, fmt.Errorf("article は文字列でなければなりません")
	}
	if raw, exists := fields["provision"]; exists {
		if isJSONNull(raw) {
			return getarticle.Request{}, fmt.Errorf("provision に null は使用できません")
		}
		var provision model.LawArticleProvision
		if err := json.Unmarshal(raw, &provision); err != nil {
			return getarticle.Request{}, fmt.Errorf("provision は文字列でなければなりません")
		}
		values.Provision = &provision
	}
	if raw, exists := fields["paragraph"]; exists {
		if isJSONNull(raw) {
			return getarticle.Request{}, fmt.Errorf("paragraph に null は使用できません")
		}
		var paragraph int
		if err := json.Unmarshal(raw, &paragraph); err != nil {
			return getarticle.Request{}, fmt.Errorf("paragraph は整数でなければなりません")
		}
		values.Paragraph = &paragraph
	}
	if raw, exists := fields["asOf"]; exists {
		if isJSONNull(raw) {
			return getarticle.Request{}, fmt.Errorf("asOf に null は使用できません")
		}
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return getarticle.Request{}, fmt.Errorf("asOf は YYYY-MM-DD 文字列でなければなりません")
		}
		asOf, err := model.NewDate(text)
		if err != nil {
			return getarticle.Request{}, fmt.Errorf("asOf は YYYY-MM-DD でなければなりません")
		}
		values.AsOf = &asOf
	}
	return getarticle.NewRequest(values)
}

func mapGetArticleOutput(fragment model.LawArticleFragment) getArticleOutput {
	citation := fragment.Citation()
	outputCitation := getLawOutputCitation{
		Source:     mapGetLawSource(citation.Source()),
		LawID:      citation.LawID(),
		RevisionID: citation.RevisionID(),
		URL:        citation.URL(),
	}
	if location, exists := citation.Location(); exists {
		outputCitation.Location = location
	}
	return getArticleOutput{
		Format:   fragment.Format(),
		Content:  fragment.Content(),
		Citation: outputCitation,
	}
}

func mapGetArticleError(err error) model.ErrorResult {
	switch {
	case errors.Is(err, lawarticleread.ErrNotFound):
		return newFixedErrorResult(model.ErrorCodeNotFound)
	case errors.Is(err, lawarticleread.ErrAmbiguousLocation):
		return newFixedErrorResult(model.ErrorCodeAmbiguousLocation)
	}
	var sourceError model.SourceError
	if errors.As(err, &sourceError) {
		result, buildErr := model.NewErrorResultFromSourceError(sourceError)
		if buildErr == nil {
			return result
		}
	}
	return newInternalErrorResult()
}

func newFixedErrorResult(code model.ErrorCode) model.ErrorResult {
	result, err := model.NewErrorResult(model.ErrorResultValues{Code: code})
	if err != nil {
		return newInternalErrorResult()
	}
	return result
}

func successGetArticleToolResult(
	output getArticleOutput,
) (*sdk.CallToolResult, error) {
	payload, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}
	return &sdk.CallToolResult{
		Content: []sdk.Content{
			&sdk.TextContent{Text: string(payload)},
		},
		StructuredContent: json.RawMessage(payload),
		IsError:           false,
	}, nil
}
