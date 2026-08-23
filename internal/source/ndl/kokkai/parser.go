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

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
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
	speechSearchJSONErrorShape     speechSearchJSONError = "JSON shape が契約と一致しません"
)

func (e speechSearchJSONError) Error() string {
	return string(e)
}

func parseAndMapSpeechSearchPage(
	parent context.Context,
	fetched fetchedSpeechSearchResponse,
	limit int,
) (parliamentspeechsearch.Page, error) {
	if err := parent.Err(); err != nil {
		return parliamentspeechsearch.Page{}, normalizeSpeechSearchContextError(err)
	}
	parseContext, cancel := context.WithTimeout(parent, speechSearchParseTimeout)
	defer cancel()

	body, err := decodeSpeechSearchResponseBody(parent, parseContext, fetched)
	if err != nil {
		if parentErr := parent.Err(); parentErr != nil {
			return parliamentspeechsearch.Page{}, normalizeSpeechSearchContextError(parentErr)
		}
		return parliamentspeechsearch.Page{}, err
	}
	response, err := parseSpeechSearchResponse(parent, parseContext, body, limit)
	if err != nil {
		return parliamentspeechsearch.Page{}, err
	}
	page, err := mapSpeechSearchPage(parseContext, response, fetched.fetchedURL, fetched.retrievedAt)
	if err != nil {
		if parentErr := parent.Err(); parentErr != nil {
			return parliamentspeechsearch.Page{}, normalizeSpeechSearchContextError(parentErr)
		}
		if errors.Is(parseContext.Err(), context.DeadlineExceeded) {
			return parliamentspeechsearch.Page{}, newSpeechSearchSourceError(
				model.SourceErrorCodeSourceProcessingLimit,
				"",
			)
		}
		return parliamentspeechsearch.Page{}, err
	}
	if parentErr := parent.Err(); parentErr != nil {
		return parliamentspeechsearch.Page{}, normalizeSpeechSearchContextError(parentErr)
	}
	if errors.Is(parseContext.Err(), context.DeadlineExceeded) {
		return parliamentspeechsearch.Page{}, newSpeechSearchSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			"",
		)
	}
	return page, nil
}

func parseSpeechSearchResponse(
	parent context.Context,
	parseContext context.Context,
	body []byte,
	limit int,
) (speechSearchResponse, error) {
	if err := parseContext.Err(); err != nil {
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
	if err := validateSpeechSearchJSON(parseContext, body); err != nil {
		return speechSearchResponse{}, classifySpeechSearchParseError(
			parent,
			parseContext,
			err,
		)
	}

	raw, err := decodeSpeechSearchRawResponse(parseContext, body)
	if err != nil {
		return speechSearchResponse{}, classifySpeechSearchDecodeError(
			parent,
			parseContext,
		)
	}
	if err := classifySpeechSearchAPIError(raw); err != nil {
		return speechSearchResponse{}, err
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
	if err := scanSpeechSearchJSONRootObject(
		ctx,
		decoder,
		&valueCount,
		speechSearchJSONValues,
		speechSearchJSONDepth,
		nil,
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

func scanSpeechSearchJSONRootObject(
	ctx context.Context,
	decoder *json.Decoder,
	valueCount *int,
	maximumValues int,
	maximumDepth int,
	allowedKeys map[string]struct{},
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if 1 > maximumDepth {
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
	if !isDelimiter || delimiter != '{' {
		return speechSearchJSONErrorShape
	}
	return scanSpeechSearchJSONObjectContents(
		ctx,
		decoder,
		1,
		valueCount,
		maximumValues,
		maximumDepth,
		allowedKeys,
	)
}

func scanSpeechSearchJSONObjectContents(
	ctx context.Context,
	decoder *json.Decoder,
	depth int,
	valueCount *int,
	maximumValues int,
	maximumDepth int,
	allowedKeys map[string]struct{},
) error {
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
			return speechSearchJSONErrorShape
		}
		if _, exists := seen[key]; exists {
			return speechSearchJSONErrorDuplicate
		}
		seen[key] = struct{}{}
		if allowedKeys != nil {
			if _, allowed := allowedKeys[key]; !allowed {
				return speechSearchJSONErrorShape
			}
		}
		if err := scanSpeechSearchJSONValue(
			ctx,
			decoder,
			depth+1,
			valueCount,
			maximumValues,
			maximumDepth,
			nil,
		); err != nil {
			return err
		}
	}
	_, err := decoder.Token()
	return err
}

func scanSpeechSearchJSONValue(
	ctx context.Context,
	decoder *json.Decoder,
	depth int,
	valueCount *int,
	maximumValues int,
	maximumDepth int,
	objectKeys map[string]struct{},
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
		return scanSpeechSearchJSONObjectContents(
			ctx,
			decoder,
			depth,
			valueCount,
			maximumValues,
			maximumDepth,
			objectKeys,
		)
	case '[':
		for decoder.More() {
			if err := scanSpeechSearchJSONValue(
				ctx,
				decoder,
				depth+1,
				valueCount,
				maximumValues,
				maximumDepth,
				objectKeys,
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

func decodeSpeechSearchRawResponse(
	ctx context.Context,
	body []byte,
) (rawSpeechSearchResponse, error) {
	var object map[string]json.RawMessage
	if err := decodeSpeechSearchJSON(ctx, body, &object); err != nil {
		return rawSpeechSearchResponse{}, err
	}
	return rawSpeechSearchResponse{
		NumberOfRecords:    object["numberOfRecords"],
		NumberOfReturn:     object["numberOfReturn"],
		StartRecord:        object["startRecord"],
		NextRecordPosition: object["nextRecordPosition"],
		SpeechRecord:       object["speechRecord"],
		Message:            object["message"],
		Details:            object["details"],
	}, nil
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

func classifySpeechSearchAPIError(raw rawSpeechSearchResponse) error {
	if len(raw.NumberOfRecords) != 0 ||
		len(raw.NumberOfReturn) != 0 ||
		len(raw.StartRecord) != 0 ||
		len(raw.NextRecordPosition) != 0 ||
		len(raw.SpeechRecord) != 0 ||
		len(raw.Message) == 0 {
		return nil
	}

	message, err := decodeSpeechSearchRequiredString(raw.Message)
	if err != nil {
		return invalidSpeechSearchResponse()
	}
	if err := validateSpeechSearchErrorDetails(raw.Details); err != nil {
		return err
	}
	if message == speechSearchBusyMessage {
		return newSpeechSearchSourceError(model.SourceErrorCodeSourceUnavailable, "")
	}
	return invalidSpeechSearchResponse()
}

func validateSpeechSearchErrorDetails(raw json.RawMessage) error {
	if len(raw) == 0 || isSpeechSearchJSONNull(raw) {
		return nil
	}
	var details []string
	if err := json.Unmarshal(raw, &details); err != nil {
		return invalidSpeechSearchResponse()
	}
	return nil
}

func decodeSpeechRecords(raw json.RawMessage, returnedCount int) ([]speechRecord, error) {
	if returnedCount == 0 {
		if len(raw) == 0 {
			return []speechRecord{}, nil
		}
	}
	if len(raw) == 0 || isSpeechSearchJSONNull(raw) {
		return nil, invalidSpeechSearchResponse()
	}
	var objects []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &objects); err != nil {
		return nil, invalidSpeechSearchResponse()
	}
	records := make([]speechRecord, len(objects))
	for index, object := range objects {
		record, err := decodeSpeechRecord(rawSpeechRecordFromObject(object))
		if err != nil {
			return nil, err
		}
		records[index] = record
	}
	return records, nil
}

func rawSpeechRecordFromObject(object map[string]json.RawMessage) rawSpeechRecord {
	return rawSpeechRecord{
		SpeechID:      object["speechID"],
		SpeechOrder:   object["speechOrder"],
		Speaker:       object["speaker"],
		SpeakerYomi:   object["speakerYomi"],
		SpeakerGroup:  object["speakerGroup"],
		SpeakerPos:    object["speakerPosition"],
		SpeakerRole:   object["speakerRole"],
		Speech:        object["speech"],
		StartPage:     object["startPage"],
		SpeechURL:     object["speechURL"],
		IssueID:       object["issueID"],
		ImageKind:     object["imageKind"],
		Session:       object["session"],
		NameOfHouse:   object["nameOfHouse"],
		NameOfMeeting: object["nameOfMeeting"],
		Issue:         object["issue"],
		Date:          object["date"],
		Closing:       object["closing"],
		MeetingURL:    object["meetingURL"],
		PDFURL:        object["pdfURL"],
	}
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

func speechSearchTopLevelKeys() map[string]struct{} {
	return map[string]struct{}{
		"numberOfRecords":    {},
		"numberOfReturn":     {},
		"startRecord":        {},
		"nextRecordPosition": {},
		"speechRecord":       {},
	}
}

func speechSearchRecordKeys() map[string]struct{} {
	return map[string]struct{}{
		"speechID":        {},
		"speechOrder":     {},
		"speaker":         {},
		"speakerYomi":     {},
		"speakerGroup":    {},
		"speakerPosition": {},
		"speakerRole":     {},
		"speech":          {},
		"startPage":       {},
		"speechURL":       {},
		"issueID":         {},
		"imageKind":       {},
		"session":         {},
		"nameOfHouse":     {},
		"nameOfMeeting":   {},
		"issue":           {},
		"date":            {},
		"closing":         {},
		"meetingURL":      {},
		"pdfURL":          {},
	}
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
