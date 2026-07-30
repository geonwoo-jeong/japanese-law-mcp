package defaultprofile

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestEvaluatorは内蔵DefaultProfileで代表Holdoutを評価する(t *testing.T) {
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("corpus-v4 を読み込めません: %v", err)
	}
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}

	holdoutByID := make(map[string]legalquerycorpus.SemanticCase)
	for _, semanticCase := range corpus.Holdout() {
		holdoutByID[semanticCase.CaseID()] = semanticCase
	}
	caseIDs := []string{
		"holdout-input-01",
		"holdout-input-17",
		"holdout-input-18",
		"holdout-intent-01",
		"holdout-intent-04",
		"holdout-intent-06",
		"holdout-ambiguity-01",
		"holdout-ambiguity-02",
		"holdout-ambiguity-04",
		"holdout-ambiguity-06",
		"holdout-ambiguity-07",
		"holdout-ambiguity-08",
		"holdout-ambiguity-09",
		"holdout-pack-01",
		"holdout-pack-11",
		"holdout-pack-08",
		"holdout-name-19",
		"holdout-name-20",
		"holdout-reference-05",
		"holdout-reference-06",
		"holdout-reference-07",
		"holdout-reference-08",
		"holdout-structure-16",
		"holdout-structure-18",
		"holdout-typo-02",
		"holdout-typo-03",
		"holdout-typo-05",
	}
	for _, caseID := range caseIDs {
		semanticCase, exists := holdoutByID[caseID]
		if !exists {
			t.Fatalf("fixture %q が corpus-v4 にありません", caseID)
		}
		t.Run(caseID, func(t *testing.T) {
			evaluation, err := evaluator.Evaluate(
				context.Background(),
				semanticCase,
			)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			switch semanticCase.Expected().Kind() {
			case legalquerycorpus.SemanticExpectedKindPlan:
				if !evaluation.PlanOutcomeMatched() {
					request, requestErr := productRequest(semanticCase.Request())
					if requestErr != nil {
						t.Fatalf("診断用 request を構築できません: %v", requestErr)
					}
					plan, planErr := evaluator.selectPlan(
						context.Background(),
						semanticCase,
						request,
					)
					if planErr != nil {
						t.Fatalf("診断用 plan を構築できません: %v", planErr)
					}
					t.Fatalf(
						"decision、reason または selection が一致しません: decision=%q reasons=%#v selected=%#v ranked=%#v",
						plan.Decision(),
						plan.ReasonCodes(),
						plan.Selected(),
						plan.RankedCandidates(),
					)
				}
				for _, meaning := range evaluation.Meanings() {
					if !meaning.SignatureMatched() {
						t.Fatalf("meaning %q の意味署名が一致しません", meaning.MeaningID())
					}
					if matched, applicable := meaning.EvidenceAssertion(); applicable && !matched {
						t.Fatalf("meaning %q の根拠 assertion が一致しません", meaning.MeaningID())
					}
					if matched, applicable := meaning.ConceptAssertion(); applicable && !matched {
						t.Fatalf("meaning %q の法概念 assertion が一致しません", meaning.MeaningID())
					}
				}
			case legalquerycorpus.SemanticExpectedKindRequestError:
				if !evaluation.RequestErrorMatched() {
					t.Fatal("request error の code または field が一致しません")
				}
			default:
				t.Fatalf("expected.kind = %q は未対応です", semanticCase.Expected().Kind())
			}
		})
	}
}

func TestEvaluatorはPlanCaseだけに再現可能なPlanを返す(t *testing.T) {
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("corpus-v4 を読み込めません: %v", err)
	}
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}

	holdoutByID := make(map[string]legalquerycorpus.SemanticCase)
	for _, semanticCase := range corpus.Holdout() {
		holdoutByID[semanticCase.CaseID()] = semanticCase
	}
	planCase := holdoutByID["holdout-intent-01"]
	firstEvaluation, firstPlan, firstHasPlan, err := evaluator.EvaluateWithPlan(
		context.Background(),
		planCase,
	)
	if err != nil {
		t.Fatalf("plan case の EvaluateWithPlan() error = %v", err)
	}
	secondEvaluation, secondPlan, secondHasPlan, err := evaluator.EvaluateWithPlan(
		context.Background(),
		planCase,
	)
	if err != nil {
		t.Fatalf("plan case の二回目の EvaluateWithPlan() error = %v", err)
	}
	if !firstHasPlan || !secondHasPlan ||
		firstEvaluation.CaseID() != planCase.CaseID() ||
		secondEvaluation.CaseID() != planCase.CaseID() {
		t.Fatal("SOT-ENG-024: plan case の評価または plan が返りませんでした")
	}
	if firstPlan.ProfileVersion() != secondPlan.ProfileVersion() ||
		firstPlan.Decision() != secondPlan.Decision() {
		t.Fatal("SOT-ENG-024: 同じ plan case の plan が再現しません")
	}

	requestErrorCase := holdoutByID["holdout-input-01"]
	evaluation, _, hasPlan, err := evaluator.EvaluateWithPlan(
		context.Background(),
		requestErrorCase,
	)
	if err != nil {
		t.Fatalf("request_error case の EvaluateWithPlan() error = %v", err)
	}
	if hasPlan || evaluation.CaseID() != requestErrorCase.CaseID() {
		t.Fatal("SOT-ENG-024: request_error case に plan を返しました")
	}
}

func TestEvaluatorはCorpusV4のExecutionFixture全件を製品Executorで評価する(
	t *testing.T,
) {
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("corpus-v4 を読み込めません: %v", err)
	}
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}

	report, err := evaluator.EvaluateExecution(
		context.Background(),
		corpus,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024: execution fixture を評価できません: %v", err)
	}
	if report.CaseCount() != len(corpus.Execution()) {
		t.Fatalf(
			"execution case count = %d, want %d",
			report.CaseCount(),
			len(corpus.Execution()),
		)
	}
	if report.WrongResourceCallCount() != 0 ||
		report.BudgetViolationCount() != 0 ||
		report.AttemptOrderViolationCount() != 0 ||
		report.ImplicitFirstReadCount() != 0 ||
		report.EmptyReclassificationCount() != 0 ||
		len(report.FailedCaseIDs()) != 0 {
		t.Fatalf("SOT-ENG-024: execution report = %#v", report)
	}
	for _, metric := range report.Metrics() {
		if metric.Numerator() != metric.Denominator() {
			t.Fatalf(
				"SOT-ENG-024: execution metric %q = %d/%d",
				metric.MetricID(),
				metric.Numerator(),
				metric.Denominator(),
			)
		}
	}
}

func TestExecutionEvaluatorは誤Resource予算超過と順序逆転を検出する(
	t *testing.T,
) {
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("corpus-v4 を読み込めません: %v", err)
	}
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}
	development := make(map[string]legalquerycorpus.SemanticCase)
	for _, semanticCase := range corpus.Development() {
		development[semanticCase.CaseID()] = semanticCase
	}
	executionCase := corpus.Execution()[0]
	semanticCase := development[executionCase.SemanticCaseID()]
	semanticEvaluation, plan, hasPlan, err := evaluator.EvaluateWithPlan(
		context.Background(),
		semanticCase,
	)
	if err != nil || !hasPlan {
		t.Fatalf("試験用 plan を作成できません: hasPlan=%t error=%v", hasPlan, err)
	}
	fixture, err := resolveExecutionFixture(
		semanticCase,
		semanticEvaluation,
		plan,
		executionCase,
	)
	if err != nil {
		t.Fatalf("試験用 execution fixture を解決できません: %v", err)
	}

	facade, err := newExecutionFixtureFacade(fixture)
	if err != nil {
		t.Fatalf("試験用 fake facade を作成できません: %v", err)
	}
	first := fixture.actions[0]
	second := fixture.actions[1]
	if _, err := facade.takeAction(
		context.Background(),
		first.step.InputKind(),
		second.step.LogicalInput(),
		first.budget,
	); err == nil {
		t.Fatal("SOT-ENG-024: plan と異なる resource/input 呼出しを受理しました")
	}
	if facade.diagnostics().wrongResourceCallCount == 0 {
		t.Fatal("SOT-ENG-024: 誤 resource 呼出しを計数しませんでした")
	}

	if got := resultBudgetViolationCount(
		plan,
		[]observedExecutionAttempt{{publishedItemCount: 41}},
	); got == 0 {
		t.Fatal("SOT-ENG-024: item 予算超過を検出しませんでした")
	}

	expected := executionCase.Expected().Attempts()
	reversed := []observedExecutionAttempt{
		{
			meaningID:   expected[1].MeaningID(),
			stepOrdinal: expected[1].StepOrdinal(),
		},
		{
			meaningID:   expected[0].MeaningID(),
			stepOrdinal: expected[0].StepOrdinal(),
		},
	}
	if compareAttemptOrder(expected, reversed) ||
		attemptOrderDifferenceCount(expected, reversed) == 0 {
		t.Fatal("SOT-ENG-024: attempt 順序の逆転を検出しませんでした")
	}
}

func TestExecutionEvaluatorはStepIDで結果予算を照合する(t *testing.T) {
	corpus, evaluator := loadExecutionTestRuntime(t)
	semanticCase, executionCase := executionTestCases(
		t,
		corpus,
		"execution-mixed-composition",
	)
	semanticEvaluation, plan, hasPlan, err := evaluator.EvaluateWithPlan(
		context.Background(),
		semanticCase,
	)
	if err != nil || !hasPlan {
		t.Fatalf("試験用 plan を作成できません: hasPlan=%t error=%v", hasPlan, err)
	}
	fixture, err := resolveExecutionFixture(
		semanticCase,
		semanticEvaluation,
		plan,
		executionCase,
	)
	if err != nil {
		t.Fatalf("試験用 execution fixture を解決できません: %v", err)
	}
	observed := make(
		[]observedExecutionAttempt,
		0,
		len(fixture.actions),
	)
	for _, action := range fixture.actions {
		count := 1
		if limit, exists := action.budget.EffectiveLimit(); exists {
			count = limit
		}
		observed = append(observed, observedExecutionAttempt{
			stepID:             action.step.StepID(),
			publishedItemCount: count,
		})
	}
	observed[1], observed[2] = observed[2], observed[1]
	if got := resultBudgetViolationCount(plan, observed); got != 0 {
		t.Fatalf("SOT-ENG-024: step ID 照合の予算違反件数 = %d", got)
	}
}

func TestProfileVersionReportsはProfileID順へ正規化する(t *testing.T) {
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}
	metadata := evaluator.planning.ProfileMetadata()
	metadata[0], metadata[1] = metadata[1], metadata[0]
	reports, err := profileVersionReports(metadata)
	if err != nil {
		t.Fatalf("profile version report を構築できません: %v", err)
	}
	if reports[0].ProfileID() != "core" ||
		reports[1].ProfileID() != "judicial-cases" {
		t.Fatalf("SOT-ENG-024: profile version 順 = %#v", reports)
	}
}

func TestExecutionFixtureFacadeは空結果後の計画外再分類を検出する(
	t *testing.T,
) {
	corpus, evaluator := loadExecutionTestRuntime(t)
	semanticCase, executionCase := executionTestCases(
		t,
		corpus,
		"execution-empty",
	)
	semanticEvaluation, plan, hasPlan, err := evaluator.EvaluateWithPlan(
		context.Background(),
		semanticCase,
	)
	if err != nil || !hasPlan {
		t.Fatalf("試験用 plan を作成できません: hasPlan=%t error=%v", hasPlan, err)
	}
	fixture, err := resolveExecutionFixture(
		semanticCase,
		semanticEvaluation,
		plan,
		executionCase,
	)
	if err != nil {
		t.Fatalf("試験用 execution fixture を解決できません: %v", err)
	}
	facade, err := newExecutionFixtureFacade(fixture)
	if err != nil {
		t.Fatalf("試験用 fake facade を作成できません: %v", err)
	}
	action := fixture.actions[0]
	if _, err := facade.takeAction(
		context.Background(),
		action.step.InputKind(),
		action.step.LogicalInput(),
		action.budget,
	); err != nil {
		t.Fatalf("空結果 action を解放できません: %v", err)
	}
	if _, err := facade.takeAction(
		context.Background(),
		"law_read",
		action.step.LogicalInput(),
		action.budget,
	); err == nil {
		t.Fatal("空結果後の計画外 read を受理しました")
	}
	diagnostics := facade.diagnostics()
	if diagnostics.emptyReclassificationCount != 1 ||
		diagnostics.implicitFirstReadCount != 1 {
		t.Fatalf(
			"SOT-ENG-024: 空結果後の診断 = %#v",
			diagnostics,
		)
	}
}

func TestExecutionFixtureFacadeはRelease待機中もContext中断を返す(
	t *testing.T,
) {
	corpus, evaluator := loadExecutionTestRuntime(t)
	semanticCase, executionCase := executionTestCases(
		t,
		corpus,
		"execution-reversed-completion",
	)
	semanticEvaluation, plan, hasPlan, err := evaluator.EvaluateWithPlan(
		context.Background(),
		semanticCase,
	)
	if err != nil || !hasPlan {
		t.Fatalf("試験用 plan を作成できません: hasPlan=%t error=%v", hasPlan, err)
	}
	fixture, err := resolveExecutionFixture(
		semanticCase,
		semanticEvaluation,
		plan,
		executionCase,
	)
	if err != nil {
		t.Fatalf("試験用 execution fixture を解決できません: %v", err)
	}
	facade, err := newExecutionFixtureFacade(fixture)
	if err != nil {
		t.Fatalf("試験用 fake facade を作成できません: %v", err)
	}
	var blocked resolvedExecutionAction
	for _, action := range fixture.actions {
		if action.releaseOrder > 1 {
			blocked = action
			break
		}
	}
	if blocked.releaseOrder == 0 {
		t.Fatal("releaseOrder > 1 の action がありません")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := facade.takeAction(
		cancelled,
		blocked.step.InputKind(),
		blocked.step.LogicalInput(),
		blocked.budget,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-ENG-024: release 待機の中断 error = %v", err)
	}
}

func TestEvaluatorは標準Reportへ全測定値とProfile版を集約する(t *testing.T) {
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("corpus-v4 を読み込めません: %v", err)
	}
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}

	report, err := evaluator.BuildStandardReport(
		context.Background(),
		corpus,
		"default-1",
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024: standard report を構築できません: %v", err)
	}
	if report.CorpusVersion() != corpus.Manifest().CorpusVersion() ||
		report.HoldoutDigest() != corpus.Manifest().HoldoutDigest() ||
		report.Sets().Development().CaseCount() != len(corpus.Development()) ||
		report.Sets().Holdout().CaseCount() != len(corpus.Holdout()) ||
		report.Sets().Execution().CaseCount() != len(corpus.Execution()) ||
		len(report.ProfileSet().Profiles()) != 2 {
		t.Fatalf("SOT-ENG-024: standard report = %#v", report)
	}
	observations := report.Sets().Holdout().DerivedObservations()
	expectedObservationIDs := []string{
		"composition-core-pack",
		"composition-pack-disabled",
		"composition-ref-read-search",
		"composition-four-step-budget",
	}
	if len(observations) != len(expectedObservationIDs) {
		t.Fatalf("SOT-ENG-026: derived observations = %#v", observations)
	}
	for index, expectedID := range expectedObservationIDs {
		if observations[index].MetricID() != expectedID ||
			observations[index].Denominator() == 0 {
			t.Fatalf(
				"SOT-ENG-026: derived observations[%d] = %#v",
				index,
				observations[index],
			)
		}
	}
	if err := report.Validate(); err != nil {
		t.Fatalf("SOT-ENG-024: standard report が有効ではありません: %v", err)
	}
}

func loadExecutionTestRuntime(
	t *testing.T,
) (legalquerycorpus.Corpus, *Evaluator) {
	t.Helper()

	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("corpus-v4 を読み込めません: %v", err)
	}
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}
	return corpus, evaluator
}

func executionTestCases(
	t *testing.T,
	corpus legalquerycorpus.Corpus,
	executionCaseID string,
) (legalquerycorpus.SemanticCase, legalquerycorpus.ExecutionCase) {
	t.Helper()

	development := make(map[string]legalquerycorpus.SemanticCase)
	for _, semanticCase := range corpus.Development() {
		development[semanticCase.CaseID()] = semanticCase
	}
	for _, executionCase := range corpus.Execution() {
		if executionCase.CaseID() != executionCaseID {
			continue
		}
		semanticCase, exists := development[executionCase.SemanticCaseID()]
		if !exists {
			t.Fatalf(
				"execution case %q の semantic case がありません",
				executionCaseID,
			)
		}
		return semanticCase, executionCase
	}
	t.Fatalf("execution case %q がありません", executionCaseID)
	return legalquerycorpus.SemanticCase{}, legalquerycorpus.ExecutionCase{}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test file path を取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}
