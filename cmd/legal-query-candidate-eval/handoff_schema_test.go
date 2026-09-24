package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// SOT-ENG-045: result の閉じた世代選択は report の schema version を変更しない。
func TestCandidateHandoffは固定世代だけを受理する(t *testing.T) {
	t.Parallel()

	reportRaw := syntheticCandidateReport(t)
	tests := []struct {
		name         string
		versionField string
		accepted     bool
	}{
		{name: "version 2", versionField: `"schemaVersion":2,`, accepted: true},
		{name: "version 3", versionField: `"schemaVersion":3,`, accepted: true},
		{name: "version 4", versionField: `"schemaVersion":4,`, accepted: true},
		{name: "未知の旧版", versionField: `"schemaVersion":1,`},
		{name: "未知の新版", versionField: `"schemaVersion":5,`},
		{name: "版省略"},
		{name: "文字列版", versionField: `"schemaVersion":"4",`},
		{name: "小数版", versionField: `"schemaVersion":4.0,`},
		{name: "null版", versionField: `"schemaVersion":null,`},
		{name: "重複版", versionField: `"schemaVersion":4,"schemaVersion":4,`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := writeCandidateHandoffFixture(t, reportRaw, func(raw []byte) []byte {
				return []byte(strings.Replace(string(raw), `"schemaVersion":2,`, test.versionField, 1))
			})
			handoff, err := readCandidateHandoff(root)
			if test.accepted {
				if err != nil {
					t.Fatalf("固定世代の handoff を拒否しました: %v", err)
				}
				if handoff.EvaluationID != candidateHandoffFixtureEvaluationID || handoff.Outcome != "passed" {
					t.Fatalf("handoff = %#v", handoff)
				}
			} else if err == nil {
				t.Fatal("未対応の版を持つ handoff を受理しました")
			}
		})
	}
}

func syntheticCandidateReport(t *testing.T) []byte {
	t.Helper()

	one, zero := 1, 0
	document := candidateReportDocument{
		ArtifactKind: "legal_query_evaluation", SchemaVersion: &one,
		CorpusVersion: "corpus-v1", HoldoutDigest: strings.Repeat("c", 64),
		BaselineVersion: "default-1",
		ProfileSet: candidateReportProfileSet{
			ProfileSetID: "synthetic", ProfileSetVersion: "v1", RankingVersion: "v1",
			Profiles: []candidateReportProfile{{ProfileID: "synthetic", ProfileVersion: "v1"}},
		},
		Sets: candidateReportSets{
			Development: candidateReportDevelopment{CaseCount: &one},
			Holdout: candidateReportHoldout{
				CaseCount: &one,
				Metrics: syntheticCandidateMetrics(append(
					[]string{"plan-reproducibility"}, semanticReportMetricIDs()...)),
				Categories: []candidateReportCategory{{
					CategoryID: "synthetic", CaseCount: &one,
					Metrics: syntheticCandidateMetrics(semanticReportMetricIDs()),
				}},
				DerivedObservations: syntheticCandidateMetrics(derivedReportMetricIDs()),
				FailedCaseIDs:       []string{},
			},
			Execution: candidateReportExecution{
				CaseCount: &one, Metrics: syntheticCandidateMetrics(executionReportMetricIDs()),
				WrongResourceCallCount: &zero, BudgetViolationCount: &zero,
				AttemptOrderViolationCount: &zero, ImplicitFirstReadCount: &zero,
				EmptyReclassificationCount: &zero, FailedCaseIDs: []string{},
			},
		},
	}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("合成 report を encode できません: %v", err)
	}
	return append(raw, '\n')
}

func syntheticCandidateMetrics(ids []string) []candidateReportMetric {
	one := 1
	metrics := make([]candidateReportMetric, len(ids))
	for index, id := range ids {
		metrics[index] = candidateReportMetric{
			MetricID: id, Numerator: &one, Denominator: &one,
			Ratio: floatPointer(1), FailedCaseIDs: []string{},
		}
	}
	return metrics
}
