package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type getJudicialCaseInputSchema struct {
	Ref getJudicialCaseInputRef `json:"ref"`
}

type getJudicialCaseInputRef struct {
	ProviderID string                  `json:"providerId"`
	Key        getJudicialCaseInputKey `json:"key"`
}

type getJudicialCaseInputKey struct {
	SourceID     string `json:"sourceId"`
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
}

type getJudicialCaseOutput struct {
	CoverageNotice string                    `json:"coverageNotice"`
	Item           getJudicialCaseOutputItem `json:"item"`
}

type getJudicialCaseOutputItem struct {
	Ref        searchJudicialCasesOutputRef    `json:"ref"`
	Provenance []searchJudicialCasesOutputProv `json:"provenance"`
	Data       getJudicialCaseOutputDetails    `json:"data"`
}

type getJudicialCaseOutputDetails struct {
	Summary                  searchJudicialCasesOutputSummary `json:"summary"`
	ReporterCitation         string                           `json:"reporterCitation,omitempty"`
	LowerCourtName           string                           `json:"lowerCourtName,omitempty"`
	LowerCourtCaseNumber     string                           `json:"lowerCourtCaseNumber,omitempty"`
	LowerCourtDecisionDate   string                           `json:"lowerCourtDecisionDate,omitempty"`
	HoldingText              string                           `json:"holdingText,omitempty"`
	SummaryText              string                           `json:"summaryText,omitempty"`
	ReferencedProvisionsText string                           `json:"referencedProvisionsText,omitempty"`
}

func addGetJudicialCaseTool(
	server toolRegistrar,
	reader judicialdecisionread.Port,
) {
	server.AddTool(&sdk.Tool{
		Name:         "get_judicial_case",
		Description:  "検索結果の参照を使い、同じ裁判所プロバイダーから公式裁判例の詳細を取得します。",
		InputSchema:  mustSchemaFor[getJudicialCaseInputSchema](),
		OutputSchema: mustSchemaFor[getJudicialCaseOutput](),
	}, func(ctx context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return callGetJudicialCase(ctx, reader, request.Params.Arguments)
	})
}

func callGetJudicialCase(
	ctx context.Context,
	reader judicialdecisionread.Port,
	arguments json.RawMessage,
) (*sdk.CallToolResult, error) {
	if isNilJudicialReadPort(reader) {
		return errorToolResult(newInternalErrorResult())
	}
	request, err := decodeGetJudicialCaseInput(arguments)
	if err != nil {
		return errorToolResult(mapGetJudicialCaseInvalidArgument(err))
	}
	item, err := reader.Read(ctx, request)
	if err != nil {
		return errorToolResult(mapGetJudicialCaseError(err))
	}
	if err := item.Validate(); err != nil ||
		!sameJudicialResourceRef(item.Ref(), request.Ref()) {
		return errorToolResult(newInternalErrorResult())
	}
	result, err := successGetJudicialCaseToolResult(
		mapGetJudicialCaseOutput(item),
	)
	if err != nil {
		return errorToolResult(newInternalErrorResult())
	}
	return result, nil
}

func isNilJudicialReadPort(reader judicialdecisionread.Port) bool {
	if reader == nil {
		return true
	}
	value := reflect.ValueOf(reader)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func decodeGetJudicialCaseInput(
	arguments json.RawMessage,
) (judicialdecisionread.Request, error) {
	input := bytes.TrimSpace(arguments)
	if len(input) == 0 ||
		!utf8.Valid(input) ||
		input[0] != '{' ||
		input[len(input)-1] != '}' {
		return judicialdecisionread.Request{},
			newGetJudicialCaseInputError("は JSON object でなければなりません")
	}
	fields, err := decodeGetJudicialCaseObject(
		input,
		map[string]struct{}{"ref": {}},
	)
	if err != nil {
		return judicialdecisionread.Request{}, err
	}
	refRaw, exists := fields["ref"]
	if !exists || isJSONNull(refRaw) {
		return judicialdecisionread.Request{},
			newGetJudicialCaseInputError("は必須です")
	}
	refFields, err := decodeGetJudicialCaseObject(
		refRaw,
		map[string]struct{}{
			"providerId": {},
			"key":        {},
		},
	)
	if err != nil {
		return judicialdecisionread.Request{}, err
	}
	providerID, err := decodeGetJudicialCaseRequiredString(
		refFields,
		"providerId",
	)
	if err != nil {
		return judicialdecisionread.Request{}, err
	}
	keyRaw, exists := refFields["key"]
	if !exists || isJSONNull(keyRaw) {
		return judicialdecisionread.Request{},
			newGetJudicialCaseInputError("の key は必須です")
	}
	keyFields, err := decodeGetJudicialCaseObject(
		keyRaw,
		map[string]struct{}{
			"sourceId":     {},
			"resourceType": {},
			"resourceId":   {},
			"versionId":    {},
		},
	)
	if err != nil {
		return judicialdecisionread.Request{}, err
	}
	if _, specified := keyFields["versionId"]; specified {
		return judicialdecisionread.Request{},
			newGetJudicialCaseInputError("に versionId は指定できません")
	}
	sourceID, err := decodeGetJudicialCaseRequiredString(
		keyFields,
		"sourceId",
	)
	if err != nil {
		return judicialdecisionread.Request{}, err
	}
	resourceType, err := decodeGetJudicialCaseRequiredString(
		keyFields,
		"resourceType",
	)
	if err != nil {
		return judicialdecisionread.Request{}, err
	}
	resourceID, err := decodeGetJudicialCaseRequiredString(
		keyFields,
		"resourceId",
	)
	if err != nil {
		return judicialdecisionread.Request{}, err
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
	if err != nil {
		return judicialdecisionread.Request{},
			newGetJudicialCaseInputError("の key が有効ではありません")
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		return judicialdecisionread.Request{},
			newGetJudicialCaseInputError("が有効ではありません")
	}
	return judicialdecisionread.NewRequest(
		judicialdecisionread.RequestValues{Ref: ref},
	)
}

func decodeGetJudicialCaseObject(
	raw json.RawMessage,
	allowed map[string]struct{},
) (map[string]json.RawMessage, error) {
	value := bytes.TrimSpace(raw)
	if len(value) == 0 ||
		!utf8.Valid(value) ||
		value[0] != '{' ||
		value[len(value)-1] != '}' {
		return nil, newGetJudicialCaseInputError(
			"の構造は JSON object でなければなりません",
		)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(value, &fields); err != nil || fields == nil {
		return nil, newGetJudicialCaseInputError(
			"の構造は JSON object でなければなりません",
		)
	}
	for key := range fields {
		if _, exists := allowed[key]; !exists {
			return nil, newGetJudicialCaseInputError(
				"に定義していない項目は使用できません",
			)
		}
	}
	return fields, nil
}

func decodeGetJudicialCaseRequiredString(
	fields map[string]json.RawMessage,
	name string,
) (string, error) {
	raw, exists := fields[name]
	if !exists || isJSONNull(raw) {
		return "", newGetJudicialCaseInputError(
			"の " + name + " は必須です",
		)
	}
	var value string
	if err := decodeStrictJSONString(raw, &value); err != nil {
		return "", newGetJudicialCaseInputError(
			"の " + name + " は文字列でなければなりません",
		)
	}
	if value == "" {
		return "", newGetJudicialCaseInputError(
			"の " + name + " は空にできません",
		)
	}
	return value, nil
}

func newGetJudicialCaseInputError(reason string) error {
	result, err := judicialdecisionread.NewArgumentError("ref", reason)
	if err != nil {
		return fmt.Errorf("裁判例詳細取得の入力エラーを作成できません: %w", err)
	}
	return result
}

func sameJudicialResourceRef(
	left model.SourceResourceRef,
	right model.SourceResourceRef,
) bool {
	if left.ProviderID() != right.ProviderID() {
		return false
	}
	leftKey := left.Key()
	rightKey := right.Key()
	leftVersion, leftHasVersion := leftKey.VersionID()
	rightVersion, rightHasVersion := rightKey.VersionID()
	return leftKey.SourceID() == rightKey.SourceID() &&
		leftKey.ResourceType() == rightKey.ResourceType() &&
		leftKey.ResourceID() == rightKey.ResourceID() &&
		leftHasVersion == rightHasVersion &&
		leftVersion == rightVersion
}

func mapGetJudicialCaseOutput(
	item model.SourcedResource[model.JudicialDecisionDetails],
) getJudicialCaseOutput {
	data := item.Data()
	outputData := getJudicialCaseOutputDetails{
		Summary: mapSearchJudicialCasesSummary(data.Summary()),
	}
	if value, exists := data.ReporterCitation(); exists {
		outputData.ReporterCitation = value
	}
	if value, exists := data.LowerCourtName(); exists {
		outputData.LowerCourtName = value
	}
	if value, exists := data.LowerCourtCaseNumber(); exists {
		outputData.LowerCourtCaseNumber = value
	}
	if value, exists := data.LowerCourtDecisionDate(); exists {
		outputData.LowerCourtDecisionDate = value.String()
	}
	if value, exists := data.HoldingText(); exists {
		outputData.HoldingText = value
	}
	if value, exists := data.SummaryText(); exists {
		outputData.SummaryText = value
	}
	if value, exists := data.ReferencedProvisionsText(); exists {
		outputData.ReferencedProvisionsText = value
	}
	return getJudicialCaseOutput{
		CoverageNotice: judicialCasesCoverageNotice,
		Item: getJudicialCaseOutputItem{
			Ref:        mapSearchJudicialCasesRef(item.Ref()),
			Provenance: mapSearchJudicialCasesProvenance(item.Provenance()),
			Data:       outputData,
		},
	}
}

func mapGetJudicialCaseInvalidArgument(err error) model.ErrorResult {
	if result, exists := judicialReadArgumentErrorResult(err); exists {
		return result
	}
	return newInvalidArgumentResult(err)
}

func judicialReadArgumentErrorResult(
	err error,
) (model.ErrorResult, bool) {
	var argumentError judicialdecisionread.ArgumentError
	if !errors.As(err, &argumentError) {
		return model.ErrorResult{}, false
	}
	result, buildErr := model.NewErrorResult(model.ErrorResultValues{
		Code: model.ErrorCodeInvalidArgument,
		Details: map[string]any{
			"field":  argumentError.Field(),
			"reason": argumentError.Reason(),
		},
	})
	if buildErr != nil {
		return model.ErrorResult{}, false
	}
	return result, true
}

func mapGetJudicialCaseError(err error) model.ErrorResult {
	if errors.Is(err, judicialdecisionread.ErrNotFound) {
		return newFixedErrorResult(model.ErrorCodeNotFound)
	}
	if result, exists := judicialReadArgumentErrorResult(err); exists {
		return result
	}
	if result, exists := judicialReadResolutionErrorResult(err); exists {
		return result
	}
	var sourceError model.SourceError
	if errors.As(err, &sourceError) &&
		getJudicialCaseSourceErrorAllowed(sourceError.Code()) {
		result, buildErr := model.NewErrorResultFromSourceError(sourceError)
		if buildErr == nil {
			return result
		}
	}
	return newInternalErrorResult()
}

func judicialReadResolutionErrorResult(
	err error,
) (model.ErrorResult, bool) {
	var resolutionError judicialdecisionread.ResolutionError
	if !errors.As(err, &resolutionError) {
		return model.ErrorResult{}, false
	}
	values := model.ErrorResultValues{Code: resolutionError.Code()}
	switch resolutionError.Code() {
	case model.ErrorCodeInvalidArgument:
		values.Details = map[string]any{
			"field":  "ref",
			"reason": resolutionError.Error(),
		}
	case model.ErrorCodeConfigurationRequired,
		model.ErrorCodeUnsupportedCapability:
		values.Details = map[string]any{
			"providerId":   resolutionError.ProviderID(),
			"capabilityId": judicialdecisionread.CapabilityID,
		}
	default:
		return model.ErrorResult{}, false
	}
	result, buildErr := model.NewErrorResult(values)
	if buildErr != nil {
		return model.ErrorResult{}, false
	}
	return result, true
}

func getJudicialCaseSourceErrorAllowed(code model.SourceErrorCode) bool {
	switch code {
	case model.SourceErrorCodeUnsupportedCapability,
		model.SourceErrorCodeConfigurationRequired,
		model.SourceErrorCodeSourceAuthFailed,
		model.SourceErrorCodeRateLimited,
		model.SourceErrorCodeSourceTimeout,
		model.SourceErrorCodeSourceUnavailable,
		model.SourceErrorCodeSourceBusy,
		model.SourceErrorCodeSourceContractChanged,
		model.SourceErrorCodeInvalidSourceResponse,
		model.SourceErrorCodeSourceResponseTooLarge,
		model.SourceErrorCodeSourceProcessingLimit,
		model.SourceErrorCodeUnsafeSourceContent:
		return true
	default:
		return false
	}
}

func successGetJudicialCaseToolResult(
	output getJudicialCaseOutput,
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
