package legalquery

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type legalQueryReadAttempt[T legalQueryResultData] struct {
	header legalQueryAttemptHeader
	item   model.SourcedResource[T]
}

func newLegalQueryReadAttempt[T legalQueryResultData](
	interpretationID string,
	step LegalQueryCandidateStep,
	item model.SourcedResource[T],
	expectedKind LogicalInputKind,
) (legalQueryReadAttempt[T], error) {
	header, err := newLegalQueryAttemptHeader(interpretationID, step)
	if err != nil {
		return legalQueryReadAttempt[T]{}, err
	}
	attempt := legalQueryReadAttempt[T]{
		header: header,
		item:   item,
	}
	if err := attempt.validate(expectedKind); err != nil {
		return legalQueryReadAttempt[T]{}, err
	}
	return attempt, nil
}

func (a legalQueryReadAttempt[T]) validate(
	expectedKind LogicalInputKind,
) error {
	if err := a.header.validate(expectedKind); err != nil {
		return err
	}
	if err := a.item.Validate(); err != nil {
		return fmt.Errorf("item が有効ではありません: %w", err)
	}
	return nil
}

// LegalQueryLawDocumentAttemptValues は、法令本文読取り attempt の作成値を保持する。
type LegalQueryLawDocumentAttemptValues struct {
	InterpretationID string
	Step             LegalQueryCandidateStep
	Item             model.SourcedResource[model.LawDocumentRepresentation]
}

// LegalQueryLawDocumentAttempt は、型付き法令本文を保持する。
type LegalQueryLawDocumentAttempt struct {
	data legalQueryReadAttempt[model.LawDocumentRepresentation]
}

// NewLegalQueryLawDocumentAttempt は、法令本文 step と結果を結び付ける。
func NewLegalQueryLawDocumentAttempt(
	values LegalQueryLawDocumentAttemptValues,
) (LegalQueryLawDocumentAttempt, error) {
	data, err := newLegalQueryReadAttempt(
		values.InterpretationID,
		values.Step,
		values.Item,
		InputKindLawRead,
	)
	if err != nil {
		return LegalQueryLawDocumentAttempt{}, err
	}
	attempt := LegalQueryLawDocumentAttempt{data: data}
	if err := attempt.Validate(); err != nil {
		return LegalQueryLawDocumentAttempt{}, err
	}
	return attempt, nil
}

func (a LegalQueryLawDocumentAttempt) InterpretationID() string {
	return a.data.header.interpretationID
}
func (a LegalQueryLawDocumentAttempt) StepID() string { return a.data.header.stepID }
func (a LegalQueryLawDocumentAttempt) CapabilityID() string {
	return a.data.header.capabilityID
}
func (a LegalQueryLawDocumentAttempt) CapabilityMajorVersion() int {
	return a.data.header.capabilityMajorVersion
}
func (LegalQueryLawDocumentAttempt) Outcome() LegalQueryAttemptOutcome {
	return LegalQueryAttemptOutcomeCompleted
}
func (LegalQueryLawDocumentAttempt) ResultKind() LegalQueryAttemptResultKind {
	return LegalQueryResultKindLawDocument
}
func (a LegalQueryLawDocumentAttempt) Item() model.SourcedResource[model.LawDocumentRepresentation] {
	return a.data.item
}
func (a LegalQueryLawDocumentAttempt) Validate() error {
	if err := a.data.validate(InputKindLawRead); err != nil {
		return err
	}
	return validateLawDocumentAttemptItem(a.data.header, a.data.item)
}
func (a LegalQueryLawDocumentAttempt) MarshalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return marshalSuccessfulAttempt(
		a.data.header,
		a.Outcome(),
		a.ResultKind(),
		struct {
			Item model.SourcedResource[model.LawDocumentRepresentation] `json:"item"`
		}{Item: a.data.item},
	)
}
func (*LegalQueryLawDocumentAttempt) UnmarshalJSON(_ []byte) error {
	return directAttemptRestoreError("LegalQueryLawDocumentAttempt")
}
func (LegalQueryLawDocumentAttempt) isLegalQueryAttempt() {}
func (a LegalQueryLawDocumentAttempt) cloneLegalQueryAttempt() LegalQueryAttempt {
	return a
}

// LegalQueryLawArticleAttemptValues は、条文読取り attempt の作成値を保持する。
type LegalQueryLawArticleAttemptValues struct {
	InterpretationID string
	Step             LegalQueryCandidateStep
	Item             model.SourcedResource[model.LawArticleFragment]
}

// LegalQueryLawArticleAttempt は、型付き条文 fragment を保持する。
type LegalQueryLawArticleAttempt struct {
	data legalQueryReadAttempt[model.LawArticleFragment]
}

// NewLegalQueryLawArticleAttempt は、条文 step と結果を結び付ける。
func NewLegalQueryLawArticleAttempt(
	values LegalQueryLawArticleAttemptValues,
) (LegalQueryLawArticleAttempt, error) {
	data, err := newLegalQueryReadAttempt(
		values.InterpretationID,
		values.Step,
		values.Item,
		InputKindLawArticleRead,
	)
	if err != nil {
		return LegalQueryLawArticleAttempt{}, err
	}
	attempt := LegalQueryLawArticleAttempt{data: data}
	if err := attempt.Validate(); err != nil {
		return LegalQueryLawArticleAttempt{}, err
	}
	return attempt, nil
}

func (a LegalQueryLawArticleAttempt) InterpretationID() string {
	return a.data.header.interpretationID
}
func (a LegalQueryLawArticleAttempt) StepID() string { return a.data.header.stepID }
func (a LegalQueryLawArticleAttempt) CapabilityID() string {
	return a.data.header.capabilityID
}
func (a LegalQueryLawArticleAttempt) CapabilityMajorVersion() int {
	return a.data.header.capabilityMajorVersion
}
func (LegalQueryLawArticleAttempt) Outcome() LegalQueryAttemptOutcome {
	return LegalQueryAttemptOutcomeCompleted
}
func (LegalQueryLawArticleAttempt) ResultKind() LegalQueryAttemptResultKind {
	return LegalQueryResultKindLawArticle
}
func (a LegalQueryLawArticleAttempt) Item() model.SourcedResource[model.LawArticleFragment] {
	return a.data.item
}
func (a LegalQueryLawArticleAttempt) Validate() error {
	if err := a.data.validate(InputKindLawArticleRead); err != nil {
		return err
	}
	return validateLawArticleAttemptItem(a.data.header, a.data.item)
}
func (a LegalQueryLawArticleAttempt) MarshalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return marshalSuccessfulAttempt(
		a.data.header,
		a.Outcome(),
		a.ResultKind(),
		struct {
			Item model.SourcedResource[model.LawArticleFragment] `json:"item"`
		}{Item: a.data.item},
	)
}
func (*LegalQueryLawArticleAttempt) UnmarshalJSON(_ []byte) error {
	return directAttemptRestoreError("LegalQueryLawArticleAttempt")
}
func (LegalQueryLawArticleAttempt) isLegalQueryAttempt() {}
func (a LegalQueryLawArticleAttempt) cloneLegalQueryAttempt() LegalQueryAttempt {
	return a
}

// LegalQueryJudicialDecisionAttemptValues は、裁判例読取り attempt の作成値を保持する。
type LegalQueryJudicialDecisionAttemptValues struct {
	InterpretationID string
	Step             LegalQueryCandidateStep
	Item             model.SourcedResource[model.JudicialDecisionDetails]
}

// LegalQueryJudicialDecisionAttempt は、型付き裁判例詳細と固定注意を保持する。
type LegalQueryJudicialDecisionAttempt struct {
	data legalQueryReadAttempt[model.JudicialDecisionDetails]
}

// NewLegalQueryJudicialDecisionAttempt は、裁判例 read step と結果を結び付ける。
func NewLegalQueryJudicialDecisionAttempt(
	values LegalQueryJudicialDecisionAttemptValues,
) (LegalQueryJudicialDecisionAttempt, error) {
	data, err := newLegalQueryReadAttempt(
		values.InterpretationID,
		values.Step,
		values.Item,
		InputKindJudicialDecisionRead,
	)
	if err != nil {
		return LegalQueryJudicialDecisionAttempt{}, err
	}
	attempt := LegalQueryJudicialDecisionAttempt{data: data}
	if err := attempt.Validate(); err != nil {
		return LegalQueryJudicialDecisionAttempt{}, err
	}
	return attempt, nil
}

func (a LegalQueryJudicialDecisionAttempt) InterpretationID() string {
	return a.data.header.interpretationID
}
func (a LegalQueryJudicialDecisionAttempt) StepID() string { return a.data.header.stepID }
func (a LegalQueryJudicialDecisionAttempt) CapabilityID() string {
	return a.data.header.capabilityID
}
func (a LegalQueryJudicialDecisionAttempt) CapabilityMajorVersion() int {
	return a.data.header.capabilityMajorVersion
}
func (LegalQueryJudicialDecisionAttempt) Outcome() LegalQueryAttemptOutcome {
	return LegalQueryAttemptOutcomeCompleted
}
func (LegalQueryJudicialDecisionAttempt) ResultKind() LegalQueryAttemptResultKind {
	return LegalQueryResultKindJudicialDecision
}
func (LegalQueryJudicialDecisionAttempt) CoverageNotice() string {
	return JudicialCasesCoverageNotice
}
func (a LegalQueryJudicialDecisionAttempt) Item() model.SourcedResource[model.JudicialDecisionDetails] {
	return a.data.item
}
func (a LegalQueryJudicialDecisionAttempt) Validate() error {
	if err := a.data.validate(InputKindJudicialDecisionRead); err != nil {
		return err
	}
	return validateJudicialDecisionAttemptItem(a.data.header, a.data.item)
}
func (a LegalQueryJudicialDecisionAttempt) MarshalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return marshalSuccessfulAttempt(
		a.data.header,
		a.Outcome(),
		a.ResultKind(),
		struct {
			CoverageNotice string                                               `json:"coverageNotice"`
			Item           model.SourcedResource[model.JudicialDecisionDetails] `json:"item"`
		}{
			CoverageNotice: JudicialCasesCoverageNotice,
			Item:           a.data.item,
		},
	)
}
func (*LegalQueryJudicialDecisionAttempt) UnmarshalJSON(_ []byte) error {
	return directAttemptRestoreError("LegalQueryJudicialDecisionAttempt")
}
func (LegalQueryJudicialDecisionAttempt) isLegalQueryAttempt() {}
func (a LegalQueryJudicialDecisionAttempt) cloneLegalQueryAttempt() LegalQueryAttempt {
	return a
}
