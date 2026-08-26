package hanrei

import (
	"context"
	"errors"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitingcandidatesearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func judicialCitingCandidateCapability() model.ProviderCapability {
	return mustJudicialCapability(judicialcitingcandidatesearch.CapabilityID)
}

// citingCandidateErrorDescriptor は、最終接続前にも capability 固有エラーを型付けする内部記述子である。
func citingCandidateErrorDescriptor() model.ProviderDescriptor {
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             providerID,
		Source:                 informationSource(),
		AdapterContractVersion: "1.2.0",
		VerifiedAt:             mustDescriptorDate("2026-07-26"),
		InterfaceType:          model.InterfaceTypeHTML,
		CredentialRequired:     false,
		Capabilities: []model.ProviderCapability{
			judicialCitingCandidateCapability(),
			mustJudicialCapability("judicial-decision.read"),
			mustJudicialCapability("judicial-decision.search"),
		},
	})
	if err != nil {
		panic(fmt.Sprintf("裁判所候補検索の内部 descriptor が不正です: %v", err))
	}
	return descriptor
}

func newCitingCandidateSourceError(
	code model.SourceErrorCode,
	retryAfter string,
) model.SourceError {
	if code == model.SourceErrorCodeSourceAuthFailed {
		code = model.SourceErrorCodeInvalidSourceResponse
		retryAfter = ""
	}
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   citingCandidateErrorDescriptor(),
		Capability: judicialCitingCandidateCapability(),
		Operation:  operationSearch,
		RetryAfter: retryAfter,
	})
	if err != nil {
		panic(fmt.Sprintf("裁判所候補検索の情報源エラーを正規化できません: %v", err))
	}
	return sourceError
}

func normalizeCitingCandidateError(ctx context.Context, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return context.Canceled
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return newCitingCandidateSourceError(model.SourceErrorCodeSourceTimeout, "")
	}
	var sourceError model.SourceError
	if errors.As(err, &sourceError) {
		retryAfter, _ := sourceError.RetryAfter()
		return newCitingCandidateSourceError(sourceError.Code(), retryAfter)
	}
	return newCitingCandidateSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
}
