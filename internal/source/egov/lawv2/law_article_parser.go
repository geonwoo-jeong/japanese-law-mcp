package lawv2

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type lawArticleResponse struct {
	law      lawSearchLaw
	location model.LawArticleLocation
	content  string
}

type lawArticleSlice struct {
	start int64
	end   int64
	depth int
}

type lawArticleCandidate struct {
	element        lawArticleSlice
	paragraph      lawArticleSlice
	paragraphCount int
}

type lawArticleXMLState struct {
	location              model.LawArticleLocation
	stack                 []xml.Name
	elementCount          int
	rootCount             int
	lawInfoCount          int
	revisionInfoCount     int
	fullTextCount         int
	lawCount              int
	fields                map[string]string
	fieldSeen             map[string]struct{}
	capture               *xmlFieldCapture
	activeProvisionDepth  int
	candidateCount        int
	candidate             lawArticleCandidate
	currentArticleDepth   int
	currentParagraphDepth int
}

func parseLawArticleResponse(
	ctx context.Context,
	body []byte,
	location model.LawArticleLocation,
) (lawArticleResponse, error) {
	if ctx == nil {
		return lawArticleResponse{}, fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return lawArticleResponse{}, err
	}
	if err := location.Validate(); err != nil {
		return lawArticleResponse{}, err
	}
	if len(body) > lawDocumentParserInputBytes {
		return lawArticleResponse{}, newLawArticleSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	if !utf8.Valid(body) {
		return lawArticleResponse{}, newLawArticleSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}

	parseContext, cancel := context.WithTimeout(ctx, lawDocumentParseTimeout)
	defer cancel()

	state := lawArticleXMLState{
		location:  location,
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
			return lawArticleResponse{}, classifyLawArticleXMLError(
				ctx,
				parseContext,
				err,
			)
		}
		if err := state.consume(token, before, after); err != nil {
			return lawArticleResponse{}, classifyLawArticleXMLError(
				ctx,
				parseContext,
				err,
			)
		}
	}
	if err := parseContext.Err(); err != nil {
		return lawArticleResponse{}, classifyLawArticleXMLError(
			ctx,
			parseContext,
			err,
		)
	}
	return state.response(body)
}

func (s *lawArticleXMLState) consume(
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

func (s *lawArticleXMLState) consumeStart(
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
	}

	if key, exists := mappedLawDocumentField(parent, element.Name, depth); exists {
		if _, duplicate := s.fieldSeen[key]; duplicate {
			return lawDocumentXMLErrorStructure
		}
		s.fieldSeen[key] = struct{}{}
		s.capture = &xmlFieldCapture{key: key, depth: depth}
	}

	if s.isRequestedProvision(element) {
		s.activeProvisionDepth = depth
	}
	if s.isArticleCandidate(element) {
		s.candidateCount++
		if s.candidateCount == 1 {
			s.candidate = lawArticleCandidate{
				element: lawArticleSlice{
					start: offset,
					depth: depth,
				},
			}
			s.currentArticleDepth = depth
		}
	}
	if err := s.captureParagraph(element, offset); err != nil {
		return err
	}

	s.stack = append(s.stack, element.Name)
	return nil
}

func (s *lawArticleXMLState) consumeEnd(
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
	if s.currentParagraphDepth == depth &&
		isXMLName(element.Name, "Paragraph") {
		s.candidate.paragraph.end = offset
		s.currentParagraphDepth = 0
	}
	if s.currentArticleDepth == depth &&
		isXMLName(element.Name, "Article") {
		s.candidate.element.end = offset
		s.currentArticleDepth = 0
	}
	if s.activeProvisionDepth == depth {
		s.activeProvisionDepth = 0
	}
	s.stack = s.stack[:len(s.stack)-1]
	return nil
}

func (s lawArticleXMLState) isRequestedProvision(
	element xml.StartElement,
) bool {
	if len(s.stack) != 4 ||
		!isXMLName(s.stack[0], "law_data_response") ||
		!isXMLName(s.stack[1], "law_full_text") ||
		!isXMLName(s.stack[2], "Law") ||
		!isXMLName(s.stack[3], "LawBody") {
		return false
	}
	switch s.location.Provision() {
	case model.LawArticleProvisionMain:
		return isXMLName(element.Name, "MainProvision")
	case model.LawArticleProvisionSupplementary:
		return isXMLName(element.Name, "SupplProvision") &&
			!hasUnqualifiedAttribute(element, "AmendLawNum")
	default:
		return false
	}
}

func (s lawArticleXMLState) isArticleCandidate(
	element xml.StartElement,
) bool {
	if s.activeProvisionDepth == 0 ||
		!isXMLName(element.Name, "Article") ||
		!hasOnlyAllowedArticleContainers(
			s.stack[s.activeProvisionDepth:],
		) {
		return false
	}
	number, exists, err := unqualifiedAttribute(element, "Num")
	return err == nil && exists && number == s.location.ArticleNumber()
}

func (s *lawArticleXMLState) captureParagraph(
	element xml.StartElement,
	offset int64,
) error {
	target, wantsParagraph := s.location.ParagraphNumber()
	if !wantsParagraph ||
		s.candidateCount != 1 ||
		s.currentArticleDepth == 0 ||
		len(s.stack)+1 != s.currentArticleDepth+1 ||
		!isXMLName(element.Name, "Paragraph") {
		return nil
	}
	number, exists, err := unqualifiedAttribute(element, "Num")
	if err != nil {
		return err
	}
	if !exists || !decimalIntegerEquals(number, target) {
		return nil
	}
	s.candidate.paragraphCount++
	if s.candidate.paragraphCount == 1 {
		s.candidate.paragraph = lawArticleSlice{
			start: offset,
			depth: len(s.stack) + 1,
		}
		s.currentParagraphDepth = len(s.stack) + 1
	}
	return nil
}

func (s lawArticleXMLState) response(body []byte) (lawArticleResponse, error) {
	if len(s.stack) != 0 || s.rootCount != 1 {
		return lawArticleResponse{}, newLawArticleSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if s.lawInfoCount == 0 ||
		s.revisionInfoCount == 0 ||
		s.fullTextCount == 0 {
		return lawArticleResponse{}, newLawArticleSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	if s.lawInfoCount != 1 ||
		s.revisionInfoCount != 1 ||
		s.fullTextCount != 1 ||
		s.lawCount != 1 {
		return lawArticleResponse{}, newLawArticleSourceError(
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
			return lawArticleResponse{}, newLawArticleSourceError(
				model.SourceErrorCodeSourceContractChanged,
				"",
			)
		}
		if s.fields[required] == "" {
			return lawArticleResponse{}, newLawArticleSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
	}

	if s.candidateCount == 0 {
		return lawArticleResponse{}, lawarticleread.ErrNotFound
	}
	if s.candidateCount > 1 {
		return lawArticleResponse{}, lawarticleread.ErrAmbiguousLocation
	}
	selected := s.candidate.element
	if _, wantsParagraph := s.location.ParagraphNumber(); wantsParagraph {
		if s.candidate.paragraphCount == 0 {
			return lawArticleResponse{}, lawarticleread.ErrNotFound
		}
		if s.candidate.paragraphCount > 1 {
			return lawArticleResponse{}, lawarticleread.ErrAmbiguousLocation
		}
		selected = s.candidate.paragraph
	}
	if selected.start < 0 ||
		selected.end <= selected.start ||
		selected.end > int64(len(body)) {
		return lawArticleResponse{}, newLawArticleSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}

	return lawArticleResponse{
		law: lawSearchLaw{
			lawID:                 s.fields["law_info/law_id"],
			revisionID:            s.fields["revision_info/law_revision_id"],
			title:                 s.fields["revision_info/law_title"],
			lawNumber:             s.fields["law_info/law_num"],
			promulgationDate:      s.fields["law_info/promulgation_date"],
			revisionEffectiveDate: s.fields["revision_info/amendment_enforcement_date"],
		},
		location: s.location,
		content:  string(body[selected.start:selected.end]),
	}, nil
}

func hasOnlyAllowedArticleContainers(path []xml.Name) bool {
	for _, name := range path {
		switch {
		case isXMLName(name, "Part"),
			isXMLName(name, "Chapter"),
			isXMLName(name, "Section"),
			isXMLName(name, "Subsection"),
			isXMLName(name, "Division"):
		default:
			return false
		}
	}
	return true
}

func hasUnqualifiedAttribute(element xml.StartElement, local string) bool {
	for _, attribute := range element.Attr {
		if isXMLName(attribute.Name, local) {
			return true
		}
	}
	return false
}

func unqualifiedAttribute(
	element xml.StartElement,
	local string,
) (string, bool, error) {
	value := ""
	found := false
	for _, attribute := range element.Attr {
		if !isXMLName(attribute.Name, local) {
			continue
		}
		if found {
			return "", false, lawDocumentXMLErrorStructure
		}
		value = attribute.Value
		found = true
	}
	return value, found, nil
}

func decimalIntegerEquals(value string, target int) bool {
	if value == "" || target < 1 {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	parsed, err := strconv.ParseUint(value, 10, 63)
	return err == nil && parsed > 0 && parsed == uint64(target)
}

func classifyLawArticleXMLError(
	parent context.Context,
	parseContext context.Context,
	err error,
) error {
	if parentErr := parent.Err(); parentErr != nil {
		return parentErr
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(parseContext.Err(), context.DeadlineExceeded) {
		return newLawArticleSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			"",
		)
	}
	switch {
	case errors.Is(err, lawDocumentXMLErrorElements):
		return newLawArticleSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	case errors.Is(err, lawDocumentXMLErrorDepth),
		errors.Is(err, lawDocumentXMLErrorUnsafe),
		strings.Contains(err.Error(), "invalid character entity"):
		return newLawArticleSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	default:
		return newLawArticleSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
}
