package querypreprocess

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

var (
	japaneseDatePattern = regexp.MustCompile(
		`[0-9０-９]{4}年[0-9０-９]{1,2}月[0-9０-９]{1,2}日`,
	)
	isoDatePattern = regexp.MustCompile(
		`[0-9０-９]{4}-[0-9０-９]{1,2}-[0-9０-９]{1,2}`,
	)
	articlePattern = regexp.MustCompile(
		`(附則)?第[0-9０-９〇零一二三四五六七八九十百千万]+条(の[0-9０-９〇零一二三四五六七八九十百千万]+){0,3}`,
	)
	paragraphPattern = regexp.MustCompile(
		`第[0-9０-９〇零一二三四五六七八九十百千万]+項`,
	)
)

type byteSpan struct {
	startByte int
	endByte   int
}

type identifierDraft struct {
	startByte int
	endByte   int
	target    identifierTarget
}

func (p *Preprocessor) extractIdentifiers(
	query string,
) ([]legalquery.IdentifierMention, []byteSpan, error) {
	hits := p.identifiers.find(rawRunes(query))
	drafts := make([]identifierDraft, 0, len(hits))
	seen := make(map[string]struct{})
	for _, hit := range hits {
		identity := strings.Join([]string{
			strconv.Itoa(hit.startByte),
			strconv.Itoa(hit.endByte),
			string(hit.value.kind),
			hit.value.lawID,
			hit.value.revisionID,
			hit.value.lawNumber,
		}, "\x00")
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		drafts = append(drafts, identifierDraft{
			startByte: hit.startByte,
			endByte:   hit.endByte,
			target:    hit.value,
		})
	}
	drafts = suppressIdentifiersInsideRevisions(drafts)
	slices.SortFunc(drafts, func(left identifierDraft, right identifierDraft) int {
		return compareDraft(
			left.startByte,
			left.endByte,
			identifierIdentity(left.target),
			right.startByte,
			right.endByte,
			identifierIdentity(right.target),
		)
	})

	mentions := make([]legalquery.IdentifierMention, 0, len(drafts))
	spans := make([]byteSpan, 0, len(drafts))
	for _, draft := range drafts {
		span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
			StartByte: draft.startByte,
			EndByte:   draft.endByte,
		})
		if err != nil {
			return nil, nil, err
		}
		values := legalquery.IdentifierMentionValues{
			Span:    span,
			Surface: query[draft.startByte:draft.endByte],
			Kind:    draft.target.kind,
			LawID:   draft.target.lawID,
		}
		if draft.target.revisionID != "" {
			revisionID := draft.target.revisionID
			values.RevisionID = &revisionID
		}
		if draft.target.lawNumber != "" {
			lawNumber := draft.target.lawNumber
			values.LawNumber = &lawNumber
		}
		mention, err := legalquery.NewIdentifierMention(values)
		if err != nil {
			return nil, nil, err
		}
		mentions = append(mentions, mention)
		spans = append(spans, byteSpan{
			startByte: draft.startByte,
			endByte:   draft.endByte,
		})
	}
	return mentions, spans, nil
}

func suppressIdentifiersInsideRevisions(
	values []identifierDraft,
) []identifierDraft {
	revisions := make([]identifierDraft, 0)
	for _, value := range values {
		if value.target.kind == legalquery.IdentifierMentionLawRevisionID {
			revisions = append(revisions, value)
		}
	}
	filtered := make([]identifierDraft, 0, len(values))
	for _, value := range values {
		if value.target.kind == legalquery.IdentifierMentionLawRevisionID {
			filtered = append(filtered, value)
			continue
		}
		contained := false
		for _, revision := range revisions {
			if revision.startByte <= value.startByte &&
				value.endByte <= revision.endByte {
				contained = true
				break
			}
		}
		if !contained {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func identifierIdentity(value identifierTarget) string {
	return strings.Join([]string{
		string(value.kind),
		value.lawID,
		value.revisionID,
		value.lawNumber,
	}, "\x00")
}

func extractDates(
	query string,
	excluded []byteSpan,
) ([]legalquery.DateMention, error) {
	indices := append(
		japaneseDatePattern.FindAllStringIndex(query, -1),
		isoDatePattern.FindAllStringIndex(query, -1)...,
	)
	slices.SortFunc(indices, func(left []int, right []int) int {
		switch {
		case left[0] < right[0]:
			return -1
		case left[0] > right[0]:
			return 1
		case left[1] > right[1]:
			return -1
		case left[1] < right[1]:
			return 1
		default:
			return 0
		}
	})
	mentions := make([]legalquery.DateMention, 0, len(indices))
	seen := make(map[string]struct{})
	for _, index := range indices {
		if overlapsAny(index[0], index[1], excluded) ||
			hasAdjacentDigit(query, index[0], index[1]) {
			continue
		}
		surface := query[index[0]:index[1]]
		date, err := parseDate(surface)
		if err != nil {
			continue
		}
		identity := fmt.Sprintf("%d:%d:%s", index[0], index[1], date.String())
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
			StartByte: index[0],
			EndByte:   index[1],
		})
		if err != nil {
			return nil, err
		}
		mention, err := legalquery.NewDateMention(legalquery.DateMentionValues{
			Span:    span,
			Surface: surface,
			Date:    date,
		})
		if err != nil {
			return nil, err
		}
		mentions = append(mentions, mention)
	}
	return mentions, nil
}

func parseDate(surface string) (model.Date, error) {
	ascii := normalizeDecimalDigits(surface)
	var year int
	var month int
	var day int
	if strings.Contains(ascii, "年") {
		if _, err := fmt.Sscanf(
			ascii,
			"%d年%d月%d日",
			&year,
			&month,
			&day,
		); err != nil {
			return model.Date{}, err
		}
	} else if _, err := fmt.Sscanf(
		ascii,
		"%d-%d-%d",
		&year,
		&month,
		&day,
	); err != nil {
		return model.Date{}, err
	}
	return model.NewDate(fmt.Sprintf("%04d-%02d-%02d", year, month, day))
}

func hasAdjacentDigit(query string, startByte int, endByte int) bool {
	if startByte > 0 {
		previous, _ := previousRune(query, startByte)
		if unicode.IsDigit(previous) {
			return true
		}
	}
	if endByte < len(query) {
		next, _ := nextRune(query, endByte)
		if unicode.IsDigit(next) {
			return true
		}
	}
	return false
}

func previousRune(value string, endByte int) (rune, int) {
	for start := endByte - 1; start >= 0; start-- {
		if value[start]&0xc0 != 0x80 {
			current := []rune(value[start:endByte])
			if len(current) == 1 {
				return current[0], start
			}
		}
	}
	return 0, -1
}

func nextRune(value string, startByte int) (rune, int) {
	for end := startByte + 1; end <= len(value); end++ {
		current := []rune(value[startByte:end])
		if len(current) == 1 {
			return current[0], end
		}
	}
	return 0, -1
}

func overlapsAny(startByte int, endByte int, values []byteSpan) bool {
	for _, value := range values {
		if startByte < value.endByte && value.startByte < endByte {
			return true
		}
	}
	return false
}

func extractProvisions(
	query string,
) ([]legalquery.ArticleMention, []legalquery.ParagraphMention, error) {
	articles := make([]legalquery.ArticleMention, 0)
	for _, index := range articlePattern.FindAllStringIndex(query, -1) {
		surface := query[index[0]:index[1]]
		provision := model.LawArticleProvisionMain
		numberExpression := surface
		if strings.HasPrefix(surface, "附則") {
			provision = model.LawArticleProvisionSupplementary
			numberExpression = strings.TrimPrefix(surface, "附則")
		}
		numberExpression = strings.TrimPrefix(numberExpression, "第")
		numberExpression = strings.TrimSuffix(numberExpression, "条")
		numberExpression = strings.ReplaceAll(numberExpression, "条の", "_")
		numberExpression = strings.ReplaceAll(numberExpression, "の", "_")
		segments := strings.Split(numberExpression, "_")
		normalized := make([]string, 0, len(segments))
		valid := true
		for _, segment := range segments {
			number, err := parsePositiveNumber(segment)
			if err != nil {
				valid = false
				break
			}
			normalized = append(normalized, strconv.Itoa(number))
		}
		if !valid {
			continue
		}
		span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
			StartByte: index[0],
			EndByte:   index[1],
		})
		if err != nil {
			return nil, nil, err
		}
		mention, err := legalquery.NewArticleMention(legalquery.ArticleMentionValues{
			Span:          span,
			Surface:       surface,
			Provision:     provision,
			ArticleNumber: strings.Join(normalized, "_"),
		})
		if err != nil {
			return nil, nil, err
		}
		articles = append(articles, mention)
	}

	paragraphs := make([]legalquery.ParagraphMention, 0)
	for _, index := range paragraphPattern.FindAllStringIndex(query, -1) {
		surface := query[index[0]:index[1]]
		numberExpression := strings.TrimSuffix(
			strings.TrimPrefix(surface, "第"),
			"項",
		)
		number, err := parsePositiveNumber(numberExpression)
		if err != nil {
			continue
		}
		span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
			StartByte: index[0],
			EndByte:   index[1],
		})
		if err != nil {
			return nil, nil, err
		}
		mention, err := legalquery.NewParagraphMention(
			legalquery.ParagraphMentionValues{
				Span:            span,
				Surface:         surface,
				ParagraphNumber: number,
			},
		)
		if err != nil {
			return nil, nil, err
		}
		paragraphs = append(paragraphs, mention)
	}
	return articles, paragraphs, nil
}

func parsePositiveNumber(value string) (int, error) {
	ascii := normalizeDecimalDigits(value)
	if isASCIIDecimal(ascii) {
		number, err := strconv.Atoi(ascii)
		if err != nil || number < 1 {
			return 0, fmt.Errorf("正の整数ではありません")
		}
		return number, nil
	}

	var total int64
	var section int64
	var current int64
	for _, character := range value {
		if digit, exists := kanjiDigit(character); exists {
			current = current*10 + digit
			continue
		}
		unit, exists := kanjiUnit(character)
		if !exists {
			return 0, fmt.Errorf("日本語の正数ではありません")
		}
		if unit == 10000 {
			section += current
			if section == 0 {
				section = 1
			}
			total += section * unit
			section = 0
			current = 0
			continue
		}
		if current == 0 {
			current = 1
		}
		section += current * unit
		current = 0
	}
	total += section + current
	if total < 1 || total > int64(math.MaxInt) {
		return 0, fmt.Errorf("正の整数の範囲外です")
	}
	return int(total), nil
}

func normalizeDecimalDigits(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, current := range value {
		if current >= '０' && current <= '９' {
			current = '0' + (current - '０')
		}
		builder.WriteRune(current)
	}
	return builder.String()
}

func isASCIIDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, current := range value {
		if current < '0' || current > '9' {
			return false
		}
	}
	return true
}

func kanjiDigit(value rune) (int64, bool) {
	switch value {
	case '〇', '零':
		return 0, true
	case '一':
		return 1, true
	case '二':
		return 2, true
	case '三':
		return 3, true
	case '四':
		return 4, true
	case '五':
		return 5, true
	case '六':
		return 6, true
	case '七':
		return 7, true
	case '八':
		return 8, true
	case '九':
		return 9, true
	default:
		return 0, false
	}
}

func kanjiUnit(value rune) (int64, bool) {
	switch value {
	case '十':
		return 10, true
	case '百':
		return 100, true
	case '千':
		return 1000, true
	case '万':
		return 10000, true
	default:
		return 0, false
	}
}
