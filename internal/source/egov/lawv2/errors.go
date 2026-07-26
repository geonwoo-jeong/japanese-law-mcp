package lawv2

import (
	"fmt"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

type operation string

const (
	operationLaws    operation = "GET /laws"
	operationKeyword operation = "GET /keyword"
	operationLawData operation = "GET /law_data/{law_id_or_num_or_revision_id}"
)

func (operation) SourceOperationProviderID() string {
	return providerID
}

func (o operation) SourceOperationName() string {
	return string(o)
}

func (o operation) ValidateSourceOperation() error {
	if o != operationLaws && o != operationKeyword && o != operationLawData {
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

func newLawDocumentSourceError(
	code model.SourceErrorCode,
	retryAfter string,
) error {
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   Descriptor(),
		Capability: lawDocumentCapability(),
		Operation:  operationLawData,
		RetryAfter: retryAfter,
	})
	if err != nil {
		return fmt.Errorf("e-Gov 法令本文エラーを正規化できません: %w", err)
	}
	return sourceError
}

func newLawContentSourceError(
	code model.SourceErrorCode,
	retryAfter string,
) error {
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   Descriptor(),
		Capability: lawContentSearchCapability(),
		Operation:  operationKeyword,
		RetryAfter: retryAfter,
	})
	if err != nil {
		return fmt.Errorf("e-Gov 法令本文検索エラーを正規化できません: %w", err)
	}
	return sourceError
}

func newLawArticleSourceError(
	code model.SourceErrorCode,
	retryAfter string,
) error {
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   Descriptor(),
		Capability: lawArticleCapability(),
		Operation:  operationLawData,
		RetryAfter: retryAfter,
	})
	if err != nil {
		return fmt.Errorf("e-Gov 条文取得エラーを正規化できません: %w", err)
	}
	return sourceError
}
