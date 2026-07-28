package legalquery

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
)

var _ JudicialCasesRequestMaterializer = JudicialCasesMaterializer{}

// MaterializeJudicialDecisionSearch は、裁判例検索の first-page request を作る。
func (m JudicialCasesMaterializer) MaterializeJudicialDecisionSearch(
	input JudicialDecisionSearchIntentV1,
	binding SelectedCapabilityBinding,
	budget LegalQueryStepBudget,
) (judicialdecisionsearch.Request, error) {
	if err := validateMaterializer(
		m.initialized,
		"JudicialCasesMaterializer",
	); err != nil {
		return judicialdecisionsearch.Request{}, err
	}
	if err := input.Validate(); err != nil {
		return judicialdecisionsearch.Request{},
			fmt.Errorf("judicial decision search logical input が有効ではありません: %w", err)
	}
	if _, err := snapshotCapabilityBinding(
		binding,
		judicialdecisionsearch.CapabilityID,
		judicialdecisionsearch.MajorVersion,
	); err != nil {
		return judicialdecisionsearch.Request{}, err
	}
	limit, err := materializerCollectionLimit(budget)
	if err != nil {
		return judicialdecisionsearch.Request{}, err
	}
	return judicialdecisionsearch.NewRequest(
		judicialdecisionsearch.RequestValues{
			Query: input.Query(),
			Limit: &limit,
		},
	)
}

// MaterializeJudicialDecisionRead は、入力 ref の exact read request を作る。
func (m JudicialCasesMaterializer) MaterializeJudicialDecisionRead(
	input JudicialDecisionReadIntentV1,
	binding SelectedCapabilityBinding,
	budget LegalQueryStepBudget,
) (judicialdecisionread.Request, error) {
	if err := validateMaterializer(
		m.initialized,
		"JudicialCasesMaterializer",
	); err != nil {
		return judicialdecisionread.Request{}, err
	}
	if err := input.Validate(); err != nil {
		return judicialdecisionread.Request{},
			fmt.Errorf("judicial decision read logical input が有効ではありません: %w", err)
	}
	selected, err := snapshotCapabilityBinding(
		binding,
		judicialdecisionread.CapabilityID,
		judicialdecisionread.MajorVersion,
	)
	if err != nil {
		return judicialdecisionread.Request{}, err
	}
	if err := validateMaterializerReadBudget(budget); err != nil {
		return judicialdecisionread.Request{}, err
	}
	ref := input.Ref()
	if err := validateMaterializerRef(
		ref,
		selected,
		"judicial-decision",
	); err != nil {
		return judicialdecisionread.Request{}, err
	}
	return judicialdecisionread.NewRequest(
		judicialdecisionread.RequestValues{Ref: ref},
	)
}
