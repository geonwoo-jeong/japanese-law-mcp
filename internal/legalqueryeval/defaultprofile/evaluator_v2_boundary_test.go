package defaultprofile

import (
	"context"
	"fmt"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryplanning"
)

func TestEvaluatorV2はRequest境界不一致だけを評価失敗へ変換する(
	t *testing.T,
) {
	strict, err := New()
	if err != nil {
		t.Fatalf("SOT-ENG-024: v1 evaluator を構築できません: %v", err)
	}
	v2, err := NewV2()
	if err != nil {
		t.Fatalf("SOT-ENG-024/038: v2 evaluator を構築できません: %v", err)
	}

	planCase := syntheticBoundaryPlanCase(t, "合成した計画境界入力")
	argumentError, err := legalquery.NewArgumentError(
		"query",
		"は一文字以上でなければなりません",
	)
	if err != nil {
		t.Fatalf("合成 ArgumentError を作成できません: %v", err)
	}
	if _, err := evaluateRequestError(
		planCase,
		argumentError,
		strict.boundaryPolicy,
	); err == nil {
		t.Fatal("SOT-ENG-024: v1 evaluator が plan と ArgumentError の不一致を受理しました")
	}
	planEvaluation, err := evaluateRequestError(
		planCase,
		argumentError,
		v2.boundaryPolicy,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024/038/040: v2 plan 境界不一致 error = %v", err)
	}
	if planEvaluation.PlanOutcomeMatched() ||
		len(planEvaluation.Meanings()) != 1 ||
		planEvaluation.Meanings()[0].SignatureMatched() {
		t.Fatalf("SOT-ENG-024: v2 plan 境界不一致 = %#v", planEvaluation)
	}

	requestErrorCase := syntheticAcceptedRequestErrorCase(t)
	if _, err := evaluateAcceptedRequestError(
		requestErrorCase,
		strict.boundaryPolicy,
	); err == nil {
		t.Fatal("SOT-ENG-024: v1 evaluator が受理済み request_error を許可しました")
	}
	requestEvaluation, err := evaluateAcceptedRequestError(
		requestErrorCase,
		v2.boundaryPolicy,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024/038/040: v2 request_error 境界不一致 error = %v", err)
	}
	if requestEvaluation.RequestErrorMatched() {
		t.Fatalf("SOT-ENG-024: v2 request_error 境界不一致 = %#v", requestEvaluation)
	}
}

func TestEvaluatorV2は前処理内部ErrorをHardErrorのまま返す(t *testing.T) {
	base, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ENG-024: planning を構築できません: %v", err)
	}
	evaluator, err := NewWithPlanningV2(failingPreprocessPlanning{Planning: base})
	if err != nil {
		t.Fatalf("SOT-ENG-024/038: v2 evaluator を構築できません: %v", err)
	}
	_, _, _, err = evaluator.EvaluateWithPlan(
		context.Background(),
		syntheticBoundaryPlanCase(t, "合成した前処理内部エラー入力"),
	)
	if err == nil {
		t.Fatal("SOT-ENG-024/040: 前処理内部 error を semantic failure に変換しました")
	}
}

type failingPreprocessPlanning struct {
	Planning
}

func (failingPreprocessPlanning) Preprocessor() legalquery.QueryPreprocessor {
	return failingPreprocessor{}
}

type failingPreprocessor struct{}

func (failingPreprocessor) Preprocess(
	context.Context,
	legalquery.Request,
) (legalquery.PreprocessResult, error) {
	return legalquery.PreprocessResult{}, fmt.Errorf("合成した前処理内部エラー")
}

func syntheticBoundaryPlanCase(
	t *testing.T,
	query string,
) legalquerycorpus.SemanticCase {
	t.Helper()

	input, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{Query: "行政手続法"},
	)
	if err != nil {
		t.Fatalf("合成 law search input を作成できません: %v", err)
	}
	step, err := legalquerycorpus.NewExpectedStep(
		legalquerycorpus.ExpectedStepValues{
			Task:         legalquery.TaskSearch,
			Resource:     legalquery.ResourceLaw,
			InputKind:    legalquery.InputKindLawSearch,
			LogicalInput: input,
		},
	)
	if err != nil {
		t.Fatalf("合成 expected step を作成できません: %v", err)
	}
	meaning, err := legalquerycorpus.NewExpectedMeaning(
		legalquerycorpus.ExpectedMeaningValues{
			MeaningID:     "meaning-synthetic-boundary",
			EvidenceCodes: []legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			Steps:         []legalquerycorpus.ExpectedStep{step},
		},
	)
	if err != nil {
		t.Fatalf("合成 expected meaning を作成できません: %v", err)
	}
	expected, err := legalquerycorpus.NewExpectedPlan(
		legalquerycorpus.ExpectedPlanValues{
			Decision: legalquery.PlanDecisionSingle,
			ReasonCodes: []legalquery.ReasonCode{
				legalquery.ReasonCodeSingleClearCandidate,
			},
			Meanings:           []legalquerycorpus.ExpectedMeaning{meaning},
			SelectedMeaningIDs: []string{"meaning-synthetic-boundary"},
		},
	)
	if err != nil {
		t.Fatalf("合成 expected plan を作成できません: %v", err)
	}
	return mustSyntheticSemanticCase(
		t,
		"holdout-synthetic-plan-argument-error",
		"input-query-empty",
		query,
		expected,
	)
}

func syntheticAcceptedRequestErrorCase(
	t *testing.T,
) legalquerycorpus.SemanticCase {
	t.Helper()

	expected, err := legalquerycorpus.NewExpectedRequestError(
		legalquerycorpus.ExpectedRequestErrorValues{
			ErrorCode: "invalid_argument",
			Field:     legalquerycorpus.RequestErrorFieldQuery,
		},
	)
	if err != nil {
		t.Fatalf("合成 expected request error を作成できません: %v", err)
	}
	return mustSyntheticSemanticCase(
		t,
		"holdout-synthetic-accepted-request",
		"input-query-empty",
		"",
		expected,
	)
}

func mustSyntheticSemanticCase(
	t *testing.T,
	caseID string,
	coverageID string,
	query string,
	expected legalquerycorpus.SemanticExpected,
) legalquerycorpus.SemanticCase {
	t.Helper()

	request, err := legalquerycorpus.NewRequest(
		legalquerycorpus.RequestValues{Query: query},
	)
	if err != nil {
		t.Fatalf("合成 corpus request を作成できません: %v", err)
	}
	semanticCase, err := legalquerycorpus.NewSemanticCase(
		legalquerycorpus.SemanticCaseValues{
			ArtifactKind:   legalquerycorpus.ArtifactKindSemanticCase,
			SchemaVersion:  1,
			CaseID:         caseID,
			LeakageGroupID: caseID + "-group",
			CoverageIDs:    []string{coverageID},
			Request:        request,
			Expected:       expected,
		},
	)
	if err != nil {
		t.Fatalf("合成 semantic case を作成できません: %v", err)
	}
	return semanticCase
}
