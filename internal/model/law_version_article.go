package model

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// LawVersionArticleLocationValues は、版間比較用の条位置を構築する値を保持する。
type LawVersionArticleLocationValues struct {
	Provision        LawArticleProvision
	ArticleNumber    string
	PartNumber       string
	ChapterNumber    string
	SectionNumber    string
	SubsectionNumber string
	DivisionNumber   string
}

// LawVersionArticleLocation は、条の同一性と補助的な構造位置を保持する。
type LawVersionArticleLocation struct {
	provision        LawArticleProvision
	articleNumber    string
	partNumber       string
	chapterNumber    string
	sectionNumber    string
	subsectionNumber string
	divisionNumber   string
}

// NewLawVersionArticleLocation は、検証済みの版間比較用条位置を返す。
func NewLawVersionArticleLocation(
	values LawVersionArticleLocationValues,
) (LawVersionArticleLocation, error) {
	location := LawVersionArticleLocation{
		provision:        values.Provision,
		articleNumber:    values.ArticleNumber,
		partNumber:       values.PartNumber,
		chapterNumber:    values.ChapterNumber,
		sectionNumber:    values.SectionNumber,
		subsectionNumber: values.SubsectionNumber,
		divisionNumber:   values.DivisionNumber,
	}
	if err := location.Validate(); err != nil {
		return LawVersionArticleLocation{}, err
	}
	return location, nil
}

func (l LawVersionArticleLocation) Provision() LawArticleProvision { return l.provision }
func (l LawVersionArticleLocation) ArticleNumber() string          { return l.articleNumber }
func (l LawVersionArticleLocation) PartNumber() (string, bool) {
	return l.partNumber, l.partNumber != ""
}
func (l LawVersionArticleLocation) ChapterNumber() (string, bool) {
	return l.chapterNumber, l.chapterNumber != ""
}
func (l LawVersionArticleLocation) SectionNumber() (string, bool) {
	return l.sectionNumber, l.sectionNumber != ""
}
func (l LawVersionArticleLocation) SubsectionNumber() (string, bool) {
	return l.subsectionNumber, l.subsectionNumber != ""
}
func (l LawVersionArticleLocation) DivisionNumber() (string, bool) {
	return l.divisionNumber, l.divisionNumber != ""
}

// HasSameIdentity は、SOT-MODEL-033 の条同一性が同じかを返す。
func (l LawVersionArticleLocation) HasSameIdentity(other LawVersionArticleLocation) bool {
	return l.provision == other.provision && l.articleNumber == other.articleNumber
}

// HasSameAuxiliaryLocation は、同一性以外の階層位置が同じかを返す。
func (l LawVersionArticleLocation) HasSameAuxiliaryLocation(
	other LawVersionArticleLocation,
) bool {
	return l.partNumber == other.partNumber &&
		l.chapterNumber == other.chapterNumber &&
		l.sectionNumber == other.sectionNumber &&
		l.subsectionNumber == other.subsectionNumber &&
		l.divisionNumber == other.divisionNumber
}

// Validate は、条の同一性と省略可能な階層番号を検証する。
func (l LawVersionArticleLocation) Validate() error {
	switch l.provision {
	case LawArticleProvisionMain, LawArticleProvisionSupplementary:
	default:
		return fmt.Errorf("provision は main または supplementary でなければなりません")
	}
	if err := validateLawVersionArticleNumber(l.articleNumber); err != nil {
		return err
	}
	for _, value := range []string{
		l.partNumber,
		l.chapterNumber,
		l.sectionNumber,
		l.subsectionNumber,
		l.divisionNumber,
	} {
		if !utf8.ValidString(value) {
			return fmt.Errorf("階層番号は有効な UTF-8 でなければなりません")
		}
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-033 の条位置を表す。
func (l LawVersionArticleLocation) MarshalJSON() ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Provision        LawArticleProvision `json:"provision"`
		ArticleNumber    string              `json:"articleNumber"`
		PartNumber       string              `json:"partNumber,omitempty"`
		ChapterNumber    string              `json:"chapterNumber,omitempty"`
		SectionNumber    string              `json:"sectionNumber,omitempty"`
		SubsectionNumber string              `json:"subsectionNumber,omitempty"`
		DivisionNumber   string              `json:"divisionNumber,omitempty"`
	}{
		Provision:        l.provision,
		ArticleNumber:    l.articleNumber,
		PartNumber:       l.partNumber,
		ChapterNumber:    l.chapterNumber,
		SectionNumber:    l.sectionNumber,
		SubsectionNumber: l.subsectionNumber,
		DivisionNumber:   l.divisionNumber,
	})
}

func (*LawVersionArticleLocation) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("LawVersionArticleLocation は JSON から直接復元できません。NewLawVersionArticleLocation を使用してください")
}

// LawVersionArticleValues は、比較対象の一条を構築する値を保持する。
type LawVersionArticleValues struct {
	Location             LawVersionArticleLocation
	ArticleTitle         string
	ArticleCaption       string
	Text                 string
	Citation             Citation
	DocumentOrder        int
	StructureFingerprint string
}

// LawVersionArticle は、版間比較に使う条の共通表現である。
type LawVersionArticle struct {
	location             LawVersionArticleLocation
	articleTitle         string
	articleCaption       string
	text                 string
	citation             Citation
	documentOrder        int
	structureFingerprint string
}

// NewLawVersionArticle は、空文字の本文も保持する検証済みの条を返す。
func NewLawVersionArticle(values LawVersionArticleValues) (LawVersionArticle, error) {
	article := LawVersionArticle{
		location:             values.Location,
		articleTitle:         values.ArticleTitle,
		articleCaption:       values.ArticleCaption,
		text:                 values.Text,
		citation:             values.Citation,
		documentOrder:        values.DocumentOrder,
		structureFingerprint: values.StructureFingerprint,
	}
	if err := article.Validate(); err != nil {
		return LawVersionArticle{}, err
	}
	return article, nil
}

func (a LawVersionArticle) Location() LawVersionArticleLocation { return a.location }
func (a LawVersionArticle) ArticleTitle() (string, bool) {
	return a.articleTitle, a.articleTitle != ""
}
func (a LawVersionArticle) ArticleCaption() (string, bool) {
	return a.articleCaption, a.articleCaption != ""
}
func (a LawVersionArticle) Text() string       { return a.text }
func (a LawVersionArticle) Citation() Citation { return a.citation }

// Validate は、比較用文字列を空文字のまま許可し、各構成値を検証する。
func (a LawVersionArticle) Validate() error {
	if err := a.location.Validate(); err != nil {
		return fmt.Errorf("location が有効ではありません: %w", err)
	}
	for name, value := range map[string]string{
		"articleTitle":         a.articleTitle,
		"articleCaption":       a.articleCaption,
		"text":                 a.text,
		"structureFingerprint": a.structureFingerprint,
	} {
		if !utf8.ValidString(value) {
			return fmt.Errorf("%s は有効な UTF-8 でなければなりません", name)
		}
	}
	if a.documentOrder < 1 {
		return fmt.Errorf("documentOrder は 1 以上でなければなりません")
	}
	if a.structureFingerprint == "" {
		return fmt.Errorf("structureFingerprint は必須です")
	}
	if err := a.citation.Validate(); err != nil {
		return fmt.Errorf("citation が有効ではありません: %w", err)
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-033 の条表現を返す。
func (a LawVersionArticle) MarshalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Location       LawVersionArticleLocation `json:"location"`
		ArticleTitle   string                    `json:"articleTitle,omitempty"`
		ArticleCaption string                    `json:"articleCaption,omitempty"`
		Text           string                    `json:"text"`
		Citation       Citation                  `json:"citation"`
	}{
		Location:       a.location,
		ArticleTitle:   a.articleTitle,
		ArticleCaption: a.articleCaption,
		Text:           a.text,
		Citation:       a.citation,
	})
}

func (*LawVersionArticle) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("LawVersionArticle は JSON から直接復元できません。NewLawVersionArticle を使用してください")
}

// LawVersionSnapshotValues は、一つの確定版を構築する値を保持する。
type LawVersionSnapshotValues struct {
	Law      LawSummary
	AsOf     *Date
	Citation Citation
}

// LawVersionSnapshot は、比較前又は比較後として確定した法令版を表す。
type LawVersionSnapshot struct {
	law      LawSummary
	asOf     *Date
	citation Citation
}

// NewLawVersionSnapshot は、検証済みの確定版を返す。
func NewLawVersionSnapshot(values LawVersionSnapshotValues) (LawVersionSnapshot, error) {
	snapshot := LawVersionSnapshot{
		law:      values.Law,
		asOf:     cloneOptionalDate(values.AsOf),
		citation: values.Citation,
	}
	if err := snapshot.Validate(); err != nil {
		return LawVersionSnapshot{}, err
	}
	return snapshot, nil
}

func (s LawVersionSnapshot) Law() LawSummary    { return s.law }
func (s LawVersionSnapshot) Citation() Citation { return s.citation }
func (s LawVersionSnapshot) AsOf() (Date, bool) {
	if s.asOf == nil {
		return Date{}, false
	}
	return *s.asOf, true
}

// Validate は、版概要と版全体の出典の一致を検証する。
func (s LawVersionSnapshot) Validate() error {
	if err := s.law.Validate(); err != nil {
		return fmt.Errorf("law が有効ではありません: %w", err)
	}
	if err := s.citation.Validate(); err != nil {
		return fmt.Errorf("citation が有効ではありません: %w", err)
	}
	if s.asOf != nil {
		if err := s.asOf.Validate(); err != nil {
			return fmt.Errorf("asOf が有効ではありません: %w", err)
		}
	}
	if s.law.LawID() != s.citation.LawID() ||
		s.law.RevisionID() != s.citation.RevisionID() ||
		s.law.Source() != s.citation.Source() {
		return fmt.Errorf("law と citation が同じ法令版及び情報源を指していません")
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-033 の確定版表現を返す。
func (s LawVersionSnapshot) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Law      LawSummary `json:"law"`
		AsOf     *Date      `json:"asOf,omitempty"`
		Citation Citation   `json:"citation"`
	}{
		Law:      s.law,
		AsOf:     cloneOptionalDate(s.asOf),
		Citation: s.citation,
	})
}

func (*LawVersionSnapshot) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("LawVersionSnapshot は JSON から直接復元できません。NewLawVersionSnapshot を使用してください")
}
