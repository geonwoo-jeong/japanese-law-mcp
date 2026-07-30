package legalquery

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

// PreprocessResultValues は、前処理結果を構築する値である。
type PreprocessResultValues struct {
	Query                string
	ComparisonKey        string
	Ref                  *model.SourceResourceRef
	LawNameMentions      []LawNameMention
	LegalConceptMentions []LegalConceptMention
	CueMentions          []CueMention
	IdentifierMentions   []IdentifierMention
	DateMentions         []DateMention
	ArticleMentions      []ArticleMention
	ParagraphMentions    []ParagraphMention
	CaseNumberMentions   []JudicialCaseNumberMention
	QueryTermMentions    []QueryTermMention
	CueTaskRelations     []CueTaskRelation
}

// PreprocessResult は、原文上の位置を保持した provider 非依存の事実である。
type PreprocessResult struct {
	query                string
	comparisonKey        string
	ref                  *model.SourceResourceRef
	lawNameMentions      []LawNameMention
	legalConceptMentions []LegalConceptMention
	cueMentions          []CueMention
	identifierMentions   []IdentifierMention
	dateMentions         []DateMention
	articleMentions      []ArticleMention
	paragraphMentions    []ParagraphMention
	caseNumberMentions   []JudicialCaseNumberMention
	queryTermMentions    []QueryTermMention
	cueTaskRelations     []CueTaskRelation
}

// NewPreprocessResult は、入力を複製し、正規順と上限を検証する。
func NewPreprocessResult(values PreprocessResultValues) (PreprocessResult, error) {
	result := PreprocessResult{
		query:                values.Query,
		comparisonKey:        values.ComparisonKey,
		lawNameMentions:      append([]LawNameMention(nil), values.LawNameMentions...),
		legalConceptMentions: append([]LegalConceptMention(nil), values.LegalConceptMentions...),
		cueMentions:          append([]CueMention(nil), values.CueMentions...),
		identifierMentions:   append([]IdentifierMention(nil), values.IdentifierMentions...),
		dateMentions:         append([]DateMention(nil), values.DateMentions...),
		articleMentions:      append([]ArticleMention(nil), values.ArticleMentions...),
		paragraphMentions:    append([]ParagraphMention(nil), values.ParagraphMentions...),
		caseNumberMentions:   append([]JudicialCaseNumberMention(nil), values.CaseNumberMentions...),
		queryTermMentions:    append([]QueryTermMention(nil), values.QueryTermMentions...),
		cueTaskRelations:     cloneCueTaskRelations(values.CueTaskRelations),
	}
	if values.Ref != nil {
		ref := *values.Ref
		result.ref = &ref
	}
	if err := result.Validate(); err != nil {
		return PreprocessResult{}, err
	}
	return result, nil
}

// Query は、検証済みの原文を返す。
func (r PreprocessResult) Query() string {
	return r.query
}

// ComparisonKey は、内部比較だけに使う正規化値を返す。
func (r PreprocessResult) ComparisonKey() string {
	return r.comparisonKey
}

// Ref は、入力で受け取った参照の複製と有無を返す。
func (r PreprocessResult) Ref() (model.SourceResourceRef, bool) {
	if r.ref == nil {
		return model.SourceResourceRef{}, false
	}
	return *r.ref, true
}

// LawNameMentions は、法令名出現の複製を返す。
func (r PreprocessResult) LawNameMentions() []LawNameMention {
	return append([]LawNameMention(nil), r.lawNameMentions...)
}

// LegalConceptMentions は、法概念出現の複製を返す。
func (r PreprocessResult) LegalConceptMentions() []LegalConceptMention {
	return append([]LegalConceptMention(nil), r.legalConceptMentions...)
}

// CueMentions は、構文 cue 出現の複製を返す。
func (r PreprocessResult) CueMentions() []CueMention {
	return append([]CueMention(nil), r.cueMentions...)
}

// IdentifierMentions は、法令識別子出現の複製を返す。
func (r PreprocessResult) IdentifierMentions() []IdentifierMention {
	return append([]IdentifierMention(nil), r.identifierMentions...)
}

// DateMentions は、完全な暦日出現の複製を返す。
func (r PreprocessResult) DateMentions() []DateMention {
	return append([]DateMention(nil), r.dateMentions...)
}

// ArticleMentions は、条番号出現の複製を返す。
func (r PreprocessResult) ArticleMentions() []ArticleMention {
	return append([]ArticleMention(nil), r.articleMentions...)
}

// ParagraphMentions は、項番号出現の複製を返す。
func (r PreprocessResult) ParagraphMentions() []ParagraphMention {
	return append([]ParagraphMention(nil), r.paragraphMentions...)
}

// CaseNumberMentions は、裁判事件番号出現の複製を返す。
func (r PreprocessResult) CaseNumberMentions() []JudicialCaseNumberMention {
	return append([]JudicialCaseNumberMention(nil), r.caseNumberMentions...)
}

// QueryTermMentions は、一般検索語出現の複製を返す。
func (r PreprocessResult) QueryTermMentions() []QueryTermMention {
	return append([]QueryTermMention(nil), r.queryTermMentions...)
}

// CueTaskRelations は、構文 cue と task の関係を深く複製して返す。
func (r PreprocessResult) CueTaskRelations() []CueTaskRelation {
	return cloneCueTaskRelations(r.cueTaskRelations)
}

// Validate は、前処理結果の構造、位置、順序および上限を確認する。
func (r PreprocessResult) Validate() error {
	request, err := NewRequest(RequestValues{Query: r.query, Ref: r.ref})
	if err != nil {
		return fmt.Errorf("query が有効ではありません: %w", err)
	}
	if request.Query() != r.query {
		return fmt.Errorf("query は検証済み原文と一致しなければなりません")
	}
	if !utf8.ValidString(r.comparisonKey) ||
		len(r.comparisonKey) > maxPreprocessComparisonBytes {
		return fmt.Errorf(
			"comparisonKey は有効な UTF-8 で %d byte 以下でなければなりません",
			maxPreprocessComparisonBytes,
		)
	}
	if expected := querynormalization.ComparisonKey(r.query); r.comparisonKey != expected {
		return fmt.Errorf("comparisonKey は query の比較用正規化値と一致しなければなりません")
	}

	total := len(r.lawNameMentions) +
		len(r.legalConceptMentions) +
		len(r.cueMentions) +
		len(r.identifierMentions) +
		len(r.dateMentions) +
		len(r.articleMentions) +
		len(r.paragraphMentions) +
		len(r.caseNumberMentions) +
		len(r.queryTermMentions)
	if total > maxPreprocessTotalMentions {
		return fmt.Errorf(
			"前処理の全出現は %d 件以下でなければなりません",
			maxPreprocessTotalMentions,
		)
	}
	if err := validateMentionSequence(
		r.query,
		"lawNameMentions",
		r.lawNameMentions,
		maxPreprocessMentions,
		func(value LawNameMention) string { return value.LawID() },
	); err != nil {
		return err
	}
	if err := validateMentionSequence(
		r.query,
		"legalConceptMentions",
		r.legalConceptMentions,
		maxPreprocessMentions,
		func(value LegalConceptMention) string { return value.ConceptID() },
	); err != nil {
		return err
	}
	if err := validateMentionSequence(
		r.query,
		"cueMentions",
		r.cueMentions,
		maxPreprocessCueMentions,
		func(value CueMention) string {
			return value.ProfileID() + "\x00" + value.CueID()
		},
	); err != nil {
		return err
	}
	if err := validateMentionSequence(
		r.query,
		"identifierMentions",
		r.identifierMentions,
		maxPreprocessMentions,
		identifierMentionIdentity,
	); err != nil {
		return err
	}
	if err := validateMentionSequence(
		r.query,
		"dateMentions",
		r.dateMentions,
		maxPreprocessMentions,
		func(value DateMention) string { return value.Date().String() },
	); err != nil {
		return err
	}
	if err := validateMentionSequence(
		r.query,
		"articleMentions",
		r.articleMentions,
		maxPreprocessMentions,
		func(value ArticleMention) string {
			return string(value.Provision()) + "\x00" + value.ArticleNumber()
		},
	); err != nil {
		return err
	}
	if err := validateMentionSequence(
		r.query,
		"paragraphMentions",
		r.paragraphMentions,
		maxPreprocessMentions,
		func(value ParagraphMention) string {
			return strconv.Itoa(value.ParagraphNumber())
		},
	); err != nil {
		return err
	}
	if err := validateMentionSequence(
		r.query,
		"caseNumberMentions",
		r.caseNumberMentions,
		maxPreprocessMentions,
		func(value JudicialCaseNumberMention) string {
			return strings.Join([]string{
				value.Era(),
				strconv.Itoa(value.Year()),
				value.CaseCode(),
				strconv.Itoa(value.SerialNumber()),
			}, "\x00")
		},
	); err != nil {
		return err
	}
	if err := validateMentionSequence(
		r.query,
		"queryTermMentions",
		r.queryTermMentions,
		maxPreprocessMentions,
		func(value QueryTermMention) string {
			return string(value.Kind())
		},
	); err != nil {
		return err
	}
	if err := validateCueTaskRelationSequence(
		r.query,
		r.cueTaskRelations,
		r.cueMentions,
	); err != nil {
		return err
	}
	return nil
}

type preprocessMention interface {
	Span() QuerySpan
	Surface() string
	Validate() error
}

func validateMentionSequence[T preprocessMention](
	query string,
	name string,
	values []T,
	maximum int,
	identity func(T) string,
) error {
	if len(values) > maximum {
		return fmt.Errorf("%s は %d 件以下でなければなりません", name, maximum)
	}
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return fmt.Errorf("%s[%d] が有効ではありません: %w", name, index, err)
		}
		if err := validateMentionSurface(query, value.Span(), value.Surface()); err != nil {
			return fmt.Errorf("%s[%d]: %w", name, index, err)
		}
		if index == 0 {
			continue
		}
		previous := values[index-1]
		if compareMention(
			previous.Span(),
			identity(previous),
			value.Span(),
			identity(value),
		) >= 0 {
			return fmt.Errorf("%s は正規順で重複なく保持しなければなりません", name)
		}
	}
	return nil
}

func validateMentionSurface(query string, span QuerySpan, surface string) error {
	if span.endByte > len(query) {
		return fmt.Errorf("query span が原文の範囲を超えています")
	}
	if !isRuneBoundary(query, span.startByte) || !isRuneBoundary(query, span.endByte) {
		return fmt.Errorf("query span の両端は UTF-8 rune 境界でなければなりません")
	}
	if query[span.startByte:span.endByte] != surface {
		return fmt.Errorf("surface は query span の原文と一致しなければなりません")
	}
	return nil
}

func isRuneBoundary(value string, index int) bool {
	if index < 0 || index > len(value) {
		return false
	}
	return index == 0 || index == len(value) || utf8.RuneStart(value[index])
}

func compareMention(
	leftSpan QuerySpan,
	leftID string,
	rightSpan QuerySpan,
	rightID string,
) int {
	switch {
	case leftSpan.startByte < rightSpan.startByte:
		return -1
	case leftSpan.startByte > rightSpan.startByte:
		return 1
	case leftSpan.endByte > rightSpan.endByte:
		return -1
	case leftSpan.endByte < rightSpan.endByte:
		return 1
	default:
		return strings.Compare(leftID, rightID)
	}
}

func identifierMentionIdentity(value IdentifierMention) string {
	revisionID, _ := value.RevisionID()
	lawNumber, _ := value.LawNumber()
	return strings.Join([]string{
		string(value.Kind()),
		value.LawID(),
		revisionID,
		lawNumber,
	}, "\x00")
}
