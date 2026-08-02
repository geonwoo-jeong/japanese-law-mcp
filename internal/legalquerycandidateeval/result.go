package legalquerycandidateeval

import (
	"bytes"
	"fmt"
)

const maximumEvaluationReportBytes = 4 << 20

// NewEvaluationResult は request と report の原 byte に結合した不変 result を作る。
func NewEvaluationResult(
	request EvaluationRequest,
	requestRaw []byte,
	reportRaw []byte,
	outcome string,
) (EvaluationResult, error) {
	if err := validateEvaluationRequest(request); err != nil {
		return EvaluationResult{}, fmt.Errorf("result の request が不正です: %w", err)
	}
	canonicalRequest, err := MarshalCanonicalJSON(request)
	if err != nil {
		return EvaluationResult{}, err
	}
	if !bytes.Equal(requestRaw, canonicalRequest) {
		return EvaluationResult{}, fmt.Errorf("request 原 byte が canonical request と一致しません")
	}
	if len(reportRaw) == 0 || len(reportRaw) > maximumEvaluationReportBytes {
		return EvaluationResult{}, fmt.Errorf("report 原 byte の size が上限外です")
	}
	if !validEvaluationOutcome(outcome) {
		return EvaluationResult{}, fmt.Errorf("評価 outcome が不正です")
	}

	result := EvaluationResult{
		ArtifactKind:  ArtifactKindEvaluationResult,
		SchemaVersion: request.SchemaVersion,
		EvaluationID:  request.EvaluationID,
		RequestSHA256: RawSHA256(requestRaw),
		Outcome:       outcome,
		ReportSHA256:  RawSHA256(reportRaw),
	}
	if err := validateEvaluationResult(result); err != nil {
		return EvaluationResult{}, err
	}
	return result, nil
}

func validateEvaluationResult(result EvaluationResult) error {
	if result.ArtifactKind != ArtifactKindEvaluationResult ||
		!isSupportedSchemaVersion(result.SchemaVersion) ||
		!evaluationIDPattern.MatchString(result.EvaluationID) {
		return fmt.Errorf("candidate evaluation result の版または ID が不正です")
	}
	if err := validateSHA256("requestSha256", result.RequestSHA256); err != nil {
		return err
	}
	if !validEvaluationOutcome(result.Outcome) {
		return fmt.Errorf("candidate evaluation result の outcome が不正です")
	}
	return validateSHA256("reportSha256", result.ReportSHA256)
}

func validEvaluationOutcome(outcome string) bool {
	return outcome == EvaluationOutcomePassed || outcome == EvaluationOutcomeFailed
}
