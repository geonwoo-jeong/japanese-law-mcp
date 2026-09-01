package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/getlaw"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type getLawInputSchema struct {
	LawID string `json:"lawId"`
	AsOf  string `json:"asOf,omitempty"`
}

type getLawOutput struct {
	Law      getLawOutputSummary  `json:"law"`
	AsOf     string               `json:"asOf,omitempty"`
	Format   string               `json:"format"`
	Content  string               `json:"content"`
	Citation getLawOutputCitation `json:"citation"`
}

type getLawOutputSummary struct {
	LawID                 string             `json:"lawId"`
	RevisionID            string             `json:"revisionId"`
	Title                 string             `json:"title"`
	LawNumber             string             `json:"lawNumber,omitempty"`
	PromulgationDate      string             `json:"promulgationDate,omitempty"`
	RevisionEffectiveDate string             `json:"revisionEffectiveDate,omitempty"`
	Source                getLawOutputSource `json:"source"`
}

type getLawOutputCitation struct {
	Source     getLawOutputSource `json:"source"`
	LawID      string             `json:"lawId"`
	RevisionID string             `json:"revisionId"`
	Location   string             `json:"location,omitempty"`
	URL        string             `json:"url"`
}

type getLawOutputSource struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Authority  string `json:"authority"`
	ServiceURL string `json:"serviceUrl"`
}

func addGetLawTool(server toolRegistrar, getter getlaw.Port) {
	server.AddTool(&sdk.Tool{
		Name:         "get_law",
		Description:  "法令識別子を使い、e-Gov で確認できる法令本文を XML で取得します。",
		InputSchema:  mustSchemaFor[getLawInputSchema](),
		OutputSchema: mustSchemaFor[getLawOutput](),
	}, func(ctx context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return callGetLaw(ctx, getter, request.Params.Arguments)
	})
}

func callGetLaw(
	ctx context.Context,
	getter getlaw.Port,
	arguments json.RawMessage,
) (*sdk.CallToolResult, error) {
	if getter == nil {
		return errorToolResult(newInternalErrorResult())
	}
	request, err := decodeGetLawInput(arguments)
	if err != nil {
		return errorToolResult(newInvalidArgumentResult(err))
	}
	document, err := getter.Get(ctx, request)
	if err != nil {
		return errorToolResult(mapGetLawError(err))
	}
	return successGetLawToolResult(mapGetLawOutput(document))
}

func decodeGetLawInput(arguments json.RawMessage) (getlaw.Request, error) {
	if len(arguments) == 0 {
		return getlaw.Request{}, fmt.Errorf("lawId は必須です")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(arguments, &fields); err != nil {
		return getlaw.Request{}, fmt.Errorf("入力は JSON object でなければなりません")
	}
	for key := range fields {
		switch key {
		case "lawId", "asOf":
		default:
			return getlaw.Request{}, fmt.Errorf("定義していない入力項目は使用できません")
		}
	}

	lawIDRaw, exists := fields["lawId"]
	if !exists || isJSONNull(lawIDRaw) {
		return getlaw.Request{}, fmt.Errorf("lawId は必須です")
	}
	var values getlaw.RequestValues
	if err := json.Unmarshal(lawIDRaw, &values.LawID); err != nil {
		return getlaw.Request{}, fmt.Errorf("lawId は文字列でなければなりません")
	}
	if asOfRaw, exists := fields["asOf"]; exists {
		if isJSONNull(asOfRaw) {
			return getlaw.Request{}, fmt.Errorf("asOf に null は使用できません")
		}
		var text string
		if err := json.Unmarshal(asOfRaw, &text); err != nil {
			return getlaw.Request{}, fmt.Errorf("asOf は YYYY-MM-DD 文字列でなければなりません")
		}
		asOf, err := model.NewDate(text)
		if err != nil {
			return getlaw.Request{}, fmt.Errorf("asOf は YYYY-MM-DD でなければなりません")
		}
		values.AsOf = &asOf
	}
	return getlaw.NewRequest(values)
}

func mapGetLawOutput(document model.LawDocument) getLawOutput {
	law := document.Law()
	citation := document.Citation()
	output := getLawOutput{
		Law: getLawOutputSummary{
			LawID:      law.LawID(),
			RevisionID: law.RevisionID(),
			Title:      law.Title(),
			Source:     mapGetLawSource(law.Source()),
		},
		Format:  document.Format(),
		Content: document.Content(),
		Citation: getLawOutputCitation{
			Source:     mapGetLawSource(citation.Source()),
			LawID:      citation.LawID(),
			RevisionID: citation.RevisionID(),
			URL:        citation.URL(),
		},
	}
	if asOf, exists := document.AsOf(); exists {
		output.AsOf = asOf.String()
	}
	if lawNumber, exists := law.LawNumber(); exists {
		output.Law.LawNumber = lawNumber
	}
	if date, exists := law.PromulgationDate(); exists {
		output.Law.PromulgationDate = date.String()
	}
	if date, exists := law.RevisionEffectiveDate(); exists {
		output.Law.RevisionEffectiveDate = date.String()
	}
	if location, exists := citation.Location(); exists {
		output.Citation.Location = location
	}
	return output
}

func mapGetLawSource(source model.LegalSource) getLawOutputSource {
	return getLawOutputSource{
		ID:         source.ID(),
		Name:       source.Name(),
		Authority:  string(source.Authority()),
		ServiceURL: source.ServiceURL(),
	}
}

func mapGetLawError(err error) model.ErrorResult {
	if errors.Is(err, lawdocumentread.ErrNotFound) {
		result, buildErr := model.NewErrorResult(model.ErrorResultValues{
			Code: model.ErrorCodeNotFound,
		})
		if buildErr == nil {
			return result
		}
		return newInternalErrorResult()
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

func successGetLawToolResult(output getLawOutput) (*sdk.CallToolResult, error) {
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
