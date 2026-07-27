package legalquery

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestLegalQueryInterpretationsDerivePublicIDsAndSelectedFields(t *testing.T) {
	t.Parallel()

	first := planTestCandidate(
		t,
		"internal-candidate-first",
		900,
		nil,
		planTestStep(t, "法令検索", "step-first"),
	)
	second, err := NewLegalQueryCandidate(LegalQueryCandidateValues{
		CandidateID:    "internal-candidate-second",
		SemanticScore:  850,
		Confidence:     ConfidenceMedium,
		EvidenceCodes:  []EvidenceCode{EvidenceExplicitResource},
		ConceptSources: []LegalConceptSource{},
		RequiredPacks:  []string{"tax"},
		Steps: []LegalQueryCandidateStep{
			planTestStep(t, "法令本文検索", "step-second"),
		},
	})
	if err != nil {
		t.Fatalf("試験用 candidate を作成できません: %v", err)
	}
	plan, err := NewLegalQueryPlan(planTestValues(
		PlanDecisionHedged,
		[]LegalQueryCandidate{first, second},
		[]LegalQueryPlanSelection{
			planTestSelection(t, first, SelectionAvailabilityAvailable),
			planTestSelection(t, second, SelectionAvailabilityAvailable),
		},
		[]ReasonCode{ReasonCodeHedgedCloseCandidates},
	))
	if err != nil {
		t.Fatalf("試験用 plan を作成できません: %v", err)
	}

	interpretations, err := NewLegalQueryInterpretations(plan)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: interpretations を作成できません: %v", err)
	}
	if len(interpretations) != 2 {
		t.Fatalf("SOT-MODEL-024: interpretations 件数 = %d", len(interpretations))
	}
	if interpretations[0].InterpretationID() != "interpretation-1" ||
		interpretations[1].InterpretationID() != "interpretation-2" ||
		interpretations[0].Confidence() != ConfidenceHigh ||
		interpretations[1].Confidence() != ConfidenceMedium ||
		interpretations[1].Availability() != SelectionAvailabilityAvailable ||
		!reflect.DeepEqual(
			interpretations[1].EvidenceCodes(),
			[]EvidenceCode{EvidenceExplicitResource},
		) ||
		!reflect.DeepEqual(interpretations[1].RequiredPacks(), []string{"tax"}) {
		t.Fatalf("SOT-MODEL-024: interpretations = %#v", interpretations)
	}
	steps := interpretations[1].Steps()
	if len(steps) != 1 ||
		steps[0].StepID() != "step-second" ||
		steps[0].Task() != TaskSearch ||
		steps[0].Resource() != ResourceLawProvision ||
		steps[0].CapabilityID() != second.Steps()[0].CapabilityID() ||
		steps[0].CapabilityMajorVersion() != 1 {
		t.Fatalf("SOT-MODEL-024: steps = %#v", steps)
	}

	encoded, err := json.Marshal(interpretations)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: JSON に変換できません: %v", err)
	}
	for _, forbidden := range []string{
		"candidateId",
		"internal-candidate",
		"semanticScore",
		`"score"`,
	} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("SOT-MODEL-024: 内部値 %q が公開されました: %s", forbidden, encoded)
		}
	}
}

func TestLegalQueryInterpretationPublishesLegalConceptSourceFields(t *testing.T) {
	t.Parallel()

	source, err := NewLegalConceptSource(LegalConceptSourceValues{
		ConceptID:   "permanent-residence",
		Title:       "永住許可に関するガイドライン",
		URL:         "https://www.moj.go.jp/isa/applications/resources/nyukan_nyukan50.html",
		ConfirmedOn: mustDate(t, "2026-07-27"),
	})
	if err != nil {
		t.Fatalf("試験用 concept source を作成できません: %v", err)
	}
	candidate, err := NewLegalQueryCandidate(LegalQueryCandidateValues{
		CandidateID:    "candidate-concept-source",
		SemanticScore:  900,
		Confidence:     ConfidenceHigh,
		EvidenceCodes:  []EvidenceCode{EvidenceLegalConcept},
		ConceptSources: []LegalConceptSource{source},
		RequiredPacks:  []string{},
		Steps: []LegalQueryCandidateStep{
			planTestStep(t, "法令検索", "step-concept-source"),
		},
	})
	if err != nil {
		t.Fatalf("試験用 candidate を作成できません: %v", err)
	}
	plan, err := NewLegalQueryPlan(planTestValues(
		PlanDecisionSingle,
		[]LegalQueryCandidate{candidate},
		[]LegalQueryPlanSelection{
			planTestSelection(t, candidate, SelectionAvailabilityAvailable),
		},
		[]ReasonCode{ReasonCodeSingleClearCandidate},
	))
	if err != nil {
		t.Fatalf("試験用 plan を作成できません: %v", err)
	}
	interpretations, err := NewLegalQueryInterpretations(plan)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: interpretations を作成できません: %v", err)
	}

	encoded, err := json.Marshal(interpretations)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: JSON に変換できません: %v", err)
	}
	for _, expected := range []string{
		`"conceptId":"permanent-residence"`,
		`"title":"永住許可に関するガイドライン"`,
		`"url":"https://www.moj.go.jp/isa/applications/resources/nyukan_nyukan50.html"`,
		`"confirmedOn":"2026-07-27"`,
	} {
		if !strings.Contains(string(encoded), expected) {
			t.Fatalf("SOT-MODEL-024: concept source の %s がありません: %s", expected, encoded)
		}
	}
}

func TestLegalQueryInterpretationAndStepSummaryAreImmutable(t *testing.T) {
	t.Parallel()

	candidate := planTestCandidate(
		t,
		"candidate-immutable",
		900,
		[]string{"tax"},
		planTestStep(t, "法令検索", "step-immutable"),
	)
	plan, err := NewLegalQueryPlan(planTestValues(
		PlanDecisionSingle,
		[]LegalQueryCandidate{candidate},
		[]LegalQueryPlanSelection{
			planTestSelection(t, candidate, SelectionAvailabilityAvailable),
		},
		[]ReasonCode{ReasonCodeSingleClearCandidate},
	))
	if err != nil {
		t.Fatalf("試験用 plan を作成できません: %v", err)
	}
	interpretations, err := NewLegalQueryInterpretations(plan)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: interpretation を作成できません: %v", err)
	}
	summary, err := NewLegalQueryStepSummary(candidate.Steps()[0])
	if err != nil {
		t.Fatalf("SOT-MODEL-024: step summary を作成できません: %v", err)
	}

	evidence := interpretations[0].EvidenceCodes()
	packs := interpretations[0].RequiredPacks()
	steps := interpretations[0].Steps()
	evidence[0] = EvidenceGeneralTerm
	packs[0] = "changed"
	steps[0] = LegalQueryStepSummary{}
	interpretations[0] = LegalQueryInterpretation{}
	if got := interpretationsFromPlan(t, plan)[0]; got.EvidenceCodes()[0] != EvidenceExplicitTask ||
		got.RequiredPacks()[0] != "tax" ||
		got.Steps()[0].StepID() != "step-immutable" {
		t.Fatal("SOT-MODEL-024: getter から interpretation を変更できました")
	}
	if summary.StepID() != "step-immutable" ||
		summary.CapabilityID() != candidate.Steps()[0].CapabilityID() {
		t.Fatalf("SOT-MODEL-024: summary = %#v", summary)
	}
}

func TestLegalQueryClarificationUsesOnlyFixedOrderedQuestions(t *testing.T) {
	t.Parallel()

	candidate := planTestCandidate(
		t,
		"candidate-question",
		600,
		nil,
		planTestStep(t, "法令検索", "step-question"),
	)
	plan, err := NewLegalQueryPlan(planTestValues(
		PlanDecisionNeedsClarification,
		[]LegalQueryCandidate{candidate},
		[]LegalQueryPlanSelection{
			planTestSelection(t, candidate, SelectionAvailabilityAvailable),
		},
		[]ReasonCode{
			ReasonCodeBelowExecutionThreshold,
			ReasonCodeAmbiguousCandidates,
		},
	))
	if err != nil {
		t.Fatalf("試験用 plan を作成できません: %v", err)
	}
	questions := []LegalQueryQuestion{
		LegalQueryQuestionTask,
		LegalQueryQuestionJudicialDecision,
	}
	clarification, err := NewLegalQueryClarification(plan, questions)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: clarification を作成できません: %v", err)
	}
	questions[0] = LegalQueryQuestionResource
	gotQuestions := clarification.Questions()
	gotReasons := clarification.ReasonCodes()
	if !reflect.DeepEqual(gotQuestions, []LegalQueryQuestion{
		LegalQueryQuestionTask,
		LegalQueryQuestionJudicialDecision,
	}) || !reflect.DeepEqual(gotReasons, []ReasonCode{
		ReasonCodeBelowExecutionThreshold,
		ReasonCodeAmbiguousCandidates,
	}) {
		t.Fatalf("SOT-MODEL-024: clarification = %#v", clarification)
	}
	gotQuestions[0] = LegalQueryQuestionResource
	gotReasons[0] = ReasonCodeAmbiguousCandidates
	if clarification.Questions()[0] != LegalQueryQuestionTask ||
		clarification.ReasonCodes()[0] != ReasonCodeBelowExecutionThreshold {
		t.Fatal("SOT-MODEL-024: clarification を getter から変更できました")
	}

	encoded, err := json.Marshal(clarification)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: JSON に変換できません: %v", err)
	}
	for _, question := range []string{
		"検索、読取りまたは更新一覧のどれを行うか指定してください。",
		"対象の裁判例を検索する語または裁判例の ref を指定してください。",
	} {
		if !strings.Contains(string(encoded), question) {
			t.Fatalf("SOT-MODEL-024: 固定質問がありません: %s", encoded)
		}
	}

	for name, invalid := range map[string][]LegalQueryQuestion{
		"空":  {},
		"重複": {LegalQueryQuestionTask, LegalQueryQuestionTask},
		"逆順": {
			LegalQueryQuestionLaw,
			LegalQueryQuestionResource,
		},
		"未知": {LegalQueryQuestion("自由記述")},
		"三件": {
			LegalQueryQuestionTask,
			LegalQueryQuestionResource,
			LegalQueryQuestionLaw,
		},
	} {
		if _, err := NewLegalQueryClarification(plan, invalid); err == nil {
			t.Fatalf("SOT-MODEL-024: %sの questions を受理しました", name)
		}
	}
}

func TestLegalQueryInterpretationObjectsRejectDirectJSONRestore(t *testing.T) {
	t.Parallel()

	for _, target := range []any{
		&LegalQueryStepSummary{},
		&LegalQueryInterpretation{},
		&LegalQueryClarification{},
	} {
		if err := json.Unmarshal([]byte(`{}`), target); err == nil {
			t.Fatalf("SOT-MODEL-024: %T を JSON から直接復元できました", target)
		}
	}
}

func interpretationsFromPlan(
	t *testing.T,
	plan LegalQueryPlan,
) []LegalQueryInterpretation {
	t.Helper()
	values, err := NewLegalQueryInterpretations(plan)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: interpretations を再作成できません: %v", err)
	}
	return values
}
