package legalquerycandidateeval

import "fmt"

func validateReviewAttestation(document ReviewAttestation) error {
	if document.ArtifactKind != ArtifactKindReviewAttestation ||
		!isSupportedSchemaVersion(document.SchemaVersion) ||
		!reviewAttestationIDPattern.MatchString(document.AttestationID) ||
		!candidateContentIDPattern.MatchString(document.CandidateContentID) {
		return fmt.Errorf("review attestation の版または ID が不正です")
	}
	if err := validateSHA256("candidateContentManifestSha256", document.CandidateContentManifestSHA256); err != nil {
		return err
	}
	if document.ReviewScope != ReviewScopeArchitecture && document.ReviewScope != ReviewScopeTestability {
		return fmt.Errorf("reviewScope が不正です")
	}
	if document.RubricVersion != ReviewRubricVersion || document.RubricSHA256 != ReviewRubricSHA256() {
		return fmt.Errorf("review rubric の版または digest が不正です")
	}
	if !authorityIDPattern.MatchString(document.ReviewerAuthorityID) {
		return fmt.Errorf("reviewerAuthorityId が不正です")
	}
	if err := validateSOTReferences(document.ReviewedSOTs, document.SchemaVersion, true); err != nil {
		return err
	}
	if document.ReviewedSOTSetSHA256 != SOTSetSHA256(document.ReviewedSOTs) {
		return fmt.Errorf("reviewedSOTSetSha256 が SOT 集合と一致しません")
	}
	if err := validateCriterionScores(document); err != nil {
		return err
	}
	if document.BlockerCount != 0 || document.MajorCount < 0 || document.MinorCount < 0 ||
		document.Decision != ReviewDecisionApproved {
		return fmt.Errorf("review の finding count または decision が不正です")
	}
	expectedID, err := CanonicalReviewAttestationID(document)
	if err != nil || document.AttestationID != expectedID {
		return fmt.Errorf("attestationId が canonical tuple digest と一致しません")
	}
	return nil
}

func validateSOTReferences(
	references []SOTReference,
	schemaVersion int,
	requireExact bool,
) error {
	//nolint:staticcheck // SOT-ENG-038: JSON null と空配列を別状態として閉じて検証する。
	if references == nil || len(references) < 1 || len(references) > 128 {
		return fmt.Errorf("review SOT 集合の件数が不正です")
	}
	expected, err := RequiredReviewSOTIDsForSchema(schemaVersion)
	if err != nil {
		return err
	}
	if requireExact && len(references) != len(expected) {
		return fmt.Errorf("review SOT 集合が schema version %d の固定集合と一致しません", schemaVersion)
	}
	previous := ""
	for index, reference := range references {
		if index > 0 && previous >= reference.SOTID {
			return fmt.Errorf("review SOT 集合は sotId の byte 昇順でなければなりません")
		}
		if requireExact && reference.SOTID != expected[index] {
			return fmt.Errorf("review SOT 集合が schema version %d の固定集合と一致しません", schemaVersion)
		}
		if err := validateSHA256("sotDocumentSha256", reference.SOTDocumentSHA256); err != nil {
			return err
		}
		previous = reference.SOTID
	}
	return nil
}

func validateCriterionScores(document ReviewAttestation) error {
	expected := ArchitectureCriterionIDs()
	if document.ReviewScope == ReviewScopeTestability {
		expected = TestabilityCriterionIDs()
	}
	if document.CriterionScores == nil || len(document.CriterionScores) != len(expected) {
		return fmt.Errorf("criterionScores の件数が不正です")
	}
	total := 0
	for index, score := range document.CriterionScores {
		if score.CriterionID != expected[index] || !allowedCriterionScore(score.Score20) || score.Score20 < 16 {
			return fmt.Errorf("criterionScores の ID、順序または approved score が不正です")
		}
		total += score.Score20
	}
	if document.Score100 != total || document.Score100 < 80 || document.Score100 > 100 {
		return fmt.Errorf("score100 が criterionScores の和または許可範囲と一致しません")
	}
	return nil
}

func allowedCriterionScore(score int) bool {
	return score == 0 || score == 10 || score == 16 || score == 20
}
