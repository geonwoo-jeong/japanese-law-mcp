package lawv2

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/continuation"
)

// NewProviderBindings は、e-Gov Version 2 の六つの共通 capability binding を一括構成する。
func NewProviderBindings(
	manager *continuation.Manager,
) (application.ProviderBindings, error) {
	lawSearch, err := NewLawSearchAdapter(manager)
	if err != nil {
		return application.ProviderBindings{}, fmt.Errorf(
			"e-Gov law.search binding を初期化できません: %w",
			err,
		)
	}
	lawContentSearch, err := NewLawContentSearchAdapter(manager)
	if err != nil {
		return application.ProviderBindings{}, fmt.Errorf(
			"e-Gov law.content.search binding を初期化できません: %w",
			err,
		)
	}
	lawDocumentRead, err := NewLawDocumentAdapter()
	if err != nil {
		return application.ProviderBindings{}, fmt.Errorf(
			"e-Gov law.document.read binding を初期化できません: %w",
			err,
		)
	}
	lawArticleRead, err := NewLawArticleAdapter()
	if err != nil {
		return application.ProviderBindings{}, fmt.Errorf(
			"e-Gov law.article.read binding を初期化できません: %w",
			err,
		)
	}
	lawRevisionList, err := NewLawRevisionListAdapter()
	if err != nil {
		return application.ProviderBindings{}, fmt.Errorf(
			"e-Gov law.revision.list binding を初期化できません: %w",
			err,
		)
	}
	lawVersionCompare, err := NewLawVersionCompareAdapter()
	if err != nil {
		return application.ProviderBindings{}, fmt.Errorf(
			"e-Gov law.version.compare binding を初期化できません: %w",
			err,
		)
	}
	return application.ProviderBindings{
		Descriptor:        Descriptor(),
		LawSearch:         lawSearch,
		LawContentSearch:  lawContentSearch,
		LawDocumentRead:   lawDocumentRead,
		LawArticleRead:    lawArticleRead,
		LawRevisionList:   lawRevisionList,
		LawVersionCompare: lawVersionCompare,
	}, nil
}
