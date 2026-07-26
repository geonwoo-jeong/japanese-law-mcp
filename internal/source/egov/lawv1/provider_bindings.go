package lawv1

import (
	"fmt"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application"
)

// NewProviderBindings は、e-Gov Version 1 更新一覧の binding を構成する。
func NewProviderBindings() (application.ProviderBindings, error) {
	adapter, err := NewLawUpdateListAdapter()
	if err != nil {
		return application.ProviderBindings{}, fmt.Errorf(
			"e-Gov law.update.list binding を初期化できません: %w",
			err,
		)
	}
	return application.ProviderBindings{
		Descriptor:    Descriptor(),
		LawUpdateList: adapter,
	}, nil
}
