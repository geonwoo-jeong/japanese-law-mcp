package lawv2

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type operation string

const (
	operationLaws         operation = "GET /laws"
	operationKeyword      operation = "GET /keyword"
	operationLawData      operation = "GET /law_data/{law_id_or_num_or_revision_id}"
	operationLawRevisions operation = "GET /law_revisions/{law_id_or_num}"
)

func (operation) SourceOperationProviderID() string {
	return providerID
}

func (o operation) SourceOperationName() string {
	return string(o)
}

func (o operation) ValidateSourceOperation() error {
	if o != operationLaws && o != operationKeyword && o != operationLawData &&
		o != operationLawRevisions {
		return fmt.Errorf("e-Gov operation が定義されていません")
	}
	return nil
}

func newLawRevisionSourceError(
	code model.SourceErrorCode,
	retryAfter string,
) error {
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   Descriptor(),
		Capability: lawRevisionListCapability(),
		Operation:  operationLawRevisions,
		RetryAfter: retryAfter,
	})
	if err != nil {
		return fmt.Errorf("e-Gov 法令改正履歴エラーを正規化できません: %w", err)
	}
	return sourceError
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

func newLawVersionCompareSourceError(
	code model.SourceErrorCode,
	retryAfter string,
) error {
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   Descriptor(),
		Capability: lawVersionCompareCapability(),
		Operation:  operationLawData,
		RetryAfter: retryAfter,
	})
	if err != nil {
		return fmt.Errorf("e-Gov 法令版間比較エラーを正規化できません: %w", err)
	}
	return sourceError
}
