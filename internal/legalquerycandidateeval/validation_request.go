package legalquerycandidateeval

import "fmt"

func validateEvaluationRequest(document EvaluationRequest) error {
	if document.ArtifactKind != ArtifactKindEvaluationRequest ||
		document.SchemaVersion != SchemaVersionV2 ||
		!evaluationIDPattern.MatchString(document.EvaluationID) {
		return fmt.Errorf("candidate evaluation request の版または ID が不正です")
	}
	if !evaluatorVersionPattern.MatchString(document.EvaluatorVersion) || len(document.EvaluatorVersion) > 64 {
		return fmt.Errorf("evaluatorVersion が不正です")
	}
	if err := validateRequestCorpus(document); err != nil {
		return err
	}
	if !candidateContentIDPattern.MatchString(document.CandidateContentID) {
		return fmt.Errorf("candidateContentId が不正です")
	}
	if err := validateSHA256("candidateContentManifestSha256", document.CandidateContentManifestSHA256); err != nil {
		return err
	}
	if document.ReviewRubricVersion != ReviewRubricVersion ||
		document.ReviewRubricSHA256 != ReviewRubricSHA256() {
		return fmt.Errorf("request の review rubric が不正です")
	}
	if err := validateSOTReferences(document.RequiredReviewSOTs, true); err != nil {
		return err
	}
	if document.RequiredReviewSOTSetSHA256 != SOTSetSHA256(document.RequiredReviewSOTs) {
		return fmt.Errorf("requiredReviewSOTSetSha256 が SOT 集合と一致しません")
	}
	if err := validateAttestationReferences(document.ReviewAttestations); err != nil {
		return err
	}
	if !baselineVersionPattern.MatchString(document.BaselineVersion) || len(document.BaselineVersion) > 64 {
		return fmt.Errorf("baselineVersion が不正です")
	}
	expectedID, err := CanonicalEvaluationID(document)
	if err != nil || document.EvaluationID != expectedID {
		return fmt.Errorf("evaluationId が canonical tuple digest と一致しません")
	}
	return nil
}

func validateRequestCorpus(document EvaluationRequest) error {
	if err := validateMachineString("corpusVersion", document.CorpusVersion, 64); err != nil {
		return err
	}
	if err := validateSHA256("corpusManifestSha256", document.CorpusManifestSHA256); err != nil {
		return err
	}
	if err := validateSHA256("holdoutDigest", document.HoldoutDigest); err != nil {
		return err
	}
	if err := validateSortedUniqueStrings(
		"holdoutLeakageGroupDigests",
		document.HoldoutLeakageGroupDigests,
		1,
		400,
	); err != nil {
		return err
	}
	for _, digest := range document.HoldoutLeakageGroupDigests {
		if err := validateSHA256("holdout leakage group digest", digest); err != nil {
			return err
		}
	}
	return nil
}

func validateAttestationReferences(references []ReviewAttestationReference) error {
	if references == nil || len(references) != 2 {
		return fmt.Errorf("reviewAttestations は二件必要です")
	}
	expectedScopes := []string{ReviewScopeArchitecture, ReviewScopeTestability}
	for index, reference := range references {
		if reference.ReviewScope != expectedScopes[index] ||
			!reviewAttestationIDPattern.MatchString(reference.AttestationID) {
			return fmt.Errorf("reviewAttestations の scope、順序または ID が不正です")
		}
		if err := validateSHA256("attestationSha256", reference.AttestationSHA256); err != nil {
			return err
		}
	}
	return nil
}
