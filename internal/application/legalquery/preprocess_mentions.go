package legalquery

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// LawNameMentionValues は、法令名辞書との一致を構築する値である。
type LawNameMentionValues struct {
	Span       QuerySpan
	Surface    string
	LawID      string
	RevisionID string
	LawNumber  string
	Canonical  string
	MatchKind  PreprocessMatchKind
}

// LawNameMention は、法令名辞書との一つの一致を表す。
type LawNameMention struct {
	span       QuerySpan
	surface    string
	lawID      string
	revisionID string
	lawNumber  string
	canonical  string
	matchKind  PreprocessMatchKind
}

// NewLawNameMention は、検証済みの法令名出現を返す。
func NewLawNameMention(values LawNameMentionValues) (LawNameMention, error) {
	mention := LawNameMention{
		span:       values.Span,
		surface:    values.Surface,
		lawID:      values.LawID,
		revisionID: values.RevisionID,
		lawNumber:  values.LawNumber,
		canonical:  values.Canonical,
		matchKind:  values.MatchKind,
	}
	if err := mention.Validate(); err != nil {
		return LawNameMention{}, err
	}
	return mention, nil
}

func (m LawNameMention) Span() QuerySpan                { return m.span }
func (m LawNameMention) Surface() string                { return m.surface }
func (m LawNameMention) LawID() string                  { return m.lawID }
func (m LawNameMention) RevisionID() string             { return m.revisionID }
func (m LawNameMention) LawNumber() string              { return m.lawNumber }
func (m LawNameMention) Canonical() string              { return m.canonical }
func (m LawNameMention) MatchKind() PreprocessMatchKind { return m.matchKind }

// Validate は、法令名出現の必須値と照合方法を確認する。
func (m LawNameMention) Validate() error {
	if err := validateMentionBase(m.span, m.surface); err != nil {
		return err
	}
	for field, value := range map[string]string{
		"lawId":      m.lawID,
		"revisionId": m.revisionID,
		"lawNumber":  m.lawNumber,
		"canonical":  m.canonical,
	} {
		if err := validateRequiredInternalText(field, value); err != nil {
			return err
		}
	}
	return validatePreprocessMatchKind(m.matchKind)
}

// LegalConceptMentionValues は、法概念辞書との一致を構築する値である。
type LegalConceptMentionValues struct {
	Span      QuerySpan
	Surface   string
	ConceptID string
	Canonical string
	MatchKind PreprocessMatchKind
}

// LegalConceptMention は、法概念辞書との一つの一致を表す。
type LegalConceptMention struct {
	span      QuerySpan
	surface   string
	conceptID string
	canonical string
	matchKind PreprocessMatchKind
}

// NewLegalConceptMention は、検証済みの法概念出現を返す。
func NewLegalConceptMention(
	values LegalConceptMentionValues,
) (LegalConceptMention, error) {
	mention := LegalConceptMention{
		span:      values.Span,
		surface:   values.Surface,
		conceptID: values.ConceptID,
		canonical: values.Canonical,
		matchKind: values.MatchKind,
	}
	if err := mention.Validate(); err != nil {
		return LegalConceptMention{}, err
	}
	return mention, nil
}

func (m LegalConceptMention) Span() QuerySpan                { return m.span }
func (m LegalConceptMention) Surface() string                { return m.surface }
func (m LegalConceptMention) ConceptID() string              { return m.conceptID }
func (m LegalConceptMention) Canonical() string              { return m.canonical }
func (m LegalConceptMention) MatchKind() PreprocessMatchKind { return m.matchKind }

// Validate は、法概念出現の必須値と照合方法を確認する。
func (m LegalConceptMention) Validate() error {
	if err := validateMentionBase(m.span, m.surface); err != nil {
		return err
	}
	if err := validateRequiredInternalText("conceptId", m.conceptID); err != nil {
		return err
	}
	if err := validateRequiredInternalText("canonical", m.canonical); err != nil {
		return err
	}
	return validatePreprocessMatchKind(m.matchKind)
}

// CueMentionValues は、profile cue との一致を構築する値である。
type CueMentionValues struct {
	Span      QuerySpan
	Surface   string
	ProfileID string
	CueID     string
	MatchKind PreprocessMatchKind
}

// CueMention は、profile から注入された一つの構文 cue を表す。
type CueMention struct {
	span      QuerySpan
	surface   string
	profileID string
	cueID     string
	matchKind PreprocessMatchKind
}

// NewCueMention は、検証済みの cue 出現を返す。
func NewCueMention(values CueMentionValues) (CueMention, error) {
	mention := CueMention{
		span:      values.Span,
		surface:   values.Surface,
		profileID: values.ProfileID,
		cueID:     values.CueID,
		matchKind: values.MatchKind,
	}
	if err := mention.Validate(); err != nil {
		return CueMention{}, err
	}
	return mention, nil
}

func (m CueMention) Span() QuerySpan                { return m.span }
func (m CueMention) Surface() string                { return m.surface }
func (m CueMention) ProfileID() string              { return m.profileID }
func (m CueMention) CueID() string                  { return m.cueID }
func (m CueMention) MatchKind() PreprocessMatchKind { return m.matchKind }

// Validate は、cue 出現の必須値と照合方法を確認する。
func (m CueMention) Validate() error {
	if err := validateMentionBase(m.span, m.surface); err != nil {
		return err
	}
	if err := validateRequiredInternalText("profileId", m.profileID); err != nil {
		return err
	}
	if err := validateRequiredInternalText("cueId", m.cueID); err != nil {
		return err
	}
	if err := validatePreprocessMatchKind(m.matchKind); err != nil {
		return err
	}
	if m.matchKind == PreprocessMatchUniqueTypoCorrection {
		return fmt.Errorf("cue に誤記補正は使用できません")
	}
	return nil
}

// IdentifierMentionValues は、法令識別子出現を構築する値である。
type IdentifierMentionValues struct {
	Span       QuerySpan
	Surface    string
	Kind       IdentifierMentionKind
	LawID      string
	RevisionID *string
	LawNumber  *string
}

// IdentifierMention は、辞書で確認した法令識別子または法令番号を表す。
type IdentifierMention struct {
	span       QuerySpan
	surface    string
	kind       IdentifierMentionKind
	lawID      string
	revisionID string
	lawNumber  string
}

// NewIdentifierMention は、三つの閉じた形の識別子出現を返す。
func NewIdentifierMention(values IdentifierMentionValues) (IdentifierMention, error) {
	mention := IdentifierMention{
		span:    values.Span,
		surface: values.Surface,
		kind:    values.Kind,
		lawID:   values.LawID,
	}
	if values.RevisionID != nil {
		mention.revisionID = *values.RevisionID
	}
	if values.LawNumber != nil {
		mention.lawNumber = *values.LawNumber
	}
	if err := mention.Validate(); err != nil {
		return IdentifierMention{}, err
	}
	return mention, nil
}

func (m IdentifierMention) Span() QuerySpan             { return m.span }
func (m IdentifierMention) Surface() string             { return m.surface }
func (m IdentifierMention) Kind() IdentifierMentionKind { return m.kind }
func (m IdentifierMention) LawID() string               { return m.lawID }
func (m IdentifierMention) RevisionID() (string, bool)  { return m.revisionID, m.revisionID != "" }
func (m IdentifierMention) LawNumber() (string, bool)   { return m.lawNumber, m.lawNumber != "" }

// Validate は、識別子種別ごとの排他的な値を確認する。
func (m IdentifierMention) Validate() error {
	if err := validateMentionBase(m.span, m.surface); err != nil {
		return err
	}
	if err := validateRequiredInternalText("lawId", m.lawID); err != nil {
		return err
	}
	switch m.kind {
	case IdentifierMentionLawID:
		if m.revisionID != "" || m.lawNumber != "" {
			return fmt.Errorf("law_id は revisionId または lawNumber を持てません")
		}
	case IdentifierMentionLawRevisionID:
		if m.revisionID == "" || m.lawNumber != "" {
			return fmt.Errorf("law_revision_id は revisionId だけを追加で必要とします")
		}
		return validateRequiredInternalText("revisionId", m.revisionID)
	case IdentifierMentionLawNumber:
		if m.lawNumber == "" || m.revisionID != "" {
			return fmt.Errorf("law_number は lawNumber だけを追加で必要とします")
		}
		return validateRequiredInternalText("lawNumber", m.lawNumber)
	default:
		return fmt.Errorf("identifier mention kind が定義されていません")
	}
	return nil
}

// DateMentionValues は、完全な暦日の出現を構築する値である。
type DateMentionValues struct {
	Span    QuerySpan
	Surface string
	Date    model.Date
}

// DateMention は、原文に完全に明記された暦日を表す。
type DateMention struct {
	span    QuerySpan
	surface string
	date    model.Date
}

// NewDateMention は、検証済みの暦日出現を返す。
func NewDateMention(values DateMentionValues) (DateMention, error) {
	mention := DateMention{
		span:    values.Span,
		surface: values.Surface,
		date:    values.Date,
	}
	if err := mention.Validate(); err != nil {
		return DateMention{}, err
	}
	return mention, nil
}

func (m DateMention) Span() QuerySpan  { return m.span }
func (m DateMention) Surface() string  { return m.surface }
func (m DateMention) Date() model.Date { return m.date }

// Validate は、暦日出現を確認する。
func (m DateMention) Validate() error {
	if err := validateMentionBase(m.span, m.surface); err != nil {
		return err
	}
	if err := m.date.Validate(); err != nil {
		return fmt.Errorf("date が有効ではありません: %w", err)
	}
	return nil
}

// ArticleMentionValues は、条番号の出現を構築する値である。
type ArticleMentionValues struct {
	Span          QuerySpan
	Surface       string
	Provision     model.LawArticleProvision
	ArticleNumber string
}

// ArticleMention は、原文に明記された条番号を表す。
type ArticleMention struct {
	span          QuerySpan
	surface       string
	provision     model.LawArticleProvision
	articleNumber string
}

// NewArticleMention は、検証済みの条番号出現を返す。
func NewArticleMention(values ArticleMentionValues) (ArticleMention, error) {
	mention := ArticleMention{
		span:          values.Span,
		surface:       values.Surface,
		provision:     values.Provision,
		articleNumber: values.ArticleNumber,
	}
	if err := mention.Validate(); err != nil {
		return ArticleMention{}, err
	}
	return mention, nil
}

func (m ArticleMention) Span() QuerySpan                      { return m.span }
func (m ArticleMention) Surface() string                      { return m.surface }
func (m ArticleMention) Provision() model.LawArticleProvision { return m.provision }
func (m ArticleMention) ArticleNumber() string                { return m.articleNumber }

// Validate は、条番号を共通 LawArticleLocation と同じ規則で確認する。
func (m ArticleMention) Validate() error {
	if err := validateMentionBase(m.span, m.surface); err != nil {
		return err
	}
	_, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:     m.provision,
		ArticleNumber: m.articleNumber,
	})
	if err != nil {
		return fmt.Errorf("article mention が有効ではありません: %w", err)
	}
	return nil
}

// ParagraphMentionValues は、項番号の出現を構築する値である。
type ParagraphMentionValues struct {
	Span            QuerySpan
	Surface         string
	ParagraphNumber int
}

// ParagraphMention は、原文に明記された項番号を表す。
type ParagraphMention struct {
	span            QuerySpan
	surface         string
	paragraphNumber int
}

// NewParagraphMention は、検証済みの項番号出現を返す。
func NewParagraphMention(values ParagraphMentionValues) (ParagraphMention, error) {
	mention := ParagraphMention{
		span:            values.Span,
		surface:         values.Surface,
		paragraphNumber: values.ParagraphNumber,
	}
	if err := mention.Validate(); err != nil {
		return ParagraphMention{}, err
	}
	return mention, nil
}

func (m ParagraphMention) Span() QuerySpan      { return m.span }
func (m ParagraphMention) Surface() string      { return m.surface }
func (m ParagraphMention) ParagraphNumber() int { return m.paragraphNumber }

// Validate は、項番号が一以上であることを確認する。
func (m ParagraphMention) Validate() error {
	if err := validateMentionBase(m.span, m.surface); err != nil {
		return err
	}
	if m.paragraphNumber < 1 {
		return fmt.Errorf("paragraphNumber は 1 以上でなければなりません")
	}
	return nil
}

func validateMentionBase(span QuerySpan, surface string) error {
	if err := span.Validate(); err != nil {
		return err
	}
	if err := validateRequiredInternalText("surface", surface); err != nil {
		return err
	}
	if len(surface) != span.endByte-span.startByte {
		return fmt.Errorf("surface の byte 数は query span の幅と一致しなければなりません")
	}
	return nil
}
