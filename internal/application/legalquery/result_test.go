package legalquery

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLegalQueryResultAcceptsSixConcreteShapes(t *testing.T) {
	t.Parallel()

	fixtures := newAttemptPayloadFixtures(t)
	search := planTestStep(t, "法令検索", "step-search")
	content := planTestStep(t, "法令本文検索", "step-content")
	executing := resultTestPlan(
		t,
		PlanDecisionSingle,
		nil,
		SelectionAvailabilityAvailable,
		ReasonCodeSingleClearCandidate,
		search,
		content,
	)
	nonempty := resultTestLawSearchAttempt(
		t, search, []model.SourcedResource[model.LawSummary]{fixtures.lawSummary}, true,
	)
	emptyContent := resultTestLawContentEmptyAttempt(t, content)
	failed := resultTestFailedAttempt(t, content)

	completed, err := NewLegalQueryCompletedResult(
		executing,
		[]LegalQueryAttempt{nonempty, emptyContent},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: completed を作成できません: %v", err)
	}
	emptyPlan := resultTestPlan(
		t,
		PlanDecisionSingle,
		nil,
		SelectionAvailabilityAvailable,
		ReasonCodeSingleClearCandidate,
		search,
	)
	emptyAttempt := resultTestLawSearchAttempt(t, search, nil, false)
	empty, err := NewLegalQueryEmptyResult(
		emptyPlan,
		[]LegalQueryAttempt{emptyAttempt},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: empty を作成できません: %v", err)
	}
	partial, err := NewLegalQueryPartialResult(
		executing,
		[]LegalQueryAttempt{nonempty, failed},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: partial を作成できません: %v", err)
	}

	clarificationPlan := resultTestPlan(
		t,
		PlanDecisionNeedsClarification,
		nil,
		SelectionAvailabilityAvailable,
		ReasonCodeAmbiguousCandidates,
		search,
	)
	needsClarification, err := NewLegalQueryNeedsClarificationResult(
		clarificationPlan,
		[]LegalQueryQuestion{LegalQueryQuestionResource},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: needs clarification を作成できません: %v", err)
	}
	unavailablePlan := resultTestPlan(
		t,
		PlanDecisionCapabilityUnavailable,
		[]string{"judicial-cases"},
		SelectionAvailabilityPackDisabled,
		ReasonCodeRequiredPackDisabled,
		planTestStep(t, "裁判例検索", "step-pack"),
	)
	unavailable, err := NewLegalQueryCapabilityUnavailableResult(unavailablePlan)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: capability unavailable を作成できません: %v", err)
	}
	unsupportedPlan := resultTestUnsupportedPlan(t)
	unsupported, err := NewLegalQueryUnsupportedResult(unsupportedPlan)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: unsupported を作成できません: %v", err)
	}

	tests := []struct {
		name     string
		result   LegalQueryResult
		status   string
		decision string
	}{
		{"completed", completed, "completed", "single"},
		{"empty", empty, "empty", "single"},
		{"partial", partial, "partial", "single"},
		{"needs clarification", needsClarification, "needs_clarification", "no_execution"},
		{"capability unavailable", unavailable, "capability_unavailable", "no_execution"},
		{"unsupported", unsupported, "unsupported", "no_execution"},
	}
	for _, test := range tests {
		if string(test.result.Status()) != test.status ||
			string(test.result.Decision()) != test.decision ||
			test.result.Language() != "ja" {
			t.Fatalf("SOT-MODEL-024: %s = %#v", test.name, test.result)
		}
		object := attemptTestJSONObject(t, test.result)
		if object["status"] != test.status ||
			object["decision"] != test.decision ||
			object["language"] != "ja" ||
			object["interpretations"] == nil ||
			object["attempts"] == nil ||
			object["notices"] == nil {
			t.Fatalf("SOT-MODEL-024: %s JSON = %#v", test.name, object)
		}
	}
	if len(needsClarification.Attempts()) != 0 ||
		len(needsClarification.Clarification().Questions()) != 1 {
		t.Fatalf("SOT-MODEL-024: clarification result = %#v", needsClarification)
	}
}

func TestLegalQueryResultDerivesStatusAndFixedNotices(t *testing.T) {
	t.Parallel()

	fixtures := newAttemptPayloadFixtures(t)
	search := planTestStep(t, "法令検索", "step-search")
	content := planTestStep(t, "法令本文検索", "step-content")
	plan := resultTestPlan(
		t,
		PlanDecisionSingle,
		nil,
		SelectionAvailabilityAvailable,
		ReasonCodeSingleClearCandidate,
		search,
		content,
	)
	nonempty := resultTestLawSearchAttempt(
		t, search, []model.SourcedResource[model.LawSummary]{fixtures.lawSummary}, true,
	)
	empty := resultTestLawContentEmptyAttempt(t, content)
	completed, err := NewLegalQueryCompletedResult(
		plan,
		[]LegalQueryAttempt{nonempty, empty},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: completed を作成できません: %v", err)
	}
	if !reflect.DeepEqual(completed.Notices(), []string{
		"最初のページだけを返しています。続きが必要な場合は対応する専門ツールを使用してください。",
		"複数の解釈または step の件数、順位および継続位置は統合していません。",
	}) {
		t.Fatalf("SOT-MODEL-024: completed notices = %#v", completed.Notices())
	}

	lowerBoundHasMore := true
	lowerBoundTotal := 2
	lowerBoundPage, err := NewLegalQueryPagePreview(
		LegalQueryPagePreviewValues{
			ReturnedCount: 1,
			HasMore:       &lowerBoundHasMore,
			TotalCount:    &lowerBoundTotal,
			TotalRelation: model.TotalRelationLowerBound,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: lower_bound page を作成できません: %v", err)
	}
	lowerBoundAttempt, err := NewLegalQueryLawSearchAttempt(
		LegalQueryLawSearchAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             search,
			Page:             lowerBoundPage,
			Items: []model.SourcedResource[model.LawSummary]{
				fixtures.lawSummary,
			},
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: lower_bound attempt を作成できません: %v", err)
	}
	lowerBoundCompleted, err := NewLegalQueryCompletedResult(
		plan,
		[]LegalQueryAttempt{lowerBoundAttempt},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: lower_bound completed を作成できません: %v", err)
	}
	if !reflect.DeepEqual(lowerBoundCompleted.Notices(), []string{
		"最初のページだけを返しています。続きが必要な場合は対応する専門ツールを使用してください。",
	}) {
		t.Fatalf(
			"SOT-MODEL-024: lower_bound notices = %#v",
			lowerBoundCompleted.Notices(),
		)
	}

	partial, err := NewLegalQueryPartialResult(
		plan,
		[]LegalQueryAttempt{resultTestFailedAttempt(t, search), empty},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: empty+failed の partial を拒否しました: %v", err)
	}
	if !reflect.DeepEqual(partial.Notices(), []string{
		"複数の解釈または step の件数、順位および継続位置は統合していません。",
		"一部の step が失敗しました。attempts の error を確認してください。",
	}) {
		t.Fatalf("SOT-MODEL-024: partial notices = %#v", partial.Notices())
	}

	unsupported, err := NewLegalQueryUnsupportedResult(resultTestUnsupportedPlan(t))
	if err != nil {
		t.Fatalf("SOT-MODEL-024: unsupported を作成できません: %v", err)
	}
	if !reflect.DeepEqual(unsupported.Notices(), []string{
		"日本語の法情報取得要求を入力してください。",
		"法情報の取得要求と法的助言、翻訳または対象外の要求を分けて入力してください。",
		"指定した法的助言、翻訳、task または resource は統合照会の採用範囲外です。採用済みの法情報取得要求だけを入力してください。",
	}) {
		t.Fatalf("SOT-MODEL-024: unsupported notices = %#v", unsupported.Notices())
	}
}

func TestLegalQueryResultEnforcesAttemptReferencesPlanOrderAndState(t *testing.T) {
	t.Parallel()

	search := planTestStep(t, "法令検索", "step-search")
	content := planTestStep(t, "法令本文検索", "step-content")
	plan := resultTestPlan(
		t,
		PlanDecisionSingle,
		nil,
		SelectionAvailabilityAvailable,
		ReasonCodeSingleClearCandidate,
		search,
		content,
	)
	searchEmpty := resultTestLawSearchAttempt(t, search, nil, false)
	contentEmpty := resultTestLawContentEmptyAttempt(t, content)
	failedSearch := resultTestFailedAttempt(t, search)
	failedContent := resultTestFailedAttempt(t, content)

	if _, err := NewLegalQueryEmptyResult(
		plan,
		[]LegalQueryAttempt{contentEmpty, searchEmpty},
	); err == nil {
		t.Fatal("SOT-MODEL-024: plan と逆順の attempt を受理しました")
	}
	if _, err := NewLegalQueryEmptyResult(
		plan,
		[]LegalQueryAttempt{searchEmpty, searchEmpty},
	); err == nil {
		t.Fatal("SOT-MODEL-024: 同じ step の重複 attempt を受理しました")
	}
	otherInterpretation, err := NewLegalQueryLawSearchAttempt(
		LegalQueryLawSearchAttemptValues{
			InterpretationID: "interpretation-2",
			Step:             search, Page: resultTestUnknownPage(t, 0),
			Items: []model.SourcedResource[model.LawSummary]{},
		},
	)
	if err != nil {
		t.Fatalf("試験用 attempt を作成できません: %v", err)
	}
	if _, err := NewLegalQueryEmptyResult(
		plan,
		[]LegalQueryAttempt{otherInterpretation},
	); err == nil {
		t.Fatal("SOT-MODEL-024: 存在しない interpretation の attempt を受理しました")
	}
	if _, err := NewLegalQueryPartialResult(
		plan,
		[]LegalQueryAttempt{failedSearch, failedContent},
	); err == nil {
		t.Fatal("SOT-MODEL-024: 全 attempt が failed の result を受理しました")
	}
	if _, err := NewLegalQueryCompletedResult(
		plan,
		[]LegalQueryAttempt{searchEmpty},
	); err == nil {
		t.Fatal("SOT-MODEL-024: 非空成功のない completed を受理しました")
	}
	if _, err := NewLegalQueryEmptyResult(
		plan,
		[]LegalQueryAttempt{searchEmpty, failedContent},
	); err == nil {
		t.Fatal("SOT-MODEL-024: failed を含む empty を受理しました")
	}
}

func TestLegalQueryResultIsImmutableAndHidesInternalOrContinuationFields(t *testing.T) {
	t.Parallel()

	fixtures := newAttemptPayloadFixtures(t)
	search := planTestStep(t, "法令検索", "step-search")
	plan := resultTestPlan(
		t,
		PlanDecisionSingle,
		nil,
		SelectionAvailabilityAvailable,
		ReasonCodeSingleClearCandidate,
		search,
	)
	attempts := []LegalQueryAttempt{resultTestLawSearchAttempt(
		t, search, []model.SourcedResource[model.LawSummary]{fixtures.lawSummary}, true,
	)}
	result, err := NewLegalQueryCompletedResult(plan, attempts)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: completed を作成できません: %v", err)
	}
	attempts[0] = nil
	gotAttempts := result.Attempts()
	gotInterpretations := result.Interpretations()
	gotNotices := result.Notices()
	gotAttempts[0] = nil
	gotInterpretations[0] = LegalQueryInterpretation{}
	gotNotices[0] = "変更"
	if len(result.Attempts()) != 1 ||
		result.Interpretations()[0].InterpretationID() != "interpretation-1" ||
		result.Notices()[0] == "変更" {
		t.Fatal("SOT-MODEL-024: result を入力または getter から変更できました")
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: JSON に変換できません: %v", err)
	}
	for _, forbidden := range []string{
		"candidateId", "semanticScore", `"score"`, "nextToken",
		"nextOffset", "continuationToken", resultTestJudicialCoverageNotice,
	} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("SOT-MODEL-024: 禁止値 %q が公開されました: %s", forbidden, encoded)
		}
	}
}

func TestLegalQueryResultsRejectDirectJSONRestore(t *testing.T) {
	t.Parallel()

	for _, target := range []any{
		&LegalQueryCompletedResult{},
		&LegalQueryEmptyResult{},
		&LegalQueryPartialResult{},
		&LegalQueryNeedsClarificationResult{},
		&LegalQueryCapabilityUnavailableResult{},
		&LegalQueryUnsupportedResult{},
	} {
		if err := json.Unmarshal([]byte(`{}`), target); err == nil {
			t.Fatalf("SOT-MODEL-024: %T を JSON から直接復元できました", target)
		}
	}
}

func resultTestPlan(
	t *testing.T,
	decision PlanDecision,
	packs []string,
	availability SelectionAvailability,
	reason ReasonCode,
	steps ...LegalQueryCandidateStep,
) LegalQueryPlan {
	t.Helper()
	candidate := planTestCandidate(t, "candidate-result", 900, packs, steps...)
	plan, err := NewLegalQueryPlan(planTestValues(
		decision,
		[]LegalQueryCandidate{candidate},
		[]LegalQueryPlanSelection{planTestSelection(t, candidate, availability)},
		[]ReasonCode{reason},
	))
	if err != nil {
		t.Fatalf("試験用 plan を作成できません: %v", err)
	}
	return plan
}

func resultTestUnsupportedPlan(t *testing.T) LegalQueryPlan {
	t.Helper()
	candidate := planTestCandidate(
		t, "candidate-unsupported", 100, nil,
		planTestStep(t, "法令検索", "step-unsupported"),
	)
	plan, err := NewLegalQueryPlan(planTestValues(
		PlanDecisionUnsupported,
		[]LegalQueryCandidate{candidate},
		nil,
		[]ReasonCode{
			ReasonCodeNonJapaneseQuery,
			ReasonCodeMixedUnsupportedIntent,
			ReasonCodeUnsupportedTaskOrResource,
		},
	))
	if err != nil {
		t.Fatalf("試験用 unsupported plan を作成できません: %v", err)
	}
	return plan
}

func resultTestLawSearchAttempt(
	t *testing.T,
	step LegalQueryCandidateStep,
	items []model.SourcedResource[model.LawSummary],
	hasMore bool,
) LegalQueryLawSearchAttempt {
	t.Helper()
	count := len(items)
	totalCount := count
	if hasMore {
		totalCount++
	}
	attempt, err := NewLegalQueryLawSearchAttempt(LegalQueryLawSearchAttemptValues{
		InterpretationID: "interpretation-1",
		Step:             step,
		Page: resultTestPage(
			t,
			count,
			hasMore,
			totalCount,
			model.TotalRelationExact,
		),
		Items: items,
	})
	if err != nil {
		t.Fatalf("試験用 law search attempt を作成できません: %v", err)
	}
	return attempt
}

func resultTestLawContentEmptyAttempt(
	t *testing.T,
	step LegalQueryCandidateStep,
) LegalQueryLawContentSearchAttempt {
	t.Helper()
	attempt, err := NewLegalQueryLawContentSearchAttempt(
		LegalQueryLawContentSearchAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             step, Page: resultTestUnknownPage(t, 0),
			Items: []model.SourcedResource[model.LawContentMatch]{},
		},
	)
	if err != nil {
		t.Fatalf("試験用 law content attempt を作成できません: %v", err)
	}
	return attempt
}

func resultTestFailedAttempt(
	t *testing.T,
	step LegalQueryCandidateStep,
) LegalQueryFailedAttempt {
	t.Helper()
	attempt, err := NewLegalQueryFailedAttempt(LegalQueryFailedAttemptValues{
		InterpretationID: "interpretation-1",
		Step:             step,
		Error:            resultTestError(t, model.ErrorCodeInternalError),
	})
	if err != nil {
		t.Fatalf("試験用 failed attempt を作成できません: %v", err)
	}
	return attempt
}
