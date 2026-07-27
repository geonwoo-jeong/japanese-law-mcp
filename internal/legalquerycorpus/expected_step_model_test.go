package legalquerycorpus

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestExpectedStepV1Decodeは七variantを製品型へ変換する(t *testing.T) {
	t.Parallel()

	tests := map[string]map[string]any{
		"law_search":               validLawSearchStep(),
		"law_content_search":       validLawContentSearchStep(),
		"law_read":                 validLawReadStep(),
		"law_article_read":         validLawArticleReadStep(),
		"law_updates":              validLawUpdatesStep(),
		"judicial_decision_search": validJudicialSearchStep(),
		"judicial_decision_read":   validJudicialReadStep(),
	}
	for name, source := range tests {
		name, source := name, source
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			step, err := decodeExpectedStepV1(mustJSONBytes(t, source))
			if err != nil {
				t.Fatalf("SOT-ENG-026: expected step decode error = %v", err)
			}
			if err := step.Validate(); err != nil {
				t.Fatalf("SOT-ENG-026: ExpectedStep.Validate() error = %v", err)
			}
			if step.Task() != legalquery.Task(source["task"].(string)) ||
				step.Resource() != legalquery.Resource(source["resource"].(string)) ||
				step.InputKind() != legalquery.LogicalInputKind(source["inputKind"].(string)) {
				t.Fatalf("SOT-ENG-026: expected step = %#v", step)
			}
			assertExpectedLogicalInputType(t, step.LogicalInput(), step.InputKind())
		})
	}
}

func TestExpectedStepはmapping不一致とzero値を拒否する(t *testing.T) {
	t.Parallel()

	if err := (ExpectedStep{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: ExpectedStep の zero value を受理した")
	}
	if _, err := NewExpectedStep(ExpectedStepValues{}); err == nil {
		t.Fatal("SOT-ENG-026: nil logicalInput を受理した")
	}
	input, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{Query: "民法"},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-022: logical input error = %v", err)
	}
	_, err = NewExpectedStep(ExpectedStepValues{
		Task:         legalquery.TaskRead,
		Resource:     legalquery.ResourceLaw,
		InputKind:    legalquery.InputKindLawSearch,
		LogicalInput: input,
	})
	if err == nil {
		t.Fatal("SOT-ENG-026: task と inputKind の不一致を受理した")
	}
	_, err = NewExpectedStep(ExpectedStepValues{
		Task:         legalquery.TaskSearch,
		Resource:     legalquery.ResourceLaw,
		InputKind:    legalquery.InputKindJudicialDecisionSearch,
		LogicalInput: input,
	})
	if err == nil {
		t.Fatal("SOT-ENG-026: inputKind と logicalInput の不一致を受理した")
	}
}

func TestExpectedStepはJSONからの直接復元を拒否する(t *testing.T) {
	t.Parallel()

	var step ExpectedStep
	if err := json.Unmarshal([]byte(`{}`), &step); err == nil {
		t.Fatal("SOT-ENG-026: ExpectedStep の直接復元を受理した")
	}
}

func TestExpectedStep公開契約はroute情報と動的JSONを持たない(t *testing.T) {
	t.Parallel()

	valuesType := reflect.TypeOf(ExpectedStepValues{})
	expectedNames := []string{"Task", "Resource", "InputKind", "LogicalInput"}
	if valuesType.NumField() != len(expectedNames) {
		t.Fatalf("SOT-ENG-026: ExpectedStepValues の項目数 = %d", valuesType.NumField())
	}
	for index, expected := range expectedNames {
		if valuesType.Field(index).Name != expected {
			t.Fatalf(
				"SOT-ENG-026: ExpectedStepValues[%d] = %q",
				index,
				valuesType.Field(index).Name,
			)
		}
	}
	for _, typ := range []reflect.Type{
		reflect.TypeOf(ExpectedStep{}),
		valuesType,
	} {
		assertNoPublicDynamicJSONType(t, typ, map[reflect.Type]bool{})
	}
}

func TestExpectedStepGetterは並行読取りで共有状態を変更しない(t *testing.T) {
	t.Parallel()

	step, err := decodeExpectedStepV1(mustJSONBytes(t, validLawContentSearchStep()))
	if err != nil {
		t.Fatalf("SOT-ENG-026: expected step decode error = %v", err)
	}
	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for repeat := 0; repeat < 100; repeat++ {
				input := step.LogicalInput().(legalquery.LawContentSearchIntentV1)
				terms := input.AllTerms()
				terms[0] = "変更"
			}
		}()
	}
	wait.Wait()
	input := step.LogicalInput().(legalquery.LawContentSearchIntentV1)
	if input.AllTerms()[0] != "永住許可" {
		t.Fatal("SOT-ENG-026: ExpectedStep の logicalInput が getter から変更された")
	}
}

func assertExpectedLogicalInputType(
	t *testing.T,
	input legalquery.LogicalInput,
	kind legalquery.LogicalInputKind,
) {
	t.Helper()
	switch kind {
	case legalquery.InputKindLawSearch:
		if _, ok := input.(legalquery.LawSearchIntentV1); !ok {
			t.Fatalf("SOT-ENG-026: logicalInput の型 = %T", input)
		}
	case legalquery.InputKindLawContentSearch:
		if _, ok := input.(legalquery.LawContentSearchIntentV1); !ok {
			t.Fatalf("SOT-ENG-026: logicalInput の型 = %T", input)
		}
	case legalquery.InputKindLawRead:
		if _, ok := input.(legalquery.LawReadIntentV1); !ok {
			t.Fatalf("SOT-ENG-026: logicalInput の型 = %T", input)
		}
	case legalquery.InputKindLawArticleRead:
		if _, ok := input.(legalquery.LawArticleReadIntentV1); !ok {
			t.Fatalf("SOT-ENG-026: logicalInput の型 = %T", input)
		}
	case legalquery.InputKindLawUpdates:
		if _, ok := input.(legalquery.LawUpdateListIntentV1); !ok {
			t.Fatalf("SOT-ENG-026: logicalInput の型 = %T", input)
		}
	case legalquery.InputKindJudicialDecisionSearch:
		if _, ok := input.(legalquery.JudicialDecisionSearchIntentV1); !ok {
			t.Fatalf("SOT-ENG-026: logicalInput の型 = %T", input)
		}
	case legalquery.InputKindJudicialDecisionRead:
		if _, ok := input.(legalquery.JudicialDecisionReadIntentV1); !ok {
			t.Fatalf("SOT-ENG-026: logicalInput の型 = %T", input)
		}
	default:
		t.Fatalf("SOT-MODEL-022: 未知の input kind = %q", kind)
	}
	if input == nil {
		t.Fatal("SOT-ENG-026: logicalInput が nil です")
	}
}
