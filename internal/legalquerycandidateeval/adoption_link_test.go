package legalquerycandidateeval

import "testing"

const verificationAdoptionLink = "candidate-evaluation-adoption-link"

func TestAdoptionLinkはPassedResultと同じ候補BaselineEvaluatorだけを許す(t *testing.T) {
	t.Parallel()

	request := validEvaluationRequest(t, manifestWithID(t))
	result := mustSyntheticEvaluationResult(
		t,
		request,
		mustCanonicalJSON(t, request),
		syntheticEvaluationReportRaw("adoption-passed"),
		EvaluationOutcomePassed,
	)
	link := AdoptionCandidateLink{
		CandidateContentID:             request.CandidateContentID,
		CandidateContentManifestSHA256: request.CandidateContentManifestSHA256,
		EvaluatorVersion:               request.EvaluatorVersion,
		BaselineVersion:                request.BaselineVersion,
		BaselineSHA256:                 result.ReportSHA256,
	}
	if err := VerifyAdoptionLink(request, result, link); err != nil {
		t.Fatalf("%s: 完全一致する passed handoff を拒否しました: %v", verificationAdoptionLink, err)
	}

	tests := []struct {
		name   string
		mutate func(*AdoptionCandidateLink)
	}{
		{name: "candidate-content", mutate: func(value *AdoptionCandidateLink) {
			value.CandidateContentID = "candidate-content-sha256-" + repeatHex('0')
		}},
		{name: "candidate-manifest", mutate: func(value *AdoptionCandidateLink) { value.CandidateContentManifestSHA256 = repeatHex('1') }},
		{name: "evaluator", mutate: func(value *AdoptionCandidateLink) { value.EvaluatorVersion = "legal-query-evaluator-v2" }},
		{name: "baseline-version", mutate: func(value *AdoptionCandidateLink) { value.BaselineVersion = "default-3" }},
		{name: "baseline-digest", mutate: func(value *AdoptionCandidateLink) { value.BaselineSHA256 = repeatHex('2') }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := link
			test.mutate(&changed)
			if err := VerifyAdoptionLink(request, result, changed); err == nil {
				t.Fatalf("%s: %s の不一致を受理しました", verificationAdoptionLink, test.name)
			}
		})
	}
}

func TestAdoptionLinkはFailedResultとRequestBinding不一致を拒否する(t *testing.T) {
	t.Parallel()

	request := validEvaluationRequest(t, manifestWithID(t))
	failed := mustSyntheticEvaluationResult(
		t,
		request,
		mustCanonicalJSON(t, request),
		syntheticEvaluationReportRaw("adoption-failed"),
		EvaluationOutcomeFailed,
	)
	link := AdoptionCandidateLink{
		CandidateContentID:             request.CandidateContentID,
		CandidateContentManifestSHA256: request.CandidateContentManifestSHA256,
		EvaluatorVersion:               request.EvaluatorVersion,
		BaselineVersion:                request.BaselineVersion,
		BaselineSHA256:                 failed.ReportSHA256,
	}
	if err := VerifyAdoptionLink(request, failed, link); err == nil {
		t.Fatalf("%s: failed result を採用可能として受理しました", verificationAdoptionLink)
	}

	passed := mustSyntheticEvaluationResult(
		t,
		request,
		mustCanonicalJSON(t, request),
		syntheticEvaluationReportRaw("adoption-binding"),
		EvaluationOutcomePassed,
	)
	link.BaselineSHA256 = passed.ReportSHA256
	passed.RequestSHA256 = repeatHex('f')
	if err := VerifyAdoptionLink(request, passed, link); err == nil {
		t.Fatalf("%s: requestSha256 が不一致な result を受理しました", verificationAdoptionLink)
	}
}
