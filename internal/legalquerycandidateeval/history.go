package legalquerycandidateeval

import "fmt"

const (
	maximumConsumedEvaluationHistory = 4096
	legacyV2EvaluationIDDefault2     = "evaluation-sha256-1001bab1bab4c88533769e89e5ad7a4aed78e043239344b67a6d450b41adfdbd"
	legacyV2EvaluationIDDefault3     = "evaluation-sha256-398e801b2d7edd6068f36fa34fe94827d7d44891d59976fdc8630e4d5be7e89c"
)

// ConsumedEvaluation は、不変 request とその一件の result から成る compact 履歴である。
type ConsumedEvaluation struct {
	Request EvaluationRequest
	Result  EvaluationResult
}

// CheckEvaluationPreflight は、compact 履歴だけを持つ schema v2 単独時代の
// preflight を確認する。
//
// SOT-ENG-042 以降の mixed-generation request 予約検査は
// checkRequestReservationPreflight が定義元であり、この関数へ v2/v3 共存の
// 遡及除外を流用しない。
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
	if consumed.Result.SchemaVersion != consumed.Request.SchemaVersion {
		return fmt.Errorf("消費済み request と result の schemaVersion が一致しません")
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

func checkRequestReservationPreflight(
	current EvaluationRequest,
	requests map[string]loadedArtifact[EvaluationRequest],
) error {
	if err := validateEvaluationRequest(current); err != nil {
		return fmt.Errorf("current evaluation request が不正です: %w", err)
	}
	if len(requests) > maximumConsumedEvaluationHistory {
		return fmt.Errorf("candidate evaluation request 履歴が上限を超えています")
	}

	currentGroups := make(map[string]struct{}, len(current.HoldoutLeakageGroupDigests))
	for _, digest := range current.HoldoutLeakageGroupDigests {
		currentGroups[digest] = struct{}{}
	}
	for _, evaluationID := range sortedKeys(requests) {
		reserved := requests[evaluationID].document
		if err := validateEvaluationRequest(reserved); err != nil {
			return fmt.Errorf("予約済み evaluation request が不正です: %w", err)
		}
		if reserved.EvaluationID == current.EvaluationID {
			continue
		}
		if isLegacyV2ReservationPair(current, reserved) {
			continue
		}
		if reserved.BaselineVersion == current.BaselineVersion {
			return fmt.Errorf("予約済み baselineVersion は再利用できません")
		}
		if reserved.HoldoutDigest == current.HoldoutDigest {
			return fmt.Errorf("予約済み holdoutDigest は再利用できません")
		}
		for _, digest := range reserved.HoldoutLeakageGroupDigests {
			if _, collision := currentGroups[digest]; collision {
				return fmt.Errorf("予約済み holdout leakage group は再利用できません")
			}
		}
	}
	return nil
}

func isLegacyV2ReservationPair(
	current EvaluationRequest,
	reserved EvaluationRequest,
) bool {
	if current.SchemaVersion != SchemaVersionV2 ||
		reserved.SchemaVersion != SchemaVersionV2 {
		return false
	}
	return (current.EvaluationID == legacyV2EvaluationIDDefault2 &&
		reserved.EvaluationID == legacyV2EvaluationIDDefault3) ||
		(current.EvaluationID == legacyV2EvaluationIDDefault3 &&
			reserved.EvaluationID == legacyV2EvaluationIDDefault2)
}
