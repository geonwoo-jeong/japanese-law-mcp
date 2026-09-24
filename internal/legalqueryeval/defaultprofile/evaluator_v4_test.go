package defaultprofile

import (
	"context"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryplanning"
)

func TestEvaluatorV4はV3の評価写像を保持する(t *testing.T) {
	const verificationID = "candidate-evaluator-v4-scoring-equivalence"
	base, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		t.Fatalf("%s: planning を構築できません: %v", verificationID, err)
	}
	planCase := syntheticBoundaryPlanCase(t, "行政手続法を検索")
	for _, testCase := range []struct {
		name     string
		planning Planning
		input    legalquerycorpus.SemanticCase
	}{
		{name: "通常計画", planning: base, input: planCase},
		{name: "入力拒否", planning: base, input: syntheticAcceptedRequestErrorCase(t)},
		{name: "前処理の入力別失敗", planning: markedFailingPreprocessPlanning{Planning: base}, input: planCase},
		{name: "回収の入力別失敗", planning: newCollectFailurePlanning(t), input: planCase},
		{name: "未分類失敗", planning: failingPreprocessPlanning{Planning: base}, input: planCase},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			v3, previousErr := NewWithPlanningV3(testCase.planning)
			v4, currentErr := NewWithPlanningV4(testCase.planning)
			if previousErr != nil || currentErr != nil {
				t.Fatalf("%s: evaluator を構築できません: %v / %v", verificationID, previousErr, currentErr)
			}
			previous, previousPlan, previousHasPlan, previousErr := v3.EvaluateWithPlan(context.Background(), testCase.input)
			current, currentPlan, currentHasPlan, currentErr := v4.EvaluateWithPlan(context.Background(), testCase.input)
			if !reflect.DeepEqual(previous, current) || !reflect.DeepEqual(previousPlan, currentPlan) ||
				previousHasPlan != currentHasPlan || (previousErr == nil) != (currentErr == nil) {
				t.Fatalf("%s: v3/v4 の評価結果が一致しません", verificationID)
			}
			if previousErr != nil && previousErr.Error() != currentErr.Error() {
				t.Fatalf("%s: hard error の内容が一致しません", verificationID)
			}
		})
	}
}
