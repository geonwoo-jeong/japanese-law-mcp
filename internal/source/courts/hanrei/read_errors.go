package hanrei

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const operationRead operation = "GET /hanrei/{id}/detail{2..8}/index.html"

func newReadSourceError(code model.SourceErrorCode, retryAfter string) error {
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   Descriptor(),
		Capability: mustJudicialCapability("judicial-decision.read"),
		Operation:  operationRead,
		RetryAfter: retryAfter,
	})
	if err != nil {
		return fmt.Errorf("裁判所の情報源エラーを正規化できません: %w", err)
	}
	return sourceError
}
