package legalquery

import (
	"fmt"
	"unicode"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// QueryLanguage は、候補生成へ渡す照会文の script 分類である。
type QueryLanguage string

const (
	// QueryLanguageJapanese は、日本語 script を一文字以上含む照会を表す。
	QueryLanguageJapanese QueryLanguage = "ja"
	// QueryLanguageNonJapanese は、日本語 script を含まない照会を表す。
	QueryLanguageNonJapanese QueryLanguage = "non_ja"
)

// CandidateGenerationInput は、profile が利用できる位置付き前処理事実である。
// 原文と比較キーを公開せず、profile による再解析を構造的に防ぐ。
type CandidateGenerationInput struct {
	language                QueryLanguage
	standaloneStructured    bool
	ref                     *model.SourceResourceRef
	lawNameMentions         []LawNameMention
	legalConceptMentions    []LegalConceptMention
	cueMentions             []CueMention
	identifierMentions      []IdentifierMention
	dateMentions            []DateMention
	articleMentions         []ArticleMention
	paragraphMentions       []ParagraphMention
	caseNumberMentions      []JudicialCaseNumberMention
	queryTermMentions       []QueryTermMention
	cueTaskRelations        []CueTaskRelation
	sharedTerminalSequences []SharedTerminalSequence
}

// NewCandidateGenerationInput は、検証済み前処理結果から profile 入力を作る。
func NewCandidateGenerationInput(
	result PreprocessResult,
) (CandidateGenerationInput, error) {
	if err := result.Validate(); err != nil {
		return CandidateGenerationInput{}, fmt.Errorf(
			"前処理結果が有効ではありません: %w",
			err,
		)
	}
	sharedTerminalSequences, err := buildSharedTerminalSequences(result)
	if err != nil {
		return CandidateGenerationInput{}, err
	}
	input := CandidateGenerationInput{
		language:             classifyQueryLanguage(result),
		standaloneStructured: classifyStandaloneStructuredQuery(result),
		lawNameMentions:      result.LawNameMentions(),
		legalConceptMentions: result.LegalConceptMentions(),
		cueMentions:          result.CueMentions(),
		identifierMentions:   result.IdentifierMentions(),
		dateMentions:         result.DateMentions(),
		articleMentions:      result.ArticleMentions(),
		paragraphMentions:    result.ParagraphMentions(),
		caseNumberMentions:   result.CaseNumberMentions(),
		queryTermMentions:    result.QueryTermMentions(),
		cueTaskRelations:     result.CueTaskRelations(),
		sharedTerminalSequences: cloneSharedTerminalSequences(
			sharedTerminalSequences,
		),
	}
	if ref, exists := result.Ref(); exists {
		input.ref = &ref
	}
	if err := input.Validate(); err != nil {
		return CandidateGenerationInput{}, fmt.Errorf(
			"profile 入力を構築できません: %w",
			err,
		)
	}
	return input, nil
}

// Language は、原文を公開せず script 分類だけを返す。
func (i CandidateGenerationInput) Language() QueryLanguage {
	return i.language
}

// StandaloneStructuredQuery は、照会全体が決定的な構造だけかを返す。
func (i CandidateGenerationInput) StandaloneStructuredQuery() bool {
	return i.standaloneStructured
}

// Ref は、入力で受け取った参照の複製と有無を返す。
func (i CandidateGenerationInput) Ref() (model.SourceResourceRef, bool) {
	if i.ref == nil {
		return model.SourceResourceRef{}, false
	}
	return *i.ref, true
}

// LawNameMentions は、法令名出現の複製を返す。
func (i CandidateGenerationInput) LawNameMentions() []LawNameMention {
	return append([]LawNameMention(nil), i.lawNameMentions...)
}

// LegalConceptMentions は、法概念出現の複製を返す。
func (i CandidateGenerationInput) LegalConceptMentions() []LegalConceptMention {
	return append([]LegalConceptMention(nil), i.legalConceptMentions...)
}

// CueMentions は、構文 cue 出現の複製を返す。
func (i CandidateGenerationInput) CueMentions() []CueMention {
	return append([]CueMention(nil), i.cueMentions...)
}

// IdentifierMentions は、公式識別子出現の複製を返す。
func (i CandidateGenerationInput) IdentifierMentions() []IdentifierMention {
	return append([]IdentifierMention(nil), i.identifierMentions...)
}

// DateMentions は、完全な暦日出現の複製を返す。
func (i CandidateGenerationInput) DateMentions() []DateMention {
	return append([]DateMention(nil), i.dateMentions...)
}

// ArticleMentions は、条番号出現の複製を返す。
func (i CandidateGenerationInput) ArticleMentions() []ArticleMention {
	return append([]ArticleMention(nil), i.articleMentions...)
}

// ParagraphMentions は、項番号出現の複製を返す。
func (i CandidateGenerationInput) ParagraphMentions() []ParagraphMention {
	return append([]ParagraphMention(nil), i.paragraphMentions...)
}

// CaseNumberMentions は、裁判事件番号出現の複製を返す。
func (i CandidateGenerationInput) CaseNumberMentions() []JudicialCaseNumberMention {
	return append([]JudicialCaseNumberMention(nil), i.caseNumberMentions...)
}

// QueryTermMentions は、一般検索語出現の複製を返す。
func (i CandidateGenerationInput) QueryTermMentions() []QueryTermMention {
	return append([]QueryTermMention(nil), i.queryTermMentions...)
}

// CueTaskRelations は、前処理で確認した cue と task の関係を深く複製して返す。
func (i CandidateGenerationInput) CueTaskRelations() []CueTaskRelation {
	return cloneCueTaskRelations(i.cueTaskRelations)
}

// SharedTerminalSequences は、共有末尾列 sidecar の深い複製を返す。
func (i CandidateGenerationInput) SharedTerminalSequences() []SharedTerminalSequence {
	return cloneSharedTerminalSequences(i.sharedTerminalSequences)
}

// Validate は、constructor で得た型付き事実の基本不変条件を確認する。
func (i CandidateGenerationInput) Validate() error {
	if i.language != QueryLanguageJapanese &&
		i.language != QueryLanguageNonJapanese {
		return fmt.Errorf("query language が定義されていません")
	}
	if i.ref != nil {
		if err := i.ref.Validate(); err != nil {
			return fmt.Errorf("ref が有効ではありません: %w", err)
		}
	}
	if i.standaloneStructured &&
		!validStandaloneStructuredFacts(
			i.lawNameMentions,
			i.legalConceptMentions,
			i.cueMentions,
			i.identifierMentions,
			i.dateMentions,
			i.caseNumberMentions,
			i.queryTermMentions,
		) {
		return fmt.Errorf("standalone structured query の事実が有効ではありません")
	}
	for index, mention := range i.lawNameMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("lawNameMentions[%d]: %w", index, err)
		}
	}
	for index, mention := range i.legalConceptMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("legalConceptMentions[%d]: %w", index, err)
		}
	}
	for index, mention := range i.cueMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("cueMentions[%d]: %w", index, err)
		}
	}
	if err := validateCueTaskRelationReferencesAndOrder(
		i.cueTaskRelations,
		i.cueMentions,
	); err != nil {
		return err
	}
	for index, mention := range i.identifierMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("identifierMentions[%d]: %w", index, err)
		}
	}
	for index, mention := range i.dateMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("dateMentions[%d]: %w", index, err)
		}
	}
	for index, mention := range i.articleMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("articleMentions[%d]: %w", index, err)
		}
	}
	for index, mention := range i.paragraphMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("paragraphMentions[%d]: %w", index, err)
		}
	}
	for index, mention := range i.caseNumberMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("caseNumberMentions[%d]: %w", index, err)
		}
	}
	for index, mention := range i.queryTermMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("queryTermMentions[%d]: %w", index, err)
		}
	}
	if err := validateSharedTerminalSequenceReferencesAndOrder(
		i.sharedTerminalSequences,
		i.lawNameMentions,
		i.legalConceptMentions,
		i.queryTermMentions,
		i.cueTaskRelations,
	); err != nil {
		return err
	}
	return nil
}

func classifyStandaloneStructuredQuery(result PreprocessResult) bool {
	if !validStandaloneStructuredFacts(
		result.LawNameMentions(),
		result.LegalConceptMentions(),
		result.CueMentions(),
		result.IdentifierMentions(),
		result.DateMentions(),
		result.CaseNumberMentions(),
		result.QueryTermMentions(),
	) {
		return false
	}
	structuredSpans := standaloneStructuredSpans(
		result.IdentifierMentions(),
		result.DateMentions(),
		result.CaseNumberMentions(),
	)
	covered := make([]bool, len(result.Query()))
	for _, span := range structuredSpans {
		for index := span.StartByte(); index < span.EndByte(); index++ {
			covered[index] = true
		}
	}
	for startByte, character := range result.Query() {
		if unicode.IsSpace(character) ||
			unicode.IsPunct(character) ||
			unicode.IsSymbol(character) {
			continue
		}
		if !covered[startByte] {
			return false
		}
	}
	return true
}

func validStandaloneStructuredFacts(
	lawNames []LawNameMention,
	concepts []LegalConceptMention,
	cues []CueMention,
	identifiers []IdentifierMention,
	dates []DateMention,
	caseNumbers []JudicialCaseNumberMention,
	queryTerms []QueryTermMention,
) bool {
	if len(identifiers)+len(dates)+len(caseNumbers) == 0 ||
		len(lawNames) > 0 ||
		len(concepts) > 0 ||
		len(cues) > 0 {
		return false
	}
	structuredSpans := standaloneStructuredSpans(
		identifiers,
		dates,
		caseNumbers,
	)
	for _, term := range queryTerms {
		if term.Kind() != QueryTermMentionQuotedPhrase ||
			!containsQuerySpan(structuredSpans, term.Span()) {
			return false
		}
	}
	return true
}

func standaloneStructuredSpans(
	identifiers []IdentifierMention,
	dates []DateMention,
	caseNumbers []JudicialCaseNumberMention,
) []QuerySpan {
	spans := make([]QuerySpan, 0, len(identifiers)+len(dates)+len(caseNumbers))
	for _, mention := range identifiers {
		spans = append(spans, mention.Span())
	}
	for _, mention := range dates {
		spans = append(spans, mention.Span())
	}
	for _, mention := range caseNumbers {
		spans = append(spans, mention.Span())
	}
	return spans
}

func containsQuerySpan(values []QuerySpan, target QuerySpan) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func classifyQueryLanguage(result PreprocessResult) QueryLanguage {
	query := result.Query()
	for _, character := range query {
		if unicode.In(character, unicode.Katakana) ||
			isSubstantiveHiragana(character) {
			return QueryLanguageJapanese
		}
	}
	covered := make([]bool, len(query))
	mark := func(span QuerySpan) {
		for index := span.StartByte(); index < span.EndByte(); index++ {
			covered[index] = true
		}
	}
	for _, mention := range result.LawNameMentions() {
		mark(mention.Span())
	}
	for _, mention := range result.LegalConceptMentions() {
		mark(mention.Span())
	}
	for _, mention := range result.CueMentions() {
		mark(mention.Span())
	}
	for _, mention := range result.IdentifierMentions() {
		mark(mention.Span())
	}
	for _, mention := range result.DateMentions() {
		mark(mention.Span())
	}
	for _, mention := range result.ArticleMentions() {
		mark(mention.Span())
	}
	for _, mention := range result.ParagraphMentions() {
		mark(mention.Span())
	}
	for _, mention := range result.CaseNumberMentions() {
		mark(mention.Span())
	}
	for _, mention := range result.QueryTermMentions() {
		if containsSubstantiveJapaneseTerm(mention.Surface()) {
			mark(mention.Span())
		}
	}
	factFound := false
	for start, character := range query {
		if unicode.IsSpace(character) ||
			unicode.IsPunct(character) ||
			unicode.IsSymbol(character) ||
			isJapaneseParticleHiragana(character) {
			continue
		}
		if !covered[start] {
			return QueryLanguageNonJapanese
		}
		factFound = true
	}
	if factFound {
		return QueryLanguageJapanese
	}
	return QueryLanguageNonJapanese
}

func containsSubstantiveJapaneseTerm(value string) bool {
	for _, character := range value {
		if unicode.In(character, unicode.Han, unicode.Katakana) ||
			isSubstantiveHiragana(character) {
			return true
		}
	}
	return false
}

func isSubstantiveHiragana(character rune) bool {
	if !unicode.In(character, unicode.Hiragana) {
		return false
	}
	return !isJapaneseParticleHiragana(character)
}

func isJapaneseParticleHiragana(character rune) bool {
	switch character {
	case 'は', 'が', 'を', 'に', 'へ', 'と', 'で', 'の', 'も', 'や', 'か', 'ね', 'よ':
		return true
	default:
		return false
	}
}
