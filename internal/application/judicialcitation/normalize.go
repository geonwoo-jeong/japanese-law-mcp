package judicialcitation

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawtarget"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

var referencedProvisionPattern = regexp.MustCompile(
	`^(?P<law>.*?)(?P<article>(附則)?第[0-9０-９〇零一二三四五六七八九十百千万]+条(の[0-9０-９〇零一二三四五六七八九十百千万]+){0,3})(?P<paragraph>第[0-9０-９〇零一二三四五六七八九十百千万]+項)?$`,
)

type LowerCourtDecisionReference struct {
	courtName        string
	caseNumber       string
	caseNumberSearch string
	decisionDate     *model.Date
}

func (r LowerCourtDecisionReference) CourtName() string        { return r.courtName }
func (r LowerCourtDecisionReference) CaseNumber() string       { return r.caseNumber }
func (r LowerCourtDecisionReference) CaseNumberSearch() string { return r.caseNumberSearch }
func (r LowerCourtDecisionReference) DecisionDate() (model.Date, bool) {
	if r.decisionDate == nil {
		return model.Date{}, false
	}
	return *r.decisionDate, true
}

func NormalizeLowerCourtDecision(
	details model.JudicialDecisionDetails,
) (LowerCourtDecisionReference, bool) {
	courtName, hasCourt := details.LowerCourtName()
	caseNumber, hasCaseNumber := details.LowerCourtCaseNumber()
	if !hasCourt || !hasCaseNumber {
		return LowerCourtDecisionReference{}, false
	}
	parsed, err := legalquery.ParseJudicialCaseNumberPrefix(caseNumber)
	if err != nil || parsed.EndByte() != len(caseNumber) {
		return LowerCourtDecisionReference{}, false
	}
	reference := LowerCourtDecisionReference{
		courtName:        courtName,
		caseNumber:       caseNumber,
		caseNumberSearch: parsed.SearchText(),
	}
	if value, exists := details.LowerCourtDecisionDate(); exists {
		date := value
		reference.decisionDate = &date
	}
	return reference, true
}

type ProvisionNormalizationResult struct {
	references []model.JudicialCitationLawReference
	unresolved []model.JudicialCitationUnresolvedMention
}

func (r ProvisionNormalizationResult) References() []model.JudicialCitationLawReference {
	return append([]model.JudicialCitationLawReference(nil), r.references...)
}

func (r ProvisionNormalizationResult) Unresolved() []model.JudicialCitationUnresolvedMention {
	return append([]model.JudicialCitationUnresolvedMention(nil), r.unresolved...)
}

func NormalizeReferencedProvisions(
	ctx context.Context,
	resolver lawtarget.LogicalInputResolver,
	details model.JudicialDecisionDetails,
	provenance model.Provenance,
) (ProvisionNormalizationResult, error) {
	if ctx == nil {
		return ProvisionNormalizationResult{}, fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return ProvisionNormalizationResult{}, err
	}
	if resolver == nil {
		return ProvisionNormalizationResult{}, fmt.Errorf("law resolver は必須です")
	}
	if err := provenance.Validate(); err != nil {
		return ProvisionNormalizationResult{}, fmt.Errorf("provenance が有効ではありません: %w", err)
	}
	text, exists := details.ReferencedProvisionsText()
	if !exists {
		return ProvisionNormalizationResult{
			references: []model.JudicialCitationLawReference{},
			unresolved: []model.JudicialCitationUnresolvedMention{},
		}, nil
	}
	segments := splitReferencedProvisionText(text)
	references := make([]model.JudicialCitationLawReference, 0, len(segments))
	unresolved := make([]model.JudicialCitationUnresolvedMention, 0)
	var current lawtarget.ResolvedLawTarget
	var hasCurrent bool
	for _, segment := range segments {
		if err := ctx.Err(); err != nil {
			return ProvisionNormalizationResult{}, err
		}
		match := referencedProvisionPattern.FindStringSubmatch(segment)
		if len(match) == 0 {
			target, reason, ok, err := resolveLawTarget(ctx, resolver, segment)
			if err != nil {
				return ProvisionNormalizationResult{}, err
			}
			if ok {
				current = target
				hasCurrent = true
				continue
			}
			mention, err := newLawProvisionUnresolvedMention(
				segment,
				reason,
				provenance,
			)
			if err != nil {
				return ProvisionNormalizationResult{}, err
			}
			unresolved = append(unresolved, mention)
			continue
		}
		lawText := normalizeCitationText(match[1])
		target := current
		ok := hasCurrent
		if lawText != "" {
			resolved, reason, found, err := resolveLawTarget(ctx, resolver, lawText)
			if err != nil {
				return ProvisionNormalizationResult{}, err
			}
			if !found {
				mention, mentionErr := newLawProvisionUnresolvedMention(
					segment,
					reason,
					provenance,
				)
				if mentionErr != nil {
					return ProvisionNormalizationResult{}, mentionErr
				}
				unresolved = append(unresolved, mention)
				continue
			}
			target = resolved
			ok = true
			current = resolved
			hasCurrent = true
		}
		if !ok {
			mention, err := newLawProvisionUnresolvedMention(
				segment,
				model.JudicialCitationUnresolvedReasonAmbiguousLawLocation,
				provenance,
			)
			if err != nil {
				return ProvisionNormalizationResult{}, err
			}
			unresolved = append(unresolved, mention)
			continue
		}
		location, err := parseReferencedProvisionLocation(match[2], match[5])
		if err != nil {
			mention, mentionErr := newLawProvisionUnresolvedMention(
				segment,
				model.JudicialCitationUnresolvedReasonAmbiguousLawLocation,
				provenance,
			)
			if mentionErr != nil {
				return ProvisionNormalizationResult{}, mentionErr
			}
			unresolved = append(unresolved, mention)
			continue
		}
		reference, err := model.NewJudicialCitationLawReference(
			model.JudicialCitationLawReferenceValues{
				LawID:    target.LawID(),
				LawTitle: target.OfficialTitle(),
				Location: location,
			},
		)
		if err != nil {
			return ProvisionNormalizationResult{}, err
		}
		references = append(references, reference)
	}
	return ProvisionNormalizationResult{
		references: references,
		unresolved: unresolved,
	}, nil
}

func splitReferencedProvisionText(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == '、' || r == ',' || r == '，' || r == ';' || r == '；' || r == '\n'
	})
	segments := make([]string, 0, len(fields))
	for _, field := range fields {
		normalized := normalizeCitationText(field)
		if normalized != "" {
			segments = append(segments, normalized)
		}
	}
	return segments
}

func resolveLawTarget(
	ctx context.Context,
	resolver lawtarget.LogicalInputResolver,
	value string,
) (
	lawtarget.ResolvedLawTarget,
	model.JudicialCitationUnresolvedReason,
	bool,
	error,
) {
	target, found, err := resolver.ResolveLogicalInput(ctx, value)
	if err != nil || !found {
		return lawtarget.ResolvedLawTarget{},
			model.JudicialCitationUnresolvedReasonUnregisteredLawName,
			false,
			err
	}
	switch target.MatchKind() {
	case lawtarget.MatchKindExact, lawtarget.MatchKindRegisteredTerm:
		return target, "", true, nil
	case lawtarget.MatchKindComparisonNormalized, lawtarget.MatchKindUniqueTypoCorrection:
		return lawtarget.ResolvedLawTarget{},
			model.JudicialCitationUnresolvedReasonFuzzyMatchOnly,
			false,
			nil
	default:
		return lawtarget.ResolvedLawTarget{},
			model.JudicialCitationUnresolvedReasonUnregisteredLawName,
			false,
			nil
	}
}

func parseReferencedProvisionLocation(
	articleSurface string,
	paragraphSurface string,
) (model.LawArticleLocation, error) {
	provision := model.LawArticleProvisionMain
	numberExpression := articleSurface
	if strings.HasPrefix(numberExpression, "附則") {
		provision = model.LawArticleProvisionSupplementary
		numberExpression = strings.TrimPrefix(numberExpression, "附則")
	}
	numberExpression = strings.TrimPrefix(numberExpression, "第")
	numberExpression = strings.TrimSuffix(numberExpression, "条")
	numberExpression = strings.ReplaceAll(numberExpression, "条の", "_")
	numberExpression = strings.ReplaceAll(numberExpression, "の", "_")
	segments := strings.Split(numberExpression, "_")
	normalized := make([]string, 0, len(segments))
	for _, segment := range segments {
		number, err := parsePositiveJapaneseNumber(segment)
		if err != nil {
			return model.LawArticleLocation{}, err
		}
		normalized = append(normalized, strconv.Itoa(number))
	}
	var paragraphNumber *int
	if paragraphSurface != "" {
		value := strings.TrimSuffix(strings.TrimPrefix(paragraphSurface, "第"), "項")
		number, err := parsePositiveJapaneseNumber(value)
		if err != nil {
			return model.LawArticleLocation{}, err
		}
		paragraphNumber = &number
	}
	return model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:       provision,
		ArticleNumber:   strings.Join(normalized, "_"),
		ParagraphNumber: paragraphNumber,
	})
}

func newLawProvisionUnresolvedMention(
	text string,
	reason model.JudicialCitationUnresolvedReason,
	provenance model.Provenance,
) (model.JudicialCitationUnresolvedMention, error) {
	return model.NewJudicialCitationUnresolvedMention(
		model.JudicialCitationUnresolvedMentionValues{
			MentionType: model.JudicialCitationMentionTypeLawProvision,
			MentionText: text,
			Reason:      reason,
			Provenance:  provenance,
		},
	)
}

func normalizeCitationText(value string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, value))
}

func parsePositiveJapaneseNumber(value string) (int, error) {
	ascii := normalizeDecimalDigits(value)
	if isASCIIDecimal(ascii) {
		number, err := strconv.Atoi(ascii)
		if err != nil || number < 1 {
			return 0, fmt.Errorf("正の整数ではありません")
		}
		return number, nil
	}
	var total int
	var section int
	var current int
	for _, character := range value {
		if digit, exists := kanjiDigit(character); exists {
			current = current*10 + digit
			continue
		}
		unit, exists := kanjiUnit(character)
		if !exists {
			return 0, fmt.Errorf("日本語の正数ではありません")
		}
		if current == 0 {
			current = 1
		}
		if unit >= 10000 {
			total += (section + current) * unit
			section = 0
			current = 0
			continue
		}
		section += current * unit
		current = 0
	}
	number := total + section + current
	if number < 1 {
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
		return 10000, true
	default:
		return 0, false
	}
}
