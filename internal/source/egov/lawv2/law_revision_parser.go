package lawv2

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	lawRevisionParserInputBytes = 16 * 1024 * 1024
	lawRevisionJSONValues       = 200000
	lawRevisionJSONDepth        = 32
	lawRevisionParseTimeout     = 3 * time.Second
)

func parseLawRevisionResponse(
	ctx context.Context,
	body []byte,
) (lawRevisionResponse, error) {
	if ctx == nil {
		return lawRevisionResponse{}, errors.New("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return lawRevisionResponse{}, normalizeContextErrorWithFactory(
			err,
			newLawRevisionSourceError,
		)
	}
	if len(body) > lawRevisionParserInputBytes {
		return lawRevisionResponse{}, newLawRevisionSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	if !utf8.Valid(body) {
		return lawRevisionResponse{}, newLawRevisionSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}

	parseContext, cancel := context.WithTimeout(ctx, lawRevisionParseTimeout)
	defer cancel()
	if err := validateJSONWithBudget(
		parseContext,
		body,
		lawRevisionJSONValues,
		lawRevisionJSONDepth,
	); err != nil {
		return lawRevisionResponse{}, classifyLawRevisionParseError(
			ctx,
			parseContext,
			err,
		)
	}

	var raw rawLawRevisionResponse
	if err := decodeLawSearchJSON(parseContext, body, &raw); err != nil {
		return lawRevisionResponse{}, classifyLawRevisionDecodeError(
			ctx,
			parseContext,
		)
	}
	return decodeLawRevisionResponse(raw)
}

func decodeLawRevisionResponse(
	raw rawLawRevisionResponse,
) (lawRevisionResponse, error) {
	if len(raw.LawInfo) == 0 || isJSONNull(raw.LawInfo) ||
		len(raw.Revisions) == 0 || isJSONNull(raw.Revisions) {
		return invalidLawRevisionResponse()
	}
	var rawLawInfo rawLawRevisionLawInfo
	if err := json.Unmarshal(raw.LawInfo, &rawLawInfo); err != nil {
		return invalidLawRevisionResponse()
	}
	lawID, err := decodeLawRevisionRequiredString(rawLawInfo.LawID)
	if err != nil {
		return lawRevisionResponse{}, err
	}
	lawNumber, err := decodeLawRevisionOptionalString(rawLawInfo.LawNumber)
	if err != nil {
		return lawRevisionResponse{}, err
	}
	promulgationDate, err := decodeLawRevisionOptionalString(rawLawInfo.PromulgationDate)
	if err != nil {
		return lawRevisionResponse{}, err
	}

	var rawRevisions []rawLawRevisionRecord
	if err := json.Unmarshal(raw.Revisions, &rawRevisions); err != nil {
		return invalidLawRevisionResponse()
	}
	revisions := make([]lawRevisionRecord, len(rawRevisions))
	seen := make(map[string]struct{}, len(rawRevisions))
	for index, rawRevision := range rawRevisions {
		revision, decodeErr := decodeLawRevisionRecord(rawRevision)
		if decodeErr != nil {
			return lawRevisionResponse{}, decodeErr
		}
		if _, exists := seen[revision.revisionID]; exists {
			return invalidLawRevisionResponse()
		}
		seen[revision.revisionID] = struct{}{}
		revisions[index] = revision
	}
	return lawRevisionResponse{
		lawID:            lawID,
		lawNumber:        lawNumber,
		promulgationDate: promulgationDate,
		revisions:        revisions,
	}, nil
}

func decodeLawRevisionRecord(raw rawLawRevisionRecord) (lawRevisionRecord, error) {
	required := []json.RawMessage{raw.RevisionID, raw.Title}
	requiredValues := make([]string, len(required))
	for index, value := range required {
		decoded, err := decodeLawRevisionRequiredString(value)
		if err != nil {
			return lawRevisionRecord{}, err
		}
		requiredValues[index] = decoded
	}

	optional := []json.RawMessage{
		raw.LawType,
		raw.TitleKana,
		raw.Abbreviation,
		raw.Category,
		raw.SourceUpdatedAt,
		raw.AmendmentPromulgationDate,
		raw.EffectiveDate,
		raw.EffectiveDateNote,
		raw.ScheduledEffectiveDate,
		raw.AmendmentLawID,
		raw.AmendmentLawTitle,
		raw.AmendmentLawTitleKana,
		raw.AmendmentLawNumber,
		raw.AmendmentType,
		raw.RepealStatus,
		raw.RepealRecordedDate,
		raw.Mission,
		raw.CurrentStatus,
	}
	optionalValues := make([]string, len(optional))
	for index, value := range optional {
		decoded, err := decodeLawRevisionOptionalString(value)
		if err != nil {
			return lawRevisionRecord{}, err
		}
		optionalValues[index] = decoded
	}
	remainInForce, err := decodeLawRevisionOptionalBool(raw.RemainInForce)
	if err != nil {
		return lawRevisionRecord{}, err
	}
	return lawRevisionRecord{
		revisionID:                requiredValues[0],
		title:                     requiredValues[1],
		lawType:                   optionalValues[0],
		titleKana:                 optionalValues[1],
		abbreviation:              optionalValues[2],
		category:                  optionalValues[3],
		sourceUpdatedAt:           optionalValues[4],
		amendmentPromulgationDate: optionalValues[5],
		effectiveDate:             optionalValues[6],
		effectiveDateNote:         optionalValues[7],
		scheduledEffectiveDate:    optionalValues[8],
		amendmentLawID:            optionalValues[9],
		amendmentLawTitle:         optionalValues[10],
		amendmentLawTitleKana:     optionalValues[11],
		amendmentLawNumber:        optionalValues[12],
		amendmentType:             optionalValues[13],
		repealStatus:              optionalValues[14],
		repealRecordedDate:        optionalValues[15],
		mission:                   optionalValues[16],
		currentStatus:             optionalValues[17],
		remainInForce:             remainInForce,
	}, nil
}

func decodeLawRevisionRequiredString(raw json.RawMessage) (string, error) {
	value, err := decodeLawRevisionOptionalString(raw)
	if err != nil {
		return "", err
	}
	if value == "" {
		_, invalidErr := invalidLawRevisionResponse()
		return "", invalidErr
	}
	return value, nil
}

func decodeLawRevisionOptionalString(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || isJSONNull(raw) {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		_, invalidErr := invalidLawRevisionResponse()
		return "", invalidErr
	}
	return value, nil
}

func decodeLawRevisionOptionalBool(raw json.RawMessage) (*bool, error) {
	if len(raw) == 0 || isJSONNull(raw) {
		return nil, nil
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		_, invalidErr := invalidLawRevisionResponse()
		return nil, invalidErr
	}
	return &value, nil
}

func classifyLawRevisionParseError(
	parent context.Context,
	parseContext context.Context,
	err error,
) error {
	if parentErr := parent.Err(); parentErr != nil {
		return normalizeContextErrorWithFactory(parentErr, newLawRevisionSourceError)
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(parseContext.Err(), context.DeadlineExceeded) {
		return newLawRevisionSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			"",
		)
	}
	switch {
	case errors.Is(err, lawSearchJSONErrorValues):
		return newLawRevisionSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	case errors.Is(err, lawSearchJSONErrorDepth):
		return newLawRevisionSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	default:
		return newLawRevisionSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
}

func classifyLawRevisionDecodeError(
	parent context.Context,
	parseContext context.Context,
) error {
	if err := parent.Err(); err != nil {
		return normalizeContextErrorWithFactory(err, newLawRevisionSourceError)
	}
	if errors.Is(parseContext.Err(), context.DeadlineExceeded) {
		return newLawRevisionSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			"",
		)
	}
	return newLawRevisionSourceError(
		model.SourceErrorCodeInvalidSourceResponse,
		"",
	)
}

func invalidLawRevisionResponse() (lawRevisionResponse, error) {
	return lawRevisionResponse{}, newLawRevisionSourceError(
		model.SourceErrorCodeInvalidSourceResponse,
		"",
	)
}
