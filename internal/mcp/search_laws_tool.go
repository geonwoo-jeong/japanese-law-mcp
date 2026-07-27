package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchlaws"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type searchLawsInputSchema struct {
	Query  string `json:"query"`
	AsOf   string `json:"asOf,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type searchLawsOutput struct {
	TotalCount int                    `json:"totalCount"`
	Items      []searchLawsOutputItem `json:"items"`
	NextOffset *int                   `json:"nextOffset,omitempty"`
}

type searchLawsOutputItem struct {
	LawID                 string                 `json:"lawId"`
	RevisionID            string                 `json:"revisionId"`
	Title                 string                 `json:"title"`
	LawNumber             string                 `json:"lawNumber,omitempty"`
	PromulgationDate      string                 `json:"promulgationDate,omitempty"`
	RevisionEffectiveDate string                 `json:"revisionEffectiveDate,omitempty"`
	Source                searchLawsOutputSource `json:"source"`
}

type searchLawsOutputSource struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Authority  string `json:"authority"`
	ServiceURL string `json:"serviceUrl"`
}

func addSearchLawsTool(
	server *sdk.Server,
	searcher searchlaws.Port,
) {
	inputSchema := mustSchemaFor[searchLawsInputSchema]()
	outputSchema := mustSchemaFor[searchLawsOutput]()
	server.AddTool(&sdk.Tool{
		Name:         "search_laws",
		Description:  "法令名、略称、表記揺れ、自然文または軽微な誤記から、公式情報源で確認できる法令を検索します。",
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
	}, func(ctx context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		result, err := callSearchLaws(ctx, searcher, request.Params.Arguments)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

func callSearchLaws(
	ctx context.Context,
	searcher searchlaws.Port,
	arguments json.RawMessage,
) (*sdk.CallToolResult, error) {
	if searcher == nil {
		return errorToolResult(newInternalErrorResult())
	}
	request, err := decodeSearchLawsInput(arguments)
	if err != nil {
		return errorToolResult(newInvalidArgumentResult(err))
	}

	result, searchErr := searcher.Search(ctx, request)
	if searchErr != nil {
		return errorToolResult(mapSearchLawsError(searchErr))
	}
	output := mapSearchLawsOutput(result)
	return successToolResult(output)
}

func decodeSearchLawsInput(arguments json.RawMessage) (searchlaws.Request, error) {
	if len(arguments) == 0 {
		return searchlaws.Request{}, fmt.Errorf("query は必須です")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(arguments, &fields); err != nil {
		return searchlaws.Request{}, fmt.Errorf("入力は JSON object でなければなりません")
	}

	for key := range fields {
		switch key {
		case "query", "asOf", "limit", "offset":
		default:
			return searchlaws.Request{}, fmt.Errorf("定義していない入力項目は使用できません")
		}
	}

	var values searchlaws.RequestValues
	queryRaw, exists := fields["query"]
	if !exists || isJSONNull(queryRaw) {
		return searchlaws.Request{}, fmt.Errorf("query は必須です")
	}
	if err := json.Unmarshal(queryRaw, &values.Query); err != nil {
		return searchlaws.Request{}, fmt.Errorf("query は文字列でなければなりません")
	}
	if raw, ok := fields["asOf"]; ok {
		if isJSONNull(raw) {
			return searchlaws.Request{}, fmt.Errorf("asOf に null は使用できません")
		}
		var asOfString string
		if err := json.Unmarshal(raw, &asOfString); err != nil {
			return searchlaws.Request{}, fmt.Errorf("asOf は YYYY-MM-DD 文字列でなければなりません")
		}
		asOf, err := model.NewDate(asOfString)
		if err != nil {
			return searchlaws.Request{}, fmt.Errorf("asOf は YYYY-MM-DD でなければなりません")
		}
		values.AsOf = &asOf
	}
	if raw, ok := fields["limit"]; ok {
		if isJSONNull(raw) {
			return searchlaws.Request{}, fmt.Errorf("limit に null は使用できません")
		}
		var limit int
		if err := json.Unmarshal(raw, &limit); err != nil {
			return searchlaws.Request{}, fmt.Errorf("limit は整数でなければなりません")
		}
		values.Limit = &limit
	}
	if raw, ok := fields["offset"]; ok {
		if isJSONNull(raw) {
			return searchlaws.Request{}, fmt.Errorf("offset に null は使用できません")
		}
		var offset int
		if err := json.Unmarshal(raw, &offset); err != nil {
			return searchlaws.Request{}, fmt.Errorf("offset は整数でなければなりません")
		}
		values.Offset = &offset
	}
	request, err := searchlaws.NewRequest(values)
	if err != nil {
		return searchlaws.Request{}, err
	}
	return request, nil
}

func mapSearchLawsOutput(result model.LawSearchResult) searchLawsOutput {
	resultItems := result.Items()
	items := make([]searchLawsOutputItem, len(resultItems))
	for index, item := range resultItems {
		source := item.Source()
		items[index] = searchLawsOutputItem{
			LawID:      item.LawID(),
			RevisionID: item.RevisionID(),
			Title:      item.Title(),
			Source: searchLawsOutputSource{
				ID:         source.ID(),
				Name:       source.Name(),
				Authority:  string(source.Authority()),
				ServiceURL: source.ServiceURL(),
			},
		}
		if lawNumber, exists := item.LawNumber(); exists {
			items[index].LawNumber = lawNumber
		}
		if promulgationDate, exists := item.PromulgationDate(); exists {
			items[index].PromulgationDate = promulgationDate.String()
		}
		if revisionDate, exists := item.RevisionEffectiveDate(); exists {
			items[index].RevisionEffectiveDate = revisionDate.String()
		}
	}
	output := searchLawsOutput{
		TotalCount: result.TotalCount(),
		Items:      items,
	}
	if nextOffset, exists := result.NextOffset(); exists {
		output.NextOffset = &nextOffset
	}
	return output
}

func mapSearchLawsError(err error) model.ErrorResult {
	var sourceError model.SourceError
	if errors.As(err, &sourceError) {
		result, buildErr := model.NewErrorResultFromSourceError(sourceError)
		if buildErr == nil {
			return result
		}
	}
	return newInternalErrorResult()
}

func newInvalidArgumentResult(cause error) model.ErrorResult {
	result, err := model.NewErrorResult(model.ErrorResultValues{
		Code: model.ErrorCodeInvalidArgument,
		Details: map[string]any{
			"field":  "arguments",
			"reason": cause.Error(),
		},
	})
	if err != nil {
		return newInternalErrorResult()
	}
	return result
}

func newInternalErrorResult() model.ErrorResult {
	result, err := model.NewErrorResult(model.ErrorResultValues{
		Code: model.ErrorCodeInternalError,
	})
	if err != nil {
		panic(fmt.Sprintf("固定 ErrorResult を作成できません: %v", err))
	}
	return result
}

func successToolResult(output searchLawsOutput) (*sdk.CallToolResult, error) {
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

func errorToolResult(result model.ErrorResult) (*sdk.CallToolResult, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &sdk.CallToolResult{
		Content: []sdk.Content{
			&sdk.TextContent{Text: string(payload)},
		},
		IsError: true,
	}, nil
}

func mustSchemaFor[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(err)
	}
	return schema
}

func isJSONNull(value json.RawMessage) bool {
	return string(value) == "null"
}
