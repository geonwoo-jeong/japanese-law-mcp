package kokkai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	speechSearchParserInputBytes = 16 * 1024 * 1024
	speechSearchJSONValues       = 200000
	speechSearchJSONDepth        = 32
	speechSearchParseTimeout     = 2 * time.Second
)

type speechSearchJSONError string

const (
	speechSearchJSONErrorValues    speechSearchJSONError = "JSON value 数が上限を超えました"
	speechSearchJSONErrorDepth     speechSearchJSONError = "JSON depth が上限を超えました"
	speechSearchJSONErrorDuplicate speechSearchJSONError = "JSON object に重複 key があります"
)

func (e speechSearchJSONError) Error() string {
	return string(e)
}

func parseSpeechSearchResponse(
	ctx context.Context,
	body []byte,
	limit int,
) (speechSearchResponse, error) {
	if err := ctx.Err(); err != nil {
		return speechSearchResponse{}, err
	}
	if len(body) > speechSearchParserInputBytes {
		return speechSearchResponse{}, newSpeechSearchSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	if !utf8.Valid(body) {
		return speechSearchResponse{}, newSpeechSearchSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}

	parseContext, cancel := context.WithTimeout(ctx, speechSearchParseTimeout)
	defer cancel()

	if err := validateSpeechSearchJSON(parseContext, body); err != nil {
		return speechSearchResponse{}, classifySpeechSearchParseError(
			ctx,
			parseContext,
			err,
		)
	}

	var raw rawSpeechSearchResponse
	if err := decodeSpeechSearchJSON(parseContext, body, &raw); err != nil {
		return speechSearchResponse{}, classifySpeechSearchDecodeError(
			ctx,
			parseContext,
		)
	}

	response, err := decodeSpeechSearchResponse(raw)
	if err != nil {
		return speechSearchResponse{}, err
	}
	if err := validateSpeechSearchPage(response, limit); err != nil {
		return speechSearchResponse{}, err
	}
	return response, nil
}

func validateSpeechSearchJSON(ctx context.Context, body []byte) error {
	decoder := json.NewDecoder(&speechSearchContextReader{
		ctx:    ctx,
		reader: bytes.NewReader(body),
	})
	decoder.UseNumber()

	valueCount := 0
	if err := scanSpeechSearchJSONValue(
		ctx,
		decoder,
		1,
		&valueCount,
		speechSearchJSONValues,
		speechSearchJSONDepth,
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

func scanSpeechSearchJSONValue(
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
		return speechSearchJSONErrorDepth
	}
	*valueCount += 1
	if *valueCount > maximumValues {
		return speechSearchJSONErrorValues
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
				return speechSearchJSONErrorDuplicate
			}
			seen[key] = struct{}{}
			if err := scanSpeechSearchJSONValue(
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
		_, err := decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := scanSpeechSearchJSONValue(
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
		_, err := decoder.Token()
		return err
	default:
		return fmt.Errorf("予期しない JSON delimiter です")
	}
}

func decodeSpeechSearchJSON(ctx context.Context, body []byte, target any) error {
	decoder := json.NewDecoder(&speechSearchContextReader{
		ctx:    ctx,
		reader: bytes.NewReader(body),
	})
	return decoder.Decode(target)
}

func decodeSpeechSearchResponse(raw rawSpeechSearchResponse) (speechSearchResponse, error) {
	totalCount, err := decodeSpeechSearchRequiredInt(raw.NumberOfRecords)
	if err != nil {
		return speechSearchResponse{}, err
	}
	returnedCount, err := decodeSpeechSearchRequiredInt(raw.NumberOfReturn)
	if err != nil {
		return speechSearchResponse{}, err
	}
	startRecord, err := decodeSpeechSearchRequiredInt(raw.StartRecord)
	if err != nil {
		return speechSearchResponse{}, err
	}
	if startRecord != 1 {
		return speechSearchResponse{}, invalidSpeechSearchResponse()
	}
	nextRecordPosition, err := decodeSpeechSearchOptionalInt(raw.NextRecordPosition)
	if err != nil {
		return speechSearchResponse{}, err
	}
	records, err := decodeSpeechRecords(raw.SpeechRecord, returnedCount)
	if err != nil {
		return speechSearchResponse{}, err
	}
	return speechSearchResponse{
		totalCount:         totalCount,
		returnedCount:      returnedCount,
		nextRecordPosition: nextRecordPosition,
		records:            records,
	}, nil
}

func decodeSpeechRecords(raw json.RawMessage, returnedCount int) ([]speechRecord, error) {
	if returnedCount == 0 {
		if len(raw) == 0 || isSpeechSearchJSONNull(raw) {
			return []speechRecord{}, nil
		}
	}
	if len(raw) == 0 || isSpeechSearchJSONNull(raw) {
		return nil, invalidSpeechSearchResponse()
	}
	var values []rawSpeechRecord
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, invalidSpeechSearchResponse()
	}
	records := make([]speechRecord, len(values))
	for index, value := range values {
		record, err := decodeSpeechRecord(value)
		if err != nil {
			return nil, err
		}
		records[index] = record
	}
	return records, nil
}

func decodeSpeechRecord(raw rawSpeechRecord) (speechRecord, error) {
	speechID, err := decodeSpeechSearchRequiredString(raw.SpeechID)
	if err != nil {
		return speechRecord{}, err
	}
	speechOrder, err := decodeSpeechSearchRequiredInt(raw.SpeechOrder)
	if err != nil {
		return speechRecord{}, err
	}
	speakerName, err := decodeSpeechSearchRequiredString(raw.Speaker)
	if err != nil {
		return speechRecord{}, err
	}
	speakerReading, err := decodeSpeechSearchOptionalString(raw.SpeakerYomi)
	if err != nil {
		return speechRecord{}, err
	}
	speakerGroup, err := decodeSpeechSearchOptionalString(raw.SpeakerGroup)
	if err != nil {
		return speechRecord{}, err
	}
	speakerPosition, err := decodeSpeechSearchOptionalString(raw.SpeakerPos)
	if err != nil {
		return speechRecord{}, err
	}
	speakerRole, err := decodeSpeechSearchOptionalString(raw.SpeakerRole)
	if err != nil {
		return speechRecord{}, err
	}
	speechText, err := decodeSpeechSearchRequiredText(raw.Speech)
	if err != nil {
		return speechRecord{}, err
	}
	startPage, err := decodeSpeechSearchOptionalInt(raw.StartPage)
	if err != nil {
		return speechRecord{}, err
	}
	speechURL, err := decodeSpeechSearchRequiredString(raw.SpeechURL)
	if err != nil {
		return speechRecord{}, err
	}
	meetingRecordID, err := decodeSpeechSearchRequiredString(raw.IssueID)
	if err != nil {
		return speechRecord{}, err
	}
	imageKind, err := decodeSpeechSearchRequiredString(raw.ImageKind)
	if err != nil {
		return speechRecord{}, err
	}
	session, err := decodeSpeechSearchRequiredInt(raw.Session)
	if err != nil {
		return speechRecord{}, err
	}
	houseName, err := decodeSpeechSearchRequiredString(raw.NameOfHouse)
	if err != nil {
		return speechRecord{}, err
	}
	meetingName, err := decodeSpeechSearchRequiredString(raw.NameOfMeeting)
	if err != nil {
		return speechRecord{}, err
	}
	issue, err := decodeSpeechSearchRequiredString(raw.Issue)
	if err != nil {
		return speechRecord{}, err
	}
	meetingDate, err := decodeSpeechSearchRequiredString(raw.Date)
	if err != nil {
		return speechRecord{}, err
	}
	closing, err := decodeSpeechSearchOptionalBool(raw.Closing)
	if err != nil {
		return speechRecord{}, err
	}
	meetingURL, err := decodeSpeechSearchRequiredString(raw.MeetingURL)
	if err != nil {
		return speechRecord{}, err
	}
	pdfURL, err := decodeSpeechSearchOptionalString(raw.PDFURL)
	if err != nil {
		return speechRecord{}, err
	}
	return speechRecord{
		speechID:        speechID,
		speechOrder:     speechOrder,
		speakerName:     speakerName,
		speakerReading:  speakerReading,
		speakerGroup:    speakerGroup,
		speakerPosition: speakerPosition,
		speakerRole:     speakerRole,
		speechText:      speechText,
		startPage:       startPage,
		speechURL:       speechURL,
		meetingRecordID: meetingRecordID,
		imageKind:       imageKind,
		session:         session,
		houseName:       houseName,
		meetingName:     meetingName,
		issue:           issue,
		meetingDate:     meetingDate,
		closing:         closing,
		meetingURL:      meetingURL,
		pdfURL:          pdfURL,
	}, nil
}

func validateSpeechSearchPage(response speechSearchResponse, limit int) error {
	if response.totalCount < 0 ||
		response.returnedCount < 0 ||
		response.returnedCount > limit ||
		response.returnedCount > response.totalCount ||
		len(response.records) != response.returnedCount {
		return invalidSpeechSearchResponse()
	}
	if response.returnedCount == 0 {
		if response.totalCount != 0 ||
			response.nextRecordPosition != nil ||
			len(response.records) != 0 {
			return invalidSpeechSearchResponse()
		}
		return nil
	}

	expectedNext := 1 + response.returnedCount
	hasRemaining := response.returnedCount < response.totalCount
	if response.nextRecordPosition == nil {
		if hasRemaining {
			return invalidSpeechSearchResponse()
		}
		return nil
	}
	if !hasRemaining || *response.nextRecordPosition != expectedNext {
		return invalidSpeechSearchResponse()
	}
	return nil
}

func decodeSpeechSearchRequiredInt(raw json.RawMessage) (int, error) {
	if len(raw) == 0 || isSpeechSearchJSONNull(raw) {
		return 0, invalidSpeechSearchResponse()
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, invalidSpeechSearchResponse()
	}
	if value < int64(minInt()) || value > int64(maxInt()) {
		return 0, invalidSpeechSearchResponse()
	}
	return int(value), nil
}

func decodeSpeechSearchOptionalInt(raw json.RawMessage) (*int, error) {
	if len(raw) == 0 || isSpeechSearchJSONNull(raw) {
		return nil, nil
	}
	value, err := decodeSpeechSearchRequiredInt(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func decodeSpeechSearchRequiredString(raw json.RawMessage) (string, error) {
	value, err := decodeSpeechSearchString(raw)
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimFunc(value, unicode.IsSpace)
	if trimmed == "" {
		return "", invalidSpeechSearchResponse()
	}
	return trimmed, nil
}

func decodeSpeechSearchRequiredText(raw json.RawMessage) (string, error) {
	value, err := decodeSpeechSearchString(raw)
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimFunc(value, unicode.IsSpace)
	if trimmed == "" {
		return "", invalidSpeechSearchResponse()
	}
	return trimmed, nil
}

func decodeSpeechSearchOptionalString(raw json.RawMessage) (*string, error) {
	if len(raw) == 0 || isSpeechSearchJSONNull(raw) {
		return nil, nil
	}
	value, err := decodeSpeechSearchString(raw)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimFunc(value, unicode.IsSpace)
	if trimmed == "" {
		return nil, nil
	}
	return &trimmed, nil
}

func decodeSpeechSearchOptionalBool(raw json.RawMessage) (*bool, error) {
	if len(raw) == 0 || isSpeechSearchJSONNull(raw) {
		return nil, nil
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, invalidSpeechSearchResponse()
	}
	return &value, nil
}

func decodeSpeechSearchString(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || isSpeechSearchJSONNull(raw) {
		return "", invalidSpeechSearchResponse()
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", invalidSpeechSearchResponse()
	}
	return value, nil
}

func classifySpeechSearchParseError(
	parent context.Context,
	parseContext context.Context,
	err error,
) error {
	if parentErr := parent.Err(); parentErr != nil {
		return parentErr
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(parseContext.Err(), context.DeadlineExceeded) {
		return newSpeechSearchSourceError(model.SourceErrorCodeSourceProcessingLimit, "")
	}
	switch {
	case errors.Is(err, speechSearchJSONErrorValues):
		return newSpeechSearchSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	case errors.Is(err, speechSearchJSONErrorDepth):
		return newSpeechSearchSourceError(model.SourceErrorCodeUnsafeSourceContent, "")
	default:
		return newSpeechSearchSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
}

func classifySpeechSearchDecodeError(
	parent context.Context,
	parseContext context.Context,
) error {
	if err := parent.Err(); err != nil {
		return err
	}
	if errors.Is(parseContext.Err(), context.DeadlineExceeded) {
		return newSpeechSearchSourceError(model.SourceErrorCodeSourceProcessingLimit, "")
	}
	return newSpeechSearchSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
}

func invalidSpeechSearchResponse() error {
	return newSpeechSearchSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
}

func isSpeechSearchJSONNull(raw json.RawMessage) bool {
	return strings.EqualFold(string(bytes.TrimSpace(raw)), "null")
}

func maxInt() int {
	return int(^uint(0) >> 1)
}

func minInt() int {
	return -maxInt() - 1
}

type speechSearchContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *speechSearchContextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	const maximumReadSize = 32 * 1024
	if len(buffer) > maximumReadSize {
		buffer = buffer[:maximumReadSize]
	}
	return r.reader.Read(buffer)
}
