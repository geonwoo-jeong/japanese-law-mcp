package legalquerycorpus

import (
	"errors"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func validateSemanticRequestExpected(
	request Request,
	expected SemanticExpected,
) error {
	_, requestError := productRequestFromRaw(request)
	switch typed := expected.(type) {
	case ExpectedPlan:
		if requestError != nil {
			return fmt.Errorf("expected plan の request は製品入力境界で受理されなければなりません")
		}
		return nil
	case ExpectedRequestError:
		if requestError == nil {
			return fmt.Errorf("expected request error の request は製品入力境界で拒否されなければなりません")
		}
		var argumentError legalquery.ArgumentError
		if !errors.As(requestError, &argumentError) {
			return fmt.Errorf("request の拒否理由を公開入力エラーとして分類できません")
		}
		if argumentError.Code() != typed.ErrorCode() ||
			RequestErrorField(argumentError.Field()) != typed.Field() {
			return fmt.Errorf("expected request error が製品入力境界の分類と一致しません")
		}
		return nil
	default:
		return fmt.Errorf("semantic expected の variant が定義されていません")
	}
}

func productRequestFromRaw(request Request) (legalquery.Request, error) {
	limit := rawRequestLimit(request)
	base, err := legalquery.NewRequest(legalquery.RequestValues{
		Query:           request.Query(),
		LimitPerAttempt: limit,
	})
	if err != nil {
		return legalquery.Request{}, err
	}
	rawRef, exists := request.Ref()
	if !exists {
		return base, nil
	}
	ref, err := productRefFromRaw(rawRef)
	if err != nil {
		return legalquery.Request{}, semanticRefArgumentError()
	}
	return legalquery.NewRequest(legalquery.RequestValues{
		Query:           request.Query(),
		Ref:             &ref,
		LimitPerAttempt: limit,
	})
}

func rawRequestLimit(request Request) *int {
	value, exists := request.LimitPerAttempt()
	if !exists {
		return nil
	}
	return &value
}

func productRefFromRaw(rawRef RequestRef) (model.SourceResourceRef, error) {
	rawKey := rawRef.Key()
	versionID, _ := rawKey.VersionID()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     rawKey.SourceID(),
		ResourceType: rawKey.ResourceType(),
		ResourceID:   rawKey.ResourceID(),
		VersionID:    versionID,
	})
	if err != nil {
		return model.SourceResourceRef{}, err
	}
	return model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: rawRef.ProviderID(),
		Key:        key,
	})
}

func semanticRefArgumentError() error {
	argumentError, err := legalquery.NewArgumentError(
		"ref",
		"は有効な SourceResourceRef でなければなりません",
	)
	if err != nil {
		return fmt.Errorf("ref の入力エラーを分類できません")
	}
	return argumentError
}

func validateSemanticPlanAvailability(
	plan ExpectedPlan,
	enabledPacks []string,
) error {
	if plan.Decision() == legalquery.PlanDecisionUnsupported {
		return nil
	}
	meanings := make(map[string]ExpectedMeaning, len(plan.Meanings()))
	for _, meaning := range plan.Meanings() {
		meanings[meaning.MeaningID()] = meaning
	}
	enabled := make(map[string]struct{}, len(enabledPacks))
	for _, packID := range enabledPacks {
		enabled[packID] = struct{}{}
	}
	for _, meaningID := range plan.SelectedMeaningIDs() {
		meaning := meanings[meaningID]
		available := hasAllRequiredPacks(meaning.RequiredPacks(), enabled)
		switch plan.Decision() {
		case legalquery.PlanDecisionSingle,
			legalquery.PlanDecisionHedged,
			legalquery.PlanDecisionNeedsClarification:
			if !available {
				return fmt.Errorf("選択意味は enabledPacks で実行可能でなければなりません")
			}
		case legalquery.PlanDecisionCapabilityUnavailable:
			if available {
				return fmt.Errorf("capability_unavailable の選択意味には無効な pack が必要です")
			}
		default:
			return fmt.Errorf("expected plan の decision が定義されていません")
		}
	}
	return nil
}

func hasAllRequiredPacks(
	requiredPacks []string,
	enabled map[string]struct{},
) bool {
	for _, packID := range requiredPacks {
		if _, exists := enabled[packID]; !exists {
			return false
		}
	}
	return true
}
