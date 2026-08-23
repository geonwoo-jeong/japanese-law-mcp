package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawrevisionlist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/listlawrevisions"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type listLawRevisionsOutput struct {
	LawID      string                       `json:"lawId"`
	TotalCount int                          `json:"totalCount"`
	Items      []listLawRevisionsOutputItem `json:"items"`
}

type listLawRevisionsOutputItem struct {
	LawID                     string                         `json:"lawId"`
	RevisionID                string                         `json:"revisionId"`
	Title                     string                         `json:"title"`
	LawType                   string                         `json:"lawType,omitempty"`
	LawNumber                 string                         `json:"lawNumber,omitempty"`
	TitleKana                 string                         `json:"titleKana,omitempty"`
	Abbreviation              string                         `json:"abbreviation,omitempty"`
	Category                  string                         `json:"category,omitempty"`
	PromulgationDate          string                         `json:"promulgationDate,omitempty"`
	SourceUpdatedAt           string                         `json:"sourceUpdatedAt,omitempty"`
	AmendmentPromulgationDate string                         `json:"amendmentPromulgationDate,omitempty"`
	EffectiveDate             string                         `json:"effectiveDate,omitempty"`
	EffectiveDateNote         string                         `json:"effectiveDateNote,omitempty"`
	ScheduledEffectiveDate    string                         `json:"scheduledEffectiveDate,omitempty"`
	AmendmentLawID            string                         `json:"amendmentLawId,omitempty"`
	AmendmentLawTitle         string                         `json:"amendmentLawTitle,omitempty"`
	AmendmentLawTitleKana     string                         `json:"amendmentLawTitleKana,omitempty"`
	AmendmentLawNumber        string                         `json:"amendmentLawNumber,omitempty"`
	RevisionKind              model.LawRevisionKind          `json:"revisionKind,omitempty"`
	RepealStatus              model.LawRevisionRepealStatus  `json:"repealStatus,omitempty"`
	RepealRecordedDate        string                         `json:"repealRecordedDate,omitempty"`
	RemainInForce             *bool                          `json:"remainInForce,omitempty"`
	CurrentStatus             model.LawRevisionCurrentStatus `json:"currentStatus,omitempty"`
	Source                    listLawRevisionsOutputSource   `json:"source"`
}

type listLawRevisionsOutputSource struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Authority  string `json:"authority"`
	ServiceURL string `json:"serviceUrl"`
}

func callListLawRevisions(
	ctx context.Context,
	lister listlawrevisions.Port,
	arguments json.RawMessage,
) (*sdk.CallToolResult, error) {
	if isNilListLawRevisionsPort(lister) {
		return errorToolResult(newInternalErrorResult())
	}
	request, err := decodeListLawRevisionsInput(arguments)
	if err != nil {
		return errorToolResult(newInvalidArgumentResult(err))
	}
	result, err := lister.List(ctx, request)
	if err != nil {
		return errorToolResult(mapListLawRevisionsError(err))
	}
	if err := result.Validate(); err != nil {
		return errorToolResult(mapListLawRevisionsError(fmt.Errorf(
			"%w: %w",
			listlawrevisions.ErrInvalidSourceResponse,
			err,
		)))
	}
	return successListLawRevisionsToolResult(mapListLawRevisionsOutput(result))
}

func isNilListLawRevisionsPort(lister listlawrevisions.Port) bool {
	if lister == nil {
		return true
	}
	value := reflect.ValueOf(lister)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func decodeListLawRevisionsInput(arguments json.RawMessage) (listlawrevisions.Request, error) {
	if len(arguments) == 0 {
		return listlawrevisions.Request{}, fmt.Errorf("lawIdOrNumber は必須です")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(arguments, &fields); err != nil || fields == nil {
		return listlawrevisions.Request{}, fmt.Errorf("入力は JSON object でなければなりません")
	}
	for key := range fields {
		if key != "lawIdOrNumber" {
			return listlawrevisions.Request{}, fmt.Errorf("定義していない入力項目は使用できません")
		}
	}
	raw, exists := fields["lawIdOrNumber"]
	if !exists || isJSONNull(raw) {
		return listlawrevisions.Request{}, fmt.Errorf("lawIdOrNumber は必須です")
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return listlawrevisions.Request{}, fmt.Errorf("lawIdOrNumber は文字列でなければなりません")
	}
	return listlawrevisions.NewRequest(listlawrevisions.RequestValues{LawIDOrNumber: value})
}

func mapListLawRevisionsOutput(result listlawrevisions.Result) listLawRevisionsOutput {
	revisions := result.Items()
	items := make([]listLawRevisionsOutputItem, len(revisions))
	for index, revision := range revisions {
		items[index] = mapListLawRevisionOutputItem(revision)
	}
	return listLawRevisionsOutput{
		LawID: result.LawID(), TotalCount: result.TotalCount(), Items: items,
	}
}

func mapListLawRevisionOutputItem(revision model.LawRevision) listLawRevisionsOutputItem {
	source := revision.Source()
	item := listLawRevisionsOutputItem{
		LawID: revision.LawID(), RevisionID: revision.RevisionID(), Title: revision.Title(),
		Source: listLawRevisionsOutputSource{
			ID: source.ID(), Name: source.Name(), Authority: string(source.Authority()),
			ServiceURL: source.ServiceURL(),
		},
	}
	item.LawType, _ = revision.LawType()
	item.LawNumber, _ = revision.LawNumber()
	item.TitleKana, _ = revision.TitleKana()
	item.Abbreviation, _ = revision.Abbreviation()
	item.Category, _ = revision.Category()
	item.SourceUpdatedAt, _ = revision.SourceUpdatedAt()
	item.EffectiveDateNote, _ = revision.EffectiveDateNote()
	item.AmendmentLawID, _ = revision.AmendmentLawID()
	item.AmendmentLawTitle, _ = revision.AmendmentLawTitle()
	item.AmendmentLawTitleKana, _ = revision.AmendmentLawTitleKana()
	item.AmendmentLawNumber, _ = revision.AmendmentLawNumber()
	item.RevisionKind, _ = revision.RevisionKind()
	item.RepealStatus, _ = revision.RepealStatus()
	item.CurrentStatus, _ = revision.CurrentStatus()
	if value, exists := revision.PromulgationDate(); exists {
		item.PromulgationDate = value.String()
	}
	if value, exists := revision.AmendmentPromulgationDate(); exists {
		item.AmendmentPromulgationDate = value.String()
	}
	if value, exists := revision.EffectiveDate(); exists {
		item.EffectiveDate = value.String()
	}
	if value, exists := revision.ScheduledEffectiveDate(); exists {
		item.ScheduledEffectiveDate = value.String()
	}
	if value, exists := revision.RepealRecordedDate(); exists {
		item.RepealRecordedDate = value.String()
	}
	if value, exists := revision.RemainInForce(); exists {
		item.RemainInForce = boolPointer(value)
	}
	return item
}

func mapListLawRevisionsError(err error) model.ErrorResult {
	if errors.Is(err, lawrevisionlist.ErrNotFound) {
		return newFixedErrorResult(model.ErrorCodeNotFound)
	}
	if errors.Is(err, listlawrevisions.ErrInvalidSourceResponse) {
		return newFixedErrorResult(model.ErrorCodeInvalidSourceResponse)
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

func successListLawRevisionsToolResult(
	output listLawRevisionsOutput,
) (*sdk.CallToolResult, error) {
	payload, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}
	return &sdk.CallToolResult{
		Content:           []sdk.Content{&sdk.TextContent{Text: string(payload)}},
		StructuredContent: json.RawMessage(payload),
		IsError:           false,
	}, nil
}
