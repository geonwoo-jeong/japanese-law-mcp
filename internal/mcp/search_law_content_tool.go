package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchlawcontent"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type searchLawContentInputSchema struct {
	Query  string `json:"query"`
	AsOf   string `json:"asOf,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type searchLawContentOutput struct {
	TotalCount int                          `json:"totalCount"`
	Items      []searchLawContentOutputItem `json:"items"`
	NextOffset *int                         `json:"nextOffset,omitempty"`
}

type searchLawContentOutputItem struct {
	Law      searchLawContentOutputLaw      `json:"law"`
	Location string                         `json:"location"`
	Text     string                         `json:"text"`
	Citation searchLawContentOutputCitation `json:"citation"`
}

type searchLawContentOutputLaw struct {
	LawID                 string                       `json:"lawId"`
	RevisionID            string                       `json:"revisionId"`
	Title                 string                       `json:"title"`
	LawNumber             string                       `json:"lawNumber,omitempty"`
	PromulgationDate      string                       `json:"promulgationDate,omitempty"`
	RevisionEffectiveDate string                       `json:"revisionEffectiveDate,omitempty"`
	Source                searchLawContentOutputSource `json:"source"`
}

type searchLawContentOutputCitation struct {
	Source     searchLawContentOutputSource `json:"source"`
	LawID      string                       `json:"lawId"`
	RevisionID string                       `json:"revisionId"`
	Location   string                       `json:"location,omitempty"`
	URL        string                       `json:"url"`
}

type searchLawContentOutputSource struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Authority  string `json:"authority"`
	ServiceURL string `json:"serviceUrl"`
}

func addSearchLawContentTool(
	server toolRegistrar,
	searcher searchlawcontent.Port,
) {
	server.AddTool(&sdk.Tool{
		Name:         "search_law_content",
		Description:  "e-Gov の本文検索式を使い、法令本文の一致位置を検索します。",
		InputSchema:  mustSchemaFor[searchLawContentInputSchema](),
		OutputSchema: mustSchemaFor[searchLawContentOutput](),
	}, func(ctx context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return callSearchLawContent(ctx, searcher, request.Params.Arguments)
	})
}

func callSearchLawContent(
	ctx context.Context,
	searcher searchlawcontent.Port,
	arguments json.RawMessage,
) (*sdk.CallToolResult, error) {
	if searcher == nil {
		return errorToolResult(newInternalErrorResult())
	}
	request, err := decodeSearchLawContentInput(arguments)
	if err != nil {
		return errorToolResult(newInvalidArgumentResult(err))
	}

	result, searchErr := searcher.Search(ctx, request)
	if searchErr != nil {
		return errorToolResult(mapSearchLawContentError(searchErr))
	}
	return successSearchLawContentToolResult(mapSearchLawContentOutput(result))
}

func decodeSearchLawContentInput(arguments json.RawMessage) (searchlawcontent.Request, error) {
	if len(arguments) == 0 {
		return searchlawcontent.Request{}, fmt.Errorf("query は必須です")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(arguments, &fields); err != nil {
		return searchlawcontent.Request{}, fmt.Errorf("入力は JSON object でなければなりません")
	}

	for key := range fields {
		switch key {
		case "query", "asOf", "limit", "offset":
		default:
			return searchlawcontent.Request{}, fmt.Errorf("定義していない入力項目は使用できません")
		}
	}

	var values searchlawcontent.RequestValues
	queryRaw, exists := fields["query"]
	if !exists || isJSONNull(queryRaw) {
		return searchlawcontent.Request{}, fmt.Errorf("query は必須です")
	}
	if err := json.Unmarshal(queryRaw, &values.Query); err != nil {
		return searchlawcontent.Request{}, fmt.Errorf("query は文字列でなければなりません")
	}
	if raw, ok := fields["asOf"]; ok {
		if isJSONNull(raw) {
			return searchlawcontent.Request{}, fmt.Errorf("asOf に null は使用できません")
		}
		var asOfString string
		if err := json.Unmarshal(raw, &asOfString); err != nil {
			return searchlawcontent.Request{}, fmt.Errorf("asOf は YYYY-MM-DD 文字列でなければなりません")
		}
		asOf, err := model.NewDate(asOfString)
		if err != nil {
			return searchlawcontent.Request{}, fmt.Errorf("asOf は YYYY-MM-DD でなければなりません")
		}
		values.AsOf = &asOf
	}
	if raw, ok := fields["limit"]; ok {
		if isJSONNull(raw) {
			return searchlawcontent.Request{}, fmt.Errorf("limit に null は使用できません")
		}
		var limit int
		if err := json.Unmarshal(raw, &limit); err != nil {
			return searchlawcontent.Request{}, fmt.Errorf("limit は整数でなければなりません")
		}
		values.Limit = &limit
	}
	if raw, ok := fields["offset"]; ok {
		if isJSONNull(raw) {
			return searchlawcontent.Request{}, fmt.Errorf("offset に null は使用できません")
		}
		var offset int
		if err := json.Unmarshal(raw, &offset); err != nil {
			return searchlawcontent.Request{}, fmt.Errorf("offset は整数でなければなりません")
		}
		values.Offset = &offset
	}
	request, err := searchlawcontent.NewRequest(values)
	if err != nil {
		return searchlawcontent.Request{}, err
	}
	return request, nil
}

func mapSearchLawContentOutput(
	result model.LawContentSearchResult,
) searchLawContentOutput {
	resultItems := result.Items()
	items := make([]searchLawContentOutputItem, len(resultItems))
	for index, item := range resultItems {
		law := item.Law()
		citation := item.Citation()
		items[index] = searchLawContentOutputItem{
			Law: searchLawContentOutputLaw{
				LawID:      law.LawID(),
				RevisionID: law.RevisionID(),
				Title:      law.Title(),
				Source:     mapSearchLawContentSource(law.Source()),
			},
			Location: item.Location(),
			Text:     item.Text(),
			Citation: searchLawContentOutputCitation{
				Source:     mapSearchLawContentSource(citation.Source()),
				LawID:      citation.LawID(),
				RevisionID: citation.RevisionID(),
				URL:        citation.URL(),
			},
		}
		if lawNumber, exists := law.LawNumber(); exists {
			items[index].Law.LawNumber = lawNumber
		}
		if promulgationDate, exists := law.PromulgationDate(); exists {
			items[index].Law.PromulgationDate = promulgationDate.String()
		}
		if revisionDate, exists := law.RevisionEffectiveDate(); exists {
			items[index].Law.RevisionEffectiveDate = revisionDate.String()
		}
		if location, exists := citation.Location(); exists {
			items[index].Citation.Location = location
		}
	}
	output := searchLawContentOutput{
		TotalCount: result.TotalCount(),
		Items:      items,
	}
	if nextOffset, exists := result.NextOffset(); exists {
		output.NextOffset = &nextOffset
	}
	return output
}

func mapSearchLawContentSource(source model.LegalSource) searchLawContentOutputSource {
	return searchLawContentOutputSource{
		ID:         source.ID(),
		Name:       source.Name(),
		Authority:  string(source.Authority()),
		ServiceURL: source.ServiceURL(),
	}
}

func mapSearchLawContentError(err error) model.ErrorResult {
	var sourceError model.SourceError
	if errors.As(err, &sourceError) {
		result, buildErr := model.NewErrorResultFromSourceError(sourceError)
		if buildErr == nil {
			return result
		}
	}
	return newInternalErrorResult()
}

func successSearchLawContentToolResult(
	output searchLawContentOutput,
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
