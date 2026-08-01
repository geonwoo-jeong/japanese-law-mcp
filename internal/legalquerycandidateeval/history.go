package legalquerycandidateeval

import "fmt"

const maximumConsumedEvaluationHistory = 4096

// ConsumedEvaluation は、不変 request とその一件の result から成る compact 履歴である。
type ConsumedEvaluation struct {
	Request EvaluationRequest
	Result  EvaluationResult
}

// CheckEvaluationPreflight は、holdout の一回利用と leakage group の分離を確認する。
func CheckEvaluationPreflight(current EvaluationRequest, history []ConsumedEvaluation) error {
	if err := validateEvaluationRequest(current); err != nil {
		return fmt.Errorf("current evaluation request が不正です: %w", err)
	}
	if len(history) > maximumConsumedEvaluationHistory {
		return fmt.Errorf("candidate evaluation 履歴が上限を超えています")
	}

	currentGroups := make(map[string]struct{}, len(current.HoldoutLeakageGroupDigests))
	for _, digest := range current.HoldoutLeakageGroupDigests {
		currentGroups[digest] = struct{}{}
	}
	seenEvaluationIDs := make(map[string]struct{}, len(history))
	seenBaselineVersions := map[string]struct{}{current.BaselineVersion: {}}
	for _, consumed := range history {
		if err := validateConsumedEvaluation(consumed); err != nil {
			return err
		}
		if _, duplicate := seenEvaluationIDs[consumed.Request.EvaluationID]; duplicate {
			return fmt.Errorf("同じ evaluationId の履歴が重複しています")
		}
		seenEvaluationIDs[consumed.Request.EvaluationID] = struct{}{}
		if consumed.Request.EvaluationID == current.EvaluationID {
			continue
		}
		if _, duplicate := seenBaselineVersions[consumed.Request.BaselineVersion]; duplicate {
			return fmt.Errorf("同じ baselineVersion の履歴が重複しています")
		}
		seenBaselineVersions[consumed.Request.BaselineVersion] = struct{}{}
		if consumed.Request.HoldoutDigest == current.HoldoutDigest {
			return fmt.Errorf("同じ holdoutDigest は別 evaluation で再利用できません")
		}
		for _, digest := range consumed.Request.HoldoutLeakageGroupDigests {
			if _, collision := currentGroups[digest]; collision {
				return fmt.Errorf("過去 evaluation と holdout leakage group が衝突しています")
			}
		}
	}
	return nil
}

func validateConsumedEvaluation(consumed ConsumedEvaluation) error {
	if err := validateEvaluationRequest(consumed.Request); err != nil {
		return fmt.Errorf("消費済み evaluation request が不正です: %w", err)
	}
	if err := validateEvaluationResult(consumed.Result); err != nil {
		return fmt.Errorf("消費済み evaluation result が不正です: %w", err)
	}
	if consumed.Result.EvaluationID != consumed.Request.EvaluationID {
		return fmt.Errorf("消費済み request と result の evaluationId が一致しません")
	}
	requestRaw, err := MarshalCanonicalJSON(consumed.Request)
	if err != nil {
		return err
	}
	if consumed.Result.RequestSHA256 != RawSHA256(requestRaw) {
		return fmt.Errorf("消費済み result の requestSha256 が一致しません")
	}
	return nil
}
