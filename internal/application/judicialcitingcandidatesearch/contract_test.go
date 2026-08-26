package judicialcitingcandidatesearch

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestArgumentErrorExposesOnlySafeClassification(t *testing.T) {
	t.Parallel()

	err := newArgumentError("limit", "は 1 以上 10 以下でなければなりません")
	if err.Code() != model.ErrorCodeInvalidArgument || err.Field() != "limit" ||
		err.Reason() == "" || err.Error() == "" || err.Validate() != nil {
		t.Fatalf("ArgumentError = %#v", err)
	}
	for _, invalid := range []ArgumentError{
		{},
		newArgumentError("unknown", "使用できません"),
		newArgumentError("limit", "改行\nは禁止です"),
		newArgumentError("limit", strings.Repeat("長", 100)),
	} {
		if invalid.Validate() == nil {
			t.Fatalf("不正な ArgumentError を受理した: %#v", invalid)
		}
	}
}

func TestCandidateJSONCopiesEvidenceAndRejectsUnsafeVariants(t *testing.T) {
	t.Parallel()

	candidate := newTestCandidate(t)
	evidence := candidate.Evidence()
	if len(evidence) != 1 || candidate.Decision().Ref().Key().ResourceID() != "00456/detail3" {
		t.Fatal("Candidate accessor が有効ではありません")
	}
	evidence[0] = model.JudicialCitationEvidence{}
	if candidate.Evidence()[0].EvidenceLevel() != model.JudicialCitationEvidenceLevelOfficialSearchCandidate {
		t.Fatal("evidence が外部から変更されました")
	}
	encoded, err := json.Marshal(candidate)
	if err != nil || strings.Contains(string(encoded), "query1") {
		t.Fatalf("Candidate JSON = %s, %v", encoded, err)
	}
	var decoded Candidate
	if json.Unmarshal(encoded, &decoded) == nil || decoded.Validate() == nil {
		t.Fatal("Candidate の直接復元またはゼロ値を受理しました")
	}
	if _, err := NewCandidate(CandidateValues{Decision: candidate.Decision()}); err == nil {
		t.Fatal("evidence のない Candidate を受理しました")
	}

	metadata, err := model.NewJudicialCitationEvidence(model.JudicialCitationEvidenceValues{
		EvidenceLevel: model.JudicialCitationEvidenceLevelOfficialMetadata,
		Provenance:    candidate.Evidence()[0].Provenance(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewCandidate(CandidateValues{
		Decision: candidate.Decision(),
		Evidence: []model.JudicialCitationEvidence{metadata},
	}); err == nil {
		t.Fatal("official_metadata を候補根拠として受理しました")
	}
}

func TestCoverageContractAndJSON(t *testing.T) {
	t.Parallel()

	caseAttempt := mustCoverageAttempt(t, SearchKindCaseNumber, AttemptStatusComplete)
	reporterAttempt := mustCoverageAttempt(t, SearchKindReporterCitation, AttemptStatusFailed)
	coverage, err := NewCoverage(CoverageValues{
		Attempts:              []CoverageAttempt{caseAttempt, reporterAttempt},
		ObservedItemCount:     3,
		DeduplicatedItemCount: 2,
		Truncated:             true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(coverage.Attempts()) != 2 || coverage.ObservedItemCount() != 3 ||
		coverage.DedupedCandidateCount() != 2 || !coverage.Truncated() {
		t.Fatalf("Coverage = %#v", coverage)
	}
	attempts := coverage.Attempts()
	attempts[0] = CoverageAttempt{}
	if coverage.Attempts()[0].SearchKind != SearchKindCaseNumber {
		t.Fatal("attempts が外部から変更されました")
	}
	encoded, err := json.Marshal(coverage)
	if err != nil || !strings.Contains(string(encoded), `"dedupedCandidateCount":2`) {
		t.Fatalf("Coverage JSON = %s, %v", encoded, err)
	}
	var decoded Coverage
	if json.Unmarshal(encoded, &decoded) == nil || decoded.Validate() == nil {
		t.Fatal("Coverage の直接復元またはゼロ値を受理しました")
	}

	invalidValues := []CoverageValues{
		{},
		{Attempts: []CoverageAttempt{reporterAttempt}},
		{Attempts: []CoverageAttempt{caseAttempt, caseAttempt}},
		{Attempts: []CoverageAttempt{caseAttempt}, ObservedItemCount: -1},
		{Attempts: []CoverageAttempt{caseAttempt}, ObservedItemCount: 1, DedupedCandidateCount: 2},
		{Attempts: []CoverageAttempt{caseAttempt}, ObservedItemCount: 2, DedupedCandidateCount: 1, DeduplicatedItemCount: 2},
	}
	for _, values := range invalidValues {
		if _, err := NewCoverage(values); err == nil {
			t.Fatalf("不正な Coverage を受理しました: %#v", values)
		}
	}
	for _, values := range []CoverageAttemptValues{
		{SearchKind: "other", Status: AttemptStatusComplete},
		{SearchKind: SearchKindCaseNumber, Status: "other"},
	} {
		if _, err := NewCoverageAttempt(values); err == nil {
			t.Fatalf("不正な CoverageAttempt を受理しました: %#v", values)
		}
	}
}

func TestRequestAccessorsAndDirectRestore(t *testing.T) {
	t.Parallel()

	target := newTestTargetResource(t)
	request, err := NewRequest(RequestValues{Target: target})
	if err != nil {
		t.Fatal(err)
	}
	if request.Target().Ref() != target.Ref() || request.Validate() != nil {
		t.Fatal("Request accessor が入力を保持していません")
	}
	var decoded Request
	if json.Unmarshal([]byte(`{"limit":5}`), &decoded) == nil || decoded.Validate() == nil {
		t.Fatal("Request の直接復元またはゼロ値を受理しました")
	}
	invalid := Request{target: model.SourcedResource[model.JudicialDecisionDetails]{}, limit: 5, initialized: true}
	if invalid.Validate() == nil {
		t.Fatal("無効な target を受理しました")
	}
}

func TestIssueAndResultJSONContract(t *testing.T) {
	t.Parallel()

	issue, err := NewIssue(IssueValues{
		SearchKind:  SearchKindReporterCitation,
		SourceError: mustCandidateSourceError(t, model.SourceErrorCodeSourceUnavailable),
	})
	if err != nil {
		t.Fatal(err)
	}
	if issue.SearchKind() != SearchKindReporterCitation ||
		issue.ErrorResult().Code() != model.ErrorCodeSourceUnavailable ||
		issue.Validate() != nil {
		t.Fatalf("Issue = %#v", issue)
	}
	encodedIssue, err := json.Marshal(issue)
	if err != nil || strings.Contains(string(encodedIssue), "query1") {
		t.Fatalf("Issue JSON = %s, %v", encodedIssue, err)
	}
	var decodedIssue Issue
	if json.Unmarshal(encodedIssue, &decodedIssue) == nil || decodedIssue.Validate() == nil {
		t.Fatal("Issue の直接復元またはゼロ値を受理しました")
	}

	coverage, err := NewCoverage(CoverageValues{
		Attempts: []CoverageAttempt{
			mustCoverageAttempt(t, SearchKindCaseNumber, AttemptStatusComplete),
			mustCoverageAttempt(t, SearchKindReporterCitation, AttemptStatusFailed),
		},
		ObservedItemCount:     1,
		DedupedCandidateCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewResult(ResultValues{
		Status:   SearchStatusPartial,
		Items:    []Candidate{newTestCandidate(t)},
		Coverage: coverage,
		Issues:   []Issue{issue},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items()) != 1 || len(result.Issues()) != 1 ||
		result.Coverage().ObservedItemCount() != 1 || result.Validate() != nil {
		t.Fatalf("Result = %#v", result)
	}
	items := result.Items()
	items[0] = Candidate{}
	if result.Items()[0].Validate() != nil {
		t.Fatal("items が外部から変更されました")
	}
	encoded, err := json.Marshal(result)
	if err != nil || !strings.Contains(string(encoded), `"status":"partial"`) {
		t.Fatalf("Result JSON = %s, %v", encoded, err)
	}
	var decoded Result
	if json.Unmarshal(encoded, &decoded) == nil || decoded.Validate() == nil {
		t.Fatal("Result の直接復元またはゼロ値を受理しました")
	}
}

func TestIssueAndResultRejectImpossibleStates(t *testing.T) {
	t.Parallel()

	for _, values := range []IssueValues{
		{
			SearchKind:  "other",
			SourceError: mustCandidateSourceError(t, model.SourceErrorCodeSourceUnavailable),
		},
		{SearchKind: SearchKindCaseNumber},
		{
			SearchKind:  SearchKindCaseNumber,
			SourceError: mustCandidateSourceError(t, model.SourceErrorCodeSourceAuthFailed),
		},
		{
			SearchKind: SearchKindCaseNumber,
			SourceError: mustTestSourceError(
				t,
				model.SourceErrorCodeSourceUnavailable,
				"judicial-decision.search",
			),
		},
	} {
		if _, err := NewIssue(values); err == nil {
			t.Fatalf("不正な Issue を受理しました: %#v", values)
		}
	}

	failedCase := mustCoverageAttempt(t, SearchKindCaseNumber, AttemptStatusFailed)
	failedReporter := mustCoverageAttempt(t, SearchKindReporterCitation, AttemptStatusFailed)
	allFailed, err := NewCoverage(CoverageValues{Attempts: []CoverageAttempt{failedCase, failedReporter}})
	if err != nil {
		t.Fatal(err)
	}
	issue := mustIssue(t, SearchKindCaseNumber)
	reporterIssue := mustIssue(t, SearchKindReporterCitation)
	if _, err := NewResult(ResultValues{
		Status: SearchStatusPartial, Coverage: allFailed, Issues: []Issue{issue, reporterIssue},
	}); err == nil {
		t.Fatal("全検索失敗から Result を作成しました")
	}

	completeCoverage, err := NewCoverage(CoverageValues{
		Attempts:          []CoverageAttempt{mustCoverageAttempt(t, SearchKindCaseNumber, AttemptStatusComplete)},
		ObservedItemCount: 2, DedupedCandidateCount: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewResult(ResultValues{
		Status: SearchStatusComplete, Items: []Candidate{newTestCandidate(t)}, Coverage: completeCoverage,
	}); err == nil {
		t.Fatal("未返却候補があるのに truncated=false を受理しました")
	}
}

func TestIssueRequiresCompleteMatchingSourceDetails(t *testing.T) {
	t.Parallel()

	valid := mustIssue(t, SearchKindCaseNumber)
	sourceError := mustCandidateSourceError(t, model.SourceErrorCodeSourceUnavailable)
	for name, details := range map[string]map[string]any{
		"details なし": nil,
		"providerId なし": {
			"sourceId":     sourceError.SourceID(),
			"capabilityId": sourceError.CapabilityID(),
		},
		"sourceId なし": {
			"providerId":   sourceError.ProviderID(),
			"capabilityId": sourceError.CapabilityID(),
		},
		"capabilityId なし": {
			"providerId": sourceError.ProviderID(),
			"sourceId":   sourceError.SourceID(),
		},
		"providerId 不一致": {
			"providerId":   "other-provider",
			"sourceId":     sourceError.SourceID(),
			"capabilityId": sourceError.CapabilityID(),
		},
		"sourceId 不一致": {
			"providerId":   sourceError.ProviderID(),
			"sourceId":     "other-source",
			"capabilityId": sourceError.CapabilityID(),
		},
		"capabilityId 不一致": {
			"providerId":   sourceError.ProviderID(),
			"sourceId":     sourceError.SourceID(),
			"capabilityId": "judicial-decision.search",
		},
	} {
		name := name
		details := details
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			errorResult, err := model.NewErrorResult(model.ErrorResultValues{
				Code:    model.ErrorCodeSourceUnavailable,
				Details: details,
			})
			if err != nil {
				t.Fatal(err)
			}
			invalid := valid
			invalid.errorResult = errorResult
			if invalid.Validate() == nil {
				t.Fatal("SourceError と一致しない details を受理しました")
			}
		})
	}
}

func mustCoverageAttempt(t *testing.T, kind SearchKind, status AttemptStatus) CoverageAttempt {
	t.Helper()
	attempt, err := NewCoverageAttempt(CoverageAttemptValues{SearchKind: kind, Status: status})
	if err != nil {
		t.Fatal(err)
	}
	return attempt
}

func mustIssue(t *testing.T, kind SearchKind) Issue {
	t.Helper()
	issue, err := NewIssue(IssueValues{
		SearchKind:  kind,
		SourceError: mustCandidateSourceError(t, model.SourceErrorCodeSourceUnavailable),
	})
	if err != nil {
		t.Fatal(err)
	}
	return issue
}
