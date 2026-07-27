package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const judicialCasesCoverageNotice = "裁判所の裁判例検索には、すべての判決等が掲載されているわけではありません。掲載情報だけから先例性、拘束力、確定性または現在の有効性を判断できません。"

type searchJudicialCasesInputSchema struct {
	Query             string `json:"query"`
	Limit             int    `json:"limit,omitempty"`
	ContinuationToken string `json:"continuationToken,omitempty"`
}

type searchJudicialCasesOutput struct {
	CoverageNotice string                          `json:"coverageNotice"`
	Items          []searchJudicialCasesOutputItem `json:"items"`
	Page           searchJudicialCasesOutputPage   `json:"page"`
}

type searchJudicialCasesOutputItem struct {
	Ref        searchJudicialCasesOutputRef     `json:"ref"`
	Provenance []searchJudicialCasesOutputProv  `json:"provenance"`
	Data       searchJudicialCasesOutputSummary `json:"data"`
}

type searchJudicialCasesOutputRef struct {
	ProviderID string                               `json:"providerId"`
	Key        searchJudicialCasesOutputResourceKey `json:"key"`
}

type searchJudicialCasesOutputResourceKey struct {
	SourceID     string `json:"sourceId"`
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	VersionID    string `json:"versionId,omitempty"`
}

type searchJudicialCasesOutputProv struct {
	Source          searchJudicialCasesOutputSource         `json:"source"`
	ResourceKey     searchJudicialCasesOutputResourceKey    `json:"resourceKey"`
	URL             string                                  `json:"url"`
	RetrievedAt     string                                  `json:"retrievedAt"`
	SourceUpdatedAt string                                  `json:"sourceUpdatedAt,omitempty"`
	MediaType       string                                  `json:"mediaType"`
	Location        string                                  `json:"location,omitempty"`
	Transformation  model.ProvenanceTransformation          `json:"transformation"`
	MethodID        string                                  `json:"methodId,omitempty"`
	InputKeys       *[]searchJudicialCasesOutputResourceKey `json:"inputKeys,omitempty"`
	ContentDigest   string                                  `json:"contentDigest,omitempty"`
}

type searchJudicialCasesOutputSummary struct {
	DecisionID          string                              `json:"decisionId"`
	PublicationCategory model.JudicialPublicationCategory   `json:"publicationCategory"`
	SourceCategoryLabel string                              `json:"sourceCategoryLabel"`
	CaseNumber          string                              `json:"caseNumber"`
	CaseName            string                              `json:"caseName,omitempty"`
	DecisionDate        string                              `json:"decisionDate"`
	CourtName           string                              `json:"courtName"`
	BranchName          string                              `json:"branchName,omitempty"`
	DivisionName        string                              `json:"divisionName,omitempty"`
	DecisionType        string                              `json:"decisionType,omitempty"`
	Outcome             string                              `json:"outcome,omitempty"`
	DetailURL           string                              `json:"detailUrl"`
	Documents           []searchJudicialCasesOutputDocument `json:"documents"`
	Source              searchJudicialCasesOutputSource     `json:"source"`
}

type searchJudicialCasesOutputDocument struct {
	Kind      model.JudicialDocumentKind `json:"kind"`
	Label     string                     `json:"label"`
	MediaType string                     `json:"mediaType"`
	URL       string                     `json:"url"`
}

type searchJudicialCasesOutputSource struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Publisher  string `json:"publisher"`
	Authority  string `json:"authority"`
	ServiceURL string `json:"serviceUrl"`
}

type searchJudicialCasesOutputPage struct {
	ReturnedCount int                 `json:"returnedCount"`
	NextToken     string              `json:"nextToken,omitempty"`
	TotalCount    *int                `json:"totalCount,omitempty"`
	TotalRelation model.TotalRelation `json:"totalRelation,omitempty"`
}

func addSearchJudicialCasesTool(
	server *sdk.Server,
	searcher judicialdecisionsearch.Port,
) {
	inputSchema := mustSchemaFor[searchJudicialCasesInputSchema]()
	inputSchema.Properties["query"] = withUTF8ByteLimit(
		inputSchema.Properties["query"],
		jsonschema.Ptr(1),
		judicialdecisionsearch.MaxQueryBytes,
		"有効な UTF-8 で、正規化後に 1 byte 以上 512 byte 以下の検索語。",
	)
	inputSchema.Properties["limit"].Minimum = jsonschema.Ptr(1.0)
	inputSchema.Properties["limit"].Maximum = jsonschema.Ptr(30.0)
	inputSchema.Properties["limit"].Default = json.RawMessage("20")
	inputSchema.Properties["continuationToken"] = withUTF8ByteLimit(
		inputSchema.Properties["continuationToken"],
		nil,
		judicialdecisionsearch.MaxTokenBytes,
		"有効な UTF-8 で 4096 byte 以下の継続トークン。",
	)
	server.AddTool(&sdk.Tool{
		Name:         "search_judicial_cases",
		Description:  "裁判所の公式掲載裁判例を検索し、同じ参照を get_judicial_case へ引き渡せる形で返します。",
		InputSchema:  inputSchema,
		OutputSchema: mustSchemaFor[searchJudicialCasesOutput](),
	}, func(ctx context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return callSearchJudicialCases(ctx, searcher, request.Params.Arguments)
	})
}

func withUTF8ByteLimit(
	schema *jsonschema.Schema,
	minLength *int,
	maxBytes int,
	description string,
) *jsonschema.Schema {
	result := schema.CloneSchemas()
	result.MinLength = minLength
	result.MaxLength = nil
	result.Description = description
	result.Extra = map[string]any{"x-maxUtf8Bytes": maxBytes}
	return result
}

func callSearchJudicialCases(
	ctx context.Context,
	searcher judicialdecisionsearch.Port,
	arguments json.RawMessage,
) (*sdk.CallToolResult, error) {
	if isNilJudicialSearchPort(searcher) {
		return errorToolResult(newInternalErrorResult())
	}
	request, err := decodeSearchJudicialCasesInput(arguments)
	if err != nil {
		return errorToolResult(mapSearchJudicialCasesInvalidArgument(err))
	}
	page, err := searcher.Search(ctx, request)
	if err != nil {
		return errorToolResult(mapSearchJudicialCasesError(err))
	}
	if err := page.Validate(); err != nil {
		return errorToolResult(newInternalErrorResult())
	}
	result, err := successSearchJudicialCasesToolResult(
		mapSearchJudicialCasesOutput(page),
	)
	if err != nil {
		return errorToolResult(newInternalErrorResult())
	}
	return result, nil
}

func isNilJudicialSearchPort(searcher judicialdecisionsearch.Port) bool {
	if searcher == nil {
		return true
	}
	value := reflect.ValueOf(searcher)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func decodeSearchJudicialCasesInput(
	arguments json.RawMessage,
) (judicialdecisionsearch.Request, error) {
	input := bytes.TrimSpace(arguments)
	if len(input) == 0 ||
		!utf8.Valid(input) ||
		input[0] != '{' ||
		input[len(input)-1] != '}' {
		return judicialdecisionsearch.Request{},
			fmt.Errorf("入力は JSON object でなければなりません")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil || fields == nil {
		return judicialdecisionsearch.Request{}, fmt.Errorf("入力は JSON object でなければなりません")
	}
	for key := range fields {
		switch key {
		case "query", "limit", "continuationToken":
		default:
			return judicialdecisionsearch.Request{}, fmt.Errorf("定義していない入力項目は使用できません")
		}
	}

	var values judicialdecisionsearch.RequestValues
	queryRaw, exists := fields["query"]
	if !exists || isJSONNull(queryRaw) {
		return judicialdecisionsearch.Request{}, fmt.Errorf("query は必須です")
	}
	if err := decodeStrictJSONString(queryRaw, &values.Query); err != nil {
		return judicialdecisionsearch.Request{}, fmt.Errorf("query は文字列でなければなりません")
	}
	if raw, ok := fields["limit"]; ok {
		if isJSONNull(raw) {
			return judicialdecisionsearch.Request{}, fmt.Errorf("limit に null は使用できません")
		}
		limit, err := decodeExactJudicialSearchLimit(raw)
		if err != nil {
			return judicialdecisionsearch.Request{}, fmt.Errorf("limit は整数でなければなりません")
		}
		values.Limit = &limit
	}
	if raw, ok := fields["continuationToken"]; ok {
		if isJSONNull(raw) {
			return judicialdecisionsearch.Request{}, fmt.Errorf("continuationToken に null は使用できません")
		}
		if err := decodeStrictJSONString(raw, &values.ContinuationToken); err != nil {
			return judicialdecisionsearch.Request{}, fmt.Errorf("continuationToken は文字列でなければなりません")
		}
	}
	return judicialdecisionsearch.NewRequest(values)
}

func decodeStrictJSONString(raw json.RawMessage, destination *string) error {
	if !hasValidJSONSurrogatePairs(raw) {
		return fmt.Errorf("surrogate pair の Unicode 表現が有効ではありません")
	}
	return json.Unmarshal(raw, destination)
}

func hasValidJSONSurrogatePairs(raw json.RawMessage) bool {
	for index := 0; index < len(raw); index++ {
		if raw[index] != '\\' {
			continue
		}
		index++
		if index >= len(raw) {
			return false
		}
		if raw[index] != 'u' {
			continue
		}
		code, exists := decodeJSONHexCodeUnit(raw, index+1)
		if !exists {
			return false
		}
		index += 4
		switch {
		case code >= 0xd800 && code <= 0xdbff:
			if index+6 >= len(raw) ||
				raw[index+1] != '\\' ||
				raw[index+2] != 'u' {
				return false
			}
			low, lowExists := decodeJSONHexCodeUnit(raw, index+3)
			if !lowExists || low < 0xdc00 || low > 0xdfff {
				return false
			}
			index += 6
		case code >= 0xdc00 && code <= 0xdfff:
			return false
		}
	}
	return true
}

func decodeJSONHexCodeUnit(raw []byte, start int) (uint16, bool) {
	if start < 0 || start+4 > len(raw) {
		return 0, false
	}
	var value uint16
	for _, character := range raw[start : start+4] {
		value <<= 4
		switch {
		case character >= '0' && character <= '9':
			value += uint16(character - '0')
		case character >= 'a' && character <= 'f':
			value += uint16(character-'a') + 10
		case character >= 'A' && character <= 'F':
			value += uint16(character-'A') + 10
		default:
			return 0, false
		}
	}
	return value, true
}

func decodeExactJudicialSearchLimit(raw json.RawMessage) (int, error) {
	input := bytes.TrimSpace(raw)
	if len(input) == 0 ||
		(input[0] != '-' && (input[0] < '0' || input[0] > '9')) {
		return 0, fmt.Errorf("JSON number ではありません")
	}
	var number json.Number
	if err := json.Unmarshal(input, &number); err != nil {
		return 0, fmt.Errorf("JSON number ではありません")
	}
	value, err := exactSmallInteger(number.String())
	if err != nil || value < 1 || value > judicialdecisionsearch.MaxLimit {
		return 0, fmt.Errorf("1 以上 30 以下の整数ではありません")
	}
	return value, nil
}

func exactSmallInteger(number string) (int, error) {
	sign := 1
	if strings.HasPrefix(number, "-") {
		sign = -1
		number = number[1:]
	}
	coefficient := number
	exponentText := ""
	if index := strings.IndexAny(number, "eE"); index >= 0 {
		coefficient = number[:index]
		exponentText = number[index+1:]
	}
	exponent := int64(0)
	if exponentText != "" {
		parsed, err := strconv.ParseInt(exponentText, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("指数を安全に解釈できません")
		}
		exponent = parsed
	}

	integerPart := coefficient
	fractionPart := ""
	if index := strings.IndexByte(coefficient, '.'); index >= 0 {
		integerPart = coefficient[:index]
		fractionPart = coefficient[index+1:]
	}
	digits := integerPart + fractionPart
	fractionLength := int64(len(fractionPart))
	if exponent < -int64(len(digits))-1 {
		return 0, fmt.Errorf("整数ではありません")
	}
	scale := exponent - fractionLength
	if scale < 0 {
		if scale < -int64(len(digits)) {
			return 0, fmt.Errorf("整数ではありません")
		}
		remove := int(-scale)
		if strings.Trim(digits[len(digits)-remove:], "0") != "" {
			return 0, fmt.Errorf("整数ではありません")
		}
		digits = digits[:len(digits)-remove]
	} else if scale > 0 {
		trimmed := strings.TrimLeft(digits, "0")
		if trimmed == "" ||
			scale > 2 ||
			len(trimmed) > 2 ||
			int64(len(trimmed))+scale > 2 {
			return 0, fmt.Errorf("対応範囲の整数ではありません")
		}
		digits += strings.Repeat("0", int(scale))
	}
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		return 0, nil
	}
	if len(digits) > 2 {
		return 0, fmt.Errorf("対応範囲の整数ではありません")
	}
	value, err := strconv.Atoi(digits)
	if err != nil {
		return 0, fmt.Errorf("整数へ変換できません")
	}
	return sign * value, nil
}

func mapSearchJudicialCasesOutput(
	page judicialdecisionsearch.Page,
) searchJudicialCasesOutput {
	items := page.Items()
	outputItems := make([]searchJudicialCasesOutputItem, len(items))
	for index, item := range items {
		outputItems[index] = searchJudicialCasesOutputItem{
			Ref:        mapSearchJudicialCasesRef(item.Ref()),
			Provenance: mapSearchJudicialCasesProvenance(item.Provenance()),
			Data:       mapSearchJudicialCasesSummary(item.Data()),
		}
	}
	return searchJudicialCasesOutput{
		CoverageNotice: judicialCasesCoverageNotice,
		Items:          outputItems,
		Page:           mapSearchJudicialCasesPage(page.Page()),
	}
}

func mapSearchJudicialCasesRef(
	ref model.SourceResourceRef,
) searchJudicialCasesOutputRef {
	return searchJudicialCasesOutputRef{
		ProviderID: ref.ProviderID(),
		Key:        mapSearchJudicialCasesResourceKey(ref.Key()),
	}
}

func mapSearchJudicialCasesResourceKey(
	key model.SourceResourceKey,
) searchJudicialCasesOutputResourceKey {
	output := searchJudicialCasesOutputResourceKey{
		SourceID:     key.SourceID(),
		ResourceType: key.ResourceType(),
		ResourceID:   key.ResourceID(),
	}
	if versionID, exists := key.VersionID(); exists {
		output.VersionID = versionID
	}
	return output
}

func mapSearchJudicialCasesProvenance(
	provenance []model.Provenance,
) []searchJudicialCasesOutputProv {
	output := make([]searchJudicialCasesOutputProv, len(provenance))
	for index, item := range provenance {
		output[index] = searchJudicialCasesOutputProv{
			Source:         mapSearchJudicialCasesSource(item.Source()),
			ResourceKey:    mapSearchJudicialCasesResourceKey(item.ResourceKey()),
			URL:            item.URL(),
			RetrievedAt:    item.RetrievedAt().Format(time.RFC3339Nano),
			MediaType:      item.MediaType(),
			Transformation: item.Transformation(),
		}
		if sourceUpdatedAt, exists := item.SourceUpdatedAt(); exists {
			output[index].SourceUpdatedAt = sourceUpdatedAt
		}
		if location, exists := item.Location(); exists {
			output[index].Location = location
		}
		if methodID, exists := item.MethodID(); exists {
			output[index].MethodID = methodID
		}
		if inputKeys, exists := item.InputKeys(); exists {
			mappedInputKeys := make(
				[]searchJudicialCasesOutputResourceKey,
				len(inputKeys),
			)
			for inputIndex, inputKey := range inputKeys {
				mappedInputKeys[inputIndex] = mapSearchJudicialCasesResourceKey(inputKey)
			}
			output[index].InputKeys = &mappedInputKeys
		}
		if contentDigest, exists := item.ContentDigest(); exists {
			output[index].ContentDigest = contentDigest
		}
	}
	return output
}

func mapSearchJudicialCasesSummary(
	summary model.JudicialDecisionSummary,
) searchJudicialCasesOutputSummary {
	output := searchJudicialCasesOutputSummary{
		DecisionID:          summary.DecisionID(),
		PublicationCategory: summary.PublicationCategory(),
		SourceCategoryLabel: summary.SourceCategoryLabel(),
		CaseNumber:          summary.CaseNumber(),
		DecisionDate:        summary.DecisionDate().String(),
		CourtName:           summary.CourtName(),
		DetailURL:           summary.DetailURL(),
		Documents:           mapSearchJudicialCasesDocuments(summary.Documents()),
		Source:              mapSearchJudicialCasesSource(summary.Source()),
	}
	if caseName, exists := summary.CaseName(); exists {
		output.CaseName = caseName
	}
	if branchName, exists := summary.BranchName(); exists {
		output.BranchName = branchName
	}
	if divisionName, exists := summary.DivisionName(); exists {
		output.DivisionName = divisionName
	}
	if decisionType, exists := summary.DecisionType(); exists {
		output.DecisionType = decisionType
	}
	if outcome, exists := summary.Outcome(); exists {
		output.Outcome = outcome
	}
	return output
}

func mapSearchJudicialCasesDocuments(
	documents []model.JudicialDocumentLink,
) []searchJudicialCasesOutputDocument {
	output := make([]searchJudicialCasesOutputDocument, len(documents))
	for index, document := range documents {
		output[index] = searchJudicialCasesOutputDocument{
			Kind:      document.Kind(),
			Label:     document.Label(),
			MediaType: document.MediaType(),
			URL:       document.URL(),
		}
	}
	return output
}

func mapSearchJudicialCasesSource(
	source model.InformationSource,
) searchJudicialCasesOutputSource {
	return searchJudicialCasesOutputSource{
		ID:         source.ID(),
		Name:       source.Name(),
		Publisher:  source.Publisher(),
		Authority:  string(source.Authority()),
		ServiceURL: source.ServiceURL(),
	}
}

func mapSearchJudicialCasesPage(page model.SourcePage) searchJudicialCasesOutputPage {
	output := searchJudicialCasesOutputPage{
		ReturnedCount: page.ReturnedCount(),
	}
	if nextToken, exists := page.NextToken(); exists {
		output.NextToken = nextToken
	}
	if totalCount, exists := page.TotalCount(); exists {
		output.TotalCount = &totalCount
	}
	if totalRelation, exists := page.TotalRelation(); exists {
		output.TotalRelation = totalRelation
	}
	return output
}

func mapSearchJudicialCasesInvalidArgument(err error) model.ErrorResult {
	if result, exists := judicialSearchArgumentErrorResult(err); exists {
		return result
	}
	return newInvalidArgumentResult(err)
}

func judicialSearchArgumentErrorResult(
	err error,
) (model.ErrorResult, bool) {
	var argumentError judicialdecisionsearch.ArgumentError
	if errors.As(err, &argumentError) {
		result, buildErr := model.NewErrorResult(model.ErrorResultValues{
			Code: model.ErrorCodeInvalidArgument,
			Details: map[string]any{
				"field":  argumentError.Field(),
				"reason": argumentError.Reason(),
			},
		})
		if buildErr == nil {
			return result, true
		}
	}
	return model.ErrorResult{}, false
}

func mapSearchJudicialCasesError(err error) model.ErrorResult {
	if result, exists := judicialSearchArgumentErrorResult(err); exists {
		return result
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

func successSearchJudicialCasesToolResult(
	output searchJudicialCasesOutput,
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
