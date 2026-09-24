package legalquerycandidateprepare

import "testing"

func TestCandidateEvaluationCorpusVersionは新規Request境界を固定する(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name          string
		schemaVersion int
		corpusVersion string
		wantError     bool
	}{
		{name: "schema-v1", schemaVersion: 1, corpusVersion: "corpus-v13", wantError: true},
		{name: "最初の履歴版", schemaVersion: 2, corpusVersion: "corpus-v1", wantError: true},
		{name: "最後の履歴専用版", schemaVersion: 2, corpusVersion: "corpus-v12", wantError: true},
		{name: "最初の新規評価版", schemaVersion: 2, corpusVersion: "corpus-v13"},
		{name: "schema-v3", schemaVersion: 3, corpusVersion: "corpus-v99"},
		{name: "未知schema", schemaVersion: 4, corpusVersion: "corpus-v99", wantError: true},
		{name: "将来版", schemaVersion: 2, corpusVersion: "corpus-v99"},
		{name: "先頭零", schemaVersion: 2, corpusVersion: "corpus-v013", wantError: true},
		{name: "数値なし", schemaVersion: 2, corpusVersion: "corpus-v", wantError: true},
		{name: "別接頭辞", schemaVersion: 2, corpusVersion: "dataset-v13", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateCandidateEvaluationCorpus(
				test.schemaVersion,
				test.corpusVersion,
			)
			if (err != nil) != test.wantError {
				t.Fatalf(
					"candidate-evaluation-corpus-admissibility: corpusVersion=%q error=%v wantError=%t",
					test.corpusVersion,
					err,
					test.wantError,
				)
			}
		})
	}
}

// SOT-ENG-048: handoff の世代と corpus schema は明示的に結合する。
func TestCandidateCorpusはHandoff世代のExact版だけを受理する(t *testing.T) {
	t.Parallel()
	for handoff := 2; handoff <= 6; handoff++ {
		for schema := 1; schema <= 4; schema++ {
			err := validateCandidateEvaluationCorpusForSchema(handoff, schema, "corpus-v99")
			accepted := ((handoff == 3 || handoff == 4) && schema == 2) || (handoff == 5 && schema == 3)
			if (err == nil) != accepted {
				t.Fatalf("handoff=%d corpus=%d error=%v", handoff, schema, err)
			}
		}
	}
}
