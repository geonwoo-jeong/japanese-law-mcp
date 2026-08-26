// Package judicialcitationnormalize は、判例引用追跡の正確一致専用正規化を提供する。
package judicialcitationnormalize

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/judicialcasenumber"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const japanesePositiveNumberExpression = `[0-9０-９〇零一二三四五六七八九十百千万]+`

var (
	referencedProvisionPattern = regexp.MustCompile(
		`^(.+?)(附則)?第?(` + japanesePositiveNumberExpression + `)条` +
			`((?:[の之]` + japanesePositiveNumberExpression + `){0,3})` +
			`(?:第?(` + japanesePositiveNumberExpression + `)項)?$`,
	)
	provisionLikeLocationPattern = regexp.MustCompile(
		`(?:附則)?第?` + japanesePositiveNumberExpression + `(?:条|項)`,
	)
	relativeLawReferencePattern = regexp.MustCompile(
		`^(?:附則)?第?` + japanesePositiveNumberExpression + `(?:条|項)`,
	)
)

type resolvedLawAlias struct {
	lawID string
	title string
}

// ExactLawAliasResolver は、法令名辞書の原表記一致だけを解決対象にする不変索引である。
// 比較用正規化と一意な誤記補正は、edge に昇格させず理由を区別するためだけに使う。
type ExactLawAliasResolver struct {
	resolver   *searchquery.Resolver
	exactIndex map[string][]resolvedLawAlias
	exactTerms []string
}

// NewExactLawAliasResolver は、検証済み法令名辞書から引用追跡用索引を構築する。
func NewExactLawAliasResolver(
	entries []lawnamelexicon.Entry,
) (ExactLawAliasResolver, error) {
	searchEntries := make([]searchquery.EntryValues, len(entries))
	exactIndex := make(map[string][]resolvedLawAlias)
	for index, entry := range entries {
		searchEntries[index] = searchquery.EntryValues{
			ResourceID: entry.ResourceID,
			Canonical:  entry.Canonical,
			Terms:      append([]string(nil), entry.Terms...),
		}
		candidate := resolvedLawAlias{lawID: entry.ResourceID, title: entry.Canonical}
		terms := append([]string{entry.Canonical}, entry.Terms...)
		for _, term := range terms {
			exactIndex[term] = appendUniqueAlias(exactIndex[term], candidate)
		}
	}
	resolver, err := searchquery.NewResolver(searchEntries, noRegisteredTermAnalyzer{})
	if err != nil {
		return ExactLawAliasResolver{}, fmt.Errorf("法令名辞書から引用追跡用索引を構築できません: %w", err)
	}
	exactTerms := make([]string, 0, len(exactIndex))
	for term := range exactIndex {
		exactTerms = append(exactTerms, term)
	}
	sort.Slice(exactTerms, func(left, right int) bool {
		if len(exactTerms[left]) != len(exactTerms[right]) {
			return len(exactTerms[left]) > len(exactTerms[right])
		}
		return exactTerms[left] < exactTerms[right]
	})
	return ExactLawAliasResolver{
		resolver:   resolver,
		exactIndex: exactIndex,
		exactTerms: exactTerms,
	}, nil
}

// ResolvedProvision は、法令名と条文位置を一意に正規化した参照法条である。
type ResolvedProvision struct {
	lawID    string
	lawTitle string
	location model.LawArticleLocation
}

func (r ResolvedProvision) LawID() string                      { return r.lawID }
func (r ResolvedProvision) LawTitle() string                   { return r.lawTitle }
func (r ResolvedProvision) Location() model.LawArticleLocation { return r.location }

// NormalizeReferencedProvision は、一件の参照法条を正確な法令名と条文位置へ正規化する。
// 解決できない情報源の値は error にせず、SOT-MODEL-035 の reason を返す。
func NormalizeReferencedProvision(
	ctx context.Context,
	value string,
	resolver ExactLawAliasResolver,
) (ResolvedProvision, model.JudicialCitationUnresolvedReason, error) {
	if ctx == nil {
		return ResolvedProvision{}, "", fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return ResolvedProvision{}, "", err
	}
	if resolver.resolver == nil {
		return ResolvedProvision{}, "", fmt.Errorf("法令名 resolver は初期化済みでなければなりません")
	}

	trimmed := strings.TrimSpace(value)
	if hasUnsupportedRelativeLawReference(trimmed) {
		return ResolvedProvision{}, "unsupported_reference_form", nil
	}
	matches := referencedProvisionPattern.FindStringSubmatch(trimmed)
	if len(matches) != 6 {
		reason, err := classifyMalformedProvision(ctx, trimmed, resolver)
		return ResolvedProvision{}, reason, err
	}

	law, reason, err := resolver.resolve(ctx, strings.TrimSpace(matches[1]))
	if err != nil || reason != "" {
		return ResolvedProvision{}, reason, err
	}
	location, locationErr := newLawArticleLocation(
		matches[2] != "",
		matches[3],
		matches[4],
		matches[5],
	)
	if locationErr == nil {
		return ResolvedProvision{
			lawID:    law.lawID,
			lawTitle: law.title,
			location: location,
		}, "", nil
	}
	return ResolvedProvision{}, "ambiguous_law_location", nil
}

// SplitReferencedProvisionText は、詳細ページの参照法条原文を決定的な順序で分割する。
func SplitReferencedProvisionText(value string) []string {
	fields := strings.FieldsFunc(value, func(current rune) bool {
		switch current {
		case '、', ',', '，', ';', '；', '\n':
			return true
		default:
			return false
		}
	})
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// LowerCourtDecisionReference は、完全な原審事件番号から作った内部参照である。
type LowerCourtDecisionReference struct {
	courtName        string
	caseNumberSearch string
}

func (r LowerCourtDecisionReference) CourtName() string        { return r.courtName }
func (r LowerCourtDecisionReference) CaseNumberSearch() string { return r.caseNumberSearch }

// NormalizeLowerCourtDecision は、原審裁判所名と完全な事件番号を共通化する。
func NormalizeLowerCourtDecision(
	courtName string,
	caseNumber string,
) (LowerCourtDecisionReference, bool, error) {
	trimmedCourt := strings.TrimSpace(courtName)
	trimmedCase := strings.TrimSpace(caseNumber)
	if trimmedCourt == "" || trimmedCase == "" {
		return LowerCourtDecisionReference{}, false, nil
	}
	parsed, err := judicialcasenumber.ParsePrefix(trimmedCase)
	if err != nil {
		return LowerCourtDecisionReference{}, false, err
	}
	if parsed.EndByte() != len(trimmedCase) {
		return LowerCourtDecisionReference{}, false, fmt.Errorf("事件番号が完全一致しません")
	}
	return LowerCourtDecisionReference{
		courtName:        trimmedCourt,
		caseNumberSearch: parsed.SearchText(),
	}, true, nil
}

func (r ExactLawAliasResolver) resolve(
	ctx context.Context,
	value string,
) (resolvedLawAlias, model.JudicialCitationUnresolvedReason, error) {
	matches, err := r.resolver.ResolveMatches(ctx, value)
	if err != nil {
		return resolvedLawAlias{}, "", err
	}
	if len(matches) == 0 {
		return resolvedLawAlias{}, "unregistered_law_name", nil
	}
	if matches[0].Kind() != searchquery.MatchKindExact {
		return resolvedLawAlias{}, "fuzzy_match_only", nil
	}
	if len(matches) != 1 {
		return resolvedLawAlias{}, "ambiguous_target", nil
	}
	return resolvedLawAlias{
		lawID: matches[0].ResourceID(),
		title: matches[0].Canonical(),
	}, "", nil
}

func classifyMalformedProvision(
	ctx context.Context,
	value string,
	resolver ExactLawAliasResolver,
) (model.JudicialCitationUnresolvedReason, error) {
	location := provisionLikeLocationPattern.FindStringIndex(value)
	if len(location) == 2 && location[0] > 0 {
		_, reason, err := resolver.resolve(ctx, strings.TrimSpace(value[:location[0]]))
		if err != nil {
			return "", err
		}
		if reason == "" {
			return "ambiguous_law_location", nil
		}
		return reason, nil
	}
	if targets, remainder, matched := resolver.longestExactPrefix(value); matched {
		if len(targets) != 1 {
			return "ambiguous_target", nil
		}
		trimmedRemainder := strings.TrimSpace(remainder)
		if trimmedRemainder == "" {
			return "unsupported_reference_form", nil
		}
		if looksLikeLocationRemainder(trimmedRemainder) {
			return "ambiguous_law_location", nil
		}
	}
	return "unregistered_law_name", nil
}

func (r ExactLawAliasResolver) longestExactPrefix(
	value string,
) ([]resolvedLawAlias, string, bool) {
	for _, term := range r.exactTerms {
		if strings.HasPrefix(value, term) {
			return r.exactIndex[term], value[len(term):], true
		}
	}
	return nil, "", false
}

func hasUnsupportedRelativeLawReference(value string) bool {
	return strings.HasPrefix(value, "同法") ||
		strings.HasPrefix(value, "本法") ||
		relativeLawReferencePattern.MatchString(value)
}

func looksLikeLocationRemainder(value string) bool {
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "附則") || strings.HasPrefix(value, "第") {
		return true
	}
	first, _ := utf8.DecodeRuneInString(value)
	return strings.ContainsRune("0123456789０１２３４５６７８９〇零一二三四五六七八九十百千万", first)
}

func appendUniqueAlias(
	values []resolvedLawAlias,
	candidate resolvedLawAlias,
) []resolvedLawAlias {
	for _, current := range values {
		if current.lawID == candidate.lawID && current.title == candidate.title {
			return values
		}
	}
	return append(values, candidate)
}

func newLawArticleLocation(
	supplementary bool,
	articleValue string,
	subArticleValue string,
	paragraphValue string,
) (model.LawArticleLocation, error) {
	provision := model.LawArticleProvisionMain
	if supplementary {
		provision = model.LawArticleProvisionSupplementary
	}
	articleValues := []string{articleValue}
	if subArticleValue != "" {
		articleValues = append(
			articleValues,
			strings.FieldsFunc(subArticleValue, func(current rune) bool {
				return current == 'の' || current == '之'
			})...,
		)
	}
	articleNumbers := make([]string, 0, len(articleValues))
	for _, current := range articleValues {
		number, err := parsePositiveJapaneseNumber(current)
		if err != nil {
			return model.LawArticleLocation{}, err
		}
		articleNumbers = append(articleNumbers, strconv.Itoa(number))
	}
	var paragraphNumber *int
	if paragraphValue != "" {
		number, err := parsePositiveJapaneseNumber(paragraphValue)
		if err != nil {
			return model.LawArticleLocation{}, err
		}
		paragraphNumber = &number
	}
	return model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:       provision,
		ArticleNumber:   strings.Join(articleNumbers, "_"),
		ParagraphNumber: paragraphNumber,
	})
}

func parsePositiveJapaneseNumber(value string) (int, error) {
	normalizedDecimal := normalizeDecimalDigits(value)
	if isASCIIDecimal(normalizedDecimal) {
		number, err := strconv.Atoi(normalizedDecimal)
		if err != nil || number < 1 {
			return 0, fmt.Errorf("正の整数ではありません")
		}
		return number, nil
	}

	total := 0
	section := 0
	current := 0
	for _, character := range value {
		if digit, exists := kanjiDigit(character); exists {
			next, ok := checkedMultiplyAdd(current, 10, digit)
			if !ok {
				return 0, fmt.Errorf("条文番号が大きすぎます")
			}
			current = next
			continue
		}
		unit, exists := kanjiUnit(character)
		if !exists {
			return 0, fmt.Errorf("日本語の正数ではありません")
		}
		if unit >= 10_000 {
			base, ok := checkedAdd(section, current)
			if !ok {
				return 0, fmt.Errorf("条文番号が大きすぎます")
			}
			if base == 0 {
				base = 1
			}
			product, ok := checkedMultiply(base, unit)
			if !ok {
				return 0, fmt.Errorf("条文番号が大きすぎます")
			}
			total, ok = checkedAdd(total, product)
			if !ok {
				return 0, fmt.Errorf("条文番号が大きすぎます")
			}
			section = 0
			current = 0
			continue
		}
		coefficient := current
		if coefficient == 0 {
			coefficient = 1
		}
		product, ok := checkedMultiply(coefficient, unit)
		if !ok {
			return 0, fmt.Errorf("条文番号が大きすぎます")
		}
		section, ok = checkedAdd(section, product)
		if !ok {
			return 0, fmt.Errorf("条文番号が大きすぎます")
		}
		current = 0
	}
	number, ok := checkedAdd(total, section)
	if !ok {
		return 0, fmt.Errorf("条文番号が大きすぎます")
	}
	number, ok = checkedAdd(number, current)
	if !ok || number < 1 {
		return 0, fmt.Errorf("正の整数ではありません")
	}
	return number, nil
}

func normalizeDecimalDigits(value string) string {
	var builder strings.Builder
	for _, character := range value {
		switch {
		case character >= '0' && character <= '9':
			builder.WriteRune(character)
		case character >= '０' && character <= '９':
			builder.WriteRune(character - '０' + '0')
		default:
			builder.WriteRune(character)
		}
	}
	return builder.String()
}

func isASCIIDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func kanjiDigit(value rune) (int, bool) {
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

func kanjiUnit(value rune) (int, bool) {
	switch value {
	case '十':
		return 10, true
	case '百':
		return 100, true
	case '千':
		return 1000, true
	case '万':
		return 10_000, true
	default:
		return 0, false
	}
}

func checkedMultiplyAdd(left int, multiplier int, addition int) (int, bool) {
	product, ok := checkedMultiply(left, multiplier)
	if !ok {
		return 0, false
	}
	return checkedAdd(product, addition)
}

func checkedMultiply(left int, right int) (int, bool) {
	maximum := int(^uint(0) >> 1)
	if left != 0 && right > maximum/left {
		return 0, false
	}
	return left * right, true
}

func checkedAdd(left int, right int) (int, bool) {
	maximum := int(^uint(0) >> 1)
	if right > maximum-left {
		return 0, false
	}
	return left + right, true
}

type noRegisteredTermAnalyzer struct{}

func (noRegisteredTermAnalyzer) RegisteredTerms(
	ctx context.Context,
	_ string,
) ([]string, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []string{}, nil
}

var _ searchquery.Analyzer = noRegisteredTermAnalyzer{}
