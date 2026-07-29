package mcp

import (
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestQueryLegalInformationOutputSchemaはstep上限結果の直列化と一致する(
	t *testing.T,
) {
	t.Parallel()

	plan, err := legalquery.NewLegalQueryPlan(
		legalquery.LegalQueryPlanValues{
			ProfileVersion: "profile-step-limit-v1",
			Decision:       legalquery.PlanDecisionNeedsClarification,
			ReasonCodes: []legalquery.ReasonCode{
				legalquery.ReasonCodeStepLimitExceeded,
			},
			LimitPerAttempt: legalquery.DefaultLimitPerAttempt,
		},
	)
	if err != nil {
		t.Fatalf("試験用 step 上限 plan を作成できません: %v", err)
	}
	result, err := legalquery.AssembleLegalQueryNonExecutionResult(plan)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: step 上限結果を作成できません: %v", err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("SOT-IF-051: step 上限結果を直列化できません: %v", err)
	}
	var instance map[string]any
	if err := json.Unmarshal(encoded, &instance); err != nil {
		t.Fatalf("SOT-IF-051: step 上限結果を検証値へ復元できません: %v", err)
	}
	assertQuerySchemaAccepts(
		t,
		newQueryLegalInformationOutputSchema(),
		instance,
	)
}

func TestQueryLegalInformationOutputSchemaはstep上限の明確化だけを受理する(
	t *testing.T,
) {
	t.Parallel()

	schema := newQueryLegalInformationOutputSchema()
	assertQuerySchemaAccepts(
		t,
		schema,
		querySchemaStepLimitClarification(),
	)

	tests := map[string]func(map[string]any){
		"一般質問": func(instance map[string]any) {
			querySchemaClarification(instance)["questions"] = []any{
				string(legalquery.LegalQueryQuestionTask),
			}
		},
		"専用質問と通常理由": func(instance map[string]any) {
			querySchemaClarification(instance)["reasonCodes"] = []any{
				string(legalquery.ReasonCodeBelowExecutionThreshold),
			}
		},
		"他理由との併存": func(instance map[string]any) {
			querySchemaClarification(instance)["reasonCodes"] = []any{
				string(legalquery.ReasonCodeBelowExecutionThreshold),
				string(legalquery.ReasonCodeStepLimitExceeded),
			}
		},
		"部分解釈": func(instance map[string]any) {
			instance["interpretations"] = []any{
				querySchemaInterpretation(
					"available",
					[]any{},
					[]any{
						querySchemaStep(
							"step-partial",
							"search",
							"law",
							"law.search",
						),
					},
				),
			}
		},
	}
	for name, mutate := range tests {
		mutate := mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			instance := querySchemaStepLimitClarification()
			mutate(instance)
			assertQuerySchemaRejects(t, schema, instance)
		})
	}
}

func querySchemaStepLimitClarification() map[string]any {
	return map[string]any{
		"status":          "needs_clarification",
		"decision":        "no_execution",
		"language":        "ja",
		"interpretations": []any{},
		"attempts":        []any{},
		"notices":         []any{},
		"clarification": map[string]any{
			"reasonCodes": []any{
				string(legalquery.ReasonCodeStepLimitExceeded),
			},
			"questions": []any{
				string(legalquery.LegalQueryQuestionStepLimitExceeded),
			},
		},
	}
}

func querySchemaClarification(instance map[string]any) map[string]any {
	return instance["clarification"].(map[string]any)
}
