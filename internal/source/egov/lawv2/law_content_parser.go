package lawv2

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"
	"unicode/utf8"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const (
	lawContentParserInputBytes = 32 * 1024 * 1024
	lawContentJSONValues       = 500000
	lawContentJSONDepth        = 64
	lawContentParseTimeout     = 5 * time.Second
)

func parseLawContentSearchResponse(
	ctx context.Context,
	body []byte,
	limit int,
	offset int,
) (lawContentSearchResponse, *int, error) {
	if err := ctx.Err(); err != nil {
		return lawContentSearchResponse{}, nil, err
	}
	if len(body) > lawContentParserInputBytes {
		return lawContentSearchResponse{}, nil, newLawContentSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	if !utf8.Valid(body) {
		return lawContentSearchResponse{}, nil, newLawContentSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}

	parseContext, cancel := context.WithTimeout(ctx, lawContentParseTimeout)
	defer cancel()

	if err := validateJSONWithBudget(
		parseContext,
		body,
		lawContentJSONValues,
		lawContentJSONDepth,
	); err != nil {
		return lawContentSearchResponse{}, nil, classifyLawContentParseError(
			ctx,
			parseContext,
			err,
		)
	}

	var raw rawLawContentSearchResponse
	if err := decodeLawSearchJSON(parseContext, body, &raw); err != nil {
		return lawContentSearchResponse{}, nil, classifyLawContentDecodeError(
			ctx,
			parseContext,
		)
	}
	response, nextOffset, err := decodeLawContentSearchResponse(raw)
	if err != nil {
		return lawContentSearchResponse{}, nil, err
	}
	next, err := validateLawContentSearchPage(response, nextOffset, limit, offset)
	if err != nil {
		return lawContentSearchResponse{}, nil, err
	}
	return response, next, nil
}

func decodeLawContentSearchResponse(
	raw rawLawContentSearchResponse,
) (lawContentSearchResponse, responseNextOffset, error) {
	totalCount, err := decodeLawContentInteger(raw.TotalCount, true)
	if err != nil {
		return lawContentSearchResponse{}, responseNextOffset{}, err
	}
	sentenceCount, err := decodeLawContentInteger(raw.SentenceCount, true)
	if err != nil {
		return lawContentSearchResponse{}, responseNextOffset{}, err
	}
	nextOffset, err := decodeLawContentNextOffset(raw.NextOffset)
	if err != nil {
		return lawContentSearchResponse{}, responseNextOffset{}, err
	}
	items, err := decodeLawContentSearchItems(raw.Items)
	if err != nil {
		return lawContentSearchResponse{}, responseNextOffset{}, err
	}
	return lawContentSearchResponse{
		totalCount:    totalCount,
		sentenceCount: sentenceCount,
		items:         items,
	}, nextOffset, nil
}

func decodeLawContentSearchItems(raw json.RawMessage) ([]lawContentSearchLaw, error) {
	if len(raw) == 0 {
		return nil, newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if isJSONNull(raw) {
		return nil, newLawContentSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	var values []rawLawContentSearchLaw
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, newLawContentSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	items := make([]lawContentSearchLaw, len(values))
	for index, value := range values {
		item, err := decodeLawContentSearchItem(value)
		if err != nil {
			return nil, err
		}
		items[index] = item
	}
	return items, nil
}

func decodeLawContentSearchItem(raw rawLawContentSearchLaw) (lawContentSearchLaw, error) {
	law, err := decodeLawContentLaw(raw.LawInfo, raw.RevisionInfo)
	if err != nil {
		return lawContentSearchLaw{}, err
	}
	sentences, err := decodeLawContentSentences(raw.Sentences)
	if err != nil {
		return lawContentSearchLaw{}, err
	}
	if len(sentences) == 0 {
		return lawContentSearchLaw{}, newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return lawContentSearchLaw{
		law:       law,
		sentences: sentences,
	}, nil
}

func decodeLawContentSentences(raw json.RawMessage) ([]lawContentSentence, error) {
	if len(raw) == 0 {
		return nil, newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if isJSONNull(raw) {
		return nil, newLawContentSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	var values []rawLawContentSentence
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, newLawContentSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	sentences := make([]lawContentSentence, len(values))
	for index, value := range values {
		position, err := decodeLawContentRequiredString(value.Position)
		if err != nil {
			return nil, err
		}
		text, err := decodeLawContentRequiredString(value.Text)
		if err != nil {
			return nil, err
		}
		sentences[index] = lawContentSentence{
			position: position,
			text:     text,
		}
	}
	return sentences, nil
}

func decodeLawContentInteger(
	raw json.RawMessage,
	officialRequired bool,
) (int, error) {
	if len(raw) == 0 {
		code := model.SourceErrorCodeInvalidSourceResponse
		if officialRequired {
			code = model.SourceErrorCodeSourceContractChanged
		}
		return 0, newLawContentSourceError(code, "")
	}
	if isJSONNull(raw) {
		return 0, newLawContentSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, newLawContentSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	if value > int64(maxInt()) || value < int64(minInt()) {
		return 0, newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return int(value), nil
}

func decodeLawContentNextOffset(
	raw json.RawMessage,
) (responseNextOffset, error) {
	if len(raw) == 0 {
		return responseNextOffset{}, nil
	}
	if isJSONNull(raw) {
		return responseNextOffset{present: true, isNull: true}, nil
	}
	value, err := decodeLawContentInteger(raw, false)
	if err != nil {
		return responseNextOffset{}, err
	}
	return responseNextOffset{present: true, value: value}, nil
}

func decodeLawContentLaw(
	rawLaw json.RawMessage,
	rawRevision json.RawMessage,
) (lawSearchLaw, error) {
	if len(rawLaw) == 0 ||
		len(rawRevision) == 0 ||
		isJSONNull(rawLaw) ||
		isJSONNull(rawRevision) {
		return lawSearchLaw{}, newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	var lawInfo rawLawInfo
	if err := json.Unmarshal(rawLaw, &lawInfo); err != nil {
		return lawSearchLaw{}, newLawContentSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	var revisionInfo rawRevisionInfo
	if err := json.Unmarshal(rawRevision, &revisionInfo); err != nil {
		return lawSearchLaw{}, newLawContentSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}

	lawID, err := decodeLawContentRequiredString(lawInfo.LawID)
	if err != nil {
		return lawSearchLaw{}, err
	}
	revisionID, err := decodeLawContentRequiredString(revisionInfo.RevisionID)
	if err != nil {
		return lawSearchLaw{}, err
	}
	title, err := decodeLawContentRequiredString(revisionInfo.Title)
	if err != nil {
		return lawSearchLaw{}, err
	}
	lawNumber, err := decodeLawContentOptionalString(lawInfo.LawNumber)
	if err != nil {
		return lawSearchLaw{}, err
	}
	promulgationDate, err := decodeLawContentOptionalString(
		lawInfo.PromulgationDate,
	)
	if err != nil {
		return lawSearchLaw{}, err
	}
	revisionEffectiveDate, err := decodeLawContentOptionalString(
		revisionInfo.RevisionEffectiveDate,
	)
	if err != nil {
		return lawSearchLaw{}, err
	}
	return lawSearchLaw{
		lawID:                 lawID,
		revisionID:            revisionID,
		title:                 title,
		lawNumber:             lawNumber,
		promulgationDate:      promulgationDate,
		revisionEffectiveDate: revisionEffectiveDate,
	}, nil
}

func decodeLawContentRequiredString(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || isJSONNull(raw) {
		return "", newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	value, err := decodeLawContentString(raw)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return value, nil
}

func decodeLawContentOptionalString(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	if isJSONNull(raw) {
		return "", newLawContentSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	return decodeLawContentString(raw)
}

func decodeLawContentString(raw json.RawMessage) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", newLawContentSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	return value, nil
}

func validateLawContentSearchPage(
	response lawContentSearchResponse,
	nextOffset responseNextOffset,
	limit int,
	offset int,
) (*int, error) {
	if response.totalCount < 0 || response.sentenceCount < 0 || response.sentenceCount > limit {
		return nil, newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}

	itemCount := 0
	for _, item := range response.items {
		if len(item.sentences) > maxInt()-itemCount {
			return nil, newLawContentSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
		itemCount += len(item.sentences)
	}
	if itemCount != response.sentenceCount {
		return nil, newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if offset < 0 || response.sentenceCount > maxInt()-offset {
		return nil, newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	endOffset := offset + response.sentenceCount
	if endOffset > response.totalCount &&
		(offset <= response.totalCount || response.sentenceCount != 0) {
		return nil, newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}

	hasRemaining := endOffset < response.totalCount
	if nextOffset.present && nextOffset.isNull {
		if hasRemaining {
			return nil, newLawContentSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
		return nil, nil
	}
	if nextOffset.present {
		if nextOffset.value != endOffset ||
			nextOffset.value <= offset ||
			nextOffset.value < 0 ||
			nextOffset.value > math.MaxInt32 ||
			nextOffset.value > response.totalCount ||
			!hasRemaining {
			return nil, newLawContentSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
		value := nextOffset.value
		return &value, nil
	}
	if !hasRemaining {
		return nil, nil
	}
	return nil, newLawContentSourceError(
		model.SourceErrorCodeInvalidSourceResponse,
		"",
	)
}

func classifyLawContentParseError(
	parent context.Context,
	parseContext context.Context,
	err error,
) error {
	if parentErr := parent.Err(); parentErr != nil {
		return parentErr
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(parseContext.Err(), context.DeadlineExceeded) {
		return newLawContentSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			"",
		)
	}
	switch {
	case errors.Is(err, lawSearchJSONErrorValues):
		return newLawContentSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	case errors.Is(err, lawSearchJSONErrorDepth):
		return newLawContentSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	default:
		return newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
}

func classifyLawContentDecodeError(
	parent context.Context,
	parseContext context.Context,
) error {
	if err := parent.Err(); err != nil {
		return err
	}
	if errors.Is(parseContext.Err(), context.DeadlineExceeded) {
		return newLawContentSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			"",
		)
	}
	return newLawContentSourceError(
		model.SourceErrorCodeSourceContractChanged,
		"",
	)
}
