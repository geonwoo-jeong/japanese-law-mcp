package legalquery

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type legalQueryResultData interface {
	Validate() error
}

type legalQueryCollectionAttempt[T legalQueryResultData] struct {
	header legalQueryAttemptHeader
	page   LegalQueryPagePreview
	items  []model.SourcedResource[T]
}

func newLegalQueryCollectionAttempt[T legalQueryResultData](
	interpretationID string,
	step LegalQueryCandidateStep,
	page LegalQueryPagePreview,
	items []model.SourcedResource[T],
	expectedKind LogicalInputKind,
) (legalQueryCollectionAttempt[T], error) {
	header, err := newLegalQueryAttemptHeader(interpretationID, step)
	if err != nil {
		return legalQueryCollectionAttempt[T]{}, err
	}
	attempt := legalQueryCollectionAttempt[T]{
		header: header,
		page:   page,
		items:  cloneSourcedResources(items),
	}
	if err := attempt.validate(expectedKind); err != nil {
		return legalQueryCollectionAttempt[T]{}, err
	}
	return attempt, nil
}

func (a legalQueryCollectionAttempt[T]) validate(
	expectedKind LogicalInputKind,
) error {
	if err := a.header.validate(expectedKind); err != nil {
		return err
	}
	if err := a.page.Validate(); err != nil {
		return fmt.Errorf("page が有効ではありません: %w", err)
	}
	if a.page.ReturnedCount() != len(a.items) {
		return fmt.Errorf("page.returnedCount と items の件数が一致しません")
	}
	for index, item := range a.items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("items[%d] が有効ではありません: %w", index, err)
		}
	}
	return nil
}

func (a legalQueryCollectionAttempt[T]) outcome() LegalQueryAttemptOutcome {
	if len(a.items) == 0 {
		return LegalQueryAttemptOutcomeEmpty
	}
	return LegalQueryAttemptOutcomeCompleted
}

func (a legalQueryCollectionAttempt[T]) clone() legalQueryCollectionAttempt[T] {
	cloned := a
	cloned.items = cloneSourcedResources(a.items)
	return cloned
}

func cloneSourcedResources[T legalQueryResultData](
	values []model.SourcedResource[T],
) []model.SourcedResource[T] {
	cloned := make([]model.SourcedResource[T], len(values))
	copy(cloned, values)
	return cloned
}

type legalQueryCollectionResult[T legalQueryResultData] struct {
	Page  LegalQueryPagePreview      `json:"page"`
	Items []model.SourcedResource[T] `json:"items"`
}

// LegalQueryLawSearchAttemptValues は、法令検索 attempt の作成値を保持する。
type LegalQueryLawSearchAttemptValues struct {
	InterpretationID string
	Step             LegalQueryCandidateStep
	Page             LegalQueryPagePreview
	Items            []model.SourcedResource[model.LawSummary]
}

// LegalQueryLawSearchAttempt は、型付き法令検索結果を保持する。
type LegalQueryLawSearchAttempt struct {
	data legalQueryCollectionAttempt[model.LawSummary]
}

// NewLegalQueryLawSearchAttempt は、法令検索 step と結果を結び付ける。
func NewLegalQueryLawSearchAttempt(
	values LegalQueryLawSearchAttemptValues,
) (LegalQueryLawSearchAttempt, error) {
	data, err := newLegalQueryCollectionAttempt(
		values.InterpretationID,
		values.Step,
		values.Page,
		values.Items,
		InputKindLawSearch,
	)
	if err != nil {
		return LegalQueryLawSearchAttempt{}, err
	}
	attempt := LegalQueryLawSearchAttempt{data: data}
	if err := attempt.Validate(); err != nil {
		return LegalQueryLawSearchAttempt{}, err
	}
	return attempt, nil
}

func (a LegalQueryLawSearchAttempt) InterpretationID() string {
	return a.data.header.interpretationID
}
func (a LegalQueryLawSearchAttempt) StepID() string { return a.data.header.stepID }
func (a LegalQueryLawSearchAttempt) CapabilityID() string {
	return a.data.header.capabilityID
}
func (a LegalQueryLawSearchAttempt) CapabilityMajorVersion() int {
	return a.data.header.capabilityMajorVersion
}
func (a LegalQueryLawSearchAttempt) Outcome() LegalQueryAttemptOutcome {
	return a.data.outcome()
}
func (a LegalQueryLawSearchAttempt) ResultKind() LegalQueryAttemptResultKind {
	return LegalQueryResultKindLawSearch
}
func (a LegalQueryLawSearchAttempt) Page() LegalQueryPagePreview { return a.data.page }
func (a LegalQueryLawSearchAttempt) Items() []model.SourcedResource[model.LawSummary] {
	return cloneSourcedResources(a.data.items)
}
func (a LegalQueryLawSearchAttempt) Validate() error {
	if err := a.data.validate(InputKindLawSearch); err != nil {
		return err
	}
	return validateLawSearchAttemptItems(a.data.items)
}
func (a LegalQueryLawSearchAttempt) MarshalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return marshalSuccessfulAttempt(
		a.data.header,
		a.Outcome(),
		a.ResultKind(),
		legalQueryCollectionResult[model.LawSummary]{
			Page: a.data.page, Items: a.Items(),
		},
	)
}
func (*LegalQueryLawSearchAttempt) UnmarshalJSON(_ []byte) error {
	return directAttemptRestoreError("LegalQueryLawSearchAttempt")
}
func (LegalQueryLawSearchAttempt) isLegalQueryAttempt() {}
func (a LegalQueryLawSearchAttempt) cloneLegalQueryAttempt() LegalQueryAttempt {
	return LegalQueryLawSearchAttempt{data: a.data.clone()}
}

// LegalQueryLawContentSearchAttemptValues は、法令本文検索 attempt の作成値を保持する。
type LegalQueryLawContentSearchAttemptValues struct {
	InterpretationID string
	Step             LegalQueryCandidateStep
	Page             LegalQueryPagePreview
	Items            []model.SourcedResource[model.LawContentMatch]
}

// LegalQueryLawContentSearchAttempt は、型付き法令本文検索結果を保持する。
type LegalQueryLawContentSearchAttempt struct {
	data legalQueryCollectionAttempt[model.LawContentMatch]
}

// NewLegalQueryLawContentSearchAttempt は、法令本文検索 step と結果を結び付ける。
func NewLegalQueryLawContentSearchAttempt(
	values LegalQueryLawContentSearchAttemptValues,
) (LegalQueryLawContentSearchAttempt, error) {
	data, err := newLegalQueryCollectionAttempt(
		values.InterpretationID,
		values.Step,
		values.Page,
		values.Items,
		InputKindLawContentSearch,
	)
	if err != nil {
		return LegalQueryLawContentSearchAttempt{}, err
	}
	attempt := LegalQueryLawContentSearchAttempt{data: data}
	if err := attempt.Validate(); err != nil {
		return LegalQueryLawContentSearchAttempt{}, err
	}
	return attempt, nil
}

func (a LegalQueryLawContentSearchAttempt) InterpretationID() string {
	return a.data.header.interpretationID
}
func (a LegalQueryLawContentSearchAttempt) StepID() string { return a.data.header.stepID }
func (a LegalQueryLawContentSearchAttempt) CapabilityID() string {
	return a.data.header.capabilityID
}
func (a LegalQueryLawContentSearchAttempt) CapabilityMajorVersion() int {
	return a.data.header.capabilityMajorVersion
}
func (a LegalQueryLawContentSearchAttempt) Outcome() LegalQueryAttemptOutcome {
	return a.data.outcome()
}
func (a LegalQueryLawContentSearchAttempt) ResultKind() LegalQueryAttemptResultKind {
	return LegalQueryResultKindLawContentSearch
}
func (a LegalQueryLawContentSearchAttempt) Page() LegalQueryPagePreview {
	return a.data.page
}
func (a LegalQueryLawContentSearchAttempt) Items() []model.SourcedResource[model.LawContentMatch] {
	return cloneSourcedResources(a.data.items)
}
func (a LegalQueryLawContentSearchAttempt) Validate() error {
	if err := a.data.validate(InputKindLawContentSearch); err != nil {
		return err
	}
	return validateLawContentAttemptItems(a.data.items)
}
func (a LegalQueryLawContentSearchAttempt) MarshalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return marshalSuccessfulAttempt(
		a.data.header,
		a.Outcome(),
		a.ResultKind(),
		legalQueryCollectionResult[model.LawContentMatch]{
			Page: a.data.page, Items: a.Items(),
		},
	)
}
func (*LegalQueryLawContentSearchAttempt) UnmarshalJSON(_ []byte) error {
	return directAttemptRestoreError("LegalQueryLawContentSearchAttempt")
}
func (LegalQueryLawContentSearchAttempt) isLegalQueryAttempt() {}
func (a LegalQueryLawContentSearchAttempt) cloneLegalQueryAttempt() LegalQueryAttempt {
	return LegalQueryLawContentSearchAttempt{data: a.data.clone()}
}

// LegalQueryLawUpdatesAttemptValues は、法令更新一覧 attempt の作成値を保持する。
type LegalQueryLawUpdatesAttemptValues struct {
	InterpretationID string
	Step             LegalQueryCandidateStep
	Page             LegalQueryPagePreview
	Items            []model.SourcedResource[model.LawUpdate]
}

// LegalQueryLawUpdatesAttempt は、型付き法令更新一覧を保持する。
type LegalQueryLawUpdatesAttempt struct {
	data legalQueryCollectionAttempt[model.LawUpdate]
}

// NewLegalQueryLawUpdatesAttempt は、更新一覧 step と結果を結び付ける。
func NewLegalQueryLawUpdatesAttempt(
	values LegalQueryLawUpdatesAttemptValues,
) (LegalQueryLawUpdatesAttempt, error) {
	data, err := newLegalQueryCollectionAttempt(
		values.InterpretationID,
		values.Step,
		values.Page,
		values.Items,
		InputKindLawUpdates,
	)
	if err != nil {
		return LegalQueryLawUpdatesAttempt{}, err
	}
	attempt := LegalQueryLawUpdatesAttempt{data: data}
	if err := attempt.Validate(); err != nil {
		return LegalQueryLawUpdatesAttempt{}, err
	}
	return attempt, nil
}

func (a LegalQueryLawUpdatesAttempt) InterpretationID() string {
	return a.data.header.interpretationID
}
func (a LegalQueryLawUpdatesAttempt) StepID() string { return a.data.header.stepID }
func (a LegalQueryLawUpdatesAttempt) CapabilityID() string {
	return a.data.header.capabilityID
}
func (a LegalQueryLawUpdatesAttempt) CapabilityMajorVersion() int {
	return a.data.header.capabilityMajorVersion
}
func (a LegalQueryLawUpdatesAttempt) Outcome() LegalQueryAttemptOutcome {
	return a.data.outcome()
}
func (a LegalQueryLawUpdatesAttempt) ResultKind() LegalQueryAttemptResultKind {
	return LegalQueryResultKindLawUpdates
}
func (a LegalQueryLawUpdatesAttempt) Page() LegalQueryPagePreview { return a.data.page }
func (a LegalQueryLawUpdatesAttempt) Items() []model.SourcedResource[model.LawUpdate] {
	return cloneSourcedResources(a.data.items)
}
func (a LegalQueryLawUpdatesAttempt) Validate() error {
	if err := a.data.validate(InputKindLawUpdates); err != nil {
		return err
	}
	return validateLawUpdatesAttemptItems(a.data.header, a.data.items)
}
func (a LegalQueryLawUpdatesAttempt) MarshalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return marshalSuccessfulAttempt(
		a.data.header,
		a.Outcome(),
		a.ResultKind(),
		legalQueryCollectionResult[model.LawUpdate]{
			Page: a.data.page, Items: a.Items(),
		},
	)
}
func (*LegalQueryLawUpdatesAttempt) UnmarshalJSON(_ []byte) error {
	return directAttemptRestoreError("LegalQueryLawUpdatesAttempt")
}
func (LegalQueryLawUpdatesAttempt) isLegalQueryAttempt() {}
func (a LegalQueryLawUpdatesAttempt) cloneLegalQueryAttempt() LegalQueryAttempt {
	return LegalQueryLawUpdatesAttempt{data: a.data.clone()}
}

// LegalQueryJudicialSearchAttemptValues は、裁判例検索 attempt の作成値を保持する。
type LegalQueryJudicialSearchAttemptValues struct {
	InterpretationID string
	Step             LegalQueryCandidateStep
	Page             LegalQueryPagePreview
	Items            []model.SourcedResource[model.JudicialDecisionSummary]
}

// LegalQueryJudicialSearchAttempt は、型付き裁判例検索結果と固定注意を保持する。
type LegalQueryJudicialSearchAttempt struct {
	data legalQueryCollectionAttempt[model.JudicialDecisionSummary]
}

// NewLegalQueryJudicialSearchAttempt は、裁判例検索 step と結果を結び付ける。
func NewLegalQueryJudicialSearchAttempt(
	values LegalQueryJudicialSearchAttemptValues,
) (LegalQueryJudicialSearchAttempt, error) {
	data, err := newLegalQueryCollectionAttempt(
		values.InterpretationID,
		values.Step,
		values.Page,
		values.Items,
		InputKindJudicialDecisionSearch,
	)
	if err != nil {
		return LegalQueryJudicialSearchAttempt{}, err
	}
	attempt := LegalQueryJudicialSearchAttempt{data: data}
	if err := attempt.Validate(); err != nil {
		return LegalQueryJudicialSearchAttempt{}, err
	}
	return attempt, nil
}

func (a LegalQueryJudicialSearchAttempt) InterpretationID() string {
	return a.data.header.interpretationID
}
func (a LegalQueryJudicialSearchAttempt) StepID() string { return a.data.header.stepID }
func (a LegalQueryJudicialSearchAttempt) CapabilityID() string {
	return a.data.header.capabilityID
}
func (a LegalQueryJudicialSearchAttempt) CapabilityMajorVersion() int {
	return a.data.header.capabilityMajorVersion
}
func (a LegalQueryJudicialSearchAttempt) Outcome() LegalQueryAttemptOutcome {
	return a.data.outcome()
}
func (a LegalQueryJudicialSearchAttempt) ResultKind() LegalQueryAttemptResultKind {
	return LegalQueryResultKindJudicialSearch
}
func (a LegalQueryJudicialSearchAttempt) CoverageNotice() string {
	return JudicialCasesCoverageNotice
}
func (a LegalQueryJudicialSearchAttempt) Page() LegalQueryPagePreview { return a.data.page }
func (a LegalQueryJudicialSearchAttempt) Items() []model.SourcedResource[model.JudicialDecisionSummary] {
	return cloneSourcedResources(a.data.items)
}
func (a LegalQueryJudicialSearchAttempt) Validate() error {
	if err := a.data.validate(InputKindJudicialDecisionSearch); err != nil {
		return err
	}
	return validateJudicialSearchAttemptItems(a.data.items)
}
func (a LegalQueryJudicialSearchAttempt) MarshalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return marshalSuccessfulAttempt(
		a.data.header,
		a.Outcome(),
		a.ResultKind(),
		struct {
			CoverageNotice string                                                 `json:"coverageNotice"`
			Page           LegalQueryPagePreview                                  `json:"page"`
			Items          []model.SourcedResource[model.JudicialDecisionSummary] `json:"items"`
		}{
			CoverageNotice: a.CoverageNotice(),
			Page:           a.data.page,
			Items:          a.Items(),
		},
	)
}
func (*LegalQueryJudicialSearchAttempt) UnmarshalJSON(_ []byte) error {
	return directAttemptRestoreError("LegalQueryJudicialSearchAttempt")
}
func (LegalQueryJudicialSearchAttempt) isLegalQueryAttempt() {}
func (a LegalQueryJudicialSearchAttempt) cloneLegalQueryAttempt() LegalQueryAttempt {
	return LegalQueryJudicialSearchAttempt{data: a.data.clone()}
}

func directAttemptRestoreError(typeName string) error {
	return fmt.Errorf(
		"%s は JSON から直接復元できません。対応する constructor を使用してください",
		typeName,
	)
}
