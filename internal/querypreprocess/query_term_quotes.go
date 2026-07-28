package querypreprocess

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

type quoteFrame struct {
	contentStart int
	close        rune
	invalid      bool
}

type quotedPhraseScan struct {
	drafts         []queryTermDraft
	protectedSpans []byteSpan
}

func cuesOutsideQuoteProtection(
	cues []legalquery.CueMention,
	protectedSpans []byteSpan,
) []legalquery.CueMention {
	values := make([]legalquery.CueMention, 0, len(cues))
	for _, cue := range cues {
		span := cue.Span()
		insideQuote := false
		for _, protected := range protectedSpans {
			if span.StartByte() < protected.endByte &&
				protected.startByte < span.EndByte() {
				insideQuote = true
				break
			}
		}
		if !insideQuote {
			values = append(values, cue)
		}
	}
	return values
}

func scanQuotedPhrases(query string) quotedPhraseScan {
	result := quotedPhraseScan{
		drafts:         make([]queryTermDraft, 0),
		protectedSpans: make([]byteSpan, 0),
	}
	stack := make([]quoteFrame, 0, 2)
	for index, current := range query {
		if len(stack) > 0 && current == stack[len(stack)-1].close {
			if current == '"' && looksLikeNestedASCIIQuote(query, index) {
				for frameIndex := range stack {
					stack[frameIndex].invalid = true
				}
				stack = append(stack, quoteFrame{
					contentStart: index + 1,
					close:        '"',
					invalid:      true,
				})
				continue
			}
			frame := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			result.protectedSpans = appendNonEmptySpan(
				result.protectedSpans,
				frame.contentStart,
				index,
			)
			if frame.invalid || len(stack) > 0 {
				if len(stack) > 0 {
					stack[len(stack)-1].invalid = true
				}
				continue
			}
			startByte, endByte := trimUnicodeSpaceSpan(
				query,
				frame.contentStart,
				index,
			)
			if startByte < endByte {
				result.drafts = append(result.drafts, queryTermDraft{
					startByte:  startByte,
					endByte:    endByte,
					startToken: -1,
					endToken:   -1,
					kind:       legalquery.QueryTermMentionQuotedPhrase,
				})
			}
			continue
		}

		closeQuote, opens := matchingCloseQuote(current)
		if opens {
			nested := len(stack) > 0
			if nested {
				for frameIndex := range stack {
					stack[frameIndex].invalid = true
				}
			}
			stack = append(stack, quoteFrame{
				contentStart: index + utf8.RuneLen(current),
				close:        closeQuote,
				invalid:      nested,
			})
			continue
		}
		if isRecognizedCloseQuote(current) && len(stack) > 0 {
			stack[len(stack)-1].invalid = true
		}
	}
	for _, frame := range stack {
		result.protectedSpans = appendNonEmptySpan(
			result.protectedSpans,
			frame.contentStart,
			len(query),
		)
	}
	return result
}

func looksLikeNestedASCIIQuote(query string, quoteByte int) bool {
	nextRelative := strings.IndexByte(query[quoteByte+1:], '"')
	if nextRelative < 0 {
		return false
	}
	nextQuote := quoteByte + 1 + nextRelative
	finalRelative := strings.IndexByte(query[nextQuote+1:], '"')
	if finalRelative < 0 {
		return false
	}
	finalQuote := nextQuote + 1 + finalRelative
	between := query[quoteByte+1 : nextQuote]
	if hasClauseBoundary(between) || hasClauseBoundary(
		query[nextQuote+1:finalQuote],
	) {
		return false
	}
	return !isSafeASCIIQuoteSeparator(between)
}

func isSafeASCIIQuoteSeparator(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	for _, prefix := range []string{
		"と",
		"または",
		"又は",
		"若しくは",
		"および",
		"及び",
		"並びに",
		"を",
		"は",
		"が",
		"に",
		"で",
		"の",
		"へ",
		"も",
		"、",
		"，",
		",",
		"・",
	} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

func hasClauseBoundary(value string) bool {
	for _, current := range value {
		switch current {
		case '。', '！', '!', '？', '?', '；', ';':
			return true
		}
	}
	return false
}

func appendNonEmptySpan(
	values []byteSpan,
	startByte int,
	endByte int,
) []byteSpan {
	if startByte >= endByte {
		return values
	}
	return append(values, byteSpan{
		startByte: startByte,
		endByte:   endByte,
	})
}

func matchingCloseQuote(value rune) (rune, bool) {
	switch value {
	case '「':
		return '」', true
	case '『':
		return '』', true
	case '“':
		return '”', true
	case '"':
		return '"', true
	default:
		return 0, false
	}
}

func isRecognizedCloseQuote(value rune) bool {
	switch value {
	case '」', '』', '”':
		return true
	default:
		return false
	}
}

func trimUnicodeSpaceSpan(query string, startByte int, endByte int) (int, int) {
	for startByte < endByte {
		current, size := utf8.DecodeRuneInString(query[startByte:endByte])
		if !unicode.IsSpace(current) {
			break
		}
		startByte += size
	}
	for startByte < endByte {
		current, size := utf8.DecodeLastRuneInString(query[startByte:endByte])
		if !unicode.IsSpace(current) {
			break
		}
		endByte -= size
	}
	return startByte, endByte
}
