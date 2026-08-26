package hanreipdf

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
)

// NewProviderBindings は、裁判例全文 PDF の引用抽出 binding を構成する。
func NewProviderBindings() (application.ProviderBindings, error) {
	extractor, err := NewJudicialDecisionCaseCitationExtractAdapter()
	if err != nil {
		return application.ProviderBindings{}, fmt.Errorf(
			"裁判所 judicial-decision.case-citation.extract binding を初期化できません: %w",
			err,
		)
	}
	return application.ProviderBindings{
		Descriptor:                  Descriptor(),
		JudicialCaseCitationExtract: extractor,
	}, nil
}
