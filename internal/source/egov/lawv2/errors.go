package lawv2

import (
	"fmt"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

type operation string

const operationLaws operation = "GET /laws"

func (operation) SourceOperationProviderID() string {
	return providerID
}

func (o operation) SourceOperationName() string {
	return string(o)
}

func (o operation) ValidateSourceOperation() error {
	if o != operationLaws {
		return fmt.Errorf("e-Gov operation が定義されていません")
	}
	return nil
}

func newSourceError(code model.SourceErrorCode, retryAfter string) error {
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   Descriptor(),
		Capability: lawSearchCapability(),
		Operation:  operationLaws,
		RetryAfter: retryAfter,
	})
	if err != nil {
		return fmt.Errorf("e-Gov 情報源エラーを正規化できません: %w", err)
	}
	return sourceError
}
