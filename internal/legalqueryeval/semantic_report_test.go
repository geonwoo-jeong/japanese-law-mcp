package legalqueryeval

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestNewSemanticReportは指標別とcategory別の失敗CaseをJSON化する(
	t *testing.T,
) {
	evaluations := []SemanticCaseEvaluation{
		{
			caseID:             "holdout-intent-01",
			categoryIDs:        []string{"capability-intent"},
			coverageIDs:        []string{"intent-law-search"},
			expectedKind:       legalquerycorpus.SemanticExpectedKindPlan,
			planOutcomeMatched: true,
			rankingApplicable:  true,
			primaryTop1Matched: true,
			primaryTop2Matched: true,
			highConfidence:     comparisonAssertion{matched: true, applicable: true},
			meanings:           []MeaningEvaluation{matchedMeaning("meaning-intent", true)},
			initialized:        true,
		},
		{
			caseID:             "holdout-name-01",
			categoryIDs:        []string{"ambiguity", "law-name-and-concept"},
			coverageIDs:        []string{"ambiguity-multiple-concepts", "concept-single"},
			expectedKind:       legalquerycorpus.SemanticExpectedKindPlan,
			planOutcomeMatched: true,
			rankingApplicable:  true,
			primaryTop1Matched: false,
			primaryTop2Matched: true,
			highConfidence:     comparisonAssertion{matched: false, applicable: true},
			meanings:           []MeaningEvaluation{matchedMeaning("meaning-name", false)},
			initialized:        true,
		},
		{
			caseID:              "holdout-input-01",
			categoryIDs:         []string{"input-boundary"},
			coverageIDs:         []string{"input-query-empty"},
			expectedKind:        legalquerycorpus.SemanticExpectedKindRequestError,
			requestErrorMatched: false,
			initialized:         true,
		},
	}
	metrics, err := AggregateSemanticCaseEvaluations(evaluations)
	if err != nil {
		t.Fatalf("試験用 metrics を集計できません: %v", err)
	}

	report, err := NewSemanticReport(SemanticHoldoutResult{
		evaluations: evaluations,
		metrics:     metrics,
	})
	if err != nil {
		t.Fatalf("NewSemanticReport() error = %v", err)
	}
	if report.CaseCount() != 3 {
		t.Fatalf("case count = %d, want 3", report.CaseCount())
	}
	if got := report.FailedCaseIDs(); !slices.Equal(
		got,
		[]string{"holdout-name-01", "holdout-input-01"},
	) {
		t.Fatalf("failed case IDs = %#v", got)
	}

	top1 := findSemanticMetricReport(t, report.Metrics(), SemanticMetricTop1)
	if top1.Matched() != 1 ||
		top1.Total() != 2 ||
		top1.Ratio() != 0.5 ||
		!slices.Equal(top1.FailedCaseIDs(), []string{"holdout-name-01"}) {
		t.Fatalf("top-1 report = %#v", top1)
	}
	requestError := findSemanticMetricReport(
		t,
		report.Metrics(),
		SemanticMetricRequestError,
	)
	if requestError.Matched() != 0 ||
		requestError.Total() != 1 ||
		!slices.Equal(requestError.FailedCaseIDs(), []string{"holdout-input-01"}) {
		t.Fatalf("request-error report = %#v", requestError)
	}

	categories := report.Categories()
	if len(categories) != 4 ||
		categories[0].CategoryID() != "ambiguity" ||
		categories[1].CategoryID() != "capability-intent" ||
		categories[2].CategoryID() != "input-boundary" ||
		categories[3].CategoryID() != "law-name-and-concept" {
		t.Fatalf("category order = %#v", categories)
	}
	nameTop1 := findSemanticMetricReport(
		t,
		categories[3].Metrics(),
		SemanticMetricTop1,
	)
	if !slices.Equal(nameTop1.FailedCaseIDs(), []string{"holdout-name-01"}) {
		t.Fatalf("law-name-and-concept top-1 failures = %#v", nameTop1.FailedCaseIDs())
	}

	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("semantic report を JSON 化できません: %v", err)
	}
	var document struct {
		CaseCount     int      `json:"caseCount"`
		FailedCaseIDs []string `json:"failedCaseIds"`
		Metrics       []struct {
			MetricID     SemanticMetricID `json:"metricId"`
			Matched      int              `json:"matched"`
			Total        int              `json:"total"`
			Ratio        float64          `json:"ratio"`
			FailedCaseID []string         `json:"failedCaseIds"`
		} `json:"metrics"`
		Categories []json.RawMessage `json:"categories"`
	}
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("semantic report JSON を復元できません: %v", err)
	}
	if document.CaseCount != 3 ||
		len(document.Metrics) != len(semanticMetricIDs()) ||
		len(document.Categories) != 4 ||
		!slices.Equal(
			document.FailedCaseIDs,
			[]string{"holdout-name-01", "holdout-input-01"},
		) {
		t.Fatalf("semantic report JSON = %s", encoded)
	}

	failed := report.FailedCaseIDs()
	failed[0] = "changed"
	if report.FailedCaseIDs()[0] != "holdout-name-01" {
		t.Fatal("FailedCaseIDs() の戻り値から report が変更されました")
	}
}

func matchedMeaning(
	meaningID string,
	conceptMatched bool,
) MeaningEvaluation {
	return MeaningEvaluation{
		meaningID:            meaningID,
		matchedCandidateRank: 1,
		signatureMatched:     true,
		evidence: comparisonAssertion{
			matched:    true,
			applicable: true,
		},
		concept: comparisonAssertion{
			matched:    conceptMatched,
			applicable: true,
		},
	}
}

func findSemanticMetricReport(
	t *testing.T,
	reports []SemanticMetricReport,
	metricID SemanticMetricID,
) SemanticMetricReport {
	t.Helper()

	for _, report := range reports {
		if report.MetricID() == metricID {
			return report
		}
	}
	t.Fatalf("metric report %q がありません", metricID)
	return SemanticMetricReport{}
}
