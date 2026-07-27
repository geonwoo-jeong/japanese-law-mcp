package legalquerycorpus

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestSemanticExpectedV1Decodeは五decisionとRequestErrorを復元する(t *testing.T) {
	t.Parallel()

	for name, source := range validExpectedPlansForAllDecisions() {
		name, source := name, source
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			expected, err := decodeSemanticExpectedV1(mustJSONBytes(t, source))
			if err != nil {
				t.Fatalf("SOT-ENG-026: semantic expected decode error = %v", err)
			}
			plan, ok := expected.(ExpectedPlan)
			if !ok {
				t.Fatalf("SOT-ENG-026: expected の型 = %T", expected)
			}
			if err := plan.Validate(); err != nil {
				t.Fatalf("SOT-ENG-026: ExpectedPlan.Validate() error = %v", err)
			}
			if plan.Kind() != SemanticExpectedKindPlan ||
				string(plan.Decision()) != name {
				t.Fatalf("SOT-ENG-026: plan = %#v", plan)
			}
		})
	}

	requestError, err := decodeSemanticExpectedV1(mustJSONBytes(t, map[string]any{
		"kind":      "request_error",
		"errorCode": "invalid_argument",
		"field":     "query",
	}))
	if err != nil {
		t.Fatalf("SOT-ENG-026: request_error decode error = %v", err)
	}
	value, ok := requestError.(ExpectedRequestError)
	if !ok ||
		value.Kind() != SemanticExpectedKindRequestError ||
		value.ErrorCode() != model.ErrorCodeInvalidArgument ||
		value.Field() != RequestErrorFieldQuery {
		t.Fatalf("SOT-ENG-026: request_error = %#v", requestError)
	}
}

func TestExpectedMeaningとPlanのgetterは深く複製する(t *testing.T) {
	t.Parallel()

	source := validExpectedPlansForAllDecisions()["single"]
	expected, err := decodeSemanticExpectedV1(mustJSONBytes(t, source))
	if err != nil {
		t.Fatalf("SOT-ENG-026: semantic expected decode error = %v", err)
	}
	plan := expected.(ExpectedPlan)

	reasons := plan.ReasonCodes()
	reasons[0] = legalquery.ReasonCodeAmbiguousCandidates
	selected := plan.SelectedMeaningIDs()
	selected[0] = "changed"
	meanings := plan.Meanings()
	meaning := meanings[0]
	evidence := meaning.EvidenceCodes()
	evidence[0] = legalquery.EvidenceGeneralTerm
	steps := meaning.Steps()
	steps[0] = ExpectedStep{}

	if plan.ReasonCodes()[0] != legalquery.ReasonCodeSingleClearCandidate ||
		plan.SelectedMeaningIDs()[0] != "law-search" ||
		plan.Meanings()[0].EvidenceCodes()[0] != legalquery.EvidenceExplicitTask ||
		plan.Meanings()[0].Steps()[0].InputKind() != legalquery.InputKindLawSearch {
		t.Fatal("SOT-ENG-026: expected plan が getter から変更された")
	}
}

func TestExpectedMeaningは根拠と概念とstepの不変条件を検証する(t *testing.T) {
	t.Parallel()

	step, err := decodeExpectedStepV1(mustJSONBytes(t, validLawSearchStep()))
	if err != nil {
		t.Fatalf("SOT-ENG-026: expected step decode error = %v", err)
	}
	valid := ExpectedMeaningValues{
		MeaningID: "law-search",
		EvidenceCodes: []legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
		},
		ConceptIDs:    []string{},
		RequiredPacks: []string{},
		Steps:         []ExpectedStep{step},
	}
	if _, err := NewExpectedMeaning(valid); err != nil {
		t.Fatalf("SOT-ENG-026: NewExpectedMeaning() error = %v", err)
	}
	tests := map[string]func(ExpectedMeaningValues) ExpectedMeaningValues{
		"meaningId": func(value ExpectedMeaningValues) ExpectedMeaningValues {
			value.MeaningID = "UPPER"
			return value
		},
		"evidence順序": func(value ExpectedMeaningValues) ExpectedMeaningValues {
			value.EvidenceCodes = []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceExplicitTask,
			}
			return value
		},
		"legal concept不足": func(value ExpectedMeaningValues) ExpectedMeaningValues {
			value.EvidenceCodes = []legalquery.EvidenceCode{
				legalquery.EvidenceLegalConcept,
			}
			return value
		},
		"concept過剰": func(value ExpectedMeaningValues) ExpectedMeaningValues {
			value.ConceptIDs = []string{"opaque-concept"}
			return value
		},
		"concept順序": func(value ExpectedMeaningValues) ExpectedMeaningValues {
			value.EvidenceCodes = []legalquery.EvidenceCode{
				legalquery.EvidenceLegalConcept,
			}
			value.ConceptIDs = []string{"z-opaque", "a-opaque"}
			return value
		},
		"pack順序": func(value ExpectedMeaningValues) ExpectedMeaningValues {
			value.RequiredPacks = []string{"z-pack", "a-pack"}
			return value
		},
		"step不足": func(value ExpectedMeaningValues) ExpectedMeaningValues {
			value.Steps = []ExpectedStep{}
			return value
		},
		"step超過": func(value ExpectedMeaningValues) ExpectedMeaningValues {
			value.Steps = []ExpectedStep{step, step, step, step, step}
			return value
		},
	}
	for name, change := range tests {
		name, change := name, change
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewExpectedMeaning(change(valid)); err == nil {
				t.Fatal("SOT-ENG-026: 不正な ExpectedMeaning を受理した")
			}
		})
	}
}

func TestSemanticExpectedはzero値と直接JSON復元を拒否する(t *testing.T) {
	t.Parallel()

	for name, validate := range map[string]func() error{
		"meaning":       func() error { return (ExpectedMeaning{}).Validate() },
		"plan":          func() error { return (ExpectedPlan{}).Validate() },
		"request_error": func() error { return (ExpectedRequestError{}).Validate() },
	} {
		name, validate := name, validate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := validate(); err == nil {
				t.Fatal("SOT-ENG-026: zero value を受理した")
			}
		})
	}

	var plan ExpectedPlan
	if err := json.Unmarshal([]byte(`{}`), &plan); err == nil {
		t.Fatal("SOT-ENG-026: ExpectedPlan の直接復元を受理した")
	}
}

func TestSemanticExpected公開型は動的JSONを持たない(t *testing.T) {
	t.Parallel()

	for _, typ := range []reflect.Type{
		reflect.TypeOf(ExpectedMeaning{}),
		reflect.TypeOf(ExpectedMeaningValues{}),
		reflect.TypeOf(ExpectedPlan{}),
		reflect.TypeOf(ExpectedPlanValues{}),
		reflect.TypeOf(ExpectedRequestError{}),
		reflect.TypeOf(ExpectedRequestErrorValues{}),
	} {
		assertNoPublicDynamicJSONType(t, typ, map[reflect.Type]bool{})
	}
}

func TestExpectedPlanGetterは並行読取りで共有状態を変更しない(t *testing.T) {
	t.Parallel()

	expected, err := decodeSemanticExpectedV1(
		mustJSONBytes(t, validExpectedPlansForAllDecisions()["hedged"]),
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: semantic expected decode error = %v", err)
	}
	plan := expected.(ExpectedPlan)
	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for repeat := 0; repeat < 100; repeat++ {
				meanings := plan.Meanings()
				meanings[0] = ExpectedMeaning{}
				selected := plan.SelectedMeaningIDs()
				selected[0] = "changed"
			}
		}()
	}
	wait.Wait()
	if plan.Meanings()[0].MeaningID() != "law-search" ||
		plan.SelectedMeaningIDs()[0] != "law-search" {
		t.Fatal("SOT-ENG-026: 並行 getter が ExpectedPlan を変更した")
	}
}

func validExpectedPlansForAllDecisions() map[string]map[string]any {
	single := validSemanticCase(validLawSearchStep())["expected"].(map[string]any)

	hedged := validSemanticCase(validLawSearchStep())["expected"].(map[string]any)
	hedged["decision"] = "hedged"
	hedged["reasonCodes"] = []any{"hedged_close_candidates"}
	hedged["meanings"] = append(
		hedged["meanings"].([]any),
		validExpectedMeaning("law-content-search", validLawContentSearchStep()),
	)
	hedged["selectedMeaningIds"] = []any{"law-search", "law-content-search"}

	clarification := validSemanticCase(validLawSearchStep())["expected"].(map[string]any)
	clarification["decision"] = "needs_clarification"
	clarification["reasonCodes"] = []any{"ambiguous_candidates"}
	clarification["selectedMeaningIds"] = []any{}

	unavailable := validSemanticCase(validJudicialSearchStep())["expected"].(map[string]any)
	unavailable["decision"] = "capability_unavailable"
	unavailable["reasonCodes"] = []any{"required_pack_disabled"}
	unavailable["selectedMeaningIds"] = []any{"judicial-search"}
	unavailableMeaning := unavailable["meanings"].([]any)[0].(map[string]any)
	unavailableMeaning["meaningId"] = "judicial-search"
	unavailableMeaning["requiredPacks"] = []any{"judicial-cases"}

	unsupported := validSemanticCase(validLawSearchStep())["expected"].(map[string]any)
	unsupported["decision"] = "unsupported"
	unsupported["reasonCodes"] = []any{"unsupported_task_or_resource"}
	unsupported["meanings"] = []any{}
	unsupported["selectedMeaningIds"] = []any{}

	return map[string]map[string]any{
		"single":                 single,
		"hedged":                 hedged,
		"needs_clarification":    clarification,
		"capability_unavailable": unavailable,
		"unsupported":            unsupported,
	}
}
