package lawv1

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
	maximumUpdateItems = 512
	maximumXMLNodes    = 16 * 1024
	maximumXMLDepth    = 4
	parseTimeout       = 2 * time.Second
)

var (
	errXMLContract = errors.New("XML 構造が確認済みの契約と一致しません")
	errXMLUnsafe   = errors.New("安全でない XML 構造です")
	errXMLNodes    = errors.New("XML node 数が上限を超えました")
	errUpdateItems = errors.New("更新法令項目数が上限を超えました")
)

type xmlCapture struct {
	name  string
	depth int
	value strings.Builder
}

type xmlParserState struct {
	stack          []xml.Name
	nodeCount      int
	rootCount      int
	resultCount    int
	applDataCount  int
	itemCount      int
	resultSeen     map[string]struct{}
	applDataSeen   map[string]struct{}
	itemSeen       map[string]struct{}
	currentItem    *updateListItem
	capture        *xmlCapture
	response       updateListResponse
	xmlDeclaration bool
}

func parseResponse(
	ctx context.Context,
	body []byte,
) (updateListResponse, error) {
	return parseResponseWithTimeout(ctx, body, parseTimeout)
}

func parseResponseWithTimeout(
	ctx context.Context,
	body []byte,
	timeout time.Duration,
) (updateListResponse, error) {
	if ctx == nil {
		return updateListResponse{}, fmt.Errorf("context は必須です")
	}
	processingContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return parseResponseWithBudget(ctx, processingContext, body)
}

func parseResponseWithBudget(
	parent context.Context,
	processingContext context.Context,
	body []byte,
) (updateListResponse, error) {
	if parent == nil || processingContext == nil {
		return updateListResponse{}, fmt.Errorf("context は必須です")
	}
	if err := parent.Err(); err != nil {
		return updateListResponse{}, normalizeContextError(err)
	}
	if err := processingContext.Err(); err != nil {
		return updateListResponse{}, classifyParserError(
			parent,
			processingContext,
			err,
		)
	}
	if len(body) > maximumDecompressedBytes {
		return updateListResponse{}, newSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	if !utf8.Valid(body) {
		return updateListResponse{}, newSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}

	state := xmlParserState{
		resultSeen:   make(map[string]struct{}),
		applDataSeen: make(map[string]struct{}),
	}
	decoder := xml.NewDecoder(&xmlContextReader{
		ctx:    processingContext,
		reader: bytes.NewReader(body),
	})
	decoder.Strict = true

	for {
		token, err := decoder.RawToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return updateListResponse{}, classifyParserError(
				parent,
				processingContext,
				err,
			)
		}
		if err := state.consume(token); err != nil {
			return updateListResponse{}, classifyParserError(
				parent,
				processingContext,
				err,
			)
		}
	}
	if err := processingContext.Err(); err != nil {
		return updateListResponse{}, classifyParserError(
			parent,
			processingContext,
			err,
		)
	}
	if err := state.validateComplete(); err != nil {
		return updateListResponse{}, classifyParserError(
			parent,
			processingContext,
			err,
		)
	}
	return state.response, nil
}

func (s *xmlParserState) consume(token xml.Token) error {
	if err := s.countNode(token); err != nil {
		return err
	}
	switch value := token.(type) {
	case xml.StartElement:
		return s.consumeStart(value)
	case xml.EndElement:
		return s.consumeEnd(value)
	case xml.CharData:
		if s.capture != nil {
			_, _ = s.capture.value.Write([]byte(value))
			return nil
		}
		if strings.TrimSpace(string(value)) != "" {
			return errXMLContract
		}
		return nil
	case xml.ProcInst:
		if value.Target != "xml" ||
			s.xmlDeclaration ||
			len(s.stack) != 0 ||
			s.rootCount != 0 {
			return errXMLUnsafe
		}
		s.xmlDeclaration = true
		return nil
	case xml.Directive:
		return errXMLUnsafe
	case xml.Comment:
		return nil
	default:
		return errXMLContract
	}
}

func (s *xmlParserState) consumeStart(element xml.StartElement) error {
	depth := len(s.stack) + 1
	if depth > maximumXMLDepth {
		return errXMLUnsafe
	}
	if element.Name.Space != "" || len(element.Attr) != 0 || s.capture != nil {
		return errXMLContract
	}

	parent := ""
	if len(s.stack) != 0 {
		parent = s.stack[len(s.stack)-1].Local
	}
	switch depth {
	case 1:
		if element.Name.Local != "DataRoot" || s.rootCount != 0 {
			return errXMLContract
		}
		s.rootCount++
	case 2:
		switch element.Name.Local {
		case "Result":
			if parent != "DataRoot" || s.resultCount != 0 {
				return errXMLContract
			}
			s.resultCount++
		case "ApplData":
			if parent != "DataRoot" || s.applDataCount != 0 {
				return errXMLContract
			}
			s.applDataCount++
		default:
			return errXMLContract
		}
	case 3:
		if err := s.startContainerField(parent, element.Name.Local); err != nil {
			return err
		}
	case 4:
		if parent != "LawNameListInfo" ||
			!isUpdateListItemField(element.Name.Local) {
			return errXMLContract
		}
		if _, exists := s.itemSeen[element.Name.Local]; exists {
			return errXMLContract
		}
		s.itemSeen[element.Name.Local] = struct{}{}
		s.capture = &xmlCapture{name: element.Name.Local, depth: depth}
	default:
		return errXMLContract
	}
	s.stack = append(s.stack, element.Name)
	return nil
}

func (s *xmlParserState) startContainerField(
	parent string,
	name string,
) error {
	switch parent {
	case "Result":
		if name != "Code" && name != "Message" {
			return errXMLContract
		}
		if _, exists := s.resultSeen[name]; exists {
			return errXMLContract
		}
		s.resultSeen[name] = struct{}{}
		s.capture = &xmlCapture{name: name, depth: 3}
		return nil
	case "ApplData":
		switch name {
		case "Date":
			if _, exists := s.applDataSeen[name]; exists {
				return errXMLContract
			}
			s.applDataSeen[name] = struct{}{}
			s.capture = &xmlCapture{name: name, depth: 3}
			return nil
		case "LawNameListInfo":
			if s.itemCount >= maximumUpdateItems {
				return errUpdateItems
			}
			s.itemCount++
			s.currentItem = &updateListItem{}
			s.itemSeen = make(map[string]struct{})
			return nil
		default:
			return errXMLContract
		}
	default:
		return errXMLContract
	}
}

func (s *xmlParserState) countNode(token xml.Token) error {
	count := 0
	switch value := token.(type) {
	case xml.StartElement:
		count = 1 + len(value.Attr)
	case xml.CharData:
		if strings.TrimSpace(string(value)) != "" {
			count = 1
		}
	case xml.Comment, xml.ProcInst:
		count = 1
	}
	s.nodeCount += count
	if s.nodeCount > maximumXMLNodes {
		return errXMLNodes
	}
	return nil
}

func (s *xmlParserState) consumeEnd(element xml.EndElement) error {
	if len(s.stack) == 0 ||
		s.stack[len(s.stack)-1] != element.Name {
		return errXMLContract
	}
	depth := len(s.stack)
	if s.capture != nil && s.capture.depth == depth {
		s.assignCapture(
			s.capture.name,
			strings.TrimSpace(s.capture.value.String()),
		)
		s.capture = nil
	}
	if element.Name.Local == "LawNameListInfo" {
		if s.currentItem == nil {
			return errXMLContract
		}
		s.response.items = append(s.response.items, *s.currentItem)
		s.currentItem = nil
		s.itemSeen = nil
	}
	s.stack = s.stack[:len(s.stack)-1]
	return nil
}

func (s *xmlParserState) assignCapture(name string, value string) {
	if s.currentItem == nil {
		switch name {
		case "Code":
			s.response.code = value
		case "Message":
			s.response.message = value
		case "Date":
			s.response.date = value
		}
		return
	}
	switch name {
	case "LawTypeName":
		s.currentItem.lawTypeName = value
	case "LawNo":
		s.currentItem.lawNumber = value
	case "LawName":
		s.currentItem.lawName = value
	case "LawNameKana":
		s.currentItem.lawNameKana = value
	case "OldLawName":
		s.currentItem.oldLawName = value
	case "PromulgationDate":
		s.currentItem.promulgationDate = value
	case "AmendName":
		s.currentItem.amendmentName = value
	case "AmendNo":
		s.currentItem.amendmentNumber = value
	case "AmendPromulgationDate":
		s.currentItem.amendmentPromulgation = value
	case "EnforcementDate":
		s.currentItem.enforcementDate = value
	case "EnforcementComment":
		s.currentItem.enforcementComment = value
	case "LawId":
		s.currentItem.lawID = value
	case "LawUrl":
		s.currentItem.lawURL = value
	case "EnforcementFlg":
		s.currentItem.enforcementFlag = value
	case "AuthFlg":
		s.currentItem.authorityReviewFlag = value
	}
}

func (s *xmlParserState) validateComplete() error {
	if len(s.stack) != 0 ||
		s.capture != nil ||
		s.currentItem != nil ||
		s.rootCount != 1 ||
		s.resultCount != 1 ||
		s.applDataCount != 1 {
		return errXMLContract
	}
	if _, exists := s.resultSeen["Code"]; !exists {
		return errXMLContract
	}
	if _, exists := s.resultSeen["Message"]; !exists {
		return errXMLContract
	}
	if _, exists := s.applDataSeen["Date"]; !exists {
		return errXMLContract
	}
	return nil
}

func isUpdateListItemField(name string) bool {
	switch name {
	case "LawTypeName",
		"LawNo",
		"LawName",
		"LawNameKana",
		"OldLawName",
		"PromulgationDate",
		"AmendName",
		"AmendNo",
		"AmendPromulgationDate",
		"EnforcementDate",
		"EnforcementComment",
		"LawId",
		"LawUrl",
		"EnforcementFlg",
		"AuthFlg":
		return true
	default:
		return false
	}
}

func classifyParserError(
	parent context.Context,
	parseContext context.Context,
	err error,
) error {
	if parentErr := parent.Err(); parentErr != nil {
		return normalizeContextError(parentErr)
	}
	if errors.Is(parseContext.Err(), context.DeadlineExceeded) {
		return newSourceError(model.SourceErrorCodeSourceProcessingLimit, "")
	}
	switch {
	case errors.Is(err, errXMLUnsafe):
		return newSourceError(model.SourceErrorCodeUnsafeSourceContent, "")
	case errors.Is(err, errXMLNodes), errors.Is(err, errUpdateItems):
		return newSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	default:
		return newSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
}

type xmlContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *xmlContextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}
