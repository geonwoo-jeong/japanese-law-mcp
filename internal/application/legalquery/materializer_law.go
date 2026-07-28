package legalquery

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

var _ CoreRequestMaterializer = CoreMaterializer{}

// MaterializeLawSearch は、法令名検索の first-page request を作る。
func (m CoreMaterializer) MaterializeLawSearch(
	input LawSearchIntentV1,
	binding SelectedCapabilityBinding,
	budget LegalQueryStepBudget,
) (lawsearch.Request, error) {
	if err := validateMaterializer(m.initialized, "CoreMaterializer"); err != nil {
		return lawsearch.Request{}, err
	}
	if err := input.Validate(); err != nil {
		return lawsearch.Request{}, fmt.Errorf("law search logical input が有効ではありません: %w", err)
	}
	if _, err := snapshotCapabilityBinding(
		binding,
		lawsearch.CapabilityID,
		lawsearch.MajorVersion,
	); err != nil {
		return lawsearch.Request{}, err
	}
	limit, err := materializerCollectionLimit(budget)
	if err != nil {
		return lawsearch.Request{}, err
	}
	return lawsearch.NewRequest(lawsearch.RequestValues{
		Query: input.Query(),
		AsOf:  materializerDatePointer(input.AsOf),
		Limit: &limit,
	})
}

// MaterializeLawContentSearch は、構造化した法令本文検索 request を作る。
func (m CoreMaterializer) MaterializeLawContentSearch(
	input LawContentSearchIntentV1,
	binding SelectedCapabilityBinding,
	budget LegalQueryStepBudget,
) (lawcontentsearch.Request, error) {
	if err := validateMaterializer(m.initialized, "CoreMaterializer"); err != nil {
		return lawcontentsearch.Request{}, err
	}
	if err := input.Validate(); err != nil {
		return lawcontentsearch.Request{},
			fmt.Errorf("law content search logical input が有効ではありません: %w", err)
	}
	if _, err := snapshotCapabilityBinding(
		binding,
		lawcontentsearch.CapabilityID,
		lawcontentsearch.MajorVersion,
	); err != nil {
		return lawcontentsearch.Request{}, err
	}
	limit, err := materializerCollectionLimit(budget)
	if err != nil {
		return lawcontentsearch.Request{}, err
	}
	return lawcontentsearch.NewRequest(lawcontentsearch.RequestValues{
		AllTerms:     input.AllTerms(),
		AnyTerms:     input.AnyTerms(),
		ExcludeTerms: input.ExcludeTerms(),
		AsOf:         materializerDatePointer(input.AsOf),
		Limit:        &limit,
	})
}

// MaterializeLawDocumentRead は、法令本文の exact read request を作る。
func (m CoreMaterializer) MaterializeLawDocumentRead(
	input LawReadIntentV1,
	binding SelectedCapabilityBinding,
	budget LegalQueryStepBudget,
) (lawdocumentread.Request, error) {
	if err := validateMaterializer(m.initialized, "CoreMaterializer"); err != nil {
		return lawdocumentread.Request{}, err
	}
	if err := input.Validate(); err != nil {
		return lawdocumentread.Request{},
			fmt.Errorf("law read logical input が有効ではありません: %w", err)
	}
	selected, err := snapshotCapabilityBinding(
		binding,
		lawdocumentread.CapabilityID,
		lawdocumentread.MajorVersion,
	)
	if err != nil {
		return lawdocumentread.Request{}, err
	}
	if err := validateMaterializerReadBudget(budget); err != nil {
		return lawdocumentread.Request{}, err
	}
	ref, asOf, err := materializeLawDocumentTarget(input, selected)
	if err != nil {
		return lawdocumentread.Request{}, err
	}
	return lawdocumentread.NewRequest(lawdocumentread.RequestValues{
		Resource: ref,
		AsOf:     asOf,
	})
}

func materializeLawDocumentTarget(
	input LawReadIntentV1,
	binding selectedCapabilityBindingSnapshot,
) (model.SourceResourceRef, *model.Date, error) {
	if ref, exists := input.Ref(); exists {
		if err := validateMaterializerRef(ref, binding, "law"); err != nil {
			return model.SourceResourceRef{}, nil, err
		}
		return ref, nil, nil
	}
	lawID, exists := input.LawID()
	if !exists {
		return model.SourceResourceRef{}, nil,
			fmt.Errorf("law read logical input に lawId または ref がありません")
	}
	revisionID, _ := input.RevisionID()
	ref, err := newMaterializedLawRef(binding, lawID, revisionID)
	if err != nil {
		return model.SourceResourceRef{}, nil, err
	}
	return ref, materializerDatePointer(input.AsOf), nil
}

// MaterializeLawArticleRead は、法令条文の exact read request を作る。
func (m CoreMaterializer) MaterializeLawArticleRead(
	input LawArticleReadIntentV1,
	binding SelectedCapabilityBinding,
	budget LegalQueryStepBudget,
) (lawarticleread.Request, error) {
	if err := validateMaterializer(m.initialized, "CoreMaterializer"); err != nil {
		return lawarticleread.Request{}, err
	}
	if err := input.Validate(); err != nil {
		return lawarticleread.Request{},
			fmt.Errorf("law article read logical input が有効ではありません: %w", err)
	}
	selected, err := snapshotCapabilityBinding(
		binding,
		lawarticleread.CapabilityID,
		lawarticleread.MajorVersion,
	)
	if err != nil {
		return lawarticleread.Request{}, err
	}
	if err := validateMaterializerReadBudget(budget); err != nil {
		return lawarticleread.Request{}, err
	}
	ref, err := materializeLawArticleTarget(input, selected)
	if err != nil {
		return lawarticleread.Request{}, err
	}
	return lawarticleread.NewRequest(lawarticleread.RequestValues{
		Resource: ref,
		AsOf:     materializerDatePointer(input.AsOf),
		Location: input.Location(),
	})
}

func materializeLawArticleTarget(
	input LawArticleReadIntentV1,
	binding selectedCapabilityBindingSnapshot,
) (model.SourceResourceRef, error) {
	if ref, exists := input.Ref(); exists {
		if err := validateMaterializerRef(ref, binding, "law"); err != nil {
			return model.SourceResourceRef{}, err
		}
		return ref, nil
	}
	lawID, exists := input.LawID()
	if !exists {
		return model.SourceResourceRef{},
			fmt.Errorf("law article read logical input に lawId または ref がありません")
	}
	return newMaterializedLawRef(binding, lawID, "")
}

// MaterializeLawUpdateList は、日付別の完全一覧 request を作る。
func (m CoreMaterializer) MaterializeLawUpdateList(
	input LawUpdateListIntentV1,
	binding SelectedCapabilityBinding,
	budget LegalQueryStepBudget,
) (lawupdatelist.Request, error) {
	if err := validateMaterializer(m.initialized, "CoreMaterializer"); err != nil {
		return lawupdatelist.Request{}, err
	}
	if err := input.Validate(); err != nil {
		return lawupdatelist.Request{},
			fmt.Errorf("law update list logical input が有効ではありません: %w", err)
	}
	if _, err := snapshotCapabilityBinding(
		binding,
		lawupdatelist.CapabilityID,
		lawupdatelist.MajorVersion,
	); err != nil {
		return lawupdatelist.Request{}, err
	}
	if _, err := materializerCollectionLimit(budget); err != nil {
		return lawupdatelist.Request{}, err
	}
	return lawupdatelist.NewRequest(lawupdatelist.RequestValues{
		Date: input.Date(),
	})
}
