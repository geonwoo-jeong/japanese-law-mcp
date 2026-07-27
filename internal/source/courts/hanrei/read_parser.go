package hanrei

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"golang.org/x/net/html"
)

const (
	maximumReadHTMLNodes = 50000
	maximumReadHTMLDepth = 64
)

func parseReadResponse(ctx context.Context, body []byte) (readResponse, error) {
	if ctx == nil {
		return readResponse{}, errors.New("context は必須です")
	}
	if len(body) > maximumReadDecompressedBytes {
		return readResponse{}, newReadSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	}
	if !utf8.Valid(body) {
		return readResponse{}, newReadSourceError(model.SourceErrorCodeUnsafeSourceContent, "")
	}
	if err := readHTMLContextError(ctx); err != nil {
		return readResponse{}, err
	}
	if err := validateReadHTMLTokenBudget(ctx, body); err != nil {
		return readResponse{}, err
	}
	document, err := html.Parse(&htmlContextReader{
		ctx:    ctx,
		reader: bytes.NewReader(body),
	})
	if err != nil {
		return readResponse{}, classifyReadHTMLParseError(ctx)
	}
	if err := validateReadHTMLBudget(ctx, document); err != nil {
		return readResponse{}, err
	}
	return decodeReadDocument(ctx, document)
}

func validateReadHTMLTokenBudget(ctx context.Context, body []byte) error {
	tokenizer := html.NewTokenizer(&htmlContextReader{ctx: ctx, reader: bytes.NewReader(body)})
	count := 0
	openElements := make([]string, 0, maximumReadHTMLDepth)
	for {
		if err := readHTMLContextError(ctx); err != nil {
			return err
		}
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			if errors.Is(tokenizer.Err(), io.EOF) {
				return nil
			}
			return invalidReadResponseError()
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			count += 1 + len(token.Attr)
			if count > maximumReadHTMLNodes {
				return newReadSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
			}
			if tokenType == html.StartTagToken && !isVoidSearchHTMLElement(token.Data) {
				openElements = append(openElements, token.Data)
				if len(openElements) > maximumReadHTMLDepth {
					return newReadSourceError(model.SourceErrorCodeUnsafeSourceContent, "")
				}
			}
		case html.EndTagToken:
			token := tokenizer.Token()
			for index := len(openElements) - 1; index >= 0; index-- {
				if openElements[index] == token.Data {
					openElements = openElements[:index]
					break
				}
			}
		case html.TextToken:
			if strings.TrimSpace(string(tokenizer.Text())) != "" {
				count++
			}
		case html.CommentToken:
			count++
		}
		if count > maximumReadHTMLNodes {
			return newReadSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
		}
	}
}

func validateReadHTMLBudget(ctx context.Context, root *html.Node) error {
	type pendingNode struct {
		node         *html.Node
		elementDepth int
	}
	pending := []pendingNode{{node: root}}
	count := 0
	for len(pending) > 0 {
		if err := readHTMLContextError(ctx); err != nil {
			return err
		}
		last := len(pending) - 1
		current := pending[last]
		pending = pending[:last]
		if err := countReadHTMLNode(current, &count); err != nil {
			return err
		}
		for child := current.node.LastChild; child != nil; child = child.PrevSibling {
			depth := current.elementDepth
			if child.Type == html.ElementNode {
				depth++
			}
			pending = append(pending, pendingNode{node: child, elementDepth: depth})
		}
	}
	return nil
}

func countReadHTMLNode(current struct {
	node         *html.Node
	elementDepth int
}, count *int) error {
	nodeCount := 0
	switch current.node.Type {
	case html.ElementNode:
		if current.elementDepth > maximumReadHTMLDepth {
			return newReadSourceError(model.SourceErrorCodeUnsafeSourceContent, "")
		}
		nodeCount = 1 + len(current.node.Attr)
	case html.TextNode:
		if strings.TrimSpace(current.node.Data) != "" {
			nodeCount = 1
		}
	case html.CommentNode:
		nodeCount = 1
	}
	*count += nodeCount
	if *count > maximumReadHTMLNodes {
		return newReadSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	}
	return nil
}

func decodeReadDocument(ctx context.Context, document *html.Node) (readResponse, error) {
	hasTitle, err := hasReadPageTitle(ctx, document)
	if err != nil {
		return readResponse{}, err
	}
	if !hasTitle {
		return readResponse{}, newReadSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	mains, err := collectReadElements(ctx, document, func(node *html.Node) bool {
		return node.Data == "main"
	})
	if err != nil {
		return readResponse{}, err
	}
	if len(mains) != 1 {
		return readResponse{}, newReadSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	categoryLabel, err := decodeReadCategoryLabel(ctx, mains[0])
	if err != nil {
		return readResponse{}, err
	}
	dls, err := readDescendantElements(ctx, mains[0], "dl")
	if err != nil {
		return readResponse{}, err
	}
	if len(dls) == 0 {
		return readResponse{}, newReadSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	fields := make([]readField, 0, len(dls))
	documents := make([]searchDocumentLink, 0)
	for _, dl := range dls {
		field, fieldDocuments, ok, fieldErr := decodeReadDL(ctx, dl)
		if fieldErr != nil {
			return readResponse{}, fieldErr
		}
		if !ok {
			continue
		}
		fields = append(fields, field)
		documents = append(documents, fieldDocuments...)
	}
	if len(fields) == 0 {
		return readResponse{}, invalidReadResponseError()
	}
	return readResponse{
		sourceCategoryLabel: categoryLabel,
		fields:              fields,
		documents:           documents,
	}, nil
}

func decodeReadCategoryLabel(ctx context.Context, main *html.Node) (string, error) {
	headers, err := readDescendantElements(ctx, main, "h4")
	if err != nil {
		return "", err
	}
	labels := make([]string, 0, 1)
	for _, header := range headers {
		text, textErr := readNodeText(ctx, header)
		if textErr != nil {
			return "", textErr
		}
		value := normalizeDisplayText(text)
		if value != "" {
			labels = append(labels, value)
		}
	}
	if len(labels) != 1 {
		return "", invalidReadResponseError()
	}
	return labels[0], nil
}

func decodeReadDL(
	ctx context.Context,
	dl *html.Node,
) (readField, []searchDocumentLink, bool, error) {
	var dt *html.Node
	var dd *html.Node
	for child := dl.FirstChild; child != nil; child = child.NextSibling {
		if err := readHTMLContextError(ctx); err != nil {
			return readField{}, nil, false, err
		}
		if child.Type != html.ElementNode || isIgnoredSearchElement(child) {
			continue
		}
		switch child.Data {
		case "dt":
			if dt != nil {
				return readField{}, nil, false, invalidReadResponseError()
			}
			dt = child
		case "dd":
			if dd != nil {
				return readField{}, nil, false, invalidReadResponseError()
			}
			dd = child
		}
	}
	if dt == nil || dd == nil {
		return readField{}, nil, false, invalidReadResponseError()
	}
	labelText, err := readNodeText(ctx, dt)
	if err != nil {
		return readField{}, nil, false, err
	}
	label := normalizeDisplayText(labelText)
	if label == "" {
		return readField{}, nil, false, invalidReadResponseError()
	}
	value, err := displayLinesWithoutLinks(ctx, dd)
	if err != nil {
		return readField{}, nil, false, err
	}
	links, err := decodeReadDocuments(ctx, dd)
	if err != nil {
		return readField{}, nil, false, err
	}
	return readField{label: label, value: strings.Join(value, "\n")}, links, true, nil
}

func decodeReadDocuments(ctx context.Context, dd *html.Node) ([]searchDocumentLink, error) {
	links, err := readDescendantElements(ctx, dd, "a")
	if err != nil {
		return nil, err
	}
	documents := make([]searchDocumentLink, 0, len(links))
	for _, link := range links {
		href, exists := singleAttribute(link, "href")
		if !exists || href == "" {
			return nil, invalidReadResponseError()
		}
		labelText, err := readNodeText(ctx, link)
		if err != nil {
			return nil, err
		}
		label := normalizeDisplayText(labelText)
		if label == "" {
			return nil, invalidReadResponseError()
		}
		documents = append(documents, searchDocumentLink{label: label, href: href})
	}
	return documents, nil
}

func displayLinesWithoutLinks(ctx context.Context, node *html.Node) ([]string, error) {
	value, err := nodeTextWithBreaksWithoutLinks(ctx, node)
	if err != nil {
		return nil, err
	}
	text := strings.ReplaceAll(value, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	raw := strings.Split(text, "\n")
	lines := make([]string, len(raw))
	for index, line := range raw {
		lines[index] = normalizeDisplayText(line)
	}
	start, end := 0, len(lines)
	for start < end && lines[start] == "" {
		start++
	}
	for end > start && lines[end-1] == "" {
		end--
	}
	return append([]string(nil), lines[start:end]...), nil
}

func nodeTextWithBreaksWithoutLinks(ctx context.Context, root *html.Node) (string, error) {
	var builder strings.Builder
	var visit func(*html.Node) error
	visit = func(node *html.Node) error {
		if err := readHTMLContextError(ctx); err != nil {
			return err
		}
		if node.Type == html.ElementNode && isIgnoredSearchElement(node) {
			return nil
		}
		if node.Type == html.ElementNode && node.Data == "a" {
			return nil
		}
		if node.Type == html.TextNode {
			builder.WriteString(node.Data)
		}
		if node.Type == html.ElementNode && node.Data == "br" {
			builder.WriteByte('\n')
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func hasReadPageTitle(ctx context.Context, root *html.Node) (bool, error) {
	titles, err := collectReadElements(ctx, root, func(node *html.Node) bool {
		return node.Data == "title"
	})
	if err != nil {
		return false, err
	}
	if len(titles) != 1 {
		return false, nil
	}
	text, err := readNodeText(ctx, titles[0])
	if err != nil {
		return false, err
	}
	return strings.Contains(
		normalizeDisplayText(text),
		"裁判例結果詳細",
	), nil
}

func collectReadElements(
	ctx context.Context,
	root *html.Node,
	match func(*html.Node) bool,
) ([]*html.Node, error) {
	found := make([]*html.Node, 0)
	pending := []*html.Node{root}
	for len(pending) > 0 {
		if err := readHTMLContextError(ctx); err != nil {
			return nil, err
		}
		last := len(pending) - 1
		node := pending[last]
		pending = pending[:last]
		if node.Type == html.ElementNode && isIgnoredSearchElement(node) {
			continue
		}
		if node.Type == html.ElementNode && match(node) {
			found = append(found, node)
		}
		for child := node.LastChild; child != nil; child = child.PrevSibling {
			pending = append(pending, child)
		}
	}
	return found, nil
}

func readDescendantElements(
	ctx context.Context,
	root *html.Node,
	name string,
) ([]*html.Node, error) {
	return collectReadElements(ctx, root, func(node *html.Node) bool {
		return node != root && node.Data == name
	})
}

func readNodeText(ctx context.Context, root *html.Node) (string, error) {
	var builder strings.Builder
	pending := []*html.Node{root}
	for len(pending) > 0 {
		if err := readHTMLContextError(ctx); err != nil {
			return "", err
		}
		last := len(pending) - 1
		node := pending[last]
		pending = pending[:last]
		if node.Type == html.ElementNode && isIgnoredSearchElement(node) {
			continue
		}
		if node.Type == html.TextNode {
			builder.WriteString(node.Data)
		}
		for child := node.LastChild; child != nil; child = child.PrevSibling {
			pending = append(pending, child)
		}
	}
	return builder.String(), nil
}

func readHTMLContextError(ctx context.Context) error {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return newReadSourceError(model.SourceErrorCodeSourceProcessingLimit, "")
	case errors.Is(ctx.Err(), context.Canceled):
		return context.Canceled
	default:
		return nil
	}
}

func classifyReadHTMLParseError(ctx context.Context) error {
	if err := readHTMLContextError(ctx); err != nil {
		return err
	}
	return newReadSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
}

func invalidReadResponseError() error {
	return newReadSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
}
