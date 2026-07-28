package legalquery

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

// QueryTermMentionKind は、原文で一般検索語の境界を得た方法を表す。
type QueryTermMentionKind string

const (
	// QueryTermMentionQuotedPhrase は、対応する引用区切りの内部を表す。
	QueryTermMentionQuotedPhrase QueryTermMentionKind = "quoted_phrase"
	// QueryTermMentionMorphologicalPhrase は、形態素解析で確認した最大名詞句を表す。
	QueryTermMentionMorphologicalPhrase QueryTermMentionKind = "morphological_phrase"
)

// QueryTermMentionValues は、一般検索語の出現を構築する値である。
type QueryTermMentionValues struct {
	Span    QuerySpan
	Surface string
	Kind    QueryTermMentionKind
}

// QueryTermMention は、原文にある provider 非依存の一般検索語候補を表す。
type QueryTermMention struct {
	span    QuerySpan
	surface string
	kind    QueryTermMentionKind
}

// NewQueryTermMention は、位置と抽出方法を検証した一般検索語出現を返す。
func NewQueryTermMention(values QueryTermMentionValues) (QueryTermMention, error) {
	mention := QueryTermMention{
		span:    values.Span,
		surface: values.Surface,
		kind:    values.Kind,
	}
	if err := mention.Validate(); err != nil {
		return QueryTermMention{}, err
	}
	return mention, nil
}

// Span は、原文上の byte span を返す。
func (m QueryTermMention) Span() QuerySpan {
	return m.span
}

// Surface は、原文に現れた検索語候補を返す。
func (m QueryTermMention) Surface() string {
	return m.surface
}

// Kind は、検索語候補の抽出方法を返す。
func (m QueryTermMention) Kind() QueryTermMentionKind {
	return m.kind
}

// Validate は、一般検索語出現の必須値と閉じた種別を確認する。
func (m QueryTermMention) Validate() error {
	if err := validateMentionBase(m.span, m.surface); err != nil {
		return err
	}
	switch m.kind {
	case QueryTermMentionQuotedPhrase:
		if strings.TrimFunc(m.surface, unicode.IsSpace) != m.surface {
			return fmt.Errorf(
				"quoted phrase の先頭と末尾に Unicode 空白を含めることはできません",
			)
		}
		return nil
	case QueryTermMentionMorphologicalPhrase:
		if !containsJapaneseQueryTermScript(m.surface) {
			return fmt.Errorf(
				"morphological phrase は日本語 script を一文字以上含まなければなりません",
			)
		}
		if utf8.RuneCountInString(
			querynormalization.ComparisonKey(m.surface),
		) < 2 {
			return fmt.Errorf(
				"morphological phrase は比較用正規化後に二文字以上必要です",
			)
		}
		return nil
	default:
		return fmt.Errorf("query term mention kind が定義されていません")
	}
}

func containsJapaneseQueryTermScript(value string) bool {
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
