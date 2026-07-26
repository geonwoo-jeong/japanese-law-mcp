package model

import (
	"encoding/json"
	"fmt"
)

// LawSummaryValues は、LawSummary の作成に必要な値を保持する。
type LawSummaryValues struct {
	LawID                 string
	RevisionID            string
	Title                 string
	LawNumber             string
	PromulgationDate      *Date
	RevisionEffectiveDate *Date
	Source                LegalSource
}

// LawSummary は、一つの法令と選択されたリビジョンの概要を表す。
type LawSummary struct {
	lawID                 string
	revisionID            string
	title                 string
	lawNumber             string
	promulgationDate      *Date
	revisionEffectiveDate *Date
	source                LegalSource
}

// NewLawSummary は、入力を複製して検証済みの LawSummary を返す。
func NewLawSummary(values LawSummaryValues) (LawSummary, error) {
	summary := LawSummary{
		lawID:                 values.LawID,
		revisionID:            values.RevisionID,
		title:                 values.Title,
		lawNumber:             values.LawNumber,
		promulgationDate:      cloneOptionalDate(values.PromulgationDate),
		revisionEffectiveDate: cloneOptionalDate(values.RevisionEffectiveDate),
		source:                values.Source,
	}
	if err := summary.Validate(); err != nil {
		return LawSummary{}, err
	}
	return summary, nil
}

// LawID は、情報源が使用する法令識別子を返す。
func (s LawSummary) LawID() string {
	return s.lawID
}

// RevisionID は、選択された法令リビジョンの識別子を返す。
func (s LawSummary) RevisionID() string {
	return s.revisionID
}

// Title は、選択されたリビジョンの法令名を返す。
func (s LawSummary) Title() string {
	return s.title
}

// LawNumber は、法令番号と有無を返す。
func (s LawSummary) LawNumber() (string, bool) {
	return s.lawNumber, s.lawNumber != ""
}

// PromulgationDate は、法令の公布日と有無を返す。
func (s LawSummary) PromulgationDate() (Date, bool) {
	if s.promulgationDate == nil {
		return Date{}, false
	}
	return *s.promulgationDate, true
}

// RevisionEffectiveDate は、選択されたリビジョンの施行日と有無を返す。
func (s LawSummary) RevisionEffectiveDate() (Date, bool) {
	if s.revisionEffectiveDate == nil {
		return Date{}, false
	}
	return *s.revisionEffectiveDate, true
}

// Source は、情報を取得した法令情報源を返す。
func (s LawSummary) Source() LegalSource {
	return s.source
}

// Validate は、LawSummary の必須項目と設定された省略可能な日付を確認する。
func (s LawSummary) Validate() error {
	switch {
	case s.lawID == "":
		return fmt.Errorf("法令概要の lawId は必須です")
	case s.revisionID == "":
		return fmt.Errorf("法令概要の revisionId は必須です")
	case s.title == "":
		return fmt.Errorf("法令概要の title は必須です")
	}
	if s.promulgationDate != nil {
		if err := s.promulgationDate.Validate(); err != nil {
			return fmt.Errorf("promulgationDate が有効ではありません: %w", err)
		}
	}
	if s.revisionEffectiveDate != nil {
		if err := s.revisionEffectiveDate.Validate(); err != nil {
			return fmt.Errorf("revisionEffectiveDate が有効ではありません: %w", err)
		}
	}
	if err := s.source.Validate(); err != nil {
		return fmt.Errorf("source が有効ではありません: %w", err)
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-001 の項目名で法令概要を表す。
func (s LawSummary) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		LawID                 string      `json:"lawId"`
		RevisionID            string      `json:"revisionId"`
		Title                 string      `json:"title"`
		LawNumber             string      `json:"lawNumber,omitempty"`
		PromulgationDate      *Date       `json:"promulgationDate,omitempty"`
		RevisionEffectiveDate *Date       `json:"revisionEffectiveDate,omitempty"`
		Source                LegalSource `json:"source"`
	}{
		LawID:                 s.lawID,
		RevisionID:            s.revisionID,
		Title:                 s.title,
		LawNumber:             s.lawNumber,
		PromulgationDate:      cloneOptionalDate(s.promulgationDate),
		RevisionEffectiveDate: cloneOptionalDate(s.revisionEffectiveDate),
		Source:                s.source,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*LawSummary) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawSummary は JSON から直接復元できません。境界専用の入力型から NewLawSummary を使用してください",
	)
}

func cloneOptionalDate(value *Date) *Date {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
