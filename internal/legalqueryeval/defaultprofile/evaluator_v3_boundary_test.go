package defaultprofile

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryplanning"
)

func TestEvaluatorV3はPlan生成Errorを評価失敗にする(
	t *testing.T,
) {
	const verificationID = "candidate-evaluator-v3-case-failure-scoring"

	base, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		t.Fatalf("%s: planning を構築できません: %v", verificationID, err)
	}
	planning := markedFailingPreprocessPlanning{Planning: base}
	semanticCase := syntheticBoundaryPlanCase(
		t,
		"合成した v3 plan 生成エラー入力",
	)
	for _, previous := range []struct {
		name string
		new  func(Planning) (*Evaluator, error)
	}{
		{name: "v1", new: NewWithPlanning},
		{name: "v2", new: NewWithPlanningV2},
	} {
		previousEvaluator, previousErr := previous.new(planning)
		if previousErr != nil {
			t.Fatalf("%s: %s evaluator を構築できません: %v", verificationID, previous.name, previousErr)
		}
		if _, _, _, previousErr = previousEvaluator.EvaluateWithPlan(
			context.Background(),
			semanticCase,
		); previousErr == nil {
			t.Fatalf("SOT-ENG-041: %s が分類済み前処理 error を評価失敗へ変換しました", previous.name)
		}
	}

	evaluator, err := NewWithPlanningV3(planning)
	if err != nil {
		t.Fatalf("SOT-ENG-024/038: v3 evaluator を構築できません: %v", err)
	}

	evaluation, _, hasPlan, err := evaluator.EvaluateWithPlan(
		context.Background(),
		semanticCase,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024/040: plan 生成 error が hard error になりました: %v", err)
	}
	if hasPlan {
		t.Fatal("SOT-ENG-024: plan 生成失敗に plan を返しました")
	}
	if evaluation.PlanOutcomeMatched() || len(evaluation.Meanings()) != 1 ||
		evaluation.Meanings()[0].SignatureMatched() {
		t.Fatalf("SOT-ENG-024: plan 生成失敗の評価 = %#v", evaluation)
	}
}

func TestEvaluatorV3はProfile回収Errorを評価失敗にする(t *testing.T) {
	const verificationID = "candidate-evaluator-v3-case-failure-scoring"

	planning := newCollectFailurePlanning(t)
	semanticCase := syntheticBoundaryPlanCase(
		t,
		"合成した v3 profile 回収エラー入力",
	)
	for _, previous := range []struct {
		name string
		new  func(Planning) (*Evaluator, error)
	}{
		{name: "v1", new: NewWithPlanning},
		{name: "v2", new: NewWithPlanningV2},
	} {
		previousEvaluator, err := previous.new(planning)
		if err != nil {
			t.Fatalf("%s: %s evaluator を構築できません: %v", verificationID, previous.name, err)
		}
		if _, _, _, err := previousEvaluator.EvaluateWithPlan(
			context.Background(),
			semanticCase,
		); err == nil {
			t.Fatalf("SOT-ENG-041: %s が profile 回収 error を評価失敗へ変換しました", previous.name)
		}
	}
	v3, err := NewWithPlanningV3(planning)
	if err != nil {
		t.Fatalf("%s: v3 evaluator を構築できません: %v", verificationID, err)
	}
	evaluation, _, hasPlan, err := v3.EvaluateWithPlan(
		context.Background(),
		semanticCase,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-041: v3 profile 回収 error = %v", err)
	}
	if hasPlan || evaluation.PlanOutcomeMatched() ||
		len(evaluation.Meanings()) != 1 ||
		evaluation.Meanings()[0].SignatureMatched() {
		t.Fatalf("SOT-ENG-041: v3 profile 回収失敗の評価 = %#v", evaluation)
	}
}

func TestEvaluatorV3は分類外Errorを評価失敗にしない(t *testing.T) {
	const verificationID = "candidate-evaluator-v3-unclassified-hard-error"

	base, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		t.Fatalf("%s: planning を構築できません: %v", verificationID, err)
	}
	evaluator, err := NewWithPlanningV3(
		failingPreprocessPlanning{Planning: base},
	)
	if err != nil {
		t.Fatalf("%s: v3 evaluator を構築できません: %v", verificationID, err)
	}
	if _, _, _, err := evaluator.EvaluateWithPlan(
		context.Background(),
		syntheticBoundaryPlanCase(t, "合成した未分類の前処理エラー入力"),
	); err == nil {
		t.Fatalf("%s: 未分類の前処理 error を評価失敗へ変換しました", verificationID)
	}

	if shouldScoreCandidatePlanningFailure(
		context.Background(),
		markCandidateCaseFailure(fmt.Errorf("合成した分類済み error")),
		candidatePlanningStageNone,
		planningFailureScoredMismatch,
	) {
		t.Fatalf("%s: 分類外 error を候補 planning failure として受理しました", verificationID)
	}
}

func TestEvaluatorV3はProfile生成InvariantをHardErrorのまま返す(t *testing.T) {
	const verificationID = "candidate-evaluator-v3-profile-invariant-hard-error"

	evaluator, err := NewWithPlanningV3(newInvalidGenerationPlanning(t))
	if err != nil {
		t.Fatalf("%s: v3 evaluator を構築できません: %v", verificationID, err)
	}
	if _, _, _, err := evaluator.EvaluateWithPlan(
		context.Background(),
		syntheticBoundaryPlanCase(t, "合成した profile generation invariant 入力"),
	); err == nil {
		t.Fatalf("%s: profile generation invariant を評価失敗へ変換しました", verificationID)
	}
}

func TestEvaluatorV3はSelectorErrorをHardErrorのまま返す(t *testing.T) {
	const verificationID = "candidate-evaluator-v3-selector-hard-error"

	evaluator, err := NewWithPlanningV3(newUnadoptedPackPlanning(t))
	if err != nil {
		t.Fatalf("%s: v3 evaluator を構築できません: %v", verificationID, err)
	}
	if _, _, _, err := evaluator.EvaluateWithPlan(
		context.Background(),
		syntheticBoundaryPlanCase(t, "合成した未採用 pack 候補入力"),
	); err == nil {
		t.Fatalf("%s: selector error を評価失敗へ変換しました", verificationID)
	}
}

func TestEvaluatorV3はPack状態Errorを候補失敗より先に返す(t *testing.T) {
	const verificationID = "candidate-evaluator-v3-pack-state-precedence"

	base, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		t.Fatalf("%s: planning を構築できません: %v", verificationID, err)
	}
	evaluator, err := NewWithPlanningV3(
		markedFailingPreprocessPlanning{Planning: base},
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024/038: v3 evaluator を構築できません: %v", err)
	}
	semanticCase := syntheticBoundaryPlanCase(
		t,
		"合成した v3 pack 状態エラー入力",
	)
	semanticCase, err = legalquerycorpus.NewSemanticCase(
		legalquerycorpus.SemanticCaseValues{
			ArtifactKind:   semanticCase.ArtifactKind(),
			SchemaVersion:  semanticCase.SchemaVersion(),
			CaseID:         "holdout-synthetic-v3-pack-state",
			LeakageGroupID: "holdout-synthetic-v3-pack-state-group",
			CoverageIDs:    semanticCase.CoverageIDs(),
			EnabledPacks:   []string{"unknown-pack"},
			Request:        semanticCase.Request(),
			Expected:       semanticCase.Expected(),
		},
	)
	if err != nil {
		t.Fatalf("合成 pack state semantic case を作成できません: %v", err)
	}

	_, _, _, err = evaluator.EvaluateWithPlan(context.Background(), semanticCase)
	if err == nil {
		t.Fatal("SOT-ENG-024/040: corpus の pack 状態 error を候補失敗へ変換しました")
	}
}

func TestEvaluatorV3は取消しをHardErrorのまま返す(
	t *testing.T,
) {
	const verificationID = "candidate-evaluator-v3-cancellation-hard-error"

	base, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		t.Fatalf("%s: planning を構築できません: %v", verificationID, err)
	}
	evaluator, err := NewWithPlanningV3(
		markedFailingPreprocessPlanning{Planning: base},
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024/038: v3 evaluator を構築できません: %v", err)
	}

	contexts := []struct {
		name string
		ctx  context.Context
	}{
		{name: "cancel", ctx: cancelledContext()},
		{name: "deadline", ctx: expiredContext()},
	}
	for _, testCase := range contexts {
		t.Run(testCase.name, func(t *testing.T) {
			_, _, _, evaluateErr := evaluator.EvaluateWithPlan(
				testCase.ctx,
				syntheticBoundaryPlanCase(t, "合成した v3 取消し入力"),
			)
			if evaluateErr == nil {
				t.Fatalf("%s: 取消しを評価失敗へ変換しました", verificationID)
			}
		})
	}
}

type failingGenerationProfile struct {
	metadata legalquery.QueryProfileMetadata
}

func (p failingGenerationProfile) Metadata() legalquery.QueryProfileMetadata {
	return p.metadata
}

func (failingGenerationProfile) CueVocabulary() []legalquery.CueVocabularyEntry {
	return nil
}

func (failingGenerationProfile) Generate(
	legalquery.CandidateGenerationInput,
	legalquery.CandidateIDScope,
) (legalquery.CandidateGeneration, error) {
	return legalquery.CandidateGeneration{}, markCandidateCaseFailure(
		fmt.Errorf("合成した profile 回収 error"),
	)
}

type invalidGenerationProfile struct {
	metadata legalquery.QueryProfileMetadata
}

func (p invalidGenerationProfile) Metadata() legalquery.QueryProfileMetadata {
	return p.metadata
}

func (invalidGenerationProfile) CueVocabulary() []legalquery.CueVocabularyEntry {
	return nil
}

func (invalidGenerationProfile) Generate(
	legalquery.CandidateGenerationInput,
	legalquery.CandidateIDScope,
) (legalquery.CandidateGeneration, error) {
	return legalquery.CandidateGeneration{}, nil
}

type unadoptedPackProfile struct {
	metadata legalquery.QueryProfileMetadata
}

func (p unadoptedPackProfile) Metadata() legalquery.QueryProfileMetadata {
	return p.metadata
}

func (unadoptedPackProfile) CueVocabulary() []legalquery.CueVocabularyEntry {
	return nil
}

func (p unadoptedPackProfile) Generate(
	input legalquery.CandidateGenerationInput,
	scope legalquery.CandidateIDScope,
) (legalquery.CandidateGeneration, error) {
	logicalInput, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{Query: "行政手続法"},
	)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	candidate, err := legalquery.AssembleLegalQueryCandidate(
		legalquery.CandidateAssemblyValues{
			IDScope:          scope,
			CandidateOrdinal: 1,
			SemanticScore:    100,
			Confidence:       legalquery.ConfidenceMedium,
			EvidenceCodes:    []legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			RequiredPacks:    []string{"tax"},
			LogicalInputs:    []legalquery.LogicalInput{logicalInput},
		},
	)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	return legalquery.NewCandidateGeneration(
		legalquery.CandidateGenerationValues{
			ProfileID:      p.metadata.ProfileID(),
			ProfileVersion: p.metadata.ProfileVersion(),
			RankingVersion: p.metadata.RankingVersion(),
			Candidates:     []legalquery.LegalQueryCandidate{candidate},
			SelectionMode:  legalquery.QuerySelectionModeAutomatic,
		},
	)
}

type markedFailingPreprocessPlanning struct {
	Planning
}

func (markedFailingPreprocessPlanning) Preprocessor() legalquery.QueryPreprocessor {
	return markedFailingPreprocessor{}
}

type markedFailingPreprocessor struct{}

func (markedFailingPreprocessor) Preprocess(
	context.Context,
	legalquery.Request,
) (legalquery.PreprocessResult, error) {
	return legalquery.PreprocessResult{}, markCandidateCaseFailure(
		fmt.Errorf("合成した分類済み前処理 error"),
	)
}

type candidateCaseFailureMarker struct {
	cause error
}

func (m candidateCaseFailureMarker) Error() string {
	return m.cause.Error()
}

func (m candidateCaseFailureMarker) Unwrap() error {
	return m.cause
}

func (candidateCaseFailureMarker) CandidateCaseFailure() {}

func markCandidateCaseFailure(err error) error {
	if err == nil {
		return nil
	}
	return candidateCaseFailureMarker{cause: err}
}

type collectFailurePlanning struct {
	Planning
	profiles legalquery.QueryProfileSet
	metadata []legalquery.QueryProfileMetadata
}

func (p collectFailurePlanning) Profiles() legalquery.QueryProfileSet {
	return p.profiles
}

func (p collectFailurePlanning) ProfileMetadata() []legalquery.QueryProfileMetadata {
	return append([]legalquery.QueryProfileMetadata(nil), p.metadata...)
}

func newCollectFailurePlanning(t *testing.T) Planning {
	return newSingleProfilePlanning(t, func(
		metadata legalquery.QueryProfileMetadata,
	) legalquery.QueryProfile {
		return failingGenerationProfile{metadata: metadata}
	})
}

func newInvalidGenerationPlanning(t *testing.T) Planning {
	return newSingleProfilePlanning(t, func(
		metadata legalquery.QueryProfileMetadata,
	) legalquery.QueryProfile {
		return invalidGenerationProfile{metadata: metadata}
	})
}

func newUnadoptedPackPlanning(t *testing.T) Planning {
	return newSingleProfilePlanning(t, func(
		metadata legalquery.QueryProfileMetadata,
	) legalquery.QueryProfile {
		return unadoptedPackProfile{metadata: metadata}
	})
}

func newSingleProfilePlanning(
	t *testing.T,
	newProfile func(legalquery.QueryProfileMetadata) legalquery.QueryProfile,
) Planning {
	t.Helper()

	base, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ENG-041: planning を構築できません: %v", err)
	}
	metadata := base.ProfileMetadata()[0]
	profiles, err := legalquery.NewQueryProfileSet([]legalquery.QueryProfile{
		newProfile(metadata),
	})
	if err != nil {
		t.Fatalf("SOT-ENG-041: profile 回収失敗 set を構築できません: %v", err)
	}
	return collectFailurePlanning{
		Planning: base,
		profiles: profiles,
		metadata: []legalquery.QueryProfileMetadata{metadata},
	}
}

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func expiredContext() context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
	cancel()
	return ctx
}
