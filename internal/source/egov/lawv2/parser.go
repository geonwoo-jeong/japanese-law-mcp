package lawv2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	lawSearchParserInputBytes = 16 * 1024 * 1024
	lawSearchJSONValues       = 200000
	lawSearchJSONDepth        = 32
	lawSearchParseTimeout     = 3 * time.Second
)

type lawSearchJSONError string

const (
	lawSearchJSONErrorValues    lawSearchJSONError = "JSON value 数が上限を超えました"
	lawSearchJSONErrorDepth     lawSearchJSONError = "JSON depth が上限を超えました"
	lawSearchJSONErrorDuplicate lawSearchJSONError = "JSON object に重複 key があります"
)

func (e lawSearchJSONError) Error() string {
	return string(e)
}

func parseLawSearchResponse(
	ctx context.Context,
	body []byte,
	limit int,
	offset int,
) (lawSearchResponse, *int, error) {
	if err := ctx.Err(); err != nil {
		return lawSearchResponse{}, nil, err
	}
	if len(body) > lawSearchParserInputBytes {
		return lawSearchResponse{}, nil, newSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	if !utf8.Valid(body) {
		return lawSearchResponse{}, nil, newSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}

	parseContext, cancel := context.WithTimeout(ctx, lawSearchParseTimeout)
	defer cancel()

	if err := validateLawSearchJSON(parseContext, body); err != nil {
		return lawSearchResponse{}, nil, classifyLawSearchParseError(ctx, parseContext, err)
	}

	var raw rawLawSearchResponse
	if err := decodeLawSearchJSON(parseContext, body, &raw); err != nil {
		return lawSearchResponse{}, nil, classifyLawSearchDecodeError(ctx, parseContext)
	}
	response, nextOffset, err := decodeLawSearchResponse(raw)
	if err != nil {
		return lawSearchResponse{}, nil, err
	}
	next, err := validateLawSearchPage(response, nextOffset, limit, offset)
	if err != nil {
		return lawSearchResponse{}, nil, err
	}
	return response, next, nil
}

func validateLawSearchJSON(ctx context.Context, body []byte) error {
	return validateJSONWithBudget(
		ctx,
		body,
		lawSearchJSONValues,
		lawSearchJSONDepth,
	)
}

func validateJSONWithBudget(
	ctx context.Context,
	body []byte,
	maximumValues int,
	maximumDepth int,
) error {
	decoder := json.NewDecoder(&contextReader{
		ctx:    ctx,
		reader: bytes.NewReader(body),
	})
	decoder.UseNumber()

	valueCount := 0
	if err := scanJSONValueWithBudget(
		ctx,
		decoder,
		1,
		&valueCount,
		maximumValues,
		maximumDepth,
	); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("root value の後に値があります")
		}
		return err
	}
	return nil
}

func scanJSONValueWithBudget(
	ctx context.Context,
	decoder *json.Decoder,
	depth int,
	valueCount *int,
	maximumValues int,
	maximumDepth int,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if depth > maximumDepth {
		return lawSearchJSONErrorDepth
	}
	*valueCount++
	if *valueCount > maximumValues {
		return lawSearchJSONErrorValues
	}

	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}

	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			if err := ctx.Err(); err != nil {
				return err
			}
			keyToken, keyErr := decoder.Token()
			if keyErr != nil {
				return keyErr
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key が文字列ではありません")
			}
			if _, exists := seen[key]; exists {
				return lawSearchJSONErrorDuplicate
			}
			seen[key] = struct{}{}
			if err := scanJSONValueWithBudget(
				ctx,
				decoder,
				depth+1,
				valueCount,
				maximumValues,
				maximumDepth,
			); err != nil {
				return err
			}
		}
		return consumeLawSearchDelimiter(decoder)
	case '[':
		for decoder.More() {
			if err := scanJSONValueWithBudget(
				ctx,
				decoder,
				depth+1,
				valueCount,
				maximumValues,
				maximumDepth,
			); err != nil {
				return err
			}
		}
		return consumeLawSearchDelimiter(decoder)
	default:
		return fmt.Errorf("予期しない JSON delimiter です")
	}
}

func consumeLawSearchDelimiter(decoder *json.Decoder) error {
	_, err := decoder.Token()
	return err
}

func decodeLawSearchJSON(ctx context.Context, body []byte, target any) error {
	decoder := json.NewDecoder(&contextReader{
		ctx:    ctx,
		reader: bytes.NewReader(body),
	})
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func decodeLawSearchResponse(
	raw rawLawSearchResponse,
) (lawSearchResponse, responseNextOffset, error) {
	totalCount, err := decodeLawSearchInteger(raw.TotalCount, true)
	if err != nil {
		return lawSearchResponse{}, responseNextOffset{}, err
	}
	count, err := decodeLawSearchInteger(raw.Count, true)
	if err != nil {
		return lawSearchResponse{}, responseNextOffset{}, err
	}
	nextOffset, err := decodeLawSearchNextOffset(raw.NextOffset)
	if err != nil {
		return lawSearchResponse{}, responseNextOffset{}, err
	}
	laws, err := decodeLawSearchLaws(raw.Laws)
	if err != nil {
		return lawSearchResponse{}, responseNextOffset{}, err
	}
	return lawSearchResponse{
		totalCount: totalCount,
		count:      count,
		laws:       laws,
	}, nextOffset, nil
}

func decodeLawSearchInteger(raw json.RawMessage, officialRequired bool) (int, error) {
	if len(raw) == 0 {
		code := model.SourceErrorCodeInvalidSourceResponse
		if officialRequired {
			code = model.SourceErrorCodeSourceContractChanged
		}
		return 0, newSourceError(code, "")
	}
	if isJSONNull(raw) {
		return 0, newSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, newSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	if value > int64(maxInt()) || value < int64(minInt()) {
		return 0, newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	return int(value), nil
}

func decodeLawSearchNextOffset(raw json.RawMessage) (responseNextOffset, error) {
	if len(raw) == 0 {
		return responseNextOffset{}, nil
	}
	if isJSONNull(raw) {
		return responseNextOffset{present: true, isNull: true}, nil
	}
	value, err := decodeLawSearchInteger(raw, false)
	if err != nil {
		return responseNextOffset{}, err
	}
	return responseNextOffset{present: true, value: value}, nil
}

func decodeLawSearchLaws(raw json.RawMessage) ([]lawSearchLaw, error) {
	if len(raw) == 0 {
		return nil, newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	if isJSONNull(raw) {
		return nil, newSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	var values []rawLawSearchLaw
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, newSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	laws := make([]lawSearchLaw, len(values))
	for index, value := range values {
		law, err := decodeLawSearchLaw(value)
		if err != nil {
			return nil, err
		}
		laws[index] = law
	}
	return laws, nil
}

func decodeLawSearchLaw(raw rawLawSearchLaw) (lawSearchLaw, error) {
	if len(raw.LawInfo) == 0 || len(raw.RevisionInfo) == 0 ||
		isJSONNull(raw.LawInfo) || isJSONNull(raw.RevisionInfo) {
		return lawSearchLaw{}, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	var lawInfo rawLawInfo
	if err := json.Unmarshal(raw.LawInfo, &lawInfo); err != nil {
		return lawSearchLaw{}, newSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	var revisionInfo rawRevisionInfo
	if err := json.Unmarshal(raw.RevisionInfo, &revisionInfo); err != nil {
		return lawSearchLaw{}, newSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}

	lawID, err := decodeLawSearchRequiredString(lawInfo.LawID)
	if err != nil {
		return lawSearchLaw{}, err
	}
	revisionID, err := decodeLawSearchRequiredString(revisionInfo.RevisionID)
	if err != nil {
		return lawSearchLaw{}, err
	}
	title, err := decodeLawSearchRequiredString(revisionInfo.Title)
	if err != nil {
		return lawSearchLaw{}, err
	}
	lawNumber, err := decodeLawSearchOptionalString(lawInfo.LawNumber)
	if err != nil {
		return lawSearchLaw{}, err
	}
	promulgationDate, err := decodeLawSearchOptionalString(lawInfo.PromulgationDate)
	if err != nil {
		return lawSearchLaw{}, err
	}
	revisionEffectiveDate, err := decodeLawSearchOptionalString(
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

func decodeLawSearchRequiredString(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || isJSONNull(raw) {
		return "", newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	value, err := decodeLawSearchString(raw)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	return value, nil
}

func decodeLawSearchOptionalString(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	if isJSONNull(raw) {
		return "", newSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	return decodeLawSearchString(raw)
}

func decodeLawSearchString(raw json.RawMessage) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", newSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	return value, nil
}

func classifyLawSearchParseError(
	parent context.Context,
	parseContext context.Context,
	err error,
) error {
	if parentErr := parent.Err(); parentErr != nil {
		return parentErr
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(parseContext.Err(), context.DeadlineExceeded) {
		return newSourceError(model.SourceErrorCodeSourceProcessingLimit, "")
	}
	switch {
	case errors.Is(err, lawSearchJSONErrorValues):
		return newSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	case errors.Is(err, lawSearchJSONErrorDepth):
		return newSourceError(model.SourceErrorCodeUnsafeSourceContent, "")
	default:
		return newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
}

func classifyLawSearchDecodeError(
	parent context.Context,
	parseContext context.Context,
) error {
	if err := parent.Err(); err != nil {
		return err
	}
	if errors.Is(parseContext.Err(), context.DeadlineExceeded) {
		return newSourceError(model.SourceErrorCodeSourceProcessingLimit, "")
	}
	return newSourceError(model.SourceErrorCodeSourceContractChanged, "")
}

func isJSONNull(raw json.RawMessage) bool {
	return strings.EqualFold(string(bytes.TrimSpace(raw)), "null")
}

func maxInt() int {
	return int(^uint(0) >> 1)
}

func minInt() int {
	return -maxInt() - 1
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	const maximumReadSize = 32 * 1024
	if len(buffer) > maximumReadSize {
		buffer = buffer[:maximumReadSize]
	}
	return r.reader.Read(buffer)
}
