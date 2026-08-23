package kokkai

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
)

// NewProviderBindings は、国会発言検索の binding を構成する。
func NewProviderBindings() (application.ProviderBindings, error) {
	search, err := NewSpeechSearchAdapter()
	if err != nil {
		return application.ProviderBindings{}, fmt.Errorf(
			"国会発言 parliament.speech.search binding を初期化できません: %w",
			err,
		)
	}
	return application.ProviderBindings{
		Descriptor:             Descriptor(),
		ParliamentSpeechSearch: search,
	}, nil
}
