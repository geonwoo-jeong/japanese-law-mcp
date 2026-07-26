package lawv2

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	lawDocumentParserInputBytes = 32 * 1024 * 1024
	lawDocumentXMLElements      = 500000
	lawDocumentXMLDepth         = 128
	lawDocumentParseTimeout     = 5 * time.Second
)

type lawDocumentXMLError string

const (
	lawDocumentXMLErrorUnsafe    lawDocumentXMLError = "安全でない XML 構造です"
	lawDocumentXMLErrorElements  lawDocumentXMLError = "XML element 数が上限を超えました"
	lawDocumentXMLErrorDepth     lawDocumentXMLError = "XML depth が上限を超えました"
	lawDocumentXMLErrorStructure lawDocumentXMLError = "XML の構造が不正です"
)

func (e lawDocumentXMLError) Error() string {
	return string(e)
}

type lawDocumentResponse struct {
	law     lawSearchLaw
	content string
}

type xmlFieldCapture struct {
	key   string
	depth int
	value strings.Builder
}

type lawDocumentXMLState struct {
	stack             []xml.Name
	elementCount      int
	rootCount         int
	lawInfoCount      int
	revisionInfoCount int
	fullTextCount     int
	lawCount          int
	lawStart          int64
	lawEnd            int64
	lawDepth          int
	fields            map[string]string
	fieldSeen         map[string]struct{}
	capture           *xmlFieldCapture
}

func parseLawDocumentResponse(
	ctx context.Context,
	body []byte,
) (lawDocumentResponse, error) {
	if ctx == nil {
		return lawDocumentResponse{}, fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return lawDocumentResponse{}, err
	}
	if len(body) > lawDocumentParserInputBytes {
		return lawDocumentResponse{}, newLawDocumentSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	if !utf8.Valid(body) {
		return lawDocumentResponse{}, newLawDocumentSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}

	parseContext, cancel := context.WithTimeout(ctx, lawDocumentParseTimeout)
	defer cancel()

	state := lawDocumentXMLState{
		fields:    make(map[string]string),
		fieldSeen: make(map[string]struct{}),
	}
	decoder := xml.NewDecoder(&contextReader{
		ctx:    parseContext,
		reader: bytes.NewReader(body),
	})
	decoder.Strict = true

	for {
		before := decoder.InputOffset()
		token, err := decoder.RawToken()
		after := decoder.InputOffset()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return lawDocumentResponse{}, classifyLawDocumentXMLError(
				ctx,
				parseContext,
				err,
			)
		}
		if err := state.consume(token, before, after); err != nil {
			return lawDocumentResponse{}, classifyLawDocumentXMLError(
				ctx,
				parseContext,
				err,
			)
		}
	}
	if err := parseContext.Err(); err != nil {
		return lawDocumentResponse{}, classifyLawDocumentXMLError(
			ctx,
			parseContext,
			err,
		)
	}
	return state.response(body)
}

func (s *lawDocumentXMLState) consume(
	token xml.Token,
	before int64,
	after int64,
) error {
	switch value := token.(type) {
	case xml.StartElement:
		return s.consumeStart(value, before)
	case xml.EndElement:
		return s.consumeEnd(value, after)
	case xml.CharData:
		if s.capture != nil {
			_, _ = s.capture.value.Write([]byte(value))
			return nil
		}
		if len(s.stack) == 0 && strings.TrimSpace(string(value)) != "" {
			return lawDocumentXMLErrorStructure
		}
		return nil
	case xml.Directive:
		return lawDocumentXMLErrorUnsafe
	default:
		return nil
	}
}

func (s *lawDocumentXMLState) consumeStart(
	element xml.StartElement,
	offset int64,
) error {
	depth := len(s.stack) + 1
	s.elementCount++
	if s.elementCount > lawDocumentXMLElements {
		return lawDocumentXMLErrorElements
	}
	if depth > lawDocumentXMLDepth {
		return lawDocumentXMLErrorDepth
	}
	if s.capture != nil {
		return lawDocumentXMLErrorStructure
	}

	parent := xml.Name{}
	if len(s.stack) != 0 {
		parent = s.stack[len(s.stack)-1]
	}
	if depth == 1 {
		s.rootCount++
		if !isXMLName(element.Name, "law_data_response") ||
			s.rootCount != 1 {
			return lawDocumentXMLErrorStructure
		}
	}
	if depth == 2 && isXMLName(parent, "law_data_response") {
		switch {
		case isXMLName(element.Name, "law_info"):
			s.lawInfoCount++
		case isXMLName(element.Name, "revision_info"):
			s.revisionInfoCount++
		case isXMLName(element.Name, "law_full_text"):
			s.fullTextCount++
		}
	}
	if isXMLName(element.Name, "Law") {
		if depth != 3 || !isXMLName(parent, "law_full_text") {
			return lawDocumentXMLErrorStructure
		}
		s.lawCount++
		if s.lawCount != 1 {
			return lawDocumentXMLErrorStructure
		}
		s.lawStart = offset
		s.lawDepth = depth
	}

	if key, exists := mappedLawDocumentField(parent, element.Name, depth); exists {
		if _, duplicate := s.fieldSeen[key]; duplicate {
			return lawDocumentXMLErrorStructure
		}
		s.fieldSeen[key] = struct{}{}
		s.capture = &xmlFieldCapture{key: key, depth: depth}
	}
	s.stack = append(s.stack, element.Name)
	return nil
}

func (s *lawDocumentXMLState) consumeEnd(
	element xml.EndElement,
	offset int64,
) error {
	if len(s.stack) == 0 ||
		s.stack[len(s.stack)-1] != element.Name {
		return lawDocumentXMLErrorStructure
	}
	depth := len(s.stack)
	if s.capture != nil && s.capture.depth == depth {
		s.fields[s.capture.key] = s.capture.value.String()
		s.capture = nil
	}
	if isXMLName(element.Name, "Law") && s.lawDepth == depth {
		s.lawEnd = offset
	}
	s.stack = s.stack[:len(s.stack)-1]
	return nil
}

func (s lawDocumentXMLState) response(
	body []byte,
) (lawDocumentResponse, error) {
	if len(s.stack) != 0 || s.rootCount != 1 {
		return lawDocumentResponse{}, newLawDocumentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if s.lawInfoCount == 0 ||
		s.revisionInfoCount == 0 ||
		s.fullTextCount == 0 {
		return lawDocumentResponse{}, newLawDocumentSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	if s.lawInfoCount != 1 ||
		s.revisionInfoCount != 1 ||
		s.fullTextCount != 1 ||
		s.lawCount != 1 ||
		s.lawStart < 0 ||
		s.lawEnd <= s.lawStart ||
		s.lawEnd > int64(len(body)) {
		return lawDocumentResponse{}, newLawDocumentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	for _, required := range []string{
		"law_info/law_id",
		"revision_info/law_revision_id",
		"revision_info/law_title",
	} {
		if _, exists := s.fieldSeen[required]; !exists {
			return lawDocumentResponse{}, newLawDocumentSourceError(
				model.SourceErrorCodeSourceContractChanged,
				"",
			)
		}
		if s.fields[required] == "" {
			return lawDocumentResponse{}, newLawDocumentSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
	}

	return lawDocumentResponse{
		law: lawSearchLaw{
			lawID:                 s.fields["law_info/law_id"],
			revisionID:            s.fields["revision_info/law_revision_id"],
			title:                 s.fields["revision_info/law_title"],
			lawNumber:             s.fields["law_info/law_num"],
			promulgationDate:      s.fields["law_info/promulgation_date"],
			revisionEffectiveDate: s.fields["revision_info/amendment_enforcement_date"],
		},
		content: string(body[s.lawStart:s.lawEnd]),
	}, nil
}

func mappedLawDocumentField(
	parent xml.Name,
	element xml.Name,
	depth int,
) (string, bool) {
	if depth != 3 || parent.Space != "" || element.Space != "" {
		return "", false
	}
	switch parent.Local + "/" + element.Local {
	case "law_info/law_id",
		"law_info/law_num",
		"law_info/promulgation_date",
		"revision_info/law_revision_id",
		"revision_info/law_title",
		"revision_info/amendment_enforcement_date":
		return parent.Local + "/" + element.Local, true
	default:
		return "", false
	}
}

func isXMLName(name xml.Name, local string) bool {
	return name.Space == "" && name.Local == local
}

func classifyLawDocumentXMLError(
	parent context.Context,
	parseContext context.Context,
	err error,
) error {
	if parentErr := parent.Err(); parentErr != nil {
		return parentErr
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(parseContext.Err(), context.DeadlineExceeded) {
		return newLawDocumentSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			"",
		)
	}
	switch {
	case errors.Is(err, lawDocumentXMLErrorElements):
		return newLawDocumentSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	case errors.Is(err, lawDocumentXMLErrorDepth),
		errors.Is(err, lawDocumentXMLErrorUnsafe),
		strings.Contains(err.Error(), "invalid character entity"):
		return newLawDocumentSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	default:
		return newLawDocumentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
}
