package hanrei

import (
	"bytes"
	"context"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"golang.org/x/net/html"
)

const (
	maximumSearchHTMLNodes = 100000
	maximumSearchHTMLDepth = 64
	emptySearchMessage     = "該当する裁判例がありませんでした。"
)

var searchTotalPattern = regexp.MustCompile(`([0-9０-９]+)件中`)

func parseSearchResponse(
	ctx context.Context,
	body []byte,
) (searchResponse, error) {
	if ctx == nil {
		return searchResponse{}, errors.New("context は必須です")
	}
	if len(body) > maximumSearchDecompressedBytes {
		return searchResponse{}, newSearchSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	if !utf8.Valid(body) {
		return searchResponse{}, newSearchSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}
	if err := searchHTMLContextError(ctx); err != nil {
		return searchResponse{}, err
	}
	if err := validateSearchHTMLTokenBudget(ctx, body); err != nil {
		return searchResponse{}, err
	}
	document, err := html.Parse(&htmlContextReader{
		ctx:    ctx,
		reader: bytes.NewReader(body),
	})
	if err != nil {
		return searchResponse{}, classifySearchHTMLParseError(ctx)
	}
	if err := validateSearchHTMLBudget(ctx, document); err != nil {
		return searchResponse{}, err
	}
	return decodeSearchDocument(ctx, document)
}

func validateSearchHTMLTokenBudget(ctx context.Context, body []byte) error {
	tokenizer := html.NewTokenizer(&htmlContextReader{
		ctx:    ctx,
		reader: bytes.NewReader(body),
	})
	count := 0
	openElements := make([]string, 0, maximumSearchHTMLDepth)
	for {
		if err := searchHTMLContextError(ctx); err != nil {
			return err
		}
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			if errors.Is(tokenizer.Err(), io.EOF) {
				return nil
			}
			return invalidSearchResponseError()
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			count += 1 + len(token.Attr)
			if count > maximumSearchHTMLNodes {
				return newSearchSourceError(
					model.SourceErrorCodeSourceResponseTooLarge,
					"",
				)
			}
			if tokenType == html.StartTagToken &&
				!isVoidSearchHTMLElement(token.Data) {
				openElements = append(openElements, token.Data)
				if len(openElements) > maximumSearchHTMLDepth {
					return newSearchSourceError(
						model.SourceErrorCodeUnsafeSourceContent,
						"",
					)
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
		if count > maximumSearchHTMLNodes {
			return newSearchSourceError(
				model.SourceErrorCodeSourceResponseTooLarge,
				"",
			)
		}
	}
}

func isVoidSearchHTMLElement(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

func decodeSearchDocument(
	ctx context.Context,
	document *html.Node,
) (searchResponse, error) {
	if !hasSearchPageTitle(ctx, document) {
		if err := searchHTMLContextError(ctx); err != nil {
			return searchResponse{}, err
		}
		return searchResponse{}, newSearchSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	tables, err := collectElements(ctx, document, isSearchResultTable)
	if err != nil {
		return searchResponse{}, err
	}
	emptyMarkers, err := collectEmptySearchMarkers(ctx, document)
	if err != nil {
		return searchResponse{}, err
	}
	tooBroadMarkers, visibleErrorMessages, err := collectSearchErrorMarkers(
		ctx,
		document,
	)
	if err != nil {
		return searchResponse{}, err
	}
	if len(tables) == 0 {
		return decodeSearchResponseWithoutTable(
			emptyMarkers,
			tooBroadMarkers,
			visibleErrorMessages,
		)
	}
	if len(tables) != 1 {
		return searchResponse{}, invalidSearchResponseError()
	}
	if visibleErrorMessages != 0 {
		return searchResponse{}, invalidSearchResponseError()
	}
	return decodePopulatedSearchResponse(ctx, tables[0], emptyMarkers)
}

func decodePopulatedSearchResponse(
	ctx context.Context,
	table *html.Node,
	emptyMarkerCount int,
) (searchResponse, error) {
	rows, err := directSearchRows(ctx, table)
	if err != nil {
		return searchResponse{}, err
	}
	if len(rows) == 0 && emptyMarkerCount == 1 {
		return searchResponse{rows: []searchResultRow{}}, nil
	}
	if emptyMarkerCount != 0 || len(rows) == 0 {
		return searchResponse{}, invalidSearchResponseError()
	}
	decoded := make([]searchResultRow, len(rows))
	for index, row := range rows {
		if err := searchHTMLContextError(ctx); err != nil {
			return searchResponse{}, err
		}
		item, err := decodeSearchRow(ctx, row)
		if err != nil {
			return searchResponse{}, err
		}
		decoded[index] = item
	}
	total, err := decodeSearchTotal(ctx, table)
	if err != nil {
		return searchResponse{}, err
	}
	if total == 0 || total < len(decoded) {
		return searchResponse{}, invalidSearchResponseError()
	}
	return searchResponse{totalCount: total, rows: decoded}, nil
}

func decodeSearchRow(
	ctx context.Context,
	row *html.Node,
) (searchResultRow, error) {
	headers, informationCells, fileCells := classifySearchCells(row)
	if len(headers) != 1 || len(informationCells) != 1 || len(fileCells) != 1 {
		return searchResultRow{}, invalidSearchResponseError()
	}
	detailHref, categoryLabel, err := decodeSearchDetailLink(ctx, headers[0])
	if err != nil {
		return searchResultRow{}, err
	}
	values, err := decodeSearchInformation(ctx, informationCells[0])
	if err != nil {
		return searchResultRow{}, err
	}
	documents, err := decodeSearchDocuments(ctx, fileCells[0])
	if err != nil {
		return searchResultRow{}, err
	}
	values.detailHref = detailHref
	values.sourceCategoryLabel = categoryLabel
	values.documents = documents
	values.location = searchRowLocation(row)
	return values, nil
}

func classifySearchCells(
	row *html.Node,
) (headers []*html.Node, informationCells []*html.Node, fileCells []*html.Node) {
	for child := row.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode || hasIgnoredAncestor(child) {
			continue
		}
		switch child.Data {
		case "th":
			headers = append(headers, child)
		case "td":
			if hasClass(child, "file-col") {
				fileCells = append(fileCells, child)
			} else {
				informationCells = append(informationCells, child)
			}
		}
	}
	return headers, informationCells, fileCells
}

func decodeSearchDetailLink(
	ctx context.Context,
	header *html.Node,
) (string, string, error) {
	links, err := descendantElements(ctx, header, "a")
	if err != nil {
		return "", "", err
	}
	detailLinkCount := 0
	var selectedHref string
	var selectedLabel string
	for _, link := range links {
		href, exists := singleAttribute(link, "href")
		if !exists {
			continue
		}
		_, valid := canonicalDetailPath(searchEndpoint, href)
		if !valid {
			continue
		}
		labelText, textErr := nodeText(ctx, link)
		if textErr != nil {
			return "", "", textErr
		}
		label := normalizeDisplayText(labelText)
		if label == "" {
			return "", "", invalidSearchResponseError()
		}
		detailLinkCount++
		selectedHref, selectedLabel = href, label
	}
	if detailLinkCount != 1 || selectedHref == "" {
		return "", "", invalidSearchResponseError()
	}
	return selectedHref, selectedLabel, nil
}

func decodeSearchInformation(
	ctx context.Context,
	cell *html.Node,
) (searchResultRow, error) {
	paragraphs, err := descendantElements(ctx, cell, "p")
	if err != nil {
		return searchResultRow{}, err
	}
	if len(paragraphs) < 2 {
		return searchResultRow{}, invalidSearchResponseError()
	}
	caseLines, err := displayLines(ctx, paragraphs[0])
	if err != nil {
		return searchResultRow{}, err
	}
	detailLines, err := displayLines(ctx, paragraphs[1])
	if err != nil {
		return searchResultRow{}, err
	}
	caseLines = nonEmptyLines(caseLines)
	if len(caseLines) == 0 ||
		len(detailLines) < 2 ||
		detailLines[0] == "" ||
		detailLines[1] == "" {
		return searchResultRow{}, invalidSearchResponseError()
	}
	return searchResultRow{
		caseNumber:   caseLines[0],
		caseName:     strings.Join(caseLines[1:], " "),
		decisionDate: detailLines[0],
		courtName:    detailLines[1],
		branchName:   lineAt(detailLines, 2),
		divisionName: lineAt(detailLines, 3),
		decisionType: lineAt(detailLines, 6),
		outcome:      lineAt(detailLines, 7),
	}, nil
}

func decodeSearchDocuments(
	ctx context.Context,
	cell *html.Node,
) ([]searchDocumentLink, error) {
	links, err := descendantElements(ctx, cell, "a")
	if err != nil {
		return nil, err
	}
	documents := make([]searchDocumentLink, 0, len(links))
	for _, link := range links {
		href, exists := singleAttribute(link, "href")
		labelText, textErr := nodeText(ctx, link)
		if textErr != nil {
			return nil, textErr
		}
		label := normalizeDisplayText(labelText)
		if !exists || href == "" || label == "" {
			return nil, invalidSearchResponseError()
		}
		documents = append(documents, searchDocumentLink{
			label: label,
			href:  href,
		})
	}
	return documents, nil
}

func decodeSearchTotal(
	ctx context.Context,
	table *html.Node,
) (int, error) {
	root := table
	for root.Parent != nil {
		root = root.Parent
	}
	paragraphs, err := collectElements(ctx, root, func(node *html.Node) bool {
		return node.Data == "p" && !hasIgnoredAncestor(node)
	})
	if err != nil {
		return 0, err
	}
	pagingParagraphs := make([]*html.Node, 0)
	fallbackParagraphs := make([]*html.Node, 0)
	for _, paragraph := range paragraphs {
		if hasAncestorClass(paragraph, "paging-parts2") {
			pagingParagraphs = append(pagingParagraphs, paragraph)
		} else if !hasSearchTableAncestor(paragraph) {
			fallbackParagraphs = append(fallbackParagraphs, paragraph)
		}
	}
	if len(pagingParagraphs) != 0 {
		fallbackParagraphs = pagingParagraphs
	}
	values := make(map[int]struct{})
	for _, paragraph := range fallbackParagraphs {
		text, textErr := nodeText(ctx, paragraph)
		if textErr != nil {
			return 0, textErr
		}
		for _, match := range searchTotalPattern.FindAllStringSubmatch(text, -1) {
			total, parseErr := parseSearchTotalDigits(match[1])
			if parseErr != nil {
				return 0, invalidSearchResponseError()
			}
			values[total] = struct{}{}
		}
	}
	if len(values) == 0 {
		return 0, newSearchSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	if len(values) != 1 {
		return 0, invalidSearchResponseError()
	}
	for total := range values {
		return total, nil
	}
	panic("到達不能")
}

func parseSearchTotalDigits(value string) (int, error) {
	var builder strings.Builder
	for _, character := range value {
		switch {
		case character >= '0' && character <= '9':
			builder.WriteRune(character)
		case character >= '０' && character <= '９':
			builder.WriteRune('0' + character - '０')
		default:
			return 0, strconv.ErrSyntax
		}
	}
	return strconv.Atoi(builder.String())
}

func validateSearchHTMLBudget(ctx context.Context, root *html.Node) error {
	type pendingNode struct {
		node         *html.Node
		elementDepth int
	}
	pending := []pendingNode{{node: root}}
	count := 0
	for len(pending) > 0 {
		if err := searchHTMLContextError(ctx); err != nil {
			return err
		}
		last := len(pending) - 1
		current := pending[last]
		pending = pending[:last]
		if err := countSearchHTMLNode(current, &count); err != nil {
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

func countSearchHTMLNode(current struct {
	node         *html.Node
	elementDepth int
}, count *int) error {
	nodeCount := 0
	switch current.node.Type {
	case html.ElementNode:
		if current.elementDepth > maximumSearchHTMLDepth {
			return newSearchSourceError(model.SourceErrorCodeUnsafeSourceContent, "")
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
	if *count > maximumSearchHTMLNodes {
		return newSearchSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	}
	return nil
}

func hasSearchPageTitle(ctx context.Context, root *html.Node) bool {
	titles, err := collectElements(ctx, root, func(node *html.Node) bool {
		return node.Data == "title"
	})
	if err != nil || len(titles) != 1 {
		return false
	}
	text, err := nodeText(ctx, titles[0])
	return err == nil &&
		strings.Contains(normalizeDisplayText(text), "裁判例検索")
}

func collectElements(
	ctx context.Context,
	root *html.Node,
	match func(*html.Node) bool,
) ([]*html.Node, error) {
	found := make([]*html.Node, 0)
	pending := []*html.Node{root}
	for len(pending) > 0 {
		if err := searchHTMLContextError(ctx); err != nil {
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

func directSearchRows(
	ctx context.Context,
	table *html.Node,
) ([]*html.Node, error) {
	rows := make([]*html.Node, 0)
	for child := table.FirstChild; child != nil; child = child.NextSibling {
		if err := searchHTMLContextError(ctx); err != nil {
			return nil, err
		}
		if child.Type != html.ElementNode {
			continue
		}
		if hasIgnoredAncestor(child) {
			continue
		}
		if child.Data == "tr" {
			rows = append(rows, child)
			continue
		}
		if child.Data != "tbody" {
			continue
		}
		for row := child.FirstChild; row != nil; row = row.NextSibling {
			if err := searchHTMLContextError(ctx); err != nil {
				return nil, err
			}
			if row.Type == html.ElementNode &&
				row.Data == "tr" &&
				!hasIgnoredAncestor(row) {
				rows = append(rows, row)
			}
		}
	}
	return rows, nil
}

func displayLines(ctx context.Context, node *html.Node) ([]string, error) {
	value, err := nodeTextWithBreaks(ctx, node)
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

func normalizeDisplayText(value string) string {
	return strings.Join(strings.FieldsFunc(value, unicode.IsSpace), " ")
}

func nonEmptyLines(lines []string) []string {
	values := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			values = append(values, line)
		}
	}
	return values
}

func lineAt(lines []string, index int) string {
	if index < 0 || index >= len(lines) {
		return ""
	}
	return lines[index]
}

func singleAttribute(node *html.Node, name string) (string, bool) {
	var value string
	found := false
	for _, attribute := range node.Attr {
		if attribute.Namespace == "" && attribute.Key == name {
			if found {
				return "", false
			}
			value = attribute.Val
			found = true
		}
	}
	return value, found
}

func hasClass(node *html.Node, className string) bool {
	value, exists := singleAttribute(node, "class")
	if !exists {
		return false
	}
	for _, candidate := range strings.Fields(value) {
		if candidate == className {
			return true
		}
	}
	return false
}

func hasAncestorClass(node *html.Node, className string) bool {
	for current := node.Parent; current != nil; current = current.Parent {
		if current.Type == html.ElementNode && hasClass(current, className) {
			return true
		}
	}
	return false
}

func hasSearchTableAncestor(node *html.Node) bool {
	for current := node.Parent; current != nil; current = current.Parent {
		if current.Type == html.ElementNode && isSearchResultTable(current) {
			return true
		}
	}
	return false
}

func isSearchResultTable(node *html.Node) bool {
	return node.Data == "table" &&
		hasClass(node, "search-result-table") &&
		!hasIgnoredAncestor(node)
}

func searchRowLocation(row *html.Node) string {
	if row == nil || row.Parent == nil {
		return ""
	}
	index := 0
	for sibling := row.Parent.FirstChild; sibling != nil; sibling = sibling.NextSibling {
		if sibling.Type == html.ElementNode && sibling.Data == "tr" {
			index++
		}
		if sibling == row {
			break
		}
	}
	if index <= 0 {
		return ""
	}
	if row.Parent.Type == html.ElementNode && row.Parent.Data == "tbody" {
		return "table.search-result-table tbody tr[" + strconv.Itoa(index) + "]"
	}
	return "table.search-result-table tr[" + strconv.Itoa(index) + "]"
}

func searchHTMLContextError(ctx context.Context) error {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return newSearchSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			"",
		)
	case errors.Is(ctx.Err(), context.Canceled):
		return context.Canceled
	default:
		return nil
	}
}

func classifySearchHTMLParseError(ctx context.Context) error {
	if err := searchHTMLContextError(ctx); err != nil {
		return err
	}
	return newSearchSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
}

func invalidSearchResponseError() error {
	return newSearchSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
}

type htmlContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *htmlContextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	const maximumReadSize = 32 * 1024
	if len(buffer) > maximumReadSize {
		buffer = buffer[:maximumReadSize]
	}
	return r.reader.Read(buffer)
}
