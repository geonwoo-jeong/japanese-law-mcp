package judicialcitingcandidatesearch

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestNewResult(t *testing.T) {
	t.Parallel()

	candidate := newTestCandidate(t)
	caseAttempt, err := NewCoverageAttempt(CoverageAttemptValues{
		SearchKind: SearchKindCaseNumber,
		Status:     AttemptStatusComplete,
	})
	if err != nil {
		t.Fatalf("attempt: %v", err)
	}
	coverage, err := NewCoverage(CoverageValues{
		Attempts:              []CoverageAttempt{caseAttempt},
		ObservedItemCount:     1,
		DeduplicatedItemCount: 1,
		Truncated:             false,
	})
	if err != nil {
		t.Fatalf("coverage: %v", err)
	}
	result, err := NewResult(ResultValues{
		Status:   SearchStatusComplete,
		Items:    []Candidate{candidate},
		Coverage: coverage,
		Issues:   []Issue{},
	})
	if err != nil {
		t.Fatalf("SOT-IF-069: NewResult() のエラー = %v", err)
	}
	if result.Status() != SearchStatusComplete {
		t.Fatalf("status = %q", result.Status())
	}
}

func TestResultRejectsInvalidStatusCombination(t *testing.T) {
	t.Parallel()

	attempt, err := NewCoverageAttempt(CoverageAttemptValues{
		SearchKind: SearchKindCaseNumber,
		Status:     AttemptStatusComplete,
	})
	if err != nil {
		t.Fatalf("attempt: %v", err)
	}
	coverage, err := NewCoverage(CoverageValues{
		Attempts:              []CoverageAttempt{attempt},
		ObservedItemCount:     0,
		DeduplicatedItemCount: 0,
	})
	if err != nil {
		t.Fatalf("coverage: %v", err)
	}
	issue, err := NewIssue(IssueValues{
		SearchKind:  SearchKindCaseNumber,
		SourceError: mustCandidateSourceError(t, model.SourceErrorCodeSourceUnavailable),
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := NewResult(ResultValues{
		Status:   SearchStatusComplete,
		Items:    []Candidate{},
		Coverage: coverage,
		Issues:   []Issue{issue},
	}); err == nil {
		t.Fatal("complete なのに issue ありを受理しました")
	}
	if _, err := NewResult(ResultValues{
		Status:   SearchStatusPartial,
		Items:    []Candidate{},
		Coverage: coverage,
		Issues:   []Issue{},
	}); err == nil {
		t.Fatal("partial なのに issue なしを受理しました")
	}
}

func newTestCandidate(t *testing.T) Candidate {
	t.Helper()

	target := newTestTargetResource(t)
	summary := target.Data().Summary()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   "00456/detail3",
	})
	if err != nil {
		t.Fatalf("candidate key: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("candidate ref: %v", err)
	}
	decisionDate := summary.DecisionDate()
	candidateSummary, err := model.NewJudicialDecisionSummary(model.JudicialDecisionSummaryValues{
		DecisionID:          "00456",
		PublicationCategory: model.JudicialPublicationCategoryHighCourt,
		SourceCategoryLabel: "高裁判例",
		CaseNumber:          "令和5(ネ)99",
		DecisionDate:        decisionDate,
		CourtName:           "東京高等裁判所",
		DetailURL:           "https://www.courts.go.jp/hanrei/00456/detail3/index.html",
		Documents:           []model.JudicialDocumentLink{},
		Source:              summary.Source(),
	})
	if err != nil {
		t.Fatalf("candidate summary: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         summary.Source(),
		ResourceKey:    key,
		URL:            "https://www.courts.go.jp/hanrei/search1/index.html",
		RetrievedAt:    target.Provenance()[0].RetrievedAt(),
		MediaType:      "text/html",
		Location:       "/html/body/table/tbody/tr[1]",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-073",
	})
	if err != nil {
		t.Fatalf("candidate provenance: %v", err)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[model.JudicialDecisionSummary]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       candidateSummary,
	})
	if err != nil {
		t.Fatalf("candidate resource: %v", err)
	}
	evidence, err := model.NewJudicialCitationEvidence(model.JudicialCitationEvidenceValues{
		EvidenceLevel: model.JudicialCitationEvidenceLevelOfficialSearchCandidate,
		Provenance:    provenance,
	})
	if err != nil {
		t.Fatalf("candidate evidence: %v", err)
	}
	candidate, err := NewCandidate(CandidateValues{
		Decision: resource,
		Evidence: []model.JudicialCitationEvidence{evidence},
	})
	if err != nil {
		t.Fatalf("candidate: %v", err)
	}
	return candidate
}
