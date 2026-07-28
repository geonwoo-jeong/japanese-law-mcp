package legalquery

import (
	"context"
	"fmt"
)

const (
	maxPreprocessComparisonBytes = 4096
	maxPreprocessMentions        = 64
	maxPreprocessCueMentions     = 128
	maxPreprocessTotalMentions   = 256
)

// QueryPreprocessor は、provider 非依存の照会文前処理を実行する。
type QueryPreprocessor interface {
	Preprocess(context.Context, Request) (PreprocessResult, error)
}

// PreprocessMatchKind は、検証済み辞書表記へ到達した方法を表す。
type PreprocessMatchKind string

const (
	// PreprocessMatchExact は、照会文全体と辞書表記の完全一致を表す。
	PreprocessMatchExact PreprocessMatchKind = "exact"
	// PreprocessMatchComparisonNormalized は、比較用正規化後の一致を表す。
	PreprocessMatchComparisonNormalized PreprocessMatchKind = "comparison_normalized"
	// PreprocessMatchRegisteredTerm は、Kagome が抽出した登録語を表す。
	PreprocessMatchRegisteredTerm PreprocessMatchKind = "registered_term"
	// PreprocessMatchUniqueTypoCorrection は、一意な辞書表記への誤記補正を表す。
	PreprocessMatchUniqueTypoCorrection PreprocessMatchKind = "unique_typo_correction"
)

// IdentifierMentionKind は、照会文に明記された法令識別子の種別を表す。
type IdentifierMentionKind string

const (
	// IdentifierMentionLawID は、法令 ID を表す。
	IdentifierMentionLawID IdentifierMentionKind = "law_id"
	// IdentifierMentionLawRevisionID は、法令リビジョン ID を表す。
	IdentifierMentionLawRevisionID IdentifierMentionKind = "law_revision_id"
	// IdentifierMentionLawNumber は、法令番号を表す。
	IdentifierMentionLawNumber IdentifierMentionKind = "law_number"
)

// QuerySpanValues は、原文上の half-open byte span を保持する。
type QuerySpanValues struct {
	StartByte int
	EndByte   int
}

// QuerySpan は、原文 UTF-8 上の開始 byte と終端 byte を保持する。
type QuerySpan struct {
	startByte int
	endByte   int
}

// NewQuerySpan は、正の幅を持つ half-open byte span を返す。
func NewQuerySpan(values QuerySpanValues) (QuerySpan, error) {
	span := QuerySpan{
		startByte: values.StartByte,
		endByte:   values.EndByte,
	}
	if err := span.Validate(); err != nil {
		return QuerySpan{}, err
	}
	return span, nil
}

// StartByte は、含まれる開始 byte offset を返す。
func (s QuerySpan) StartByte() int {
	return s.startByte
}

// EndByte は、含まれない終端 byte offset を返す。
func (s QuerySpan) EndByte() int {
	return s.endByte
}

// Validate は、span が非負で正の幅を持つことを確認する。
func (s QuerySpan) Validate() error {
	if s.startByte < 0 || s.endByte <= s.startByte {
		return fmt.Errorf(
			"query span は 0 以上の startByte と、それより大きい endByte が必要です",
		)
	}
	return nil
}

// CueVocabularyEntry は、一つの profile が前処理へ渡す構文 cue 語彙である。
type CueVocabularyEntry struct {
	ProfileID string
	CueID     string
	Terms     []string
}

func validatePreprocessMatchKind(kind PreprocessMatchKind) error {
	switch kind {
	case PreprocessMatchExact,
		PreprocessMatchComparisonNormalized,
		PreprocessMatchRegisteredTerm,
		PreprocessMatchUniqueTypoCorrection:
		return nil
	default:
		return fmt.Errorf("matchKind が定義されていません")
	}
}
