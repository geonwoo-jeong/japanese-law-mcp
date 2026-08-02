package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const candidateHandoffFixtureEvaluationID = "evaluation-sha256-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestCandidateHandoffはCanonicalなResultとReportだけを受理する(t *testing.T) {
	t.Parallel()

	reportRaw, err := os.ReadFile("../../testdata/legalquery/baselines/default.json")
	if err != nil {
		t.Fatalf("標準 report fixture を読めません: %v", err)
	}
	evaluationID := candidateHandoffFixtureEvaluationID

	t.Run("canonical", func(t *testing.T) {
		root := writeCandidateHandoffFixture(t, reportRaw, nil)
		handoff, err := readCandidateHandoff(root)
		if err != nil {
			t.Fatalf("canonical handoff を拒否しました: %v", err)
		}
		if handoff.EvaluationID != evaluationID || handoff.Outcome != "passed" {
			t.Fatalf("handoff = %#v", handoff)
		}
	})

	tests := []struct {
		name         string
		mutateReport func([]byte) []byte
		mutateResult func([]byte) []byte
	}{
		{
			name: "result の後方 token",
			mutateResult: func(raw []byte) []byte {
				return append(append([]byte(nil), raw...), ' ')
			},
		},
		{
			name: "result の重複 key",
			mutateResult: func(raw []byte) []byte {
				return []byte(strings.Replace(
					string(raw),
					`"outcome":"passed"`,
					`"outcome":"failed","outcome":"passed"`,
					1,
				))
			},
		},
		{
			name: "report の非 canonical byte",
			mutateReport: func(raw []byte) []byte {
				return append(append([]byte(nil), raw...), ' ')
			},
		},
		{
			name: "report の未知 privacy field",
			mutateReport: func(raw []byte) []byte {
				return []byte(strings.TrimSuffix(string(raw), "}\n") +
					`,"query":"永住許可 /private/path secret-marker"}` + "\n")
			},
		},
		{
			name: "report の重複 key",
			mutateReport: func(raw []byte) []byte {
				return []byte(strings.Replace(
					string(raw),
					`{"artifactKind":"legal_query_evaluation"`,
					`{"artifactKind":"legal_query_evaluation","artifactKind":"legal_query_evaluation"`,
					1,
				))
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidateReport := append([]byte(nil), reportRaw...)
			if test.mutateReport != nil {
				candidateReport = test.mutateReport(candidateReport)
			}
			root := writeCandidateHandoffFixture(
				t,
				candidateReport,
				test.mutateResult,
			)
			_, err := readCandidateHandoff(root)
			if err == nil {
				t.Fatal("closed canonical ではない handoff を受理しました")
			}
			if strings.Contains(err.Error(), "secret-marker") ||
				strings.Contains(err.Error(), "永住許可") ||
				strings.Contains(err.Error(), "/private/path") {
				t.Fatalf("handoff error が privacy 対象を漏らしました: %v", err)
			}
		})
	}
}

func TestCandidateReportDecoderはTypedSchemaの不正値を拒否する(t *testing.T) {
	t.Parallel()

	reportRaw, err := os.ReadFile("../../testdata/legalquery/baselines/default.json")
	if err != nil {
		t.Fatalf("標準 report fixture を読めません: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*candidateReportDocument)
	}{
		{
			name: "identity",
			mutate: func(document *candidateReportDocument) {
				document.CorpusVersion = "corpus-v0"
			},
		},
		{
			name: "profile",
			mutate: func(document *candidateReportDocument) {
				document.ProfileSet.Profiles[0].ProfileID = "../private"
			},
		},
		{
			name: "development",
			mutate: func(document *candidateReportDocument) {
				zero := 0
				document.Sets.Development.CaseCount = &zero
			},
		},
		{
			name: "holdout caseCount",
			mutate: func(document *candidateReportDocument) {
				document.Sets.Holdout.CaseCount = nil
			},
		},
		{
			name: "holdout metric count",
			mutate: func(document *candidateReportDocument) {
				document.Sets.Holdout.Metrics = document.Sets.Holdout.Metrics[:8]
			},
		},
		{
			name: "plan metric population",
			mutate: func(document *candidateReportDocument) {
				zero := 0
				document.Sets.Holdout.Metrics[0].Numerator = &zero
				document.Sets.Holdout.Metrics[0].Denominator = &zero
				document.Sets.Holdout.Metrics[0].Ratio = floatPointer(0)
				document.Sets.Holdout.Metrics[0].FailedCaseIDs = []string{}
			},
		},
		{
			name: "category count",
			mutate: func(document *candidateReportDocument) {
				document.Sets.Holdout.Categories = []candidateReportCategory{}
			},
		},
		{
			name: "derived denominator",
			mutate: func(document *candidateReportDocument) {
				zero := 0
				metric := &document.Sets.Holdout.DerivedObservations[0]
				metric.Numerator = &zero
				metric.Denominator = &zero
				metric.Ratio = floatPointer(0)
				metric.FailedCaseIDs = []string{}
			},
		},
		{
			name: "holdout failedCaseIds",
			mutate: func(document *candidateReportDocument) {
				document.Sets.Holdout.FailedCaseIDs = nil
			},
		},
		{
			name: "category order",
			mutate: func(document *candidateReportDocument) {
				document.Sets.Holdout.Categories[1].CategoryID =
					document.Sets.Holdout.Categories[0].CategoryID
			},
		},
		{
			name: "execution caseCount",
			mutate: func(document *candidateReportDocument) {
				document.Sets.Execution.CaseCount = nil
			},
		},
		{
			name: "execution metric population",
			mutate: func(document *candidateReportDocument) {
				caseCount := *document.Sets.Execution.CaseCount
				denominator := caseCount - 1
				metric := &document.Sets.Execution.Metrics[0]
				metric.Numerator = &denominator
				metric.Denominator = &denominator
				metric.Ratio = floatPointer(1)
				metric.FailedCaseIDs = []string{}
			},
		},
		{
			name: "execution violation count",
			mutate: func(document *candidateReportDocument) {
				document.Sets.Execution.WrongResourceCallCount = nil
			},
		},
		{
			name: "metric ratio",
			mutate: func(document *candidateReportDocument) {
				document.Sets.Holdout.Metrics[0].Ratio = floatPointer(0.5)
			},
		},
		{
			name: "case ID privacy boundary",
			mutate: func(document *candidateReportDocument) {
				metric := &document.Sets.Holdout.Metrics[0]
				numerator := *metric.Denominator - 1
				metric.Numerator = &numerator
				metric.Ratio = floatPointer(
					float64(numerator) / float64(*metric.Denominator),
				)
				metric.FailedCaseIDs = []string{"/private/secret"}
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var document candidateReportDocument
			if err := json.Unmarshal(reportRaw, &document); err != nil {
				t.Fatalf("report fixture を decode できません: %v", err)
			}
			test.mutate(&document)
			invalidRaw, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("不正 report fixture を encode できません: %v", err)
			}
			invalidRaw = append(invalidRaw, '\n')
			if err := validateCanonicalCandidateReport(invalidRaw); err == nil {
				t.Fatal("typed schema に反する report を受理しました")
			}
		})
	}
}

func floatPointer(value float64) *float64 {
	return &value
}

func writeCandidateHandoffFixture(
	t *testing.T,
	reportRaw []byte,
	mutateResult func([]byte) []byte,
) string {
	t.Helper()

	evaluationID := candidateHandoffFixtureEvaluationID
	reportDigest := sha256.Sum256(reportRaw)
	result := workerResultDocument{
		ArtifactKind:  "legal_query_candidate_evaluation_result",
		SchemaVersion: 2,
		EvaluationID:  evaluationID,
		RequestSHA256: strings.Repeat("b", 64),
		Outcome:       "passed",
		ReportSHA256:  hex.EncodeToString(reportDigest[:]),
	}
	resultRaw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("result fixture を encode できません: %v", err)
	}
	resultRaw = append(resultRaw, '\n')
	if mutateResult != nil {
		resultRaw = mutateResult(resultRaw)
	}

	root := t.TempDir()
	evaluationRoot := filepath.Join(root, evaluationID)
	if err := os.Mkdir(evaluationRoot, 0o700); err != nil {
		t.Fatalf("handoff directory を作れません: %v", err)
	}
	evaluationTree, err := os.OpenRoot(evaluationRoot)
	if err != nil {
		t.Fatalf("handoff root を開けません: %v", err)
	}
	defer func() {
		if closeErr := evaluationTree.Close(); closeErr != nil {
			t.Fatalf("handoff root を閉じられません: %v", closeErr)
		}
	}()
	if err := evaluationTree.WriteFile("report.json", reportRaw, 0o600); err != nil {
		t.Fatalf("report fixture を書けません: %v", err)
	}
	if err := evaluationTree.WriteFile("result.json", resultRaw, 0o600); err != nil {
		t.Fatalf("result fixture を書けません: %v", err)
	}
	return root
}
