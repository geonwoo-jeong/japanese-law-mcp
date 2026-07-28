package querypreprocess

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/nlp/kagome"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

type queryTermDraft struct {
	startByte  int
	endByte    int
	startToken int
	endToken   int
	kind       legalquery.QueryTermMentionKind
}

func extractQueryTermMentions(
	ctx context.Context,
	query string,
	tokens []kagome.TokenOccurrence,
	protectedSpans []byteSpan,
	cues []legalquery.CueMention,
) ([]legalquery.QueryTermMention, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	quoteScan := scanQuotedPhrases(query)
	quoted := quoteScan.drafts
	morphologicalProtection := append([]byteSpan(nil), protectedSpans...)
	morphologicalProtection = append(
		morphologicalProtection,
		quoteScan.protectedSpans...,
	)
	anchoringCues := cuesOutsideQuoteProtection(
		cues,
		quoteScan.protectedSpans,
	)
	morphological := morphologicalPhraseDrafts(
		query,
		tokens,
		morphologicalProtection,
		anchoringCues,
	)
	drafts := append([]queryTermDraft(nil), quoted...)
	drafts = append(drafts, morphological...)
	drafts = deduplicateQueryTermDrafts(drafts)

	mentions := make([]legalquery.QueryTermMention, 0, len(drafts))
	for index, draft := range drafts {
		if index%16 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
			StartByte: draft.startByte,
			EndByte:   draft.endByte,
		})
		if err != nil {
			return nil, err
		}
		mention, err := legalquery.NewQueryTermMention(
			legalquery.QueryTermMentionValues{
				Span:    span,
				Surface: query[draft.startByte:draft.endByte],
				Kind:    draft.kind,
			},
		)
		if err != nil {
			return nil, err
		}
		mentions = append(mentions, mention)
	}
	return mentions, nil
}

func morphologicalPhraseDrafts(
	query string,
	tokens []kagome.TokenOccurrence,
	protectedSpans []byteSpan,
	cues []legalquery.CueMention,
) []queryTermDraft {
	phrases := nounPhraseDrafts(query, tokens, protectedSpans)
	anchored := make([]bool, len(phrases))
	for index, phrase := range phrases {
		anchored[index] = isStandaloneNounPhrase(query, tokens, phrase) ||
			hasForwardPhraseAnchor(query, tokens, phrase, cues)
	}
	for index := len(phrases) - 1; index > 0; index-- {
		if !anchored[index] {
			continue
		}
		for previous := index - 1; previous >= 0; previous-- {
			if !isParallelPhraseSeparator(
				query[phrases[previous].endByte:phrases[previous+1].startByte],
			) {
				break
			}
			anchored[previous] = true
		}
	}

	values := make([]queryTermDraft, 0, len(phrases))
	for index, phrase := range phrases {
		if anchored[index] {
			values = append(values, phrase)
		}
	}
	return values
}

func nounPhraseDrafts(
	query string,
	tokens []kagome.TokenOccurrence,
	protectedSpans []byteSpan,
) []queryTermDraft {
	values := make([]queryTermDraft, 0)
	for index := 0; index < len(tokens); {
		startToken, centerToken, found := nounPhraseStart(
			tokens,
			index,
			protectedSpans,
		)
		if !found {
			index++
			continue
		}
		endToken := nounPhraseEnd(tokens, centerToken, protectedSpans)
		startByte := tokens[startToken].StartByte()
		endByte := tokens[endToken].EndByte()
		surface := query[startByte:endByte]
		if containsJapaneseScript(surface) &&
			utf8.RuneCountInString(querynormalization.ComparisonKey(surface)) >= 2 {
			values = append(values, queryTermDraft{
				startByte:  startByte,
				endByte:    endByte,
				startToken: startToken,
				endToken:   endToken,
				kind:       legalquery.QueryTermMentionMorphologicalPhrase,
			})
		}
		index = endToken + 1
	}
	return values
}

func containsJapaneseScript(value string) bool {
	for _, current := range value {
		if unicode.In(
			current,
			unicode.Hiragana,
			unicode.Katakana,
			unicode.Han,
		) {
			return true
		}
	}
	return false
}

func nounPhraseStart(
	tokens []kagome.TokenOccurrence,
	index int,
	protectedSpans []byteSpan,
) (int, int, bool) {
	if isSearchTermNoun(tokens[index], protectedSpans) {
		return index, index, true
	}
	if isSearchTermPrefix(tokens[index], protectedSpans) {
		next := index + 1
		if next < len(tokens) &&
			isSearchTermNoun(tokens[next], protectedSpans) {
			return index, next, true
		}
	}
	if centerToken, found := numericModifierCenter(
		tokens,
		index,
		protectedSpans,
	); found {
		return index, centerToken, true
	}
	return 0, 0, false
}

func numericModifierCenter(
	tokens []kagome.TokenOccurrence,
	index int,
	protectedSpans []byteSpan,
) (int, bool) {
	cursor := index
	if isNumericConnectionPrefix(tokens[cursor], protectedSpans) {
		cursor++
	}
	if cursor >= len(tokens) ||
		!isSearchTermNumber(tokens[cursor], protectedSpans) {
		return 0, false
	}
	for cursor < len(tokens) &&
		isSearchTermNumber(tokens[cursor], protectedSpans) {
		cursor++
	}
	counterFound := false
	for cursor < len(tokens) &&
		isSearchTermSuffix(tokens[cursor], protectedSpans) {
		counterFound = true
		cursor++
	}
	if !counterFound {
		return 0, false
	}
	if cursor < len(tokens) &&
		isGenitiveConnector(tokens[cursor], protectedSpans) {
		cursor++
	}
	for cursor < len(tokens) &&
		isSearchTermPrefix(tokens[cursor], protectedSpans) {
		cursor++
	}
	if cursor >= len(tokens) ||
		!isSearchTermNoun(tokens[cursor], protectedSpans) {
		return 0, false
	}
	return cursor, true
}

func nounPhraseEnd(
	tokens []kagome.TokenOccurrence,
	startToken int,
	protectedSpans []byteSpan,
) int {
	endToken := startToken
	for next := startToken + 1; next < len(tokens); {
		switch {
		case isSearchTermNoun(tokens[next], protectedSpans),
			isSearchTermSuffix(tokens[next], protectedSpans):
			endToken = next
			next++
		case isGenitiveConnector(tokens[next], protectedSpans):
			following := next + 1
			if following < len(tokens) &&
				isSearchTermPrefix(tokens[following], protectedSpans) {
				following++
			}
			if following >= len(tokens) ||
				!isSearchTermNoun(tokens[following], protectedSpans) {
				return endToken
			}
			endToken = following
			next = following + 1
		default:
			return endToken
		}
	}
	return endToken
}

func isSearchTermNoun(
	token kagome.TokenOccurrence,
	protectedSpans []byteSpan,
) bool {
	if token.UserDictionary() ||
		overlapsAny(token.StartByte(), token.EndByte(), protectedSpans) ||
		!isLexicalToken(token.Surface()) {
		return false
	}
	partOfSpeech := token.PartOfSpeech()
	if len(partOfSpeech) == 0 || partOfSpeech[0] != "名詞" {
		return false
	}
	if len(partOfSpeech) < 2 {
		return true
	}
	switch partOfSpeech[1] {
	case "数", "代名詞", "非自立", "接尾":
		return false
	default:
		return true
	}
}

func isSearchTermPrefix(
	token kagome.TokenOccurrence,
	protectedSpans []byteSpan,
) bool {
	if token.UserDictionary() ||
		overlapsAny(token.StartByte(), token.EndByte(), protectedSpans) ||
		!isLexicalToken(token.Surface()) {
		return false
	}
	partOfSpeech := token.PartOfSpeech()
	return len(partOfSpeech) >= 2 &&
		partOfSpeech[0] == "接頭詞" &&
		partOfSpeech[1] == "名詞接続"
}

func isNumericConnectionPrefix(
	token kagome.TokenOccurrence,
	protectedSpans []byteSpan,
) bool {
	if token.UserDictionary() ||
		overlapsAny(token.StartByte(), token.EndByte(), protectedSpans) ||
		!isLexicalToken(token.Surface()) {
		return false
	}
	partOfSpeech := token.PartOfSpeech()
	return len(partOfSpeech) >= 2 &&
		partOfSpeech[0] == "接頭詞" &&
		partOfSpeech[1] == "数接続"
}

func isSearchTermNumber(
	token kagome.TokenOccurrence,
	protectedSpans []byteSpan,
) bool {
	if token.UserDictionary() ||
		overlapsAny(token.StartByte(), token.EndByte(), protectedSpans) ||
		!isLexicalToken(token.Surface()) {
		return false
	}
	partOfSpeech := token.PartOfSpeech()
	return len(partOfSpeech) >= 2 &&
		partOfSpeech[0] == "名詞" &&
		partOfSpeech[1] == "数"
}

func isSearchTermSuffix(
	token kagome.TokenOccurrence,
	protectedSpans []byteSpan,
) bool {
	if token.UserDictionary() ||
		overlapsAny(token.StartByte(), token.EndByte(), protectedSpans) ||
		!isLexicalToken(token.Surface()) {
		return false
	}
	partOfSpeech := token.PartOfSpeech()
	return len(partOfSpeech) >= 2 &&
		partOfSpeech[0] == "名詞" &&
		partOfSpeech[1] == "接尾"
}

func isGenitiveConnector(
	token kagome.TokenOccurrence,
	protectedSpans []byteSpan,
) bool {
	if overlapsAny(token.StartByte(), token.EndByte(), protectedSpans) ||
		token.Surface() != "の" {
		return false
	}
	partOfSpeech := token.PartOfSpeech()
	return len(partOfSpeech) >= 2 &&
		partOfSpeech[0] == "助詞" &&
		partOfSpeech[1] == "連体化"
}

func isStandaloneNounPhrase(
	query string,
	tokens []kagome.TokenOccurrence,
	phrase queryTermDraft,
) bool {
	for index, token := range tokens {
		if index >= phrase.startToken && index <= phrase.endToken {
			continue
		}
		if !isIgnorableOutsideStandalonePhrase(token.Surface()) {
			return false
		}
	}
	return strings.TrimSpace(query[phrase.startByte:phrase.endByte]) != ""
}

func isIgnorableOutsideStandalonePhrase(value string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	for _, current := range value {
		if !unicode.IsPunct(current) && !unicode.IsSymbol(current) {
			return false
		}
	}
	return true
}

func hasForwardPhraseAnchor(
	query string,
	tokens []kagome.TokenOccurrence,
	phrase queryTermDraft,
	cues []legalquery.CueMention,
) bool {
	anchorStart := skipUnicodeSpaceForward(query, phrase.endByte)
	for _, relation := range []string{
		"について",
		"に関する",
		"に係る",
		"を含む",
		"を含み",
		"を含ん",
		"を除く",
		"を除き",
		"を除い",
	} {
		if strings.HasPrefix(query[anchorStart:], relation) {
			return true
		}
	}

	if strings.HasPrefix(query[anchorStart:], "の") {
		afterParticle := skipUnicodeSpaceForward(query, anchorStart+len("の"))
		for _, cue := range cues {
			if cue.Span().StartByte() == afterParticle {
				return true
			}
		}
	}
	for _, particle := range []string{"を", "は"} {
		if !strings.HasPrefix(query[anchorStart:], particle) {
			continue
		}
		afterParticle := skipUnicodeSpaceForward(
			query,
			anchorStart+len(particle),
		)
		return hasCueBeforeClauseBoundary(
			query,
			tokens,
			afterParticle,
			cues,
		)
	}
	return false
}

func hasCueBeforeClauseBoundary(
	query string,
	tokens []kagome.TokenOccurrence,
	startByte int,
	cues []legalquery.CueMention,
) bool {
	endByte := len(query)
	for offset, current := range query[startByte:] {
		switch current {
		case '、', '，', ',', '。', '！', '!', '？', '?', '；', ';':
			endByte = startByte + offset
			goto foundBoundary
		}
	}

foundBoundary:
	for _, token := range tokens {
		if token.EndByte() <= startByte {
			continue
		}
		if token.StartByte() >= endByte {
			break
		}
		if isPredicateBoundaryToken(token) {
			endByte = token.StartByte()
			break
		}
	}
	for _, cue := range cues {
		cueStart := cue.Span().StartByte()
		if cueStart >= startByte && cueStart < endByte {
			return true
		}
	}
	return false
}

func isPredicateBoundaryToken(token kagome.TokenOccurrence) bool {
	if token.UserDictionary() {
		return false
	}
	partOfSpeech := token.PartOfSpeech()
	if len(partOfSpeech) == 0 {
		return false
	}
	switch partOfSpeech[0] {
	case "動詞", "形容詞", "助動詞":
		return true
	default:
		return false
	}
}

func skipUnicodeSpaceForward(query string, startByte int) int {
	for startByte < len(query) {
		current, size := utf8.DecodeRuneInString(query[startByte:])
		if !unicode.IsSpace(current) {
			return startByte
		}
		startByte += size
	}
	return startByte
}

func isParallelPhraseSeparator(value string) bool {
	withoutSpace := strings.Map(func(current rune) rune {
		if unicode.IsSpace(current) {
			return -1
		}
		return current
	}, value)
	switch withoutSpace {
	case "と",
		"または",
		"又は",
		"若しくは",
		"および",
		"及び",
		"並びに",
		"、",
		"，",
		",",
		"、または",
		"、又は":
		return true
	default:
		return false
	}
}

func deduplicateQueryTermDrafts(values []queryTermDraft) []queryTermDraft {
	byIdentity := make(map[string]queryTermDraft, len(values))
	for _, value := range values {
		identity := fmt.Sprintf(
			"%d:%d:%s",
			value.startByte,
			value.endByte,
			value.kind,
		)
		byIdentity[identity] = value
	}
	deduplicated := make([]queryTermDraft, 0, len(byIdentity))
	for _, value := range byIdentity {
		deduplicated = append(deduplicated, value)
	}
	slices.SortFunc(
		deduplicated,
		func(left queryTermDraft, right queryTermDraft) int {
			return compareDraft(
				left.startByte,
				left.endByte,
				string(left.kind),
				right.startByte,
				right.endByte,
				string(right.kind),
			)
		},
	)
	return deduplicated
}

type mentionWithSpan interface {
	Span() legalquery.QuerySpan
}

func appendMentionSpans[T mentionWithSpan](
	destination []byteSpan,
	mentions []T,
) []byteSpan {
	for _, mention := range mentions {
		span := mention.Span()
		destination = append(destination, byteSpan{
			startByte: span.StartByte(),
			endByte:   span.EndByte(),
		})
	}
	return destination
}
