package querypreprocess

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/nlp/kagome"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

const (
	maxTypoTokenWindow = 8
	maxTypoRunes       = 24
	// 六 token 窓で受理した最大八十七 token を八 token 窓でも保持する。
	maxTypoCandidates = 668
)

type lawDraft struct {
	startByte int
	endByte   int
	lawID     string
	matchKind legalquery.PreprocessMatchKind
}

type conceptDraft struct {
	startByte int
	endByte   int
	conceptID string
	matchKind legalquery.PreprocessMatchKind
}

type cueDraft struct {
	startByte int
	endByte   int
	profileID string
	cueID     string
	matchKind legalquery.PreprocessMatchKind
}

type registeredOccurrence struct {
	startByte int
	endByte   int
	surface   string
}

// Preprocess は、照会ごとに新しい provider 非依存の事実を返す。
func (p *Preprocessor) Preprocess(
	ctx context.Context,
	request legalquery.Request,
) (legalquery.PreprocessResult, error) {
	if ctx == nil {
		return legalquery.PreprocessResult{}, fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return legalquery.PreprocessResult{}, err
	}
	if p == nil ||
		isNilAnalyzer(p.analyzer) ||
		p.normalizedTerms == nil ||
		p.identifiers == nil {
		return legalquery.PreprocessResult{}, fmt.Errorf("前処理器は初期化されていません")
	}
	if err := request.Validate(); err != nil {
		return legalquery.PreprocessResult{}, fmt.Errorf(
			"統合法情報照会 request が有効ではありません: %w",
			err,
		)
	}

	query := request.Query()
	positioned, comparisonKey, err := normalizedRunes(query)
	if err != nil {
		return legalquery.PreprocessResult{}, err
	}
	tokens, err := p.analyzer.AnalyzeTokenOccurrences(ctx, query)
	if err != nil {
		return legalquery.PreprocessResult{}, fmt.Errorf(
			"照会文を形態素解析できません: %w",
			err,
		)
	}
	if err := validateTokenOccurrences(query, tokens); err != nil {
		return legalquery.PreprocessResult{}, err
	}

	registered := registeredOccurrences(tokens)
	lawDrafts, conceptDrafts, cueDrafts := p.dictionaryDrafts(
		query,
		positioned,
		registered,
	)
	caseNumberMentions, caseNumberSpans, err :=
		extractJudicialCaseNumbers(query)
	if err != nil {
		return legalquery.PreprocessResult{}, err
	}
	typoProtectedSpans := dictionaryDraftSpans(lawDrafts, conceptDrafts)
	typoProtectedSpans = append(typoProtectedSpans, caseNumberSpans...)
	typoLawDrafts, typoConceptDrafts, err := p.typoDrafts(
		ctx,
		query,
		tokens,
		typoProtectedSpans,
	)
	if err != nil {
		return legalquery.PreprocessResult{}, err
	}
	lawDrafts = append(lawDrafts, typoLawDrafts...)
	conceptDrafts = append(conceptDrafts, typoConceptDrafts...)

	lawMentions, err := p.buildLawMentions(query, lawDrafts)
	if err != nil {
		return legalquery.PreprocessResult{}, err
	}
	conceptMentions, err := p.buildConceptMentions(
		query,
		conceptDrafts,
		lawMentions,
	)
	if err != nil {
		return legalquery.PreprocessResult{}, err
	}
	cueMentions, err := p.buildCueMentions(query, cueDrafts)
	if err != nil {
		return legalquery.PreprocessResult{}, err
	}
	identifierMentions, identifierSpans, err := p.extractIdentifiers(query)
	if err != nil {
		return legalquery.PreprocessResult{}, err
	}
	dateMentions, err := extractDates(query, identifierSpans)
	if err != nil {
		return legalquery.PreprocessResult{}, err
	}
	articleMentions, paragraphMentions, err := extractProvisions(query)
	if err != nil {
		return legalquery.PreprocessResult{}, err
	}
	protectedSpans := make([]byteSpan, 0)
	protectedSpans = appendMentionSpans(protectedSpans, lawMentions)
	protectedSpans = appendMentionSpans(protectedSpans, conceptMentions)
	protectedSpans = appendMentionSpans(protectedSpans, cueMentions)
	protectedSpans = appendMentionSpans(protectedSpans, identifierMentions)
	protectedSpans = appendMentionSpans(protectedSpans, dateMentions)
	protectedSpans = appendMentionSpans(protectedSpans, articleMentions)
	protectedSpans = appendMentionSpans(protectedSpans, paragraphMentions)
	protectedSpans = appendMentionSpans(protectedSpans, caseNumberMentions)
	queryTermMentions, err := extractQueryTermMentions(
		ctx,
		query,
		tokens,
		protectedSpans,
		cueMentions,
	)
	if err != nil {
		return legalquery.PreprocessResult{}, err
	}

	values := legalquery.PreprocessResultValues{
		Query:                query,
		ComparisonKey:        comparisonKey,
		LawNameMentions:      lawMentions,
		LegalConceptMentions: conceptMentions,
		CueMentions:          cueMentions,
		IdentifierMentions:   identifierMentions,
		DateMentions:         dateMentions,
		ArticleMentions:      articleMentions,
		ParagraphMentions:    paragraphMentions,
		CaseNumberMentions:   caseNumberMentions,
		QueryTermMentions:    queryTermMentions,
	}
	if ref, exists := request.Ref(); exists {
		values.Ref = &ref
	}
	result, err := legalquery.NewPreprocessResult(values)
	if err != nil {
		return legalquery.PreprocessResult{}, fmt.Errorf(
			"前処理結果を構築できません: %w",
			err,
		)
	}
	return result, nil
}

func (p *Preprocessor) dictionaryDrafts(
	query string,
	positioned []positionedRune,
	registered []registeredOccurrence,
) ([]lawDraft, []conceptDraft, []cueDraft) {
	laws := make([]lawDraft, 0)
	concepts := make([]conceptDraft, 0)
	cues := make([]cueDraft, 0)
	for _, hit := range p.normalizedTerms.find(positioned) {
		surface := query[hit.startByte:hit.endByte]
		matchKind, matched := dictionaryMatchKind(
			query,
			surface,
			hit.startByte,
			hit.endByte,
			hit.value.term,
			registered,
		)
		if !matched {
			continue
		}
		switch hit.value.kind {
		case dictionaryLawName:
			laws = append(laws, lawDraft{
				startByte: hit.startByte,
				endByte:   hit.endByte,
				lawID:     hit.value.id,
				matchKind: matchKind,
			})
		case dictionaryLegalConcept:
			concepts = append(concepts, conceptDraft{
				startByte: hit.startByte,
				endByte:   hit.endByte,
				conceptID: hit.value.id,
				matchKind: matchKind,
			})
		case dictionaryCue:
			cues = append(cues, cueDraft{
				startByte: hit.startByte,
				endByte:   hit.endByte,
				profileID: hit.value.id,
				cueID:     hit.value.secondary,
				matchKind: matchKind,
			})
		}
	}
	return laws, concepts, cues
}

func dictionaryMatchKind(
	query string,
	surface string,
	startByte int,
	endByte int,
	term string,
	registered []registeredOccurrence,
) (legalquery.PreprocessMatchKind, bool) {
	if startByte == 0 && endByte == len(query) && surface == term {
		return legalquery.PreprocessMatchExact, true
	}
	for _, occurrence := range registered {
		if occurrence.startByte == startByte &&
			occurrence.endByte == endByte &&
			occurrence.surface == surface &&
			surface == term {
			return legalquery.PreprocessMatchRegisteredTerm, true
		}
	}
	if surface == term {
		return "", false
	}
	return legalquery.PreprocessMatchComparisonNormalized, true
}

func registeredOccurrences(
	tokens []kagome.TokenOccurrence,
) []registeredOccurrence {
	values := make([]registeredOccurrence, 0)
	for _, token := range tokens {
		if !token.UserDictionary() {
			continue
		}
		values = append(values, registeredOccurrence{
			startByte: token.StartByte(),
			endByte:   token.EndByte(),
			surface:   token.Surface(),
		})
	}
	return values
}

func validateTokenOccurrences(
	query string,
	tokens []kagome.TokenOccurrence,
) error {
	if len(tokens) == 0 {
		return fmt.Errorf("形態素解析 token が原文全体を保持していません")
	}
	previousEnd := 0
	for index, token := range tokens {
		if token.StartByte() != previousEnd ||
			token.EndByte() <= token.StartByte() ||
			token.EndByte() > len(query) ||
			query[token.StartByte():token.EndByte()] != token.Surface() {
			return fmt.Errorf(
				"形態素解析 token[%d] の原文位置が有効ではありません",
				index,
			)
		}
		previousEnd = token.EndByte()
	}
	if previousEnd != len(query) {
		return fmt.Errorf("形態素解析 token が原文全体を保持していません")
	}
	return nil
}

func (p *Preprocessor) typoDrafts(
	ctx context.Context,
	query string,
	tokens []kagome.TokenOccurrence,
	protectedSpans []byteSpan,
) ([]lawDraft, []conceptDraft, error) {
	candidates, err := typoCandidates(query, tokens, protectedSpans)
	if err != nil {
		return nil, nil, err
	}
	laws := make([]lawDraft, 0)
	concepts := make([]conceptDraft, 0)
	for index, candidate := range candidates {
		if index%16 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, nil, err
			}
		}
		if p.lawResolver != nil {
			matches, err := p.lawResolver.ResolveUniqueTypoMatches(
				ctx,
				candidate.surface,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("法令名の誤記候補を判定できません: %w", err)
			}
			for _, match := range matches {
				laws = append(laws, lawDraft{
					startByte: candidate.startByte,
					endByte:   candidate.endByte,
					lawID:     match.ResourceID(),
					matchKind: legalquery.PreprocessMatchUniqueTypoCorrection,
				})
			}
		}
		if p.conceptResolver == nil {
			continue
		}
		matches, err := p.conceptResolver.ResolveUniqueTypoMatches(
			ctx,
			candidate.surface,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("法概念の誤記候補を判定できません: %w", err)
		}
		for _, match := range matches {
			concepts = append(concepts, conceptDraft{
				startByte: candidate.startByte,
				endByte:   candidate.endByte,
				conceptID: match.ResourceID(),
				matchKind: legalquery.PreprocessMatchUniqueTypoCorrection,
			})
		}
	}
	return laws, concepts, nil
}

type typoCandidate struct {
	startByte int
	endByte   int
	surface   string
}

func typoCandidates(
	query string,
	tokens []kagome.TokenOccurrence,
	protectedSpans []byteSpan,
) ([]typoCandidate, error) {
	values := make([]typoCandidate, 0)
	seen := make(map[string]struct{})
	for start := range tokens {
		if !isLexicalToken(tokens[start].Surface()) {
			continue
		}
		for end := start; end < len(tokens) && end < start+maxTypoTokenWindow; end++ {
			if !isLexicalToken(tokens[end].Surface()) {
				break
			}
			if !isTypoCandidateTerminal(tokens[end]) {
				continue
			}
			startByte := tokens[start].StartByte()
			endByte := tokens[end].EndByte()
			if overlapsAny(
				startByte,
				endByte,
				protectedSpans,
			) {
				continue
			}
			surface := query[startByte:endByte]
			key := querynormalization.ComparisonKey(surface)
			runeCount := len([]rune(key))
			if runeCount > maxTypoRunes {
				break
			}
			if runeCount < 3 {
				continue
			}
			identity := fmt.Sprintf("%d:%d", startByte, endByte)
			if _, exists := seen[identity]; exists {
				continue
			}
			seen[identity] = struct{}{}
			values = append(values, typoCandidate{
				startByte: startByte,
				endByte:   endByte,
				surface:   surface,
			})
			if len(values) > maxTypoCandidates {
				return nil, fmt.Errorf(
					"誤記判定候補は %d 件以下でなければなりません",
					maxTypoCandidates,
				)
			}
		}
	}
	return values, nil
}

func isTypoCandidateTerminal(token kagome.TokenOccurrence) bool {
	partOfSpeech := token.PartOfSpeech()
	if len(partOfSpeech) == 0 {
		return false
	}
	return partOfSpeech[0] != "助詞" && partOfSpeech[0] != "助動詞"
}

func dictionaryDraftSpans(
	laws []lawDraft,
	concepts []conceptDraft,
) []byteSpan {
	values := make([]byteSpan, 0, len(laws)+len(concepts))
	for _, draft := range laws {
		values = append(values, byteSpan{
			startByte: draft.startByte,
			endByte:   draft.endByte,
		})
	}
	for _, draft := range concepts {
		values = append(values, byteSpan{
			startByte: draft.startByte,
			endByte:   draft.endByte,
		})
	}
	return values
}

func isLexicalToken(value string) bool {
	if value == "" {
		return false
	}
	for _, current := range value {
		if !unicode.IsLetter(current) &&
			!unicode.IsNumber(current) &&
			!unicode.IsMark(current) {
			return false
		}
	}
	return true
}

func preprocessMatchPriority(kind legalquery.PreprocessMatchKind) int {
	switch kind {
	case legalquery.PreprocessMatchExact:
		return 0
	case legalquery.PreprocessMatchComparisonNormalized:
		return 1
	case legalquery.PreprocessMatchRegisteredTerm:
		return 2
	case legalquery.PreprocessMatchUniqueTypoCorrection:
		return 3
	default:
		return 4
	}
}

func compareDraft(
	leftStart int,
	leftEnd int,
	leftID string,
	rightStart int,
	rightEnd int,
	rightID string,
) int {
	switch {
	case leftStart < rightStart:
		return -1
	case leftStart > rightStart:
		return 1
	case leftEnd > rightEnd:
		return -1
	case leftEnd < rightEnd:
		return 1
	default:
		return strings.Compare(leftID, rightID)
	}
}

func deduplicateLawDrafts(values []lawDraft) []lawDraft {
	byIdentity := make(map[string]lawDraft)
	for _, value := range values {
		key := fmt.Sprintf("%d:%d:%s", value.startByte, value.endByte, value.lawID)
		existing, exists := byIdentity[key]
		if !exists ||
			preprocessMatchPriority(value.matchKind) <
				preprocessMatchPriority(existing.matchKind) {
			byIdentity[key] = value
		}
	}
	deduplicated := make([]lawDraft, 0, len(byIdentity))
	for _, value := range byIdentity {
		deduplicated = append(deduplicated, value)
	}
	slices.SortFunc(deduplicated, func(left lawDraft, right lawDraft) int {
		return compareDraft(
			left.startByte,
			left.endByte,
			left.lawID,
			right.startByte,
			right.endByte,
			right.lawID,
		)
	})
	return deduplicated
}

func deduplicateConceptDrafts(values []conceptDraft) []conceptDraft {
	byIdentity := make(map[string]conceptDraft)
	for _, value := range values {
		key := fmt.Sprintf("%d:%d:%s", value.startByte, value.endByte, value.conceptID)
		existing, exists := byIdentity[key]
		if !exists ||
			preprocessMatchPriority(value.matchKind) <
				preprocessMatchPriority(existing.matchKind) {
			byIdentity[key] = value
		}
	}
	deduplicated := make([]conceptDraft, 0, len(byIdentity))
	for _, value := range byIdentity {
		deduplicated = append(deduplicated, value)
	}
	slices.SortFunc(deduplicated, func(left conceptDraft, right conceptDraft) int {
		return compareDraft(
			left.startByte,
			left.endByte,
			left.conceptID,
			right.startByte,
			right.endByte,
			right.conceptID,
		)
	})
	return deduplicated
}

func deduplicateCueDrafts(values []cueDraft) []cueDraft {
	byIdentity := make(map[string]cueDraft)
	for _, value := range values {
		identity := cueKey(value.profileID, value.cueID)
		key := fmt.Sprintf("%d:%d:%s", value.startByte, value.endByte, identity)
		existing, exists := byIdentity[key]
		if !exists ||
			preprocessMatchPriority(value.matchKind) <
				preprocessMatchPriority(existing.matchKind) {
			byIdentity[key] = value
		}
	}
	deduplicated := make([]cueDraft, 0, len(byIdentity))
	for _, value := range byIdentity {
		deduplicated = append(deduplicated, value)
	}
	slices.SortFunc(deduplicated, func(left cueDraft, right cueDraft) int {
		return compareDraft(
			left.startByte,
			left.endByte,
			cueKey(left.profileID, left.cueID),
			right.startByte,
			right.endByte,
			cueKey(right.profileID, right.cueID),
		)
	})
	return deduplicated
}
