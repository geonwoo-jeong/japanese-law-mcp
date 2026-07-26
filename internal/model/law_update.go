package model

import (
	"encoding/json"
	"fmt"
)

// LawUpdateValues は、LawUpdate の作成に必要な値を保持する。
type LawUpdateValues struct {
	UpdatedOn                 Date
	LawID                     string
	Title                     string
	LawType                   string
	LawNumber                 string
	TitleKana                 string
	PreviousTitle             string
	PromulgationDate          *Date
	AmendmentTitle            string
	AmendmentLawNumber        string
	AmendmentPromulgationDate *Date
	EffectiveDate             *Date
	EffectiveDateNote         string
	DocumentURL               string
	EnforcementPending        *bool
	AuthorityReviewPending    *bool
	Source                    LegalSource
}

// LawUpdate は、指定日に更新された一つの法令に関する情報を表す。
type LawUpdate struct {
	updatedOn                 Date
	lawID                     string
	title                     string
	lawType                   string
	lawNumber                 string
	titleKana                 string
	previousTitle             string
	promulgationDate          *Date
	amendmentTitle            string
	amendmentLawNumber        string
	amendmentPromulgationDate *Date
	effectiveDate             *Date
	effectiveDateNote         string
	documentURL               string
	enforcementPending        *bool
	authorityReviewPending    *bool
	source                    LegalSource
}

// NewLawUpdate は、入力を複製して検証済みの LawUpdate を返す。
func NewLawUpdate(values LawUpdateValues) (LawUpdate, error) {
	update := LawUpdate{
		updatedOn:                 values.UpdatedOn,
		lawID:                     values.LawID,
		title:                     values.Title,
		lawType:                   values.LawType,
		lawNumber:                 values.LawNumber,
		titleKana:                 values.TitleKana,
		previousTitle:             values.PreviousTitle,
		promulgationDate:          cloneOptionalDate(values.PromulgationDate),
		amendmentTitle:            values.AmendmentTitle,
		amendmentLawNumber:        values.AmendmentLawNumber,
		amendmentPromulgationDate: cloneOptionalDate(values.AmendmentPromulgationDate),
		effectiveDate:             cloneOptionalDate(values.EffectiveDate),
		effectiveDateNote:         values.EffectiveDateNote,
		documentURL:               values.DocumentURL,
		enforcementPending:        cloneOptionalBool(values.EnforcementPending),
		authorityReviewPending:    cloneOptionalBool(values.AuthorityReviewPending),
		source:                    values.Source,
	}
	if err := update.Validate(); err != nil {
		return LawUpdate{}, err
	}
	return update, nil
}

// UpdatedOn は、この更新一覧が対象とする日付を返す。
func (u LawUpdate) UpdatedOn() Date {
	return u.updatedOn
}

// LawID は、情報源が使用する法令識別子を返す。
func (u LawUpdate) LawID() string {
	return u.lawID
}

// Title は、法令名を返す。
func (u LawUpdate) Title() string {
	return u.title
}

// LawType は、法令種別と有無を返す。
func (u LawUpdate) LawType() (string, bool) {
	return u.lawType, u.lawType != ""
}

// LawNumber は、法令番号と有無を返す。
func (u LawUpdate) LawNumber() (string, bool) {
	return u.lawNumber, u.lawNumber != ""
}

// TitleKana は、法令名の読みと有無を返す。
func (u LawUpdate) TitleKana() (string, bool) {
	return u.titleKana, u.titleKana != ""
}

// PreviousTitle は、旧法令名と有無を返す。
func (u LawUpdate) PreviousTitle() (string, bool) {
	return u.previousTitle, u.previousTitle != ""
}

// PromulgationDate は、法令の公布日と有無を返す。
func (u LawUpdate) PromulgationDate() (Date, bool) {
	if u.promulgationDate == nil {
		return Date{}, false
	}
	return *u.promulgationDate, true
}

// AmendmentTitle は、改正法令名と有無を返す。
func (u LawUpdate) AmendmentTitle() (string, bool) {
	return u.amendmentTitle, u.amendmentTitle != ""
}

// AmendmentLawNumber は、改正法令番号と有無を返す。
func (u LawUpdate) AmendmentLawNumber() (string, bool) {
	return u.amendmentLawNumber, u.amendmentLawNumber != ""
}

// AmendmentPromulgationDate は、改正法令の公布日と有無を返す。
func (u LawUpdate) AmendmentPromulgationDate() (Date, bool) {
	if u.amendmentPromulgationDate == nil {
		return Date{}, false
	}
	return *u.amendmentPromulgationDate, true
}

// EffectiveDate は、施行日と有無を返す。
func (u LawUpdate) EffectiveDate() (Date, bool) {
	if u.effectiveDate == nil {
		return Date{}, false
	}
	return *u.effectiveDate, true
}

// EffectiveDateNote は、施行日に関する注記と有無を返す。
func (u LawUpdate) EffectiveDateNote() (string, bool) {
	return u.effectiveDateNote, u.effectiveDateNote != ""
}

// DocumentURL は、個別の法令を確認できる公式 URL と有無を返す。
func (u LawUpdate) DocumentURL() (string, bool) {
	return u.documentURL, u.documentURL != ""
}

// EnforcementPending は、未施行を示す値と有無を返す。
func (u LawUpdate) EnforcementPending() (bool, bool) {
	if u.enforcementPending == nil {
		return false, false
	}
	return *u.enforcementPending, true
}

// AuthorityReviewPending は、所管課確認中を示す値と有無を返す。
func (u LawUpdate) AuthorityReviewPending() (bool, bool) {
	if u.authorityReviewPending == nil {
		return false, false
	}
	return *u.authorityReviewPending, true
}

// Source は、更新情報を取得した法令情報源を返す。
func (u LawUpdate) Source() LegalSource {
	return u.source
}

// Validate は、LawUpdate の必須項目と省略可能な値を確認する。
func (u LawUpdate) Validate() error {
	if err := u.updatedOn.Validate(); err != nil {
		return fmt.Errorf("updatedOn が有効ではありません: %w", err)
	}
	switch {
	case u.lawID == "":
		return fmt.Errorf("法令更新情報の lawId は必須です")
	case u.title == "":
		return fmt.Errorf("法令更新情報の title は必須です")
	}
	for name, date := range map[string]*Date{
		"promulgationDate":          u.promulgationDate,
		"amendmentPromulgationDate": u.amendmentPromulgationDate,
		"effectiveDate":             u.effectiveDate,
	} {
		if date != nil {
			if err := date.Validate(); err != nil {
				return fmt.Errorf("%s が有効ではありません: %w", name, err)
			}
		}
	}
	if u.documentURL != "" && !isHTTPSURL(u.documentURL) {
		return fmt.Errorf("documentUrl は認証情報を含まない HTTPS URL でなければなりません")
	}
	if err := u.source.Validate(); err != nil {
		return fmt.Errorf("source が有効ではありません: %w", err)
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-019 の項目名で法令更新情報を表す。
func (u LawUpdate) MarshalJSON() ([]byte, error) {
	if err := u.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		UpdatedOn                 Date        `json:"updatedOn"`
		LawID                     string      `json:"lawId"`
		Title                     string      `json:"title"`
		LawType                   string      `json:"lawType,omitempty"`
		LawNumber                 string      `json:"lawNumber,omitempty"`
		TitleKana                 string      `json:"titleKana,omitempty"`
		PreviousTitle             string      `json:"previousTitle,omitempty"`
		PromulgationDate          *Date       `json:"promulgationDate,omitempty"`
		AmendmentTitle            string      `json:"amendmentTitle,omitempty"`
		AmendmentLawNumber        string      `json:"amendmentLawNumber,omitempty"`
		AmendmentPromulgationDate *Date       `json:"amendmentPromulgationDate,omitempty"`
		EffectiveDate             *Date       `json:"effectiveDate,omitempty"`
		EffectiveDateNote         string      `json:"effectiveDateNote,omitempty"`
		DocumentURL               string      `json:"documentUrl,omitempty"`
		EnforcementPending        *bool       `json:"enforcementPending,omitempty"`
		AuthorityReviewPending    *bool       `json:"authorityReviewPending,omitempty"`
		Source                    LegalSource `json:"source"`
	}{
		UpdatedOn:                 u.updatedOn,
		LawID:                     u.lawID,
		Title:                     u.title,
		LawType:                   u.lawType,
		LawNumber:                 u.lawNumber,
		TitleKana:                 u.titleKana,
		PreviousTitle:             u.previousTitle,
		PromulgationDate:          cloneOptionalDate(u.promulgationDate),
		AmendmentTitle:            u.amendmentTitle,
		AmendmentLawNumber:        u.amendmentLawNumber,
		AmendmentPromulgationDate: cloneOptionalDate(u.amendmentPromulgationDate),
		EffectiveDate:             cloneOptionalDate(u.effectiveDate),
		EffectiveDateNote:         u.effectiveDateNote,
		DocumentURL:               u.documentURL,
		EnforcementPending:        cloneOptionalBool(u.enforcementPending),
		AuthorityReviewPending:    cloneOptionalBool(u.authorityReviewPending),
		Source:                    u.source,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*LawUpdate) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawUpdate は JSON から直接復元できません。境界専用の入力型から NewLawUpdate を使用してください",
	)
}

func cloneOptionalBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
