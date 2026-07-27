package legalquerycorpus

import "testing"

func TestSemanticExpectedV1Decodeは必須項目とvariant未知項目を拒否する(t *testing.T) {
	t.Parallel()

	missingKind := validExpectedPlansForAllDecisions()["single"]
	delete(missingKind, "kind")
	missingDecision := validExpectedPlansForAllDecisions()["single"]
	delete(missingDecision, "decision")
	missingMeaningSteps := validExpectedPlansForAllDecisions()["single"]
	delete(missingMeaningSteps["meanings"].([]any)[0].(map[string]any), "steps")
	unknownPlanField := validExpectedPlansForAllDecisions()["single"]
	unknownPlanField["profileVersion"] = "forbidden"
	unknownMeaningField := validExpectedPlansForAllDecisions()["single"]
	unknownMeaningField["meanings"].([]any)[0].(map[string]any)["candidateId"] = "forbidden"

	for name, source := range map[string]map[string]any{
		"kind欠落":          missingKind,
		"decision欠落":      missingDecision,
		"meaning steps欠落": missingMeaningSteps,
		"plan未知項目":        unknownPlanField,
		"meaning未知項目":     unknownMeaningField,
		"未知kind": {
			"kind": "unknown",
		},
		"request error未知項目": {
			"kind":      "request_error",
			"errorCode": "invalid_argument",
			"field":     "query",
			"message":   "forbidden",
		},
	} {
		name, source := name, source
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := decodeSemanticExpectedV1(mustJSONBytes(t, source)); err == nil {
				t.Fatal("SOT-ENG-026: 不正な semantic expected を受理した")
			}
		})
	}
}

func TestExpectedPlanV1Decodeはdecisionと参照の不整合を拒否する(t *testing.T) {
	t.Parallel()

	wrongReason := validExpectedPlansForAllDecisions()["single"]
	wrongReason["reasonCodes"] = []any{"ambiguous_candidates"}

	unknownSelection := validExpectedPlansForAllDecisions()["single"]
	unknownSelection["selectedMeaningIds"] = []any{"unknown"}

	duplicateMeaning := validExpectedPlansForAllDecisions()["hedged"]
	duplicateMeaning["meanings"].([]any)[1].(map[string]any)["meaningId"] = "law-search"

	reversedSelection := validExpectedPlansForAllDecisions()["hedged"]
	reversedSelection["selectedMeaningIds"] = []any{
		"law-content-search",
		"law-search",
	}

	tooManySelected := validExpectedPlansForAllDecisions()["hedged"]
	tooManySelected["selectedMeaningIds"] = []any{
		"law-search",
		"law-content-search",
		"third",
	}

	unsupportedSelected := validExpectedPlansForAllDecisions()["unsupported"]
	unsupportedSelected["selectedMeaningIds"] = []any{"law-search"}

	tooManySelectedSteps := validExpectedPlansForAllDecisions()["hedged"]
	firstMeaning := tooManySelectedSteps["meanings"].([]any)[0].(map[string]any)
	firstMeaning["steps"] = append(
		firstMeaning["steps"].([]any),
		validLawReadStep(),
		validLawUpdatesStep(),
	)
	secondMeaning := tooManySelectedSteps["meanings"].([]any)[1].(map[string]any)
	secondMeaning["steps"] = append(
		secondMeaning["steps"].([]any),
		validJudicialSearchStep(),
	)

	for name, source := range map[string]map[string]any{
		"reason":               wrongReason,
		"unknown selection":    unknownSelection,
		"duplicate meaning":    duplicateMeaning,
		"reversed selection":   reversedSelection,
		"selected上限":           tooManySelected,
		"unsupported selected": unsupportedSelected,
		"selected step合計":      tooManySelectedSteps,
	} {
		name, source := name, source
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := decodeSemanticExpectedV1(mustJSONBytes(t, source)); err == nil {
				t.Fatal("SOT-ENG-026: decision と参照が不整合な plan を受理した")
			}
		})
	}
}

func TestExpectedMeaningV1DecodeはopaqueConceptIDを過制約しない(t *testing.T) {
	t.Parallel()

	source := validExpectedPlansForAllDecisions()["single"]
	meaning := source["meanings"].([]any)[0].(map[string]any)
	meaning["evidenceCodes"] = []any{"legal_concept"}
	meaning["conceptIds"] = []any{"法概念:永住/許可"}
	expected, err := decodeSemanticExpectedV1(mustJSONBytes(t, source))
	if err != nil {
		t.Fatalf("SOT-ENG-026: opaque conceptId decode error = %v", err)
	}
	plan := expected.(ExpectedPlan)
	if plan.Meanings()[0].ConceptIDs()[0] != "法概念:永住/許可" {
		t.Fatalf("SOT-ENG-026: conceptIds = %#v", plan.Meanings()[0].ConceptIDs())
	}
}

func TestExpectedRequestErrorV1Decodeは三fieldだけを受理する(t *testing.T) {
	t.Parallel()

	for _, field := range []string{"query", "ref", "limitPerAttempt"} {
		field := field
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			expected, err := decodeSemanticExpectedV1(mustJSONBytes(t, map[string]any{
				"kind":      "request_error",
				"errorCode": "invalid_argument",
				"field":     field,
			}))
			if err != nil {
				t.Fatalf("SOT-ENG-026: request_error decode error = %v", err)
			}
			if string(expected.(ExpectedRequestError).Field()) != field {
				t.Fatalf("SOT-ENG-026: request error field = %q", field)
			}
		})
	}

	for _, source := range []map[string]any{
		{
			"kind":      "request_error",
			"errorCode": "not_found",
			"field":     "query",
		},
		{
			"kind":      "request_error",
			"errorCode": "invalid_argument",
			"field":     "unknown",
		},
		{
			"kind":      "request_error",
			"errorCode": "invalid_argument",
		},
	} {
		if _, err := decodeSemanticExpectedV1(mustJSONBytes(t, source)); err == nil {
			t.Fatal("SOT-ENG-026: 不正な request_error を受理した")
		}
	}
}
