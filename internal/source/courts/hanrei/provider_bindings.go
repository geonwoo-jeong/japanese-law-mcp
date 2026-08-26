package hanrei

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
)

// NewProviderBindings は、裁判例の検索、詳細取得と被引用候補検索の binding を一括構成する。
func NewProviderBindings() (application.ProviderBindings, error) {
	candidateSearch, err := NewJudicialCitingCandidateSearchAdapter()
	if err != nil {
		return application.ProviderBindings{}, fmt.Errorf(
			"裁判所 judicial-decision.citing-candidate.search binding を初期化できません: %w",
			err,
		)
	}
	search, err := NewJudicialDecisionSearchAdapter()
	if err != nil {
		return application.ProviderBindings{}, fmt.Errorf(
			"裁判所 judicial-decision.search binding を初期化できません: %w",
			err,
		)
	}
	read, err := NewJudicialDecisionReadAdapter()
	if err != nil {
		return application.ProviderBindings{}, fmt.Errorf(
			"裁判所 judicial-decision.read binding を初期化できません: %w",
			err,
		)
	}
	return application.ProviderBindings{
		Descriptor:                    Descriptor(),
		JudicialCitingCandidateSearch: candidateSearch,
		JudicialDecisionRead:          read,
		JudicialDecisionSearch:        search,
	}, nil
}
