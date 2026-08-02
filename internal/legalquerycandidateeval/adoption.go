package legalquerycandidateeval

import "fmt"

// AdoptionCandidateLink は adoption manifest が候補評価から引き継ぐ tuple である。
type AdoptionCandidateLink struct {
	CandidateContentID             string
	CandidateContentManifestSHA256 string
	EvaluatorVersion               string
	BaselineVersion                string
	BaselineSHA256                 string
}

// VerifyAdoptionLink は passed result と adoption tuple の完全一致を確認する。
func VerifyAdoptionLink(
	request EvaluationRequest,
	result EvaluationResult,
	link AdoptionCandidateLink,
) error {
	if err := validateEvaluationRequest(request); err != nil {
		return fmt.Errorf("adoption 元 request が不正です: %w", err)
	}
	if err := validateEvaluationResult(result); err != nil {
		return fmt.Errorf("adoption 元 result が不正です: %w", err)
	}
	requestRaw, err := MarshalCanonicalJSON(request)
	if err != nil {
		return err
	}
	if result.EvaluationID != request.EvaluationID ||
		result.SchemaVersion != request.SchemaVersion ||
		result.RequestSHA256 != RawSHA256(requestRaw) {
		return fmt.Errorf("adoption 元 request と result の結合が一致しません")
	}
	if result.Outcome != EvaluationOutcomePassed {
		return fmt.Errorf("passed result だけを adoption へ接続できます")
	}
	if link.CandidateContentID != request.CandidateContentID ||
		link.CandidateContentManifestSHA256 != request.CandidateContentManifestSHA256 ||
		link.EvaluatorVersion != request.EvaluatorVersion ||
		link.BaselineVersion != request.BaselineVersion ||
		link.BaselineSHA256 != result.ReportSHA256 {
		return fmt.Errorf("adoption tuple が passed result と一致しません")
	}
	return nil
}
