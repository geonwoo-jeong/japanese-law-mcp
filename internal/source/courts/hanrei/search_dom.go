package hanrei

import (
	"context"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"golang.org/x/net/html"
)

const tooBroadSearchMessagePrefix = "検索結果が2000件を超えました。「全文検索」欄の検索語を追加・変更してください。"

func decodeSearchResponseWithoutTable(
	emptyMarkerCount int,
	tooBroadMarkerCount int,
	visibleErrorMessageCount int,
) (searchResponse, error) {
	if tooBroadMarkerCount == 1 &&
		visibleErrorMessageCount == 1 &&
		emptyMarkerCount == 0 {
		return searchResponse{}, newSearchSourceError(
			model.SourceErrorCodeUnsupportedQuery,
			"",
		)
	}
	if tooBroadMarkerCount != 0 {
		return searchResponse{}, invalidSearchResponseError()
	}
	if visibleErrorMessageCount != 0 && emptyMarkerCount != 0 {
		return searchResponse{}, invalidSearchResponseError()
	}
	if emptyMarkerCount == 1 {
		return searchResponse{rows: []searchResultRow{}}, nil
	}
	if emptyMarkerCount > 1 {
		return searchResponse{}, invalidSearchResponseError()
	}
	return searchResponse{}, newSearchSourceError(
		model.SourceErrorCodeSourceContractChanged,
		"",
	)
}

func collectEmptySearchMarkers(
	ctx context.Context,
	root *html.Node,
) (int, error) {
	candidates, err := collectElements(ctx, root, func(node *html.Node) bool {
		id, exists := singleAttribute(node, "id")
		return node.Data == "p" &&
			exists &&
			id == "searched" &&
			!hasIgnoredAncestor(node)
	})
	if err != nil {
		return 0, err
	}
	markerCount := 0
	for _, candidate := range candidates {
		text, textErr := nodeText(ctx, candidate)
		if textErr != nil {
			return 0, textErr
		}
		if normalizeDisplayText(text) == emptySearchMessage {
			markerCount++
		}
	}
	return markerCount, nil
}

func collectSearchErrorMarkers(
	ctx context.Context,
	root *html.Node,
) (int, int, error) {
	candidates, err := collectElements(ctx, root, func(node *html.Node) bool {
		return node.Data == "ul" &&
			hasClass(node, "errorMessage") &&
			!hasIgnoredAncestor(node)
	})
	if err != nil {
		return 0, 0, err
	}
	ambiguousStyle, err := hasInlineStyleAroundErrorMessages(
		ctx,
		root,
		candidates,
	)
	if err != nil {
		return 0, 0, err
	}
	if ambiguousStyle {
		return 0, 0, invalidSearchResponseError()
	}
	markerCount := 0
	for _, candidate := range candidates {
		text, textErr := nodeText(ctx, candidate)
		if textErr != nil {
			return 0, 0, textErr
		}
		if strings.HasPrefix(
			normalizeDisplayText(text),
			tooBroadSearchMessagePrefix,
		) {
			markerCount++
		}
	}
	return markerCount, len(candidates), nil
}

func hasInlineStyleAroundErrorMessages(
	ctx context.Context,
	root *html.Node,
	candidates []*html.Node,
) (bool, error) {
	if len(candidates) == 0 {
		return false, nil
	}
	candidateSet := make(map[*html.Node]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidateSet[candidate] = struct{}{}
	}
	styledNodes, err := collectElements(ctx, root, func(node *html.Node) bool {
		return hasInlineStyleAttribute(node) && !hasIgnoredAncestor(node)
	})
	if err != nil {
		return false, err
	}
	styledNodeSet := make(map[*html.Node]struct{}, len(styledNodes))
	for _, styledNode := range styledNodes {
		styledNodeSet[styledNode] = struct{}{}
	}
	for _, candidate := range candidates {
		found, findErr := hasNodeInAncestorSet(ctx, candidate, styledNodeSet)
		if findErr != nil || found {
			return found, findErr
		}
	}
	for _, styledNode := range styledNodes {
		found, findErr := hasNodeInAncestorSet(ctx, styledNode, candidateSet)
		if findErr != nil || found {
			return found, findErr
		}
	}
	return false, nil
}

func hasNodeInAncestorSet(
	ctx context.Context,
	node *html.Node,
	nodeSet map[*html.Node]struct{},
) (bool, error) {
	for current := node; current != nil; current = current.Parent {
		if err := searchHTMLContextError(ctx); err != nil {
			return false, err
		}
		if _, exists := nodeSet[current]; exists {
			return true, nil
		}
	}
	return false, nil
}

func descendantElements(
	ctx context.Context,
	root *html.Node,
	name string,
) ([]*html.Node, error) {
	found := make([]*html.Node, 0)
	err := walkVisibleTree(ctx, root, false, func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == name {
			found = append(found, node)
		}
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

func nodeTextWithBreaks(ctx context.Context, root *html.Node) (string, error) {
	var builder strings.Builder
	err := walkVisibleTree(ctx, root, true, func(node *html.Node) {
		if node.Type == html.TextNode {
			builder.WriteString(node.Data)
		}
		if node.Type == html.ElementNode && node.Data == "br" {
			builder.WriteByte('\n')
		}
	})
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}

func nodeText(ctx context.Context, root *html.Node) (string, error) {
	var builder strings.Builder
	err := walkVisibleTree(ctx, root, true, func(node *html.Node) {
		if node.Type == html.TextNode {
			builder.WriteString(node.Data)
		}
	})
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}

func walkVisibleTree(
	ctx context.Context,
	root *html.Node,
	includeRoot bool,
	visit func(*html.Node),
) error {
	pending := make([]*html.Node, 0)
	if includeRoot {
		pending = append(pending, root)
	} else {
		pending = appendVisibleChildrenReverse(pending, root)
	}
	for len(pending) != 0 {
		if err := searchHTMLContextError(ctx); err != nil {
			return err
		}
		last := len(pending) - 1
		node := pending[last]
		pending = pending[:last]
		if node.Type == html.ElementNode && isIgnoredSearchElement(node) {
			continue
		}
		visit(node)
		pending = appendVisibleChildrenReverse(pending, node)
	}
	return nil
}

func appendVisibleChildrenReverse(
	pending []*html.Node,
	node *html.Node,
) []*html.Node {
	if isClosedDetails(node) {
		if summary := firstSummaryChild(node); summary != nil {
			return append(pending, summary)
		}
		return pending
	}
	for child := node.LastChild; child != nil; child = child.PrevSibling {
		pending = append(pending, child)
	}
	return pending
}

func hasIgnoredAncestor(node *html.Node) bool {
	var child *html.Node
	for current := node; current != nil; current = current.Parent {
		if current.Type == html.ElementNode {
			if isClosedDetails(current) &&
				child != nil &&
				firstSummaryChild(current) != child {
				return true
			}
			if isIgnoredSearchElement(current) {
				return true
			}
		}
		child = current
	}
	return false
}

func isIgnoredSearchElement(node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode {
		return false
	}
	if _, hidden := singleAttribute(node, "hidden"); hidden {
		return true
	}
	ariaHidden, _ := singleAttribute(node, "aria-hidden")
	if strings.EqualFold(strings.TrimSpace(ariaHidden), "true") ||
		isClosedDialog(node) ||
		hasClass(node, "modal") ||
		hasClass(node, "d-none") {
		return true
	}
	switch node.Data {
	case "script", "style", "template":
		return true
	default:
		return false
	}
}

func hasInlineStyleAttribute(node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode {
		return false
	}
	for _, attribute := range node.Attr {
		if attribute.Namespace == "" && attribute.Key == "style" {
			return true
		}
	}
	return false
}

func isClosedDialog(node *html.Node) bool {
	if node.Data != "dialog" {
		return false
	}
	_, open := singleAttribute(node, "open")
	return !open
}

func isClosedDetails(node *html.Node) bool {
	if node.Data != "details" {
		return false
	}
	_, open := singleAttribute(node, "open")
	return !open
}

func firstSummaryChild(details *html.Node) *html.Node {
	for current := details.FirstChild; current != nil; current = current.NextSibling {
		if current.Type == html.ElementNode && current.Data == "summary" {
			return current
		}
	}
	return nil
}
