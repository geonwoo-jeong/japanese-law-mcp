package legalquery

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// SelectedCapabilityBinding は、選択済み binding の最小 metadata を公開する。
type SelectedCapabilityBinding interface {
	ProviderID() string
	SourceID() string
	CapabilityID() string
	CapabilityMajorVersion() int
}

// CoreRequestMaterializer は、法令コアの logical input を型付き request に変換する。
type CoreRequestMaterializer interface {
	MaterializeLawSearch(
		LawSearchIntentV1,
		SelectedCapabilityBinding,
		LegalQueryStepBudget,
	) (lawsearch.Request, error)
	MaterializeLawContentSearch(
		LawContentSearchIntentV1,
		SelectedCapabilityBinding,
		LegalQueryStepBudget,
	) (lawcontentsearch.Request, error)
	MaterializeLawDocumentRead(
		LawReadIntentV1,
		SelectedCapabilityBinding,
		LegalQueryStepBudget,
	) (lawdocumentread.Request, error)
	MaterializeLawArticleRead(
		LawArticleReadIntentV1,
		SelectedCapabilityBinding,
		LegalQueryStepBudget,
	) (lawarticleread.Request, error)
	MaterializeLawUpdateList(
		LawUpdateListIntentV1,
		SelectedCapabilityBinding,
		LegalQueryStepBudget,
	) (lawupdatelist.Request, error)
}

// JudicialCasesRequestMaterializer は、裁判例 pack の logical input を変換する。
type JudicialCasesRequestMaterializer interface {
	MaterializeJudicialDecisionSearch(
		JudicialDecisionSearchIntentV1,
		SelectedCapabilityBinding,
		LegalQueryStepBudget,
	) (judicialdecisionsearch.Request, error)
	MaterializeJudicialDecisionRead(
		JudicialDecisionReadIntentV1,
		SelectedCapabilityBinding,
		LegalQueryStepBudget,
	) (judicialdecisionread.Request, error)
}

// CoreMaterializer は、法令コアの既存 capability request を決定的に作る。
type CoreMaterializer struct {
	initialized bool
}

// NewCoreMaterializer は、検証可能な法令コア materializer を返す。
func NewCoreMaterializer() CoreMaterializer {
	return CoreMaterializer{initialized: true}
}

// JudicialCasesMaterializer は、裁判例の既存 capability request を作る。
type JudicialCasesMaterializer struct {
	initialized bool
}

// NewJudicialCasesMaterializer は、検証可能な裁判例 materializer を返す。
func NewJudicialCasesMaterializer() JudicialCasesMaterializer {
	return JudicialCasesMaterializer{initialized: true}
}

type selectedCapabilityBindingSnapshot struct {
	providerID   string
	sourceID     string
	capabilityID string
	majorVersion int
}

func validateMaterializer(
	initialized bool,
	name string,
) error {
	if !initialized {
		return fmt.Errorf("%s は constructor で作成しなければなりません", name)
	}
	return nil
}

func snapshotCapabilityBinding(
	binding SelectedCapabilityBinding,
	expectedCapabilityID string,
	expectedMajorVersion int,
) (selectedCapabilityBindingSnapshot, error) {
	if isNilInterfaceValue(binding) {
		return selectedCapabilityBindingSnapshot{},
			fmt.Errorf("選択済み capability binding は必須です")
	}
	snapshot := selectedCapabilityBindingSnapshot{
		providerID:   binding.ProviderID(),
		sourceID:     binding.SourceID(),
		capabilityID: binding.CapabilityID(),
		majorVersion: binding.CapabilityMajorVersion(),
	}
	if snapshot.capabilityID != expectedCapabilityID ||
		snapshot.majorVersion != expectedMajorVersion {
		return selectedCapabilityBindingSnapshot{},
			fmt.Errorf("選択済み binding の capability が materializer と一致しません")
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     snapshot.sourceID,
		ResourceType: "binding",
		ResourceID:   "binding",
	})
	if err != nil {
		return selectedCapabilityBindingSnapshot{},
			fmt.Errorf("選択済み binding の sourceId が有効ではありません: %w", err)
	}
	if _, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: snapshot.providerID,
		Key:        key,
	}); err != nil {
		return selectedCapabilityBindingSnapshot{},
			fmt.Errorf("選択済み binding の providerId が有効ではありません: %w", err)
	}
	return snapshot, nil
}

func materializerCollectionLimit(
	budget LegalQueryStepBudget,
) (int, error) {
	if err := budget.Validate(); err != nil {
		return 0, fmt.Errorf("collection step の予算が有効ではありません: %w", err)
	}
	if budget.ReservedItems() != 0 {
		return 0, fmt.Errorf("collection materializer に read step の予算を指定できません")
	}
	limit, exists := budget.EffectiveLimit()
	if !exists {
		return 0, fmt.Errorf("collection materializer には effectiveLimit が必要です")
	}
	return limit, nil
}

func validateMaterializerReadBudget(budget LegalQueryStepBudget) error {
	if err := budget.Validate(); err != nil {
		return fmt.Errorf("read step の予算が有効ではありません: %w", err)
	}
	if budget.ReservedItems() != 1 {
		return fmt.Errorf("read materializer に collection step の予算を指定できません")
	}
	if _, exists := budget.EffectiveLimit(); exists {
		return fmt.Errorf("read materializer に effectiveLimit を指定できません")
	}
	return nil
}

func materializerDatePointer(
	getter func() (model.Date, bool),
) *model.Date {
	value, exists := getter()
	if !exists {
		return nil
	}
	return &value
}

func validateMaterializerRef(
	ref model.SourceResourceRef,
	binding selectedCapabilityBindingSnapshot,
	resourceType string,
) error {
	if err := ref.Validate(); err != nil {
		return newMaterializerRefError(
			"は有効な SourceResourceRef でなければなりません",
		)
	}
	key := ref.Key()
	if ref.ProviderID() != binding.providerID {
		return newMaterializerRefError(
			"の providerId は選択済み binding と一致しなければなりません",
		)
	}
	if key.SourceID() != binding.sourceID {
		return newMaterializerRefError(
			"の sourceId は選択済み binding と一致しなければなりません",
		)
	}
	if key.ResourceType() != resourceType {
		return newMaterializerRefError(
			"の resourceType は選択した能力と一致しなければなりません",
		)
	}
	return nil
}

func newMaterializerRefError(reason string) error {
	result, err := NewArgumentError("ref", reason)
	if err != nil {
		return fmt.Errorf("ref の入力エラーを作成できません: %w", err)
	}
	return result
}

func newMaterializedLawRef(
	binding selectedCapabilityBindingSnapshot,
	lawID string,
	revisionID string,
) (model.SourceResourceRef, error) {
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     binding.sourceID,
		ResourceType: "law",
		ResourceID:   lawID,
		VersionID:    revisionID,
	})
	if err != nil {
		return model.SourceResourceRef{},
			fmt.Errorf("法令 ID から資源 key を作成できません: %w", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: binding.providerID,
		Key:        key,
	})
	if err != nil {
		return model.SourceResourceRef{},
			fmt.Errorf("法令 ID から資源 ref を作成できません: %w", err)
	}
	return ref, nil
}
