package legalquerycorpus

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateExecutionScenarioRequirementsは八Scenarioの網羅を受理する(t *testing.T) {
	t.Parallel()

	fixture := executionScenarioRequirementsValidFixture(t)
	if err := validateExecutionScenarioRequirements(
		fixture.manifest,
		fixture.development,
		fixture.execution,
	); err != nil {
		t.Fatalf("SOT-ENG-026: 八 scenario の網羅 error = %v", err)
	}
	if err := validateExecutionReferences(
		fixture.development,
		fixture.execution,
	); err != nil {
		t.Fatalf("SOT-ENG-026: scenario fixture の参照 error = %v", err)
	}
}

func TestExecutionScenarioRequirementsは固定八Scenarioを独立に確認する(t *testing.T) {
	t.Parallel()

	want := []string{
		"execution-all-failed",
		"execution-empty",
		"execution-item-budget",
		"execution-mixed-composition",
		"execution-nonempty",
		"execution-partial-failure",
		"execution-reversed-completion",
		"execution-timeout",
	}
	if got := manifestRequiredExecutionScenarioIDs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("SOT-ENG-026: 必須 execution scenario = %v", got)
	}
}

func TestValidateExecutionScenarioRequirementsは各Scenarioの不足を規定順で拒否する(t *testing.T) {
	t.Parallel()

	fixture := executionScenarioRequirementsValidFixture(t)
	required := fixture.manifest.RequiredExecutionScenarioIDs()
	for missingIndex, missingID := range required {
		missingIndex, missingID := missingIndex, missingID
		t.Run(missingID, func(t *testing.T) {
			t.Parallel()

			development := removeSemanticCaseAt(
				fixture.development,
				missingIndex,
			)
			execution := removeExecutionCaseAt(
				fixture.execution,
				missingIndex,
			)
			manifest := executionScenarioRequirementsManifest(
				t,
				semanticCaseIDsForScenarioTest(development),
				[]string{"holdout-placeholder"},
				executionCaseIDsForScenarioTest(execution),
			)
			err := validateExecutionScenarioRequirements(
				manifest,
				development,
				execution,
			)
			if err == nil {
				t.Fatal("SOT-ENG-026: 必須 execution scenario の不足を受理した")
			}
			if !strings.Contains(err.Error(), missingID) {
				t.Fatalf(
					"SOT-ENG-026: 不足 scenario の決定順が不正です: %v",
					err,
				)
			}
		})
	}
}

func TestValidateExecutionScenarioRequirementsは複数不足の先頭を規定順で返す(t *testing.T) {
	t.Parallel()

	fixture := executionScenarioRequirementsValidFixture(t)
	development := removeSemanticCaseAt(fixture.development, 4)
	development = removeSemanticCaseAt(development, 0)
	execution := removeExecutionCaseAt(fixture.execution, 4)
	execution = removeExecutionCaseAt(execution, 0)
	manifest := executionScenarioRequirementsManifest(
		t,
		semanticCaseIDsForScenarioTest(development),
		[]string{"holdout-placeholder"},
		executionCaseIDsForScenarioTest(execution),
	)

	err := validateExecutionScenarioRequirements(
		manifest,
		development,
		execution,
	)
	if err == nil || !strings.Contains(
		err.Error(),
		string(ExecutionScenarioIDAllFailed),
	) {
		t.Fatalf("SOT-ENG-026: 複数不足の先頭 error = %v", err)
	}
}

func TestValidateExecutionScenarioRequirementsはItemBudgetをEffectiveLimitで判定する(t *testing.T) {
	t.Parallel()

	fixture := executionScenarioRequirementsItemBudgetFixture(
		t,
		5,
		4,
	)
	if err := validateExecutionReferences(
		fixture.development,
		fixture.execution,
	); err != nil {
		t.Fatalf("SOT-ENG-026: false-positive fixture の参照 error = %v", err)
	}
	err := validateExecutionScenarioRequirements(
		fixture.manifest,
		fixture.development,
		fixture.execution,
	)
	if err == nil {
		t.Fatal("SOT-ENG-026: sourceItemCount が effectiveLimit 以下の item-budget を受理した")
	}
	if !strings.Contains(err.Error(), string(ExecutionScenarioIDItemBudget)) {
		t.Fatalf("SOT-ENG-026: item-budget error の分類 = %v", err)
	}
}

func TestValidateExecutionScenarioRequirementsはReversedCompletionにHedgedを要求する(t *testing.T) {
	t.Parallel()

	fixture := executionScenarioRequirementsSingleReversedFixture(t)
	err := validateExecutionScenarioRequirements(
		fixture.manifest,
		fixture.development,
		fixture.execution,
	)
	if err == nil {
		t.Fatal("SOT-ENG-026: single plan の reversed-completion を受理した")
	}
	if !strings.Contains(
		err.Error(),
		string(ExecutionScenarioIDReversedCompletion),
	) {
		t.Fatalf("SOT-ENG-026: reversed-completion error の分類 = %v", err)
	}
}

func TestValidateExecutionScenarioRequirementsはMixedCompositionにCoreとPackを要求する(
	t *testing.T,
) {
	t.Parallel()

	fixture := executionScenarioRequirementsValidFixture(t)
	mixedIndex := 3
	fixture.development[mixedIndex] = executionReferenceTestSemanticCase(
		t,
		executionReferenceTestSemanticCaseValues{
			caseID: "development-mixed-composition",
			meanings: []executionReferenceTestMeaningValues{{
				id: "meaning-one",
				steps: []map[string]any{
					validLawReadStep(),
					validLawSearchStep(),
				},
			}},
			selectedMeaningIDs: []string{"meaning-one"},
			enabledPacks:       []string{"judicial-cases"},
		},
	)
	err := validateExecutionScenarioRequirements(
		fixture.manifest,
		fixture.development,
		fixture.execution,
	)
	if err == nil {
		t.Fatal("SOT-ENG-026: pack step のない mixed-composition を受理した")
	}
	if !strings.Contains(
		err.Error(),
		string(ExecutionScenarioIDMixedComposition),
	) {
		t.Fatalf("SOT-ENG-026: mixed-composition error の分類 = %v", err)
	}
}

func TestValidateExecutionScenarioRequirementsは旧CorpusVersionで新Scenarioを拒否する(
	t *testing.T,
) {
	t.Parallel()

	fixture := executionScenarioRequirementsValidFixture(t)
	legacyManifest := executionScenarioRequirementsManifestForVersion(
		t,
		"corpus-v1",
		semanticCaseIDsForScenarioTest(fixture.development),
		[]string{"holdout-placeholder"},
		executionCaseIDsForScenarioTest(fixture.execution),
	)

	err := validateExecutionScenarioRequirements(
		legacyManifest,
		fixture.development,
		fixture.execution,
	)
	if err == nil {
		t.Fatal("SOT-ENG-026: 旧 corpus version に mixed-composition を混入させても受理した")
	}
	if !strings.Contains(err.Error(), string(ExecutionScenarioIDMixedComposition)) {
		t.Fatalf("SOT-ENG-026: version 境界 error の分類 = %v", err)
	}
}

func TestValidateIntegrityExecutionScenarioRequirementsは失敗時にZeroResultを返す(t *testing.T) {
	t.Parallel()

	fixture := executionScenarioRequirementsItemBudgetFixture(t, 5, 4)
	checked := integrityCheckedCorpus{
		manifest:    fixture.manifest,
		development: fixture.development,
		execution:   fixture.execution,
	}
	beforeDevelopment, err := cloneCorpusSemanticCases(checked.development)
	if err != nil {
		t.Fatalf("SOT-ENG-026: development 複製 error = %v", err)
	}
	beforeExecution, err := cloneCorpusExecutionCases(checked.execution)
	if err != nil {
		t.Fatalf("SOT-ENG-026: execution 複製 error = %v", err)
	}

	got, err := validateIntegrityExecutionScenarioRequirements(checked)
	if err == nil {
		t.Fatal("SOT-ENG-026: 不正な execution scenario requirements を受理した")
	}
	if !reflect.DeepEqual(got, integrityCheckedCorpus{}) {
		t.Fatal("SOT-ENG-026: scenario requirements 失敗時に部分結果を返した")
	}
	if !reflect.DeepEqual(beforeDevelopment, checked.development) ||
		!reflect.DeepEqual(beforeExecution, checked.execution) {
		t.Fatal("SOT-ENG-026: scenario requirements 検証が入力を変更した")
	}
}

func TestValidateIntegrityRequirementsはScenarioまで直列に検証する(t *testing.T) {
	t.Parallel()

	fixture := executionScenarioRequirementsValidFixture(t)
	holdout := holdoutRequirementsTestValidHoldout(t)
	fixture.manifest = executionScenarioRequirementsManifest(
		t,
		semanticCaseIDsForScenarioTest(fixture.development),
		semanticCaseIDsForScenarioTest(holdout),
		executionCaseIDsForScenarioTest(fixture.execution),
	)
	checked := integrityCheckedCorpus{
		manifest:    fixture.manifest,
		development: fixture.development,
		holdout:     holdout,
		execution:   fixture.execution,
	}

	got, err := validateIntegrityRequirements(checked)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 全 requirements の直列検証 error = %v", err)
	}
	if len(got.execution) != len(executionScenarioIDs()) {
		t.Fatalf("SOT-ENG-026: execution 件数 = %d", len(got.execution))
	}
}

func TestValidateIntegrityRequirementsはPhaseの失敗順を固定する(t *testing.T) {
	t.Parallel()

	fixture := executionScenarioRequirementsValidFixture(t)
	holdout := holdoutRequirementsTestValidHoldout(t)
	development := removeSemanticCaseAt(fixture.development, 0)
	execution := removeExecutionCaseAt(fixture.execution, 0)
	manifest := executionScenarioRequirementsManifest(
		t,
		semanticCaseIDsForScenarioTest(development),
		semanticCaseIDsForScenarioTest(holdout),
		executionCaseIDsForScenarioTest(execution),
	)
	scenarioInvalid := integrityCheckedCorpus{
		manifest:    manifest,
		development: development,
		holdout:     holdout,
		execution:   execution,
	}
	referenceInvalid := scenarioInvalid
	referenceInvalid.execution = append(
		[]ExecutionCase{},
		scenarioInvalid.execution...,
	)
	referenceInvalid.execution[0].semanticCaseID = "development-missing"
	holdoutInvalid := referenceInvalid
	holdoutInvalid.holdout = holdoutInvalid.holdout[:minimumHoldoutCaseCount-1]

	tests := []struct {
		name    string
		checked integrityCheckedCorpus
		want    string
	}{
		{name: "holdout", checked: holdoutInvalid, want: "holdout"},
		{name: "reference", checked: referenceInvalid, want: "semanticCaseId"},
		{
			name:    "scenario",
			checked: scenarioInvalid,
			want:    string(ExecutionScenarioIDAllFailed),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := validateIntegrityRequirements(test.checked)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("SOT-ENG-026: phase error = %v", err)
			}
			if !reflect.DeepEqual(got, integrityCheckedCorpus{}) {
				t.Fatal("SOT-ENG-026: phase 失敗時に部分結果を返した")
			}
		})
	}
}

func TestValidateExecutionScenarioRequirementsのErrorは原文を含まない(t *testing.T) {
	t.Parallel()

	const secretQuery = "secret-query-must-not-leak"
	fixture := executionScenarioRequirementsItemBudgetFixture(t, 5, 4)
	values := executionReferenceTestSemanticCaseValues{
		caseID:             fixture.development[0].CaseID(),
		query:              secretQuery,
		steps:              []map[string]any{validLawSearchStep()},
		meanings:           []executionReferenceTestMeaningValues{{id: "meaning-one", steps: []map[string]any{validLawSearchStep()}}},
		selectedMeaningIDs: []string{"meaning-one"},
	}
	fixture.development[0] = executionReferenceTestSemanticCase(t, values)

	err := validateExecutionScenarioRequirements(
		fixture.manifest,
		fixture.development,
		fixture.execution,
	)
	if err == nil {
		t.Fatal("SOT-ENG-026: 不正な item-budget を受理した")
	}
	if strings.Contains(err.Error(), secretQuery) {
		t.Fatalf("SOT-ENG-026: scenario error が query 原文を含む: %v", err)
	}
}

type executionScenarioRequirementsFixture struct {
	manifest    Manifest
	development []SemanticCase
	execution   []ExecutionCase
}

func executionScenarioRequirementsValidFixture(
	t *testing.T,
) executionScenarioRequirementsFixture {
	t.Helper()

	scenarioValues := validExecutionScenarioCases(t)
	scenarioIDs := manifestRequiredExecutionScenarioIDs()
	development := make([]SemanticCase, 0, len(scenarioIDs))
	execution := make([]ExecutionCase, 0, len(scenarioIDs))
	for _, scenarioID := range scenarioIDs {
		semanticCase := executionScenarioRequirementsSemanticCase(
			t,
			ExecutionScenarioID(scenarioID),
		)
		values := scenarioValues[scenarioID]
		values.CaseID = scenarioID
		values.SemanticCaseID = semanticCase.CaseID()
		executionCase, err := NewExecutionCase(values)
		if err != nil {
			t.Fatalf("SOT-ENG-026: scenario execution case error = %v", err)
		}
		development = append(development, semanticCase)
		execution = append(execution, executionCase)
	}
	return executionScenarioRequirementsFixture{
		manifest: executionScenarioRequirementsManifest(
			t,
			semanticCaseIDsForScenarioTest(development),
			[]string{"holdout-placeholder"},
			executionCaseIDsForScenarioTest(execution),
		),
		development: development,
		execution:   execution,
	}
}

func executionScenarioRequirementsSemanticCase(
	t *testing.T,
	scenarioID ExecutionScenarioID,
) SemanticCase {
	t.Helper()

	suffix := strings.TrimPrefix(string(scenarioID), "execution-")
	values := executionReferenceTestSemanticCaseValues{
		caseID:             "development-" + suffix,
		meanings:           []executionReferenceTestMeaningValues{{id: "meaning-one"}},
		selectedMeaningIDs: []string{"meaning-one"},
	}
	switch scenarioID {
	case ExecutionScenarioIDNonempty:
		values.meanings[0].steps = []map[string]any{validLawReadStep()}
	case ExecutionScenarioIDPartialFailure:
		values.meanings[0].steps = []map[string]any{
			validLawReadStep(),
			validLawReadStep(),
		}
	case ExecutionScenarioIDReversedCompletion:
		values.decision = "hedged"
		values.reasonCodes = []string{"hedged_close_candidates"}
		values.meanings = []executionReferenceTestMeaningValues{
			{id: "meaning-one", steps: []map[string]any{validLawReadStep()}},
			{id: "meaning-two", steps: []map[string]any{validLawReadStep()}},
		}
		values.selectedMeaningIDs = []string{"meaning-one", "meaning-two"}
	case ExecutionScenarioIDItemBudget:
		values.limit = intPointer(20)
		values.meanings[0].steps = []map[string]any{validLawSearchStep()}
	case ExecutionScenarioIDMixedComposition:
		values.meanings[0].steps = []map[string]any{
			validLawReadStep(),
			validJudicialSearchStep(),
		}
		values.meanings[0].requiredPacks = []string{"judicial-cases"}
		values.enabledPacks = []string{"judicial-cases"}
	default:
		values.meanings[0].steps = []map[string]any{validLawSearchStep()}
	}
	return executionReferenceTestSemanticCase(t, values)
}

func executionScenarioRequirementsItemBudgetFixture(
	t *testing.T,
	sourceItemCount int,
	publishedItemCount int,
) executionScenarioRequirementsFixture {
	t.Helper()

	development := []SemanticCase{
		executionReferenceTestSemanticCase(
			t,
			executionReferenceTestSemanticCaseValues{
				caseID: "development-item-budget",
				meanings: []executionReferenceTestMeaningValues{{
					id:    "meaning-one",
					steps: []map[string]any{validLawSearchStep()},
				}},
				selectedMeaningIDs: []string{"meaning-one"},
			},
		),
	}
	outcome, err := NewCollectionSuccessOutcome(sourceItemCount)
	if err != nil {
		t.Fatalf("SOT-ENG-026: item-budget outcome error = %v", err)
	}
	action := mustCaseAction(t, "meaning-one", 1, 1, outcome)
	expected := mustCaseExpectedResult(
		t,
		"completed",
		publishedItemCount,
		[]ExpectedAttempt{
			mustCompletedCollectionAttempt(
				t,
				"meaning-one",
				1,
				publishedItemCount,
				sourceItemCount > publishedItemCount,
			),
		},
	)
	values := executionCaseValuesForActions(
		t,
		[]string{string(ExecutionScenarioIDItemBudget)},
		[]ExecutionAction{action},
		expected,
	)
	values.CaseID = "execution-item-budget"
	values.SemanticCaseID = development[0].CaseID()
	executionCase, err := NewExecutionCase(values)
	if err != nil {
		t.Fatalf("SOT-ENG-026: item-budget execution case error = %v", err)
	}
	execution := []ExecutionCase{executionCase}
	return executionScenarioRequirementsFixture{
		manifest: executionScenarioRequirementsManifest(
			t,
			semanticCaseIDsForScenarioTest(development),
			[]string{"holdout-placeholder"},
			executionCaseIDsForScenarioTest(execution),
		),
		development: development,
		execution:   execution,
	}
}

func executionScenarioRequirementsSingleReversedFixture(
	t *testing.T,
) executionScenarioRequirementsFixture {
	t.Helper()

	development := []SemanticCase{
		executionReferenceTestSemanticCase(
			t,
			executionReferenceTestSemanticCaseValues{
				caseID: "development-reversed-completion",
				meanings: []executionReferenceTestMeaningValues{
					{id: "meaning-one", steps: []map[string]any{validLawReadStep()}},
					{id: "meaning-two", steps: []map[string]any{validLawReadStep()}},
				},
				selectedMeaningIDs: []string{"meaning-one"},
			},
		),
	}
	values := validExecutionScenarioCases(t)[string(ExecutionScenarioIDReversedCompletion)]
	values.CaseID = "execution-reversed-completion"
	values.SemanticCaseID = development[0].CaseID()
	executionCase, err := NewExecutionCase(values)
	if err != nil {
		t.Fatalf("SOT-ENG-026: reversed execution case error = %v", err)
	}
	execution := []ExecutionCase{executionCase}
	return executionScenarioRequirementsFixture{
		manifest: executionScenarioRequirementsManifest(
			t,
			semanticCaseIDsForScenarioTest(development),
			[]string{"holdout-placeholder"},
			executionCaseIDsForScenarioTest(execution),
		),
		development: development,
		execution:   execution,
	}
}

func removeSemanticCaseAt(
	values []SemanticCase,
	index int,
) []SemanticCase {
	result := make([]SemanticCase, 0, len(values)-1)
	result = append(result, values[:index]...)
	return append(result, values[index+1:]...)
}

func removeExecutionCaseAt(
	values []ExecutionCase,
	index int,
) []ExecutionCase {
	result := make([]ExecutionCase, 0, len(values)-1)
	result = append(result, values[:index]...)
	return append(result, values[index+1:]...)
}

func semanticCaseIDsForScenarioTest(values []SemanticCase) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.CaseID())
	}
	return result
}

func executionCaseIDsForScenarioTest(values []ExecutionCase) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.CaseID())
	}
	return result
}

func executionScenarioRequirementsManifest(
	t *testing.T,
	developmentIDs []string,
	holdoutIDs []string,
	executionIDs []string,
) Manifest {
	t.Helper()
	return executionScenarioRequirementsManifestForVersion(
		t,
		"corpus-v4",
		developmentIDs,
		holdoutIDs,
		executionIDs,
	)
}

func executionScenarioRequirementsManifestForVersion(
	t *testing.T,
	corpusVersion string,
	developmentIDs []string,
	holdoutIDs []string,
	executionIDs []string,
) Manifest {
	t.Helper()

	manifest, err := NewManifest(ManifestValues{
		ArtifactKind:        ArtifactKindCorpusManifest,
		SchemaVersion:       1,
		CorpusVersion:       corpusVersion,
		Seed:                1,
		HoldoutDigest:       strings.Repeat("f", 64),
		RequiredCategoryIDs: requiredCategoryIDs(),
		RequiredExecutionScenarioIDs: manifestRequiredExecutionScenarioIDsForVersion(
			corpusVersion,
		),
		Development: mustCorpusManifestSetForTest(
			t,
			ManifestSetDevelopment,
			developmentIDs,
			0,
		),
		Holdout: mustCorpusManifestSetForTest(
			t,
			ManifestSetHoldout,
			holdoutIDs,
			1000,
		),
		Execution: mustCorpusManifestSetForTest(
			t,
			ManifestSetExecution,
			executionIDs,
			2000,
		),
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: scenario manifest error = %v", err)
	}
	return manifest
}
