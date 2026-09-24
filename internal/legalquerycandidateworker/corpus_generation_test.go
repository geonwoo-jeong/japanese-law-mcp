package legalquerycandidateworker

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval/evaluators"
)

func TestCandidateCorpusGenerationは新旧世代と未知版を分離する(t *testing.T) {
	t.Parallel()
	const verificationID = "candidate-evaluator-v4-corpus-generation-binding"
	for _, testCase := range []struct {
		handoff  int
		corpus   int
		version  string
		accepted bool
	}{
		{handoff: 2, corpus: 1, version: evaluators.Version1, accepted: true},
		{handoff: 2, corpus: 2, version: evaluators.Version2, accepted: true},
		{handoff: 3, corpus: 2, version: evaluators.Version3, accepted: true},
		{handoff: 4, corpus: 2, version: evaluators.Version3, accepted: true},
		{handoff: 5, corpus: 3, version: evaluators.Version4, accepted: true},
		{handoff: 5, corpus: 2, version: evaluators.Version4},
		{handoff: 5, corpus: 3, version: evaluators.Version3},
		{handoff: 5, corpus: 3, version: "legal-query-evaluator-v5"},
		{handoff: 4, corpus: 3, version: evaluators.Version3},
		{handoff: 4, corpus: 2, version: evaluators.Version2},
		{handoff: 3, corpus: 2, version: evaluators.Version1},
		{handoff: 3, corpus: 3, version: evaluators.Version3},
		{handoff: 2, corpus: 3, version: evaluators.Version1},
		{handoff: 2, corpus: 2, version: evaluators.Version4},
		{handoff: 5, corpus: 4, version: evaluators.Version4},
		{handoff: 6, corpus: 3, version: evaluators.Version4},
	} {
		request := legalquerycandidateeval.EvaluationRequest{
			SchemaVersion:    testCase.handoff,
			EvaluatorVersion: testCase.version,
		}
		err := validateCandidateCorpusGeneration(request, testCase.corpus)
		if (err == nil) != testCase.accepted {
			t.Fatalf("%s: handoff=%d corpus=%d evaluator=%s の受理判定が不正です: %v", verificationID,
				testCase.handoff, testCase.corpus, testCase.version, err)
		}
	}
}
