package kokkai

import (
	"context"
	"errors"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type operation string

const operationSpeechSearch operation = "GET /api/speech"

func (operation) SourceOperationProviderID() string {
	return providerID
}

func (o operation) SourceOperationName() string {
	return string(o)
}

func (o operation) ValidateSourceOperation() error {
	if o != operationSpeechSearch {
		return fmt.Errorf("国会発言検索 operation が定義されていません")
	}
	return nil
}

func newSpeechSearchSourceError(
	code model.SourceErrorCode,
	retryAfter string,
) error {
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   Descriptor(),
		Capability: speechSearchCapability(),
		Operation:  operationSpeechSearch,
		RetryAfter: retryAfter,
	})
	if err != nil {
		return fmt.Errorf("国会発言検索の情報源エラーを正規化できません: %w", err)
	}
	return sourceError
}

func speechSearchCapability() model.ProviderCapability {
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           parliamentspeechsearch.CapabilityID,
		MajorVersion: parliamentspeechsearch.MajorVersion,
		Level:        model.CapabilityLevelExtended,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		panic(fmt.Sprintf("国会発言 capability の固定値が不正です: %v", err))
	}
	return capability
}

func normalizeSpeechSearchContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return newSpeechSearchSourceError(model.SourceErrorCodeSourceTimeout, "")
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	return newSpeechSearchSourceError(model.SourceErrorCodeSourceUnavailable, "")
}
