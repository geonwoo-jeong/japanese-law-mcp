package model

import (
	"encoding/json"
	"fmt"
)

// JudicialDecisionDetailsValues は、JudicialDecisionDetails の作成に必要な値を保持する。
type JudicialDecisionDetailsValues struct {
	Summary                  JudicialDecisionSummary
	ReporterCitation         *string
	LowerCourtName           *string
	LowerCourtCaseNumber     *string
	LowerCourtDecisionDate   *Date
	HoldingText              *string
	SummaryText              *string
	ReferencedProvisionsText *string
}

// JudicialDecisionDetails は、裁判例概要と同じ公式詳細ページの固有情報を表す。
type JudicialDecisionDetails struct {
	summary                  JudicialDecisionSummary
	reporterCitation         *string
	lowerCourtName           *string
	lowerCourtCaseNumber     *string
	lowerCourtDecisionDate   *Date
	holdingText              *string
	summaryText              *string
	referencedProvisionsText *string
}

// NewJudicialDecisionDetails は、入力を複製して検証済みの裁判例詳細を返す。
func NewJudicialDecisionDetails(
	values JudicialDecisionDetailsValues,
) (JudicialDecisionDetails, error) {
	details := JudicialDecisionDetails{
		summary:                  values.Summary,
		reporterCitation:         cloneOptionalString(values.ReporterCitation),
		lowerCourtName:           cloneOptionalString(values.LowerCourtName),
		lowerCourtCaseNumber:     cloneOptionalString(values.LowerCourtCaseNumber),
		lowerCourtDecisionDate:   cloneOptionalDate(values.LowerCourtDecisionDate),
		holdingText:              cloneOptionalString(values.HoldingText),
		summaryText:              cloneOptionalString(values.SummaryText),
		referencedProvisionsText: cloneOptionalString(values.ReferencedProvisionsText),
	}
	if err := details.Validate(); err != nil {
		return JudicialDecisionDetails{}, err
	}
	return details, nil
}

// Summary は、検索結果と同じ意味を持つ裁判例概要を返す。
func (d JudicialDecisionDetails) Summary() JudicialDecisionSummary {
	return d.summary
}

// ReporterCitation は、判例集等の巻、号および頁と有無を返す。
func (d JudicialDecisionDetails) ReporterCitation() (string, bool) {
	return optionalStringValue(d.reporterCitation)
}

// LowerCourtName は、原審裁判所名と有無を返す。
func (d JudicialDecisionDetails) LowerCourtName() (string, bool) {
	return optionalStringValue(d.lowerCourtName)
}

// LowerCourtCaseNumber は、原審事件番号と有無を返す。
func (d JudicialDecisionDetails) LowerCourtCaseNumber() (string, bool) {
	return optionalStringValue(d.lowerCourtCaseNumber)
}

// LowerCourtDecisionDate は、原審裁判年月日と有無を返す。
func (d JudicialDecisionDetails) LowerCourtDecisionDate() (Date, bool) {
	if d.lowerCourtDecisionDate == nil {
		return Date{}, false
	}
	return *d.lowerCourtDecisionDate, true
}

// HoldingText は、詳細ページに文字として掲載された判示事項と有無を返す。
func (d JudicialDecisionDetails) HoldingText() (string, bool) {
	return optionalStringValue(d.holdingText)
}

// SummaryText は、詳細ページに文字として掲載された裁判要旨と有無を返す。
func (d JudicialDecisionDetails) SummaryText() (string, bool) {
	return optionalStringValue(d.summaryText)
}

// ReferencedProvisionsText は、詳細ページに掲載された参照法条の原文と有無を返す。
func (d JudicialDecisionDetails) ReferencedProvisionsText() (string, bool) {
	return optionalStringValue(d.referencedProvisionsText)
}

// Validate は、共通概要、省略可能な文字列および原審日付を確認する。
func (d JudicialDecisionDetails) Validate() error {
	if err := d.summary.Validate(); err != nil {
		return fmt.Errorf("summary が有効ではありません: %w", err)
	}
	if err := validateJudicialOptionalStrings(d.optionalStrings()); err != nil {
		return err
	}
	if d.lowerCourtDecisionDate != nil {
		if err := d.lowerCourtDecisionDate.Validate(); err != nil {
			return fmt.Errorf("lowerCourtDecisionDate が有効ではありません: %w", err)
		}
	}
	return nil
}

func (d JudicialDecisionDetails) optionalStrings() map[string]*string {
	return map[string]*string{
		"reporterCitation":         d.reporterCitation,
		"lowerCourtName":           d.lowerCourtName,
		"lowerCourtCaseNumber":     d.lowerCourtCaseNumber,
		"holdingText":              d.holdingText,
		"summaryText":              d.summaryText,
		"referencedProvisionsText": d.referencedProvisionsText,
	}
}

// MarshalJSON は、SOT-MODEL-021 の項目名で裁判例詳細を表す。
func (d JudicialDecisionDetails) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Summary                  JudicialDecisionSummary `json:"summary"`
		ReporterCitation         *string                 `json:"reporterCitation,omitempty"`
		LowerCourtName           *string                 `json:"lowerCourtName,omitempty"`
		LowerCourtCaseNumber     *string                 `json:"lowerCourtCaseNumber,omitempty"`
		LowerCourtDecisionDate   *Date                   `json:"lowerCourtDecisionDate,omitempty"`
		HoldingText              *string                 `json:"holdingText,omitempty"`
		SummaryText              *string                 `json:"summaryText,omitempty"`
		ReferencedProvisionsText *string                 `json:"referencedProvisionsText,omitempty"`
	}{
		Summary:                  d.summary,
		ReporterCitation:         cloneOptionalString(d.reporterCitation),
		LowerCourtName:           cloneOptionalString(d.lowerCourtName),
		LowerCourtCaseNumber:     cloneOptionalString(d.lowerCourtCaseNumber),
		LowerCourtDecisionDate:   cloneOptionalDate(d.lowerCourtDecisionDate),
		HoldingText:              cloneOptionalString(d.holdingText),
		SummaryText:              cloneOptionalString(d.summaryText),
		ReferencedProvisionsText: cloneOptionalString(d.referencedProvisionsText),
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*JudicialDecisionDetails) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialDecisionDetails は JSON から直接復元できません。" +
			"境界専用の入力型から NewJudicialDecisionDetails を使用してください",
	)
}
