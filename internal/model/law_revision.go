package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// LawRevisionValues は、LawRevision の作成に必要な値を保持する。
type LawRevisionValues struct {
	LawID                     string
	RevisionID                string
	Title                     string
	LawType                   string
	LawNumber                 string
	TitleKana                 string
	Abbreviation              string
	Category                  string
	PromulgationDate          *Date
	SourceUpdatedAt           string
	AmendmentPromulgationDate *Date
	EffectiveDate             *Date
	EffectiveDateNote         string
	ScheduledEffectiveDate    *Date
	AmendmentLawID            string
	AmendmentLawTitle         string
	AmendmentLawTitleKana     string
	AmendmentLawNumber        string
	RevisionKind              LawRevisionKind
	RepealStatus              LawRevisionRepealStatus
	RepealRecordedDate        *Date
	RemainInForce             *bool
	CurrentStatus             LawRevisionCurrentStatus
	DocumentURL               string
	Source                    LegalSource
}

// LawRevision は、一つの法令履歴 ID に対応する共通改正履歴である。
type LawRevision struct {
	lawID                     string
	revisionID                string
	title                     string
	lawType                   string
	lawNumber                 string
	titleKana                 string
	abbreviation              string
	category                  string
	promulgationDate          *Date
	sourceUpdatedAt           string
	amendmentPromulgationDate *Date
	effectiveDate             *Date
	effectiveDateNote         string
	scheduledEffectiveDate    *Date
	amendmentLawID            string
	amendmentLawTitle         string
	amendmentLawTitleKana     string
	amendmentLawNumber        string
	revisionKind              LawRevisionKind
	repealStatus              LawRevisionRepealStatus
	repealRecordedDate        *Date
	remainInForce             *bool
	currentStatus             LawRevisionCurrentStatus
	documentURL               string
	source                    LegalSource
}

// NewLawRevision は、入力を複製して検証済みの LawRevision を返す。
func NewLawRevision(values LawRevisionValues) (LawRevision, error) {
	revision := LawRevision{
		lawID:                     values.LawID,
		revisionID:                values.RevisionID,
		title:                     values.Title,
		lawType:                   values.LawType,
		lawNumber:                 values.LawNumber,
		titleKana:                 values.TitleKana,
		abbreviation:              values.Abbreviation,
		category:                  values.Category,
		promulgationDate:          cloneOptionalDate(values.PromulgationDate),
		sourceUpdatedAt:           values.SourceUpdatedAt,
		amendmentPromulgationDate: cloneOptionalDate(values.AmendmentPromulgationDate),
		effectiveDate:             cloneOptionalDate(values.EffectiveDate),
		effectiveDateNote:         values.EffectiveDateNote,
		scheduledEffectiveDate:    cloneOptionalDate(values.ScheduledEffectiveDate),
		amendmentLawID:            values.AmendmentLawID,
		amendmentLawTitle:         values.AmendmentLawTitle,
		amendmentLawTitleKana:     values.AmendmentLawTitleKana,
		amendmentLawNumber:        values.AmendmentLawNumber,
		revisionKind:              values.RevisionKind,
		repealStatus:              values.RepealStatus,
		repealRecordedDate:        cloneOptionalDate(values.RepealRecordedDate),
		remainInForce:             cloneOptionalBool(values.RemainInForce),
		currentStatus:             values.CurrentStatus,
		documentURL:               values.DocumentURL,
		source:                    values.Source,
	}
	if err := revision.Validate(); err != nil {
		return LawRevision{}, err
	}
	return revision, nil
}

func (r LawRevision) LawID() string                     { return r.lawID }
func (r LawRevision) RevisionID() string                { return r.revisionID }
func (r LawRevision) Title() string                     { return r.title }
func (r LawRevision) LawType() (string, bool)           { return optionalText(r.lawType) }
func (r LawRevision) LawNumber() (string, bool)         { return optionalText(r.lawNumber) }
func (r LawRevision) TitleKana() (string, bool)         { return optionalText(r.titleKana) }
func (r LawRevision) Abbreviation() (string, bool)      { return optionalText(r.abbreviation) }
func (r LawRevision) Category() (string, bool)          { return optionalText(r.category) }
func (r LawRevision) SourceUpdatedAt() (string, bool)   { return optionalText(r.sourceUpdatedAt) }
func (r LawRevision) EffectiveDateNote() (string, bool) { return optionalText(r.effectiveDateNote) }
func (r LawRevision) AmendmentLawID() (string, bool)    { return optionalText(r.amendmentLawID) }
func (r LawRevision) AmendmentLawTitle() (string, bool) { return optionalText(r.amendmentLawTitle) }
func (r LawRevision) AmendmentLawTitleKana() (string, bool) {
	return optionalText(r.amendmentLawTitleKana)
}
func (r LawRevision) AmendmentLawNumber() (string, bool) { return optionalText(r.amendmentLawNumber) }
func (r LawRevision) DocumentURL() (string, bool)        { return optionalText(r.documentURL) }
func (r LawRevision) Source() LegalSource                { return r.source }

func (r LawRevision) RevisionKind() (LawRevisionKind, bool) {
	return r.revisionKind, r.revisionKind != ""
}

func (r LawRevision) RepealStatus() (LawRevisionRepealStatus, bool) {
	return r.repealStatus, r.repealStatus != ""
}

func (r LawRevision) CurrentStatus() (LawRevisionCurrentStatus, bool) {
	return r.currentStatus, r.currentStatus != ""
}

func (r LawRevision) PromulgationDate() (Date, bool) {
	return optionalLawRevisionDate(r.promulgationDate)
}

func (r LawRevision) AmendmentPromulgationDate() (Date, bool) {
	return optionalLawRevisionDate(r.amendmentPromulgationDate)
}

func (r LawRevision) EffectiveDate() (Date, bool) {
	return optionalLawRevisionDate(r.effectiveDate)
}

func (r LawRevision) ScheduledEffectiveDate() (Date, bool) {
	return optionalLawRevisionDate(r.scheduledEffectiveDate)
}

func (r LawRevision) RepealRecordedDate() (Date, bool) {
	return optionalLawRevisionDate(r.repealRecordedDate)
}

func (r LawRevision) RemainInForce() (bool, bool) {
	if r.remainInForce == nil {
		return false, false
	}
	return *r.remainInForce, true
}

// Validate は、LawRevision の必須項目と省略可能値を確認する。
func (r LawRevision) Validate() error {
	switch {
	case r.lawID == "":
		return fmt.Errorf("法令改正履歴の lawId は必須です")
	case r.revisionID == "":
		return fmt.Errorf("法令改正履歴の revisionId は必須です")
	case r.title == "":
		return fmt.Errorf("法令改正履歴の title は必須です")
	}
	for name, value := range map[string]*Date{
		"promulgationDate":          r.promulgationDate,
		"amendmentPromulgationDate": r.amendmentPromulgationDate,
		"effectiveDate":             r.effectiveDate,
		"scheduledEffectiveDate":    r.scheduledEffectiveDate,
		"repealRecordedDate":        r.repealRecordedDate,
	} {
		if value != nil {
			if err := value.Validate(); err != nil {
				return fmt.Errorf("%s が有効ではありません: %w", name, err)
			}
		}
	}
	if r.sourceUpdatedAt != "" {
		if _, err := time.Parse(time.RFC3339, r.sourceUpdatedAt); err != nil {
			return fmt.Errorf("sourceUpdatedAt は RFC 3339 形式でなければなりません")
		}
	}
	if err := validateLawRevisionKind(r.revisionKind); err != nil {
		return err
	}
	if err := validateLawRevisionRepealStatus(r.repealStatus); err != nil {
		return err
	}
	if err := validateLawRevisionCurrentStatus(r.currentStatus); err != nil {
		return err
	}
	if r.documentURL != "" && !isHTTPSURL(r.documentURL) {
		return fmt.Errorf("documentUrl は認証情報を含まない HTTPS URL でなければなりません")
	}
	if err := r.source.Validate(); err != nil {
		return fmt.Errorf("source が有効ではありません: %w", err)
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-032 の項目名で法令改正履歴を表す。
func (r LawRevision) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		LawID                     string                   `json:"lawId"`
		RevisionID                string                   `json:"revisionId"`
		Title                     string                   `json:"title"`
		LawType                   string                   `json:"lawType,omitempty"`
		LawNumber                 string                   `json:"lawNumber,omitempty"`
		TitleKana                 string                   `json:"titleKana,omitempty"`
		Abbreviation              string                   `json:"abbreviation,omitempty"`
		Category                  string                   `json:"category,omitempty"`
		PromulgationDate          *Date                    `json:"promulgationDate,omitempty"`
		SourceUpdatedAt           string                   `json:"sourceUpdatedAt,omitempty"`
		AmendmentPromulgationDate *Date                    `json:"amendmentPromulgationDate,omitempty"`
		EffectiveDate             *Date                    `json:"effectiveDate,omitempty"`
		EffectiveDateNote         string                   `json:"effectiveDateNote,omitempty"`
		ScheduledEffectiveDate    *Date                    `json:"scheduledEffectiveDate,omitempty"`
		AmendmentLawID            string                   `json:"amendmentLawId,omitempty"`
		AmendmentLawTitle         string                   `json:"amendmentLawTitle,omitempty"`
		AmendmentLawTitleKana     string                   `json:"amendmentLawTitleKana,omitempty"`
		AmendmentLawNumber        string                   `json:"amendmentLawNumber,omitempty"`
		RevisionKind              LawRevisionKind          `json:"revisionKind,omitempty"`
		RepealStatus              LawRevisionRepealStatus  `json:"repealStatus,omitempty"`
		RepealRecordedDate        *Date                    `json:"repealRecordedDate,omitempty"`
		RemainInForce             *bool                    `json:"remainInForce,omitempty"`
		CurrentStatus             LawRevisionCurrentStatus `json:"currentStatus,omitempty"`
		DocumentURL               string                   `json:"documentUrl,omitempty"`
		Source                    LegalSource              `json:"source"`
	}{
		LawID: r.lawID, RevisionID: r.revisionID, Title: r.title,
		LawType: r.lawType, LawNumber: r.lawNumber, TitleKana: r.titleKana,
		Abbreviation: r.abbreviation, Category: r.category,
		PromulgationDate: cloneOptionalDate(r.promulgationDate), SourceUpdatedAt: r.sourceUpdatedAt,
		AmendmentPromulgationDate: cloneOptionalDate(r.amendmentPromulgationDate),
		EffectiveDate:             cloneOptionalDate(r.effectiveDate), EffectiveDateNote: r.effectiveDateNote,
		ScheduledEffectiveDate: cloneOptionalDate(r.scheduledEffectiveDate),
		AmendmentLawID:         r.amendmentLawID, AmendmentLawTitle: r.amendmentLawTitle,
		AmendmentLawTitleKana: r.amendmentLawTitleKana, AmendmentLawNumber: r.amendmentLawNumber,
		RevisionKind: r.revisionKind, RepealStatus: r.repealStatus,
		RepealRecordedDate: cloneOptionalDate(r.repealRecordedDate),
		RemainInForce:      cloneOptionalBool(r.remainInForce), CurrentStatus: r.currentStatus,
		DocumentURL: r.documentURL, Source: r.source,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*LawRevision) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("LawRevision は JSON から直接復元できません。NewLawRevision を使用してください")
}

func optionalText(value string) (string, bool) { return value, value != "" }

func optionalLawRevisionDate(value *Date) (Date, bool) {
	if value == nil {
		return Date{}, false
	}
	return *value, true
}
