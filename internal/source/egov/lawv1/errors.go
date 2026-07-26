package lawv1

import (
	"fmt"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

type operation string

const operationUpdateLawLists operation = "GET /api/1/updatelawlists/{yyyyMMdd}"

func (operation) SourceOperationProviderID() string {
	return providerID
}

func (o operation) SourceOperationName() string {
	return string(o)
}

func (o operation) ValidateSourceOperation() error {
	if o != operationUpdateLawLists {
		return fmt.Errorf("e-Gov Version 1 operation が定義されていません")
	}
	return nil
}

func newSourceError(
	code model.SourceErrorCode,
	retryAfter string,
) error {
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   Descriptor(),
		Capability: lawUpdateListCapability(),
		Operation:  operationUpdateLawLists,
		RetryAfter: retryAfter,
	})
	if err != nil {
		return fmt.Errorf("e-Gov Version 1 情報源エラーを正規化できません: %w", err)
	}
	return sourceError
}
