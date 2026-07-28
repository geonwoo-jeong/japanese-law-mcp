package application

import (
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// CoreLegalQueryFacade は、法令コアの route、request 組立ておよび型付き port を接続する。
type CoreLegalQueryFacade struct {
	routes       ProviderRoutes
	materializer legalquery.CoreRequestMaterializer
	initialized  bool
}

var _ legalquery.CoreCapabilityFacade = CoreLegalQueryFacade{}

// NewCoreLegalQueryFacade は、起動時に検証済みの法令コア facade を返す。
func NewCoreLegalQueryFacade(
	routes ProviderRoutes,
	materializer legalquery.CoreRequestMaterializer,
) (CoreLegalQueryFacade, error) {
	result := CoreLegalQueryFacade{
		routes:       routes,
		materializer: materializer,
		initialized:  true,
	}
	if err := result.Validate(); err != nil {
		return CoreLegalQueryFacade{}, err
	}
	return result, nil
}

// Validate は、法令コアの全能力を外部呼出し前に構成できることを確認する。
func (f CoreLegalQueryFacade) Validate() error {
	if !f.initialized {
		return fmt.Errorf(
			"CoreLegalQueryFacade は NewCoreLegalQueryFacade で作成しなければなりません",
		)
	}
	if isNilTypedPort(f.materializer) {
		return fmt.Errorf("法令コア request materializer は必須です")
	}
	if err := f.materializer.Validate(); err != nil {
		return fmt.Errorf(
			"法令コア request materializer が有効ではありません: %w",
			err,
		)
	}
	if err := f.validatePrimaryCapabilities(); err != nil {
		return err
	}
	return nil
}

func (f CoreLegalQueryFacade) validatePrimaryCapabilities() error {
	checks := []struct {
		capabilityID string
		majorVersion int
		hasPort      func(string) bool
	}{
		{
			capabilityID: lawsearch.CapabilityID,
			majorVersion: lawsearch.MajorVersion,
			hasPort: func(providerID string) bool {
				_, exists := f.routes.registry.LawSearch(providerID)
				return exists
			},
		},
		{
			capabilityID: lawcontentsearch.CapabilityID,
			majorVersion: lawcontentsearch.MajorVersion,
			hasPort: func(providerID string) bool {
				_, exists := f.routes.registry.LawContentSearch(providerID)
				return exists
			},
		},
		{
			capabilityID: lawdocumentread.CapabilityID,
			majorVersion: lawdocumentread.MajorVersion,
			hasPort: func(providerID string) bool {
				_, exists := f.routes.registry.LawDocumentRead(providerID)
				return exists
			},
		},
		{
			capabilityID: lawarticleread.CapabilityID,
			majorVersion: lawarticleread.MajorVersion,
			hasPort: func(providerID string) bool {
				_, exists := f.routes.registry.LawArticleRead(providerID)
				return exists
			},
		},
		{
			capabilityID: lawupdatelist.CapabilityID,
			majorVersion: lawupdatelist.MajorVersion,
			hasPort: func(providerID string) bool {
				_, exists := f.routes.registry.LawUpdateList(providerID)
				return exists
			},
		},
	}
	for _, check := range checks {
		binding, exists := f.routes.PrimaryBindingMetadata(
			check.capabilityID,
			check.majorVersion,
		)
		if !exists || !check.hasPort(binding.ProviderID()) {
			return fmt.Errorf(
				"法令コアの primary binding %s@%d が構成されていません",
				check.capabilityID,
				check.majorVersion,
			)
		}
	}
	return nil
}

func (f CoreLegalQueryFacade) validateCallContext(ctx context.Context) error {
	if err := f.Validate(); err != nil {
		return err
	}
	if isNilTypedPort(ctx) {
		return fmt.Errorf("context は必須です")
	}
	return nil
}

func validateCoreFacadeCollectionBudget(
	budget legalquery.LegalQueryStepBudget,
) error {
	if err := budget.Validate(); err != nil {
		return fmt.Errorf("collection step 予算が有効ではありません: %w", err)
	}
	if budget.ReservedItems() != 0 {
		return fmt.Errorf("collection step 予算に read 用 reservedItems を指定できません")
	}
	if _, exists := budget.EffectiveLimit(); !exists {
		return fmt.Errorf("collection step 予算には effectiveLimit が必要です")
	}
	return nil
}

func validateCoreFacadeReadBudget(
	budget legalquery.LegalQueryStepBudget,
) error {
	if err := budget.Validate(); err != nil {
		return fmt.Errorf("read step 予算が有効ではありません: %w", err)
	}
	if budget.ReservedItems() != 1 {
		return fmt.Errorf("read step 予算の reservedItems は 1 でなければなりません")
	}
	if _, exists := budget.EffectiveLimit(); exists {
		return fmt.Errorf("read step 予算に effectiveLimit を指定できません")
	}
	return nil
}

func (f CoreLegalQueryFacade) primaryBinding(
	capabilityID string,
	majorVersion int,
) (ProviderBindingMetadata, error) {
	binding, exists := f.routes.PrimaryBindingMetadata(
		capabilityID,
		majorVersion,
	)
	if !exists {
		return ProviderBindingMetadata{}, fmt.Errorf(
			"primary binding %s@%d が構成されていません",
			capabilityID,
			majorVersion,
		)
	}
	return binding, nil
}

type coreFacadeLawRefInput interface {
	Ref() (model.SourceResourceRef, bool)
}

func (f CoreLegalQueryFacade) lawBinding(
	input coreFacadeLawRefInput,
	capabilityID string,
	majorVersion int,
) (ProviderBindingMetadata, error) {
	ref, explicit := input.Ref()
	if !explicit {
		return f.primaryBinding(capabilityID, majorVersion)
	}
	return f.explicitLawBinding(ref, capabilityID, majorVersion)
}

func (f CoreLegalQueryFacade) explicitLawBinding(
	ref model.SourceResourceRef,
	capabilityID string,
	majorVersion int,
) (ProviderBindingMetadata, error) {
	binding, exists := f.routes.ExplicitBindingMetadata(
		ref.ProviderID(),
		capabilityID,
		majorVersion,
	)
	if !exists {
		return ProviderBindingMetadata{}, coreFacadeInvalidRef(
			"に対応する採用済み capability binding がありません",
		)
	}
	if ref.Key().SourceID() != binding.SourceID() {
		return ProviderBindingMetadata{}, coreFacadeInvalidRef(
			"の sourceId が採用済み provider と一致しません",
		)
	}
	return binding, nil
}

func coreFacadeInvalidRef(reason string) error {
	result, err := legalquery.NewArgumentError("ref", reason)
	if err != nil {
		return fmt.Errorf("ref の入力エラーを作成できません: %w", err)
	}
	return result
}
