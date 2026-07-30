package legalqueryeval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestStandardReportは固定スキーマだけを決定的にJSON化する(t *testing.T) {
	t.Parallel()

	report := mustStandardReport(t, 1, []ExecutionCaseEvaluation{
		mustExecutionCaseEvaluation(t, "execution-empty"),
	})
	first, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("SOT-ENG-024: standard report を JSON 化できません: %v", err)
	}
	second, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("SOT-ENG-024: standard report を再度 JSON 化できません: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("SOT-ENG-024: JSON が非決定的です:\n%s\n%s", first, second)
	}
	if strings.Contains(string(first), "照会本文") {
		t.Fatalf("SOT-ENG-024: report に照会本文が含まれました: %s", first)
	}

	var document struct {
		ArtifactKind    string `json:"artifactKind"`
		SchemaVersion   int    `json:"schemaVersion"`
		CorpusVersion   string `json:"corpusVersion"`
		HoldoutDigest   string `json:"holdoutDigest"`
		BaselineVersion string `json:"baselineVersion"`
		ProfileSet      struct {
			ProfileSetID      string `json:"profileSetId"`
			ProfileSetVersion string `json:"profileSetVersion"`
			RankingVersion    string `json:"rankingVersion"`
			Profiles          []struct {
				ProfileID      string `json:"profileId"`
				ProfileVersion string `json:"profileVersion"`
			} `json:"profiles"`
		} `json:"profileSet"`
		Sets struct {
			Development struct {
				CaseCount int `json:"caseCount"`
			} `json:"development"`
			Holdout struct {
				CaseCount int `json:"caseCount"`
				Metrics   []struct {
					MetricID    string `json:"metricId"`
					Numerator   int    `json:"numerator"`
					Denominator int    `json:"denominator"`
				} `json:"metrics"`
			} `json:"holdout"`
			Execution struct {
				CaseCount int `json:"caseCount"`
			} `json:"execution"`
		} `json:"sets"`
	}
	if err := json.Unmarshal(first, &document); err != nil {
		t.Fatalf("SOT-ENG-024: standard report JSON を確認できません: %v", err)
	}
	if document.ArtifactKind != "legal_query_evaluation" ||
		document.SchemaVersion != 1 ||
		document.CorpusVersion != "corpus-v4" ||
		document.ProfileSet.ProfileSetID != "default" ||
		document.ProfileSet.ProfileSetVersion != "profile-set-v1" ||
		document.ProfileSet.RankingVersion != "legal-query-ranking-2026-07-28-1" ||
		document.BaselineVersion != "default-1" ||
		len(document.ProfileSet.Profiles) != 2 {
		t.Fatalf("SOT-ENG-024: standard report JSON = %s", first)
	}
	if document.Sets.Development.CaseCount != 1 ||
		document.Sets.Holdout.CaseCount != 1 ||
		document.Sets.Execution.CaseCount != 1 ||
		len(document.Sets.Holdout.Metrics) != len(semanticMetricIDs())+1 ||
		document.Sets.Holdout.Metrics[0].MetricID != "plan-reproducibility" ||
		document.Sets.Holdout.Metrics[0].Numerator != 1 ||
		document.Sets.Holdout.Metrics[0].Denominator != 1 {
		t.Fatalf("SOT-ENG-024: 件数または再現性 = %s", first)
	}
}

func TestEvaluationMetricReportは全不一致CaseIDを必要とする(t *testing.T) {
	t.Parallel()

	if _, err := NewEvaluationMetricReport(
		EvaluationMetricReportValues{
			MetricID:    "plan-outcome",
			Numerator:   0,
			Denominator: 1,
		},
	); err == nil {
		t.Fatal("SOT-ENG-024: 不一致 case ID の欠落を受理しました")
	}
}

func TestStandardReportの再現性母集団はPlanCaseだけを数える(t *testing.T) {
	t.Parallel()

	semantic := perfectSemanticReport(1)
	semantic.caseCount = 2
	reproducibility, err := NewEvaluationMetricReport(
		EvaluationMetricReportValues{
			MetricID:    "plan-reproducibility",
			Numerator:   1,
			Denominator: 1,
		},
	)
	if err != nil {
		t.Fatalf("試験用 reproducibility を作成できません: %v", err)
	}
	execution, err := NewExecutionReport([]ExecutionCaseEvaluation{
		mustExecutionCaseEvaluation(t, "execution-empty"),
	})
	if err != nil {
		t.Fatalf("試験用 execution report を作成できません: %v", err)
	}
	_, err = NewStandardReport(StandardReportValues{
		CorpusVersion:     "corpus-v4",
		HoldoutDigest:     strings.Repeat("a", 64),
		ProfileSetID:      "default",
		ProfileSetVersion: "profile-set-v1",
		RankingVersion:    "legal-query-ranking-2026-07-28-1",
		ProfileVersions: []ProfileVersionReport{
			mustProfileVersionReport(
				t,
				"core",
				"core-2026-07-30-1",
				"legal-query-ranking-2026-07-28-1",
			),
		},
		BaselineVersion:      "default-1",
		DevelopmentCaseCount: 1,
		HoldoutCaseIDs:       []string{"holdout-input-01", "holdout-intent-01"},
		Semantic:             semantic,
		Execution:            execution,
		Reproducibility:      reproducibility,
		DerivedObservations:  perfectDerivedObservations(t),
	})
	if err != nil {
		t.Fatalf("SOT-ENG-024: request_error を再現性母集団から除外できません: %v", err)
	}
}

func perfectDerivedObservations(
	t *testing.T,
) []EvaluationMetricReport {
	t.Helper()

	observations := make(
		[]EvaluationMetricReport,
		0,
		len(derivedObservationIDs()),
	)
	for _, observationID := range derivedObservationIDs() {
		report, err := NewEvaluationMetricReport(
			EvaluationMetricReportValues{
				MetricID:    observationID,
				Numerator:   1,
				Denominator: 1,
			},
		)
		if err != nil {
			t.Fatalf("試験用 derived observation を作成できません: %v", err)
		}
		observations = append(observations, report)
	}
	return observations
}

func TestStandardReportの受入判定は高確信度の分母零と実行違反を拒否する(
	t *testing.T,
) {
	t.Parallel()

	zeroHigh := mustStandardReport(t, 0, []ExecutionCaseEvaluation{
		mustExecutionCaseEvaluation(t, "execution-empty"),
	})
	if err := VerifyStandardAcceptance(zeroHigh); err == nil ||
		!strings.Contains(err.Error(), "high-confidence") {
		t.Fatalf("SOT-ENG-024: high-confidence 分母零の error = %v", err)
	}

	violation, err := NewExecutionCaseEvaluation(
		ExecutionCaseEvaluationValues{
			CaseID:                 "execution-item-budget",
			ExpectedMatched:        false,
			WrongResourceCallCount: 1,
			BudgetViolationCount:   1,
		},
	)
	if err != nil {
		t.Fatalf("試験用 execution evaluation を作成できません: %v", err)
	}
	withViolation := mustStandardReport(t, 1, []ExecutionCaseEvaluation{violation})
	if err := VerifyStandardAcceptance(withViolation); err == nil ||
		!strings.Contains(err.Error(), "execution") {
		t.Fatalf("SOT-ENG-024: execution 違反の error = %v", err)
	}

	derivedViolation := mustStandardReport(t, 1, []ExecutionCaseEvaluation{
		mustExecutionCaseEvaluation(t, "execution-empty"),
	})
	failedObservation, err := NewEvaluationMetricReport(
		EvaluationMetricReportValues{
			MetricID:      derivedObservationCorePack,
			Numerator:     0,
			Denominator:   1,
			FailedCaseIDs: []string{"holdout-intent-01"},
		},
	)
	if err != nil {
		t.Fatalf("試験用 derived observation を作成できません: %v", err)
	}
	derivedViolation.sets.holdout.derivedObservations[0] = failedObservation
	derivedViolation.sets.holdout.failedCaseIDs = []string{"holdout-intent-01"}
	if err := VerifyStandardAcceptance(derivedViolation); err == nil ||
		!strings.Contains(err.Error(), "派生観測") {
		t.Fatalf("SOT-ENG-026: derived observation 違反の error = %v", err)
	}
}

func TestStandardBaselineは閉じたJSONを読んで完全一致だけを受理する(t *testing.T) {
	t.Parallel()

	report := mustStandardReport(t, 1, []ExecutionCaseEvaluation{
		mustExecutionCaseEvaluation(t, "execution-empty"),
	})
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("試験用 baseline を JSON 化できません: %v", err)
	}
	path := filepath.Join(t.TempDir(), "default.json")
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		t.Fatalf("試験用 baseline を書き込めません: %v", err)
	}

	baseline, err := LoadStandardBaseline(path)
	if err != nil {
		t.Fatalf("SOT-ENG-024: baseline を読み込めません: %v", err)
	}
	if err := CompareStandardBaseline(report, baseline); err != nil {
		t.Fatalf("SOT-ENG-024: 同一 baseline が不一致です: %v", err)
	}

	changedEvaluation, err := NewExecutionCaseEvaluation(
		ExecutionCaseEvaluationValues{
			CaseID:              "execution-empty",
			ExpectedMatched:     false,
			AttemptOrderMatched: true,
		},
	)
	if err != nil {
		t.Fatalf("不一致用 execution evaluation を作成できません: %v", err)
	}
	changed := mustStandardReport(
		t,
		1,
		[]ExecutionCaseEvaluation{changedEvaluation},
	)
	if err := CompareStandardBaseline(changed, baseline); err == nil {
		t.Fatal("SOT-ENG-024: 異なる評価結果を baseline 一致として受理しました")
	}

	unknown := append(
		append([]byte{}, encoded[:len(encoded)-1]...),
		[]byte(`,"unknown":true}`)...,
	)
	if err := os.WriteFile(path, unknown, 0o600); err != nil {
		t.Fatalf("未知項目付き baseline を書き込めません: %v", err)
	}
	if _, err := LoadStandardBaseline(path); err == nil {
		t.Fatal("SOT-ENG-024: 未知項目を持つ baseline を受理しました")
	}
}

func mustStandardReport(
	t *testing.T,
	highConfidenceTotal int,
	executionEvaluations []ExecutionCaseEvaluation,
) StandardReport {
	t.Helper()

	semantic := perfectSemanticReport(highConfidenceTotal)
	execution, err := NewExecutionReport(executionEvaluations)
	if err != nil {
		t.Fatalf("試験用 execution report を作成できません: %v", err)
	}
	profiles := []ProfileVersionReport{
		mustProfileVersionReport(
			t,
			"core",
			"core-2026-07-30-1",
			"legal-query-ranking-2026-07-28-1",
		),
		mustProfileVersionReport(
			t,
			"judicial-cases",
			"judicial-cases-2026-07-30-1",
			"legal-query-ranking-2026-07-28-1",
		),
	}
	reproducibility, err := NewEvaluationMetricReport(
		EvaluationMetricReportValues{
			MetricID:    "plan-reproducibility",
			Numerator:   semantic.CaseCount(),
			Denominator: semantic.CaseCount(),
		},
	)
	if err != nil {
		t.Fatalf("試験用 reproducibility を作成できません: %v", err)
	}
	report, err := NewStandardReport(StandardReportValues{
		CorpusVersion:        "corpus-v4",
		HoldoutDigest:        strings.Repeat("a", 64),
		ProfileSetID:         "default",
		ProfileSetVersion:    "profile-set-v1",
		RankingVersion:       "legal-query-ranking-2026-07-28-1",
		ProfileVersions:      profiles,
		BaselineVersion:      "default-1",
		DevelopmentCaseCount: 1,
		HoldoutCaseIDs:       []string{"holdout-intent-01"},
		Semantic:             semantic,
		Execution:            execution,
		Reproducibility:      reproducibility,
		DerivedObservations:  perfectDerivedObservations(t),
	})
	if err != nil {
		t.Fatalf("試験用 standard report を作成できません: %v", err)
	}
	return report
}

func perfectSemanticReport(highConfidenceTotal int) SemanticReport {
	metrics := make([]SemanticMetricReport, 0, len(semanticMetricIDs()))
	for _, metricID := range semanticMetricIDs() {
		total := 1
		if metricID == SemanticMetricRequestError {
			total = 0
		}
		if metricID == SemanticMetricHighConfidence {
			total = highConfidenceTotal
		}
		metrics = append(metrics, SemanticMetricReport{
			metricID: metricID,
			matched:  total,
			total:    total,
		})
	}
	categoryMetrics := make([]SemanticMetricReport, len(metrics))
	copy(categoryMetrics, metrics)
	return SemanticReport{
		caseCount: 1,
		metrics:   metrics,
		categories: []SemanticCategoryReport{{
			categoryID: "capability-intent",
			caseCount:  1,
			metrics:    categoryMetrics,
		}},
		failedCaseIDs: []string{},
		initialized:   true,
	}
}

func mustProfileVersionReport(
	t *testing.T,
	profileID string,
	profileVersion string,
	rankingVersion string,
) ProfileVersionReport {
	t.Helper()

	report, err := NewProfileVersionReport(ProfileVersionReportValues{
		ProfileID:      profileID,
		ProfileVersion: profileVersion,
		RankingVersion: rankingVersion,
	})
	if err != nil {
		t.Fatalf("試験用 profile version を作成できません: %v", err)
	}
	return report
}

func mustExecutionCaseEvaluation(
	t *testing.T,
	caseID string,
) ExecutionCaseEvaluation {
	t.Helper()

	evaluation, err := NewExecutionCaseEvaluation(
		ExecutionCaseEvaluationValues{
			CaseID:              caseID,
			ExpectedMatched:     true,
			AttemptOrderMatched: true,
		},
	)
	if err != nil {
		t.Fatalf("試験用 execution evaluation を作成できません: %v", err)
	}
	if !slices.Equal(evaluation.FailedChecks(), []string{}) {
		t.Fatalf("試験用 execution evaluation に失敗があります: %#v", evaluation.FailedChecks())
	}
	return evaluation
}
