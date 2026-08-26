package mcp

import (
	"net/http"
	"strconv"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitingcandidatesearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	traceJudicialCitationsPDFProviderID   = "courts-hanrei-pdf"
	traceJudicialCitationsReadOperation   = "GET /hanrei/{id}/detail{2..8}/index.html"
	traceJudicialCitationsSearchOperation = "GET /hanrei/search1/index.html"
	traceJudicialCitationsFetchOperation  = "GET /assets/hanrei/{id}.pdf"
	traceJudicialCitationsParseOperation  = "private worker parse"
)

func sanitizeTraceJudicialCitationsOperationResult(
	result model.ErrorResult,
) (model.ErrorResult, bool) {
	if err := result.Validate(); err != nil {
		return model.ErrorResult{}, false
	}
	code := result.Code()
	switch code {
	case model.ErrorCodeNotFound, model.ErrorCodeInternalError:
		if _, exists := result.Details(); exists {
			return model.ErrorResult{}, false
		}
		return rebuildTraceJudicialCitationsError(code, nil)
	case model.ErrorCodeUnsupportedCapability,
		model.ErrorCodeUnsupportedQuery,
		model.ErrorCodeConfigurationRequired:
		return sanitizeTraceJudicialCitationsSourceDetails(result, false, false)
	case model.ErrorCodeSourceAuthFailed,
		model.ErrorCodeSourceTimeout,
		model.ErrorCodeSourceBusy,
		model.ErrorCodeSourceContractChanged,
		model.ErrorCodeInvalidSourceResponse,
		model.ErrorCodeSourceResponseTooLarge,
		model.ErrorCodeSourceProcessingLimit,
		model.ErrorCodeUnsafeSourceContent:
		return sanitizeTraceJudicialCitationsSourceDetails(result, true, false)
	case model.ErrorCodeRateLimited, model.ErrorCodeSourceUnavailable:
		return sanitizeTraceJudicialCitationsSourceDetails(result, true, true)
	default:
		return model.ErrorResult{}, false
	}
}

func sanitizeTraceJudicialCitationsSourceDetails(
	result model.ErrorResult,
	requireOperation bool,
	allowRetryAfter bool,
) (model.ErrorResult, bool) {
	details, exists := result.Details()
	if !exists {
		return model.ErrorResult{}, false
	}
	providerID, providerOK := traceJudicialCitationsDetailString(details, "providerId")
	sourceID, sourceOK := traceJudicialCitationsDetailString(details, "sourceId")
	capabilityID, capabilityOK := traceJudicialCitationsDetailString(details, "capabilityId")
	if !providerOK || !sourceOK || !capabilityOK ||
		sourceID != traceJudicialCitationsSourceID ||
		!traceJudicialCitationsSourceIdentityAllowed(providerID, capabilityID) {
		return model.ErrorResult{}, false
	}
	safe := map[string]any{
		"providerId":   providerID,
		"sourceId":     sourceID,
		"capabilityId": capabilityID,
	}
	if requireOperation {
		operation, operationOK := traceJudicialCitationsDetailString(details, "operation")
		if !operationOK || !traceJudicialCitationsOperationAllowed(
			providerID,
			capabilityID,
			operation,
		) {
			return model.ErrorResult{}, false
		}
		safe["operation"] = operation
	}
	if retryAfter, hasRetryAfter := details["retryAfter"]; hasRetryAfter {
		value, ok := retryAfter.(string)
		if !allowRetryAfter || !ok || !traceJudicialCitationsRetryAfterAllowed(value) {
			return model.ErrorResult{}, false
		}
		safe["retryAfter"] = value
	}
	if len(details) != len(safe) {
		return model.ErrorResult{}, false
	}
	return rebuildTraceJudicialCitationsError(result.Code(), safe)
}

func traceJudicialCitationsDetailString(
	details map[string]any,
	key string,
) (string, bool) {
	value, exists := details[key]
	if !exists {
		return "", false
	}
	text, ok := value.(string)
	return text, ok && text != ""
}

func traceJudicialCitationsSourceIdentityAllowed(providerID, capabilityID string) bool {
	switch providerID {
	case traceJudicialCitationsProviderID:
		return capabilityID == judicialdecisionread.CapabilityID ||
			capabilityID == judicialcitingcandidatesearch.CapabilityID
	case traceJudicialCitationsPDFProviderID:
		return capabilityID == judicialcasecitationextract.CapabilityID
	default:
		return false
	}
}

func traceJudicialCitationsOperationAllowed(providerID, capabilityID, operation string) bool {
	switch {
	case providerID == traceJudicialCitationsProviderID &&
		capabilityID == judicialdecisionread.CapabilityID:
		return operation == traceJudicialCitationsReadOperation
	case providerID == traceJudicialCitationsProviderID &&
		capabilityID == judicialcitingcandidatesearch.CapabilityID:
		return operation == traceJudicialCitationsSearchOperation
	case providerID == traceJudicialCitationsPDFProviderID &&
		capabilityID == judicialcasecitationextract.CapabilityID:
		return operation == traceJudicialCitationsFetchOperation ||
			operation == traceJudicialCitationsParseOperation
	default:
		return false
	}
}

func traceJudicialCitationsRetryAfterAllowed(value string) bool {
	if value == "" {
		return false
	}
	if _, err := strconv.ParseUint(value, 10, 31); err == nil {
		return true
	}
	_, err := http.ParseTime(value)
	return err == nil
}

func rebuildTraceJudicialCitationsError(
	code model.ErrorCode,
	details map[string]any,
) (model.ErrorResult, bool) {
	result, err := model.NewErrorResult(model.ErrorResultValues{
		Code:    code,
		Details: details,
	})
	return result, err == nil
}
