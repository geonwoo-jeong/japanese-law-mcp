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
	language             QueryLanguage
	ref                  *model.SourceResourceRef
	lawNameMentions      []LawNameMention
	legalConceptMentions []LegalConceptMention
	cueMentions          []CueMention
	identifierMentions   []IdentifierMention
	dateMentions         []DateMention
	articleMentions      []ArticleMention
	paragraphMentions    []ParagraphMention
	queryTermMentions    []QueryTermMention
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
	input := CandidateGenerationInput{
		language:             classifyQueryLanguage(result),
		lawNameMentions:      result.LawNameMentions(),
		legalConceptMentions: result.LegalConceptMentions(),
		cueMentions:          result.CueMentions(),
		identifierMentions:   result.IdentifierMentions(),
		dateMentions:         result.DateMentions(),
		articleMentions:      result.ArticleMentions(),
		paragraphMentions:    result.ParagraphMentions(),
		queryTermMentions:    result.QueryTermMentions(),
	}
	if ref, exists := result.Ref(); exists {
		input.ref = &ref
	}
	return input, nil
}

// Language は、原文を公開せず script 分類だけを返す。
func (i CandidateGenerationInput) Language() QueryLanguage {
	return i.language
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

// QueryTermMentions は、一般検索語出現の複製を返す。
func (i CandidateGenerationInput) QueryTermMentions() []QueryTermMention {
	return append([]QueryTermMention(nil), i.queryTermMentions...)
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
	for index, mention := range i.queryTermMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("queryTermMentions[%d]: %w", index, err)
		}
	}
	return nil
}

func classifyQueryLanguage(result PreprocessResult) QueryLanguage {
	query := result.Query()
	for _, character := range query {
		if unicode.In(character, unicode.Hiragana, unicode.Katakana) {
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
	for _, mention := range result.QueryTermMentions() {
		mark(mention.Span())
	}
	factFound := false
	for start, character := range query {
		if unicode.IsSpace(character) ||
			unicode.IsPunct(character) ||
			unicode.IsSymbol(character) {
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
