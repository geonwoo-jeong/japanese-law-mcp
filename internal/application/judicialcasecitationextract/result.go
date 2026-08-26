package judicialcasecitationextract

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	maximumOccurrenceCount   = 256
	maximumExaminedPageCount = 300
)

// DocumentTextStatus は、PDF text layer の可用性を表す。
type DocumentTextStatus string

const (
	DocumentTextStatusAvailable               DocumentTextStatus = "available"
	DocumentTextStatusDocumentTextUnavailable DocumentTextStatus = "document_text_unavailable"
)

// ResultValues は、判例引用抽出結果の作成値を保持する。
type ResultValues struct {
	ConfirmedDecisionMentions []model.JudicialCitationDecisionMention
	UnresolvedMentions        []model.JudicialCitationUnresolvedMention
	DocumentTextStatus        DocumentTextStatus
	ExaminedPageCount         int
	OccurrenceCount           int
	Truncated                 bool
}

// Result は、一度の PDF 抽出 capability が返す出現集合を保持する。
type Result struct {
	confirmedDecisionMentions []model.JudicialCitationDecisionMention
	unresolvedMentions        []model.JudicialCitationUnresolvedMention
	documentTextStatus        DocumentTextStatus
	examinedPageCount         int
	occurrenceCount           int
	truncated                 bool
	initialized               bool
}

// NewResult は、出現数、text layer 状態および配列整合を検証した Result を返す。
func NewResult(values ResultValues) (Result, error) {
	if values.ConfirmedDecisionMentions == nil || values.UnresolvedMentions == nil {
		return Result{}, fmt.Errorf("mention 配列は空配列または値を持つ配列でなければなりません")
	}
	result := Result{
		confirmedDecisionMentions: slices.Clone(values.ConfirmedDecisionMentions),
		unresolvedMentions:        slices.Clone(values.UnresolvedMentions),
		documentTextStatus:        values.DocumentTextStatus,
		examinedPageCount:         values.ExaminedPageCount,
		occurrenceCount:           values.OccurrenceCount,
		truncated:                 values.Truncated,
		initialized:               true,
	}
	if err := result.Validate(); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (r Result) ConfirmedDecisionMentions() []model.JudicialCitationDecisionMention {
	return slices.Clone(r.confirmedDecisionMentions)
}

func (r Result) UnresolvedMentions() []model.JudicialCitationUnresolvedMention {
	return slices.Clone(r.unresolvedMentions)
}

func (r Result) DocumentTextStatus() DocumentTextStatus { return r.documentTextStatus }
func (r Result) ExaminedPageCount() int                 { return r.examinedPageCount }
func (r Result) OccurrenceCount() int                   { return r.occurrenceCount }
func (r Result) Truncated() bool                        { return r.truncated }

// Validate は、出現配列、件数および text layer 縮退を確認する。
func (r Result) Validate() error {
	if !r.initialized {
		return fmt.Errorf("Result は NewResult で作成しなければなりません")
	}
	if r.confirmedDecisionMentions == nil || r.unresolvedMentions == nil {
		return fmt.Errorf("mention 配列は空配列または値を持つ配列でなければなりません")
	}
	switch r.documentTextStatus {
	case DocumentTextStatusAvailable, DocumentTextStatusDocumentTextUnavailable:
	default:
		return fmt.Errorf("documentTextStatus が定義されていません")
	}
	if r.examinedPageCount < 0 || r.examinedPageCount > maximumExaminedPageCount {
		return fmt.Errorf(
			"examinedPageCount は 0 以上 %d 以下でなければなりません",
			maximumExaminedPageCount,
		)
	}
	total := len(r.confirmedDecisionMentions) + len(r.unresolvedMentions)
	if r.occurrenceCount != total {
		return fmt.Errorf("occurrenceCount は返却配列の合計と一致しなければなりません")
	}
	if r.occurrenceCount > maximumOccurrenceCount {
		return fmt.Errorf("occurrenceCount は %d 件以下でなければなりません", maximumOccurrenceCount)
	}
	if r.truncated && r.occurrenceCount != maximumOccurrenceCount {
		return fmt.Errorf(
			"truncated は occurrenceCount が %d 件の場合だけ true にできます",
			maximumOccurrenceCount,
		)
	}
	if r.documentTextStatus == DocumentTextStatusDocumentTextUnavailable {
		if total != 0 || r.truncated {
			return fmt.Errorf("document_text_unavailable では mention 配列を空にし、truncated を false にしなければなりません")
		}
	}
	for index, mention := range r.confirmedDecisionMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("confirmedDecisionMentions[%d] が有効ではありません: %w", index, err)
		}
		if mention.Evidence().EvidenceLevel() != model.JudicialCitationEvidenceLevelExactTextMatch {
			return fmt.Errorf("confirmedDecisionMentions[%d].evidence は exact_text_match でなければなりません", index)
		}
	}
	for index, mention := range r.unresolvedMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("unresolvedMentions[%d] が有効ではありません: %w", index, err)
		}
		if mention.MentionType() != model.JudicialCitationMentionTypeDecision {
			return fmt.Errorf("unresolvedMentions[%d] は judicial_decision でなければなりません", index)
		}
	}
	return nil
}

func (r Result) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ConfirmedDecisionMentions []model.JudicialCitationDecisionMention   `json:"confirmedDecisionMentions"`
		UnresolvedMentions        []model.JudicialCitationUnresolvedMention `json:"unresolvedMentions"`
		DocumentTextStatus        DocumentTextStatus                        `json:"documentTextStatus"`
		ExaminedPageCount         int                                       `json:"examinedPageCount"`
		OccurrenceCount           int                                       `json:"occurrenceCount"`
		Truncated                 bool                                      `json:"truncated"`
	}{
		ConfirmedDecisionMentions: slices.Clone(r.confirmedDecisionMentions),
		UnresolvedMentions:        slices.Clone(r.unresolvedMentions),
		DocumentTextStatus:        r.documentTextStatus,
		ExaminedPageCount:         r.examinedPageCount,
		OccurrenceCount:           r.occurrenceCount,
		Truncated:                 r.truncated,
	})
}

func (*Result) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("Result は JSON から直接復元できません。NewResult を使用してください")
}
