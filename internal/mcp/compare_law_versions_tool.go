package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/comparelawversions"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawversioncompare"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type compareLawVersionsInputSchema struct {
	LawID  string                          `json:"lawId"`
	Before compareLawVersionsInputSelector `json:"before"`
	After  compareLawVersionsInputSelector `json:"after"`
}

type compareLawVersionsInputSelector struct {
	RevisionID string `json:"revisionId,omitempty"`
	AsOf       string `json:"asOf,omitempty"`
}

type compareLawVersionsOutput struct {
	LawID              string                           `json:"lawId"`
	Scope              model.LawVersionComparisonScope  `json:"scope"`
	Before             compareLawVersionsOutputSnapshot `json:"before"`
	After              compareLawVersionsOutputSnapshot `json:"after"`
	BeforeArticleCount int                              `json:"beforeArticleCount"`
	AfterArticleCount  int                              `json:"afterArticleCount"`
	AddedCount         int                              `json:"addedCount"`
	RemovedCount       int                              `json:"removedCount"`
	ModifiedCount      int                              `json:"modifiedCount"`
	UnchangedCount     int                              `json:"unchangedCount"`
	TotalCount         int                              `json:"totalCount"`
	Items              []compareLawVersionsOutputItem   `json:"items"`
}

type compareLawVersionsOutputSnapshot struct {
	Law      compareLawVersionsOutputLawSummary `json:"law"`
	AsOf     string                             `json:"asOf,omitempty"`
	Citation compareLawVersionsOutputCitation   `json:"citation"`
}

type compareLawVersionsOutputLawSummary struct {
	LawID                 string                       `json:"lawId"`
	RevisionID            string                       `json:"revisionId"`
	Title                 string                       `json:"title"`
	LawNumber             string                       `json:"lawNumber,omitempty"`
	PromulgationDate      string                       `json:"promulgationDate,omitempty"`
	RevisionEffectiveDate string                       `json:"revisionEffectiveDate,omitempty"`
	Source                listLawRevisionsOutputSource `json:"source"`
}

type compareLawVersionsOutputCitation struct {
	Source     listLawRevisionsOutputSource `json:"source"`
	LawID      string                       `json:"lawId"`
	RevisionID string                       `json:"revisionId"`
	Location   string                       `json:"location,omitempty"`
	URL        string                       `json:"url"`
}

type compareLawVersionsOutputItem struct {
	ChangeKind    model.LawVersionChangeKind       `json:"changeKind"`
	ChangeReasons []model.LawVersionChangeReason   `json:"changeReasons,omitempty"`
	Before        *compareLawVersionsOutputArticle `json:"before,omitempty"`
	After         *compareLawVersionsOutputArticle `json:"after,omitempty"`
}

type compareLawVersionsOutputArticle struct {
	Location       compareLawVersionsOutputLocation `json:"location"`
	ArticleTitle   string                           `json:"articleTitle,omitempty"`
	ArticleCaption string                           `json:"articleCaption,omitempty"`
	Text           string                           `json:"text"`
	Citation       compareLawVersionsOutputCitation `json:"citation"`
}

type compareLawVersionsOutputLocation struct {
	Provision        model.LawArticleProvision `json:"provision"`
	ArticleNumber    string                    `json:"articleNumber"`
	PartNumber       string                    `json:"partNumber,omitempty"`
	ChapterNumber    string                    `json:"chapterNumber,omitempty"`
	SectionNumber    string                    `json:"sectionNumber,omitempty"`
	SubsectionNumber string                    `json:"subsectionNumber,omitempty"`
	DivisionNumber   string                    `json:"divisionNumber,omitempty"`
}

func addCompareLawVersionsTool(server toolRegistrar, comparer comparelawversions.Port) {
	server.AddTool(&sdk.Tool{
		Name:         "compare_law_versions",
		Description:  "同じ法令の二つの版を条単位で比較し、追加・削除・変更を返します。",
		InputSchema:  mustSchemaFor[compareLawVersionsInputSchema](),
		OutputSchema: mustSchemaFor[compareLawVersionsOutput](),
	}, func(ctx context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return callCompareLawVersions(ctx, comparer, request.Params.Arguments)
	})
}

func callCompareLawVersions(
	ctx context.Context,
	comparer comparelawversions.Port,
	arguments json.RawMessage,
) (*sdk.CallToolResult, error) {
	if isNilCompareLawVersionsPort(comparer) {
		return errorToolResult(newInternalErrorResult())
	}
	request, err := decodeCompareLawVersionsInput(arguments)
	if err != nil {
		return errorToolResult(newInvalidArgumentResult(err))
	}
	result, err := comparer.Compare(ctx, request)
	if err != nil {
		return errorToolResult(mapCompareLawVersionsError(err))
	}
	if err := result.Validate(); err != nil {
		return errorToolResult(mapCompareLawVersionsError(fmt.Errorf(
			"%w: %w",
			comparelawversions.ErrInvalidSourceResponse,
			err,
		)))
	}
	return successCompareLawVersionsToolResult(mapCompareLawVersionsOutput(result))
}

func isNilCompareLawVersionsPort(comparer comparelawversions.Port) bool {
	if comparer == nil {
		return true
	}
	value := reflect.ValueOf(comparer)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func decodeCompareLawVersionsInput(arguments json.RawMessage) (comparelawversions.Request, error) {
	if len(arguments) == 0 {
		return comparelawversions.Request{}, fmt.Errorf("lawId は必須です")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(arguments, &fields); err != nil || fields == nil {
		return comparelawversions.Request{}, fmt.Errorf("入力は JSON object でなければなりません")
	}
	for key := range fields {
		if key != "lawId" && key != "before" && key != "after" {
			return comparelawversions.Request{}, fmt.Errorf("定義していない入力項目は使用できません")
		}
	}
	var lawID string
	if raw, exists := fields["lawId"]; !exists || isJSONNull(raw) {
		return comparelawversions.Request{}, fmt.Errorf("lawId は必須です")
	} else if err := json.Unmarshal(raw, &lawID); err != nil {
		return comparelawversions.Request{}, fmt.Errorf("lawId は文字列でなければなりません")
	}
	before, err := decodeCompareLawVersionsSelector(fields["before"], "before")
	if err != nil {
		return comparelawversions.Request{}, err
	}
	after, err := decodeCompareLawVersionsSelector(fields["after"], "after")
	if err != nil {
		return comparelawversions.Request{}, err
	}
	return comparelawversions.NewRequest(comparelawversions.RequestValues{
		LawID:  lawID,
		Before: before,
		After:  after,
	})
}

func decodeCompareLawVersionsSelector(
	raw json.RawMessage,
	name string,
) (lawversioncompare.Selector, error) {
	if len(raw) == 0 || isJSONNull(raw) {
		return lawversioncompare.Selector{}, fmt.Errorf("%s は必須です", name)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return lawversioncompare.Selector{}, fmt.Errorf("%s は JSON object でなければなりません", name)
	}
	for key := range fields {
		if key != "revisionId" && key != "asOf" {
			return lawversioncompare.Selector{}, fmt.Errorf("%s に定義していない入力項目は使用できません", name)
		}
	}
	values := lawversioncompare.SelectorValues{}
	if value, exists := fields["revisionId"]; exists && !isJSONNull(value) {
		if err := json.Unmarshal(value, &values.RevisionID); err != nil {
			return lawversioncompare.Selector{}, fmt.Errorf("%s.revisionId は文字列でなければなりません", name)
		}
	}
	if value, exists := fields["asOf"]; exists && !isJSONNull(value) {
		var text string
		if err := json.Unmarshal(value, &text); err != nil {
			return lawversioncompare.Selector{}, fmt.Errorf("%s.asOf は文字列でなければなりません", name)
		}
		date, err := model.NewDate(text)
		if err != nil {
			return lawversioncompare.Selector{}, fmt.Errorf("%s.asOf が有効ではありません: %w", name, err)
		}
		values.AsOf = &date
	}
	selector, err := lawversioncompare.NewSelector(values)
	if err != nil {
		return lawversioncompare.Selector{}, fmt.Errorf("%s が有効ではありません: %w", name, err)
	}
	return selector, nil
}

func mapCompareLawVersionsOutput(result model.LawVersionComparison) compareLawVersionsOutput {
	output := compareLawVersionsOutput{
		LawID:              result.LawID(),
		Scope:              result.Scope(),
		Before:             mapCompareLawVersionsSnapshot(result.Before()),
		After:              mapCompareLawVersionsSnapshot(result.After()),
		BeforeArticleCount: result.BeforeArticleCount(),
		AfterArticleCount:  result.AfterArticleCount(),
		AddedCount:         result.AddedCount(),
		RemovedCount:       result.RemovedCount(),
		ModifiedCount:      result.ModifiedCount(),
		UnchangedCount:     result.UnchangedCount(),
		TotalCount:         result.TotalCount(),
	}
	changes := result.Items()
	output.Items = make([]compareLawVersionsOutputItem, len(changes))
	for index, change := range changes {
		output.Items[index] = mapCompareLawVersionsChange(change)
	}
	return output
}

func mapCompareLawVersionsSnapshot(
	snapshot model.LawVersionSnapshot,
) compareLawVersionsOutputSnapshot {
	output := compareLawVersionsOutputSnapshot{
		Law:      mapCompareLawVersionsLawSummary(snapshot.Law()),
		Citation: mapCompareLawVersionsCitation(snapshot.Citation()),
	}
	if value, exists := snapshot.AsOf(); exists {
		output.AsOf = value.String()
	}
	return output
}

func mapCompareLawVersionsLawSummary(summary model.LawSummary) compareLawVersionsOutputLawSummary {
	source := summary.Source()
	output := compareLawVersionsOutputLawSummary{
		LawID:      summary.LawID(),
		RevisionID: summary.RevisionID(),
		Title:      summary.Title(),
		Source: listLawRevisionsOutputSource{
			ID: source.ID(), Name: source.Name(), Authority: string(source.Authority()),
			ServiceURL: source.ServiceURL(),
		},
	}
	output.LawNumber, _ = summary.LawNumber()
	if value, exists := summary.PromulgationDate(); exists {
		output.PromulgationDate = value.String()
	}
	if value, exists := summary.RevisionEffectiveDate(); exists {
		output.RevisionEffectiveDate = value.String()
	}
	return output
}

func mapCompareLawVersionsCitation(citation model.Citation) compareLawVersionsOutputCitation {
	source := citation.Source()
	output := compareLawVersionsOutputCitation{
		Source: listLawRevisionsOutputSource{
			ID: source.ID(), Name: source.Name(), Authority: string(source.Authority()),
			ServiceURL: source.ServiceURL(),
		},
		LawID:      citation.LawID(),
		RevisionID: citation.RevisionID(),
		URL:        citation.URL(),
	}
	output.Location, _ = citation.Location()
	return output
}

func mapCompareLawVersionsChange(change model.LawVersionChange) compareLawVersionsOutputItem {
	output := compareLawVersionsOutputItem{
		ChangeKind:    change.ChangeKind(),
		ChangeReasons: change.ChangeReasons(),
	}
	if value, exists := change.Before(); exists {
		article := mapCompareLawVersionsArticle(value)
		output.Before = &article
	}
	if value, exists := change.After(); exists {
		article := mapCompareLawVersionsArticle(value)
		output.After = &article
	}
	return output
}

func mapCompareLawVersionsArticle(
	article model.LawVersionArticle,
) compareLawVersionsOutputArticle {
	output := compareLawVersionsOutputArticle{
		Location: mapCompareLawVersionsLocation(article.Location()),
		Text:     article.Text(),
		Citation: mapCompareLawVersionsCitation(article.Citation()),
	}
	if value, exists := article.ArticleTitle(); exists {
		output.ArticleTitle = value
	}
	if value, exists := article.ArticleCaption(); exists {
		output.ArticleCaption = value
	}
	return output
}

func mapCompareLawVersionsLocation(
	location model.LawVersionArticleLocation,
) compareLawVersionsOutputLocation {
	output := compareLawVersionsOutputLocation{
		Provision:     location.Provision(),
		ArticleNumber: location.ArticleNumber(),
	}
	if value, exists := location.PartNumber(); exists {
		output.PartNumber = value
	}
	if value, exists := location.ChapterNumber(); exists {
		output.ChapterNumber = value
	}
	if value, exists := location.SectionNumber(); exists {
		output.SectionNumber = value
	}
	if value, exists := location.SubsectionNumber(); exists {
		output.SubsectionNumber = value
	}
	if value, exists := location.DivisionNumber(); exists {
		output.DivisionNumber = value
	}
	return output
}

func mapCompareLawVersionsError(err error) model.ErrorResult {
	if errors.Is(err, lawversioncompare.ErrNotFound) {
		return newFixedErrorResult(model.ErrorCodeNotFound)
	}
	if errors.Is(err, comparelawversions.ErrInvalidSourceResponse) {
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

func successCompareLawVersionsToolResult(
	output compareLawVersionsOutput,
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
