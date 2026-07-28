package application

import (
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// JudicialCasesLegalQueryFacade は、裁判例 pack の route、request 組立ておよび型付き port を接続する。
type JudicialCasesLegalQueryFacade struct {
	routes       ProviderRoutes
	materializer legalquery.JudicialCasesRequestMaterializer
	initialized  bool
}

var _ legalquery.JudicialCasesCapabilityFacade = JudicialCasesLegalQueryFacade{}

// NewJudicialCasesLegalQueryFacade は、起動時に検証済みの裁判例 facade を返す。
func NewJudicialCasesLegalQueryFacade(
	routes ProviderRoutes,
	materializer legalquery.JudicialCasesRequestMaterializer,
) (JudicialCasesLegalQueryFacade, error) {
	result := JudicialCasesLegalQueryFacade{
		routes:       routes,
		materializer: materializer,
		initialized:  true,
	}
	if err := result.Validate(); err != nil {
		return JudicialCasesLegalQueryFacade{}, err
	}
	return result, nil
}

// Validate は、裁判例 pack の全能力を外部呼出し前に構成できることを確認する。
func (f JudicialCasesLegalQueryFacade) Validate() error {
	if !f.initialized {
		return fmt.Errorf(
			"JudicialCasesLegalQueryFacade は NewJudicialCasesLegalQueryFacade で作成しなければなりません",
		)
	}
	if isNilTypedPort(f.materializer) {
		return fmt.Errorf("裁判例 request materializer は必須です")
	}
	if err := f.materializer.Validate(); err != nil {
		return fmt.Errorf(
			"裁判例 request materializer が有効ではありません: %w",
			err,
		)
	}
	if err := f.validatePrimaryCapabilities(); err != nil {
		return err
	}
	return nil
}

func (f JudicialCasesLegalQueryFacade) validatePrimaryCapabilities() error {
	checks := []struct {
		capabilityID string
		majorVersion int
		hasPort      func(string) bool
	}{
		{
			capabilityID: judicialdecisionsearch.CapabilityID,
			majorVersion: judicialdecisionsearch.MajorVersion,
			hasPort: func(providerID string) bool {
				_, exists := f.routes.registry.JudicialDecisionSearch(providerID)
				return exists
			},
		},
		{
			capabilityID: judicialdecisionread.CapabilityID,
			majorVersion: judicialdecisionread.MajorVersion,
			hasPort: func(providerID string) bool {
				_, exists := f.routes.registry.JudicialDecisionRead(providerID)
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
				"裁判例 pack の primary binding %s@%d が構成されていません",
				check.capabilityID,
				check.majorVersion,
			)
		}
	}
	return nil
}

func (f JudicialCasesLegalQueryFacade) validateCallContext(
	ctx context.Context,
) error {
	if err := f.Validate(); err != nil {
		return err
	}
	if isNilTypedPort(ctx) {
		return fmt.Errorf("context は必須です")
	}
	return nil
}

func (f JudicialCasesLegalQueryFacade) primaryBinding(
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

func (f JudicialCasesLegalQueryFacade) judicialDecisionReadBinding(
	ref model.SourceResourceRef,
) (ProviderBindingMetadata, error) {
	binding, exists := f.routes.ExplicitBindingMetadata(
		ref.ProviderID(),
		judicialdecisionread.CapabilityID,
		judicialdecisionread.MajorVersion,
	)
	if !exists {
		return ProviderBindingMetadata{}, legalQueryFacadeInvalidRef(
			"に対応する採用済み capability binding がありません",
		)
	}
	if ref.Key().SourceID() != binding.SourceID() {
		return ProviderBindingMetadata{}, legalQueryFacadeInvalidRef(
			"の sourceId が採用済み provider と一致しません",
		)
	}
	return binding, nil
}
