package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationtrace"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	traceJudicialCitationsProviderID = "courts-hanrei-html"
	traceJudicialCitationsSourceID   = "courts-hanrei"
)

func addTraceJudicialCitationsTool(
	server toolRegistrar,
	tracer judicialcitationtrace.Port,
) {
	server.AddTool(&sdk.Tool{
		Name:         "trace_judicial_citations",
		Description:  "一件の公式裁判例から 1-hop の判例引用関係 graph を返します。",
		InputSchema:  newTraceJudicialCitationsInputSchema(),
		OutputSchema: newTraceJudicialCitationsOutputSchema(),
	}, func(ctx context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return callTraceJudicialCitations(ctx, tracer, request.Params.Arguments)
	})
}

func callTraceJudicialCitations(
	ctx context.Context,
	tracer judicialcitationtrace.Port,
	arguments json.RawMessage,
) (*sdk.CallToolResult, error) {
	if isNilJudicialCitationTracePort(tracer) {
		return errorToolResult(newInternalErrorResult())
	}
	request, err := decodeTraceJudicialCitationsInput(arguments)
	if err != nil {
		return errorToolResult(mapTraceJudicialCitationsInvalidArgument(err))
	}
	result, err := tracer.Trace(ctx, request)
	if err != nil {
		return errorToolResult(mapTraceJudicialCitationsError(err))
	}
	if err := result.Validate(); err != nil {
		return errorToolResult(newInternalErrorResult())
	}
	if !traceJudicialCitationsResultMatchesRequest(result, request) {
		return errorToolResult(newInternalErrorResult())
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return errorToolResult(newInternalErrorResult())
	}
	return &sdk.CallToolResult{
		Content:           []sdk.Content{&sdk.TextContent{Text: string(payload)}},
		StructuredContent: json.RawMessage(payload),
		IsError:           false,
	}, nil
}

func traceJudicialCitationsResultMatchesRequest(
	result model.JudicialCitationGraphResult,
	request judicialcitationtrace.Request,
) bool {
	graph := result.Graph()
	coverage := graph.Coverage()
	if coverage.RequestedDirection() != request.Direction() {
		return false
	}
	var rootRef model.SourceResourceRef
	foundRoot := false
	for _, node := range graph.Nodes() {
		if node.NodeID() != graph.RootNodeID() {
			continue
		}
		rootRef, foundRoot = node.Ref()
		break
	}
	if !foundRoot || !sameJudicialResourceRef(rootRef, request.Ref()) {
		return false
	}
	if request.Direction() == model.JudicialCitationRequestedDirectionOutgoing {
		return true
	}
	limit, exists := coverage.Incoming().Limit()
	return exists && limit == request.IncomingLimit()
}

func decodeTraceJudicialCitationsInput(
	arguments json.RawMessage,
) (judicialcitationtrace.Request, error) {
	input := bytes.TrimSpace(arguments)
	if len(input) == 0 || !utf8.Valid(input) || input[0] != '{' || input[len(input)-1] != '}' {
		return judicialcitationtrace.Request{}, newTraceJudicialCitationsInputError(
			"ref",
			"入力全体は JSON object でなければなりません",
		)
	}
	fields, err := decodeGetJudicialCaseObject(input, map[string]struct{}{
		"ref":           {},
		"direction":     {},
		"incomingLimit": {},
	})
	if err != nil {
		return judicialcitationtrace.Request{}, newTraceJudicialCitationsInputError(
			"ref",
			"定義していない項目は使用できません",
		)
	}
	refRaw, exists := fields["ref"]
	if !exists || isJSONNull(refRaw) {
		return judicialcitationtrace.Request{}, newTraceJudicialCitationsInputError(
			"ref",
			"は必須です",
		)
	}
	refRequest, err := decodeGetJudicialCaseInput(json.RawMessage(`{"ref":` + string(refRaw) + `}`))
	if err != nil {
		return judicialcitationtrace.Request{}, newTraceJudicialCitationsInputError(
			"ref",
			"は検証済みの SourceResourceRef でなければなりません",
		)
	}
	ref := refRequest.Ref()
	if ref.ProviderID() != traceJudicialCitationsProviderID {
		return judicialcitationtrace.Request{}, newTraceJudicialCitationsInputError(
			"ref",
			"ref は courts-hanrei-html の judicial-decision 参照でなければなりません",
		)
	}
	if ref.Key().SourceID() != traceJudicialCitationsSourceID {
		return judicialcitationtrace.Request{}, newTraceJudicialCitationsInputError(
			"ref",
			"ref は courts-hanrei 情報源に属していなければなりません",
		)
	}
	values := judicialcitationtrace.RequestValues{Ref: ref}
	if raw, ok := fields["direction"]; ok {
		if isJSONNull(raw) {
			return judicialcitationtrace.Request{}, newTraceJudicialCitationsInputError(
				"direction",
				"に null は使用できません",
			)
		}
		var direction string
		if err := decodeStrictJSONString(raw, &direction); err != nil {
			return judicialcitationtrace.Request{}, newTraceJudicialCitationsInputError(
				"direction",
				"は文字列でなければなりません",
			)
		}
		values.Direction = model.JudicialCitationRequestedDirection(direction)
	}
	if raw, ok := fields["incomingLimit"]; ok {
		if isJSONNull(raw) {
			return judicialcitationtrace.Request{}, newTraceJudicialCitationsInputError(
				"incomingLimit",
				"に null は使用できません",
			)
		}
		limit, err := decodeExactJudicialSearchLimit(raw)
		if err != nil {
			return judicialcitationtrace.Request{}, newTraceJudicialCitationsInputError(
				"incomingLimit",
				"は 1 以上 10 以下の整数でなければなりません",
			)
		}
		values.IncomingLimit = &limit
	}
	return judicialcitationtrace.NewRequest(values)
}

func mapTraceJudicialCitationsInvalidArgument(err error) model.ErrorResult {
	var inputError traceJudicialCitationsInputError
	if errors.As(err, &inputError) {
		result, buildErr := model.NewErrorResult(model.ErrorResultValues{
			Code: model.ErrorCodeInvalidArgument,
			Details: map[string]any{
				"field":  inputError.field,
				"reason": inputError.reason,
			},
		})
		if buildErr == nil {
			return result
		}
	}
	var argumentError judicialcitationtrace.ArgumentError
	if errors.As(err, &argumentError) && argumentError.Validate() == nil {
		result, buildErr := model.NewErrorResult(model.ErrorResultValues{
			Code: model.ErrorCodeInvalidArgument,
			Details: map[string]any{
				"field":  argumentError.Field(),
				"reason": argumentError.Reason(),
			},
		})
		if buildErr == nil {
			return result
		}
	}
	return newInvalidArgumentResult(err)
}

func mapTraceJudicialCitationsError(err error) model.ErrorResult {
	var operationError judicialcitationtrace.OperationError
	if errors.As(err, &operationError) {
		if result, safe := sanitizeTraceJudicialCitationsOperationResult(
			operationError.Result(),
		); safe {
			return result
		}
	}
	return newInternalErrorResult()
}

type traceJudicialCitationsInputError struct {
	field  string
	reason string
}

func (e traceJudicialCitationsInputError) Error() string {
	return e.field + " " + e.reason
}

func newTraceJudicialCitationsInputError(field, reason string) error {
	return traceJudicialCitationsInputError{field: field, reason: reason}
}

func newTraceJudicialCitationsInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Schema:      queryLegalInformationSchemaVersion,
		Type:        "object",
		Description: "一件の裁判例参照から 1-hop の引用関係を取得します。",
		Properties: map[string]*jsonschema.Schema{
			"ref": {
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"providerId": queryLegalInformationConstString(
						traceJudicialCitationsProviderID,
					),
					"key": {
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"sourceId": queryLegalInformationConstString(
								traceJudicialCitationsSourceID,
							),
							"resourceType": queryLegalInformationConstString(
								"judicial-decision",
							),
							"resourceId": {
								Type:    "string",
								Pattern: `^[0-9]+/detail[2-8]$`,
							},
						},
						Required:             []string{"sourceId", "resourceType", "resourceId"},
						AdditionalProperties: queryLegalInformationFalseSchema(),
					},
				},
				Required:             []string{"providerId", "key"},
				AdditionalProperties: queryLegalInformationFalseSchema(),
			},
			"direction": {
				Type: "string",
				Enum: []any{
					string(model.JudicialCitationRequestedDirectionOutgoing),
					string(model.JudicialCitationRequestedDirectionIncoming),
					string(model.JudicialCitationRequestedDirectionBoth),
				},
				Default: json.RawMessage(`"both"`),
			},
			"incomingLimit": {
				Type:    "integer",
				Minimum: jsonschema.Ptr(1.0),
				Maximum: jsonschema.Ptr(10.0),
				Default: json.RawMessage("5"),
			},
		},
		Required:             []string{"ref"},
		AdditionalProperties: queryLegalInformationFalseSchema(),
	}
}
