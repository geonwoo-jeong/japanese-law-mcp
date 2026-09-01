package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/listlawupdates"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type listLawUpdatesInputSchema struct {
	Date  string `json:"date"`
	Limit int    `json:"limit,omitempty"`
}

type listLawUpdatesOutput struct {
	Date          string                     `json:"date"`
	TotalCount    int                        `json:"totalCount"`
	ReturnedCount int                        `json:"returnedCount"`
	OmittedCount  int                        `json:"omittedCount"`
	Truncated     bool                       `json:"truncated"`
	Items         []listLawUpdatesOutputItem `json:"items"`
}

type listLawUpdatesOutputItem struct {
	UpdatedOn                 string                     `json:"updatedOn"`
	LawID                     string                     `json:"lawId"`
	Title                     string                     `json:"title"`
	LawType                   string                     `json:"lawType,omitempty"`
	LawNumber                 string                     `json:"lawNumber,omitempty"`
	TitleKana                 string                     `json:"titleKana,omitempty"`
	PreviousTitle             string                     `json:"previousTitle,omitempty"`
	PromulgationDate          string                     `json:"promulgationDate,omitempty"`
	AmendmentTitle            string                     `json:"amendmentTitle,omitempty"`
	AmendmentLawNumber        string                     `json:"amendmentLawNumber,omitempty"`
	AmendmentPromulgationDate string                     `json:"amendmentPromulgationDate,omitempty"`
	EffectiveDate             string                     `json:"effectiveDate,omitempty"`
	EffectiveDateNote         string                     `json:"effectiveDateNote,omitempty"`
	DocumentURL               string                     `json:"documentUrl,omitempty"`
	EnforcementPending        *bool                      `json:"enforcementPending,omitempty"`
	AuthorityReviewPending    *bool                      `json:"authorityReviewPending,omitempty"`
	Source                    listLawUpdatesOutputSource `json:"source"`
}

type listLawUpdatesOutputSource struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Authority  string `json:"authority"`
	ServiceURL string `json:"serviceUrl"`
}

func addListLawUpdatesTool(
	server toolRegistrar,
	lister listlawupdates.Port,
) {
	inputSchema := mustSchemaFor[listLawUpdatesInputSchema]()
	inputSchema.Properties["limit"].Minimum = jsonschema.Ptr(1.0)
	inputSchema.Properties["limit"].Maximum = jsonschema.Ptr(float64(listlawupdates.MaxLimit))
	inputSchema.Properties["limit"].Default = json.RawMessage(
		fmt.Sprintf("%d", listlawupdates.DefaultLimit),
	)
	server.AddTool(&sdk.Tool{
		Name:         "list_law_updates",
		Description:  "指定日に e-Gov の更新一覧へ掲載された法令を、総件数と省略件数を明示して取得します。",
		InputSchema:  inputSchema,
		OutputSchema: mustSchemaFor[listLawUpdatesOutput](),
	}, func(ctx context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return callListLawUpdates(ctx, lister, request.Params.Arguments)
	})
}

func callListLawUpdates(
	ctx context.Context,
	lister listlawupdates.Port,
	arguments json.RawMessage,
) (*sdk.CallToolResult, error) {
	if lister == nil {
		return errorToolResult(newInternalErrorResult())
	}
	request, err := decodeListLawUpdatesInput(arguments)
	if err != nil {
		return errorToolResult(newInvalidArgumentResult(err))
	}
	result, err := lister.List(ctx, request)
	if err != nil {
		return errorToolResult(mapListLawUpdatesError(err))
	}
	if err := result.Validate(); err != nil {
		return errorToolResult(newInternalErrorResult())
	}
	return successListLawUpdatesToolResult(mapListLawUpdatesOutput(result))
}

func decodeListLawUpdatesInput(
	arguments json.RawMessage,
) (listlawupdates.Request, error) {
	input := bytes.TrimSpace(arguments)
	if len(input) == 0 || input[0] != '{' || input[len(input)-1] != '}' {
		return listlawupdates.Request{}, fmt.Errorf("入力は JSON object でなければなりません")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil || fields == nil {
		return listlawupdates.Request{}, fmt.Errorf("入力は JSON object でなければなりません")
	}
	for key := range fields {
		if key != "date" && key != "limit" {
			return listlawupdates.Request{}, fmt.Errorf("定義していない入力項目は使用できません")
		}
	}
	rawDate, exists := fields["date"]
	if !exists || isJSONNull(rawDate) {
		return listlawupdates.Request{}, fmt.Errorf("date は必須です")
	}
	var dateText string
	if err := json.Unmarshal(rawDate, &dateText); err != nil {
		return listlawupdates.Request{}, fmt.Errorf("date は YYYY-MM-DD 文字列でなければなりません")
	}
	date, err := model.NewDate(dateText)
	if err != nil {
		return listlawupdates.Request{}, fmt.Errorf("date は実在する YYYY-MM-DD でなければなりません")
	}
	values := listlawupdates.RequestValues{Date: date}
	if rawLimit, exists := fields["limit"]; exists {
		if isJSONNull(rawLimit) {
			return listlawupdates.Request{}, fmt.Errorf("limit に null は使用できません")
		}
		var limit int
		if err := json.Unmarshal(rawLimit, &limit); err != nil {
			return listlawupdates.Request{}, fmt.Errorf("limit は整数でなければなりません")
		}
		values.Limit = &limit
	}
	return listlawupdates.NewRequest(values)
}

func mapListLawUpdatesOutput(result listlawupdates.Result) listLawUpdatesOutput {
	updates := result.Items()
	items := make([]listLawUpdatesOutputItem, len(updates))
	for index, update := range updates {
		source := update.Source()
		items[index] = listLawUpdatesOutputItem{
			UpdatedOn: update.UpdatedOn().String(),
			LawID:     update.LawID(),
			Title:     update.Title(),
			Source: listLawUpdatesOutputSource{
				ID:         source.ID(),
				Name:       source.Name(),
				Authority:  string(source.Authority()),
				ServiceURL: source.ServiceURL(),
			},
		}
		mapListLawUpdateOptionalFields(update, &items[index])
	}
	return listLawUpdatesOutput{
		Date:          result.Date().String(),
		TotalCount:    result.TotalCount(),
		ReturnedCount: result.ReturnedCount(),
		OmittedCount:  result.OmittedCount(),
		Truncated:     result.Truncated(),
		Items:         items,
	}
}

func mapListLawUpdateOptionalFields(
	update model.LawUpdate,
	output *listLawUpdatesOutputItem,
) {
	output.LawType, _ = update.LawType()
	output.LawNumber, _ = update.LawNumber()
	output.TitleKana, _ = update.TitleKana()
	output.PreviousTitle, _ = update.PreviousTitle()
	if value, exists := update.PromulgationDate(); exists {
		output.PromulgationDate = value.String()
	}
	output.AmendmentTitle, _ = update.AmendmentTitle()
	output.AmendmentLawNumber, _ = update.AmendmentLawNumber()
	if value, exists := update.AmendmentPromulgationDate(); exists {
		output.AmendmentPromulgationDate = value.String()
	}
	if value, exists := update.EffectiveDate(); exists {
		output.EffectiveDate = value.String()
	}
	output.EffectiveDateNote, _ = update.EffectiveDateNote()
	output.DocumentURL, _ = update.DocumentURL()
	if value, exists := update.EnforcementPending(); exists {
		output.EnforcementPending = boolPointer(value)
	}
	if value, exists := update.AuthorityReviewPending(); exists {
		output.AuthorityReviewPending = boolPointer(value)
	}
}

func mapListLawUpdatesError(err error) model.ErrorResult {
	if errors.Is(err, listlawupdates.ErrInvalidSourceResponse) {
		result, buildErr := model.NewErrorResult(model.ErrorResultValues{
			Code: model.ErrorCodeInvalidSourceResponse,
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

func successListLawUpdatesToolResult(
	output listLawUpdatesOutput,
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

func boolPointer(value bool) *bool {
	return &value
}
