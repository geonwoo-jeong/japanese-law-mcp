package mcp

import (
	"fmt"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestQueryLegalInformationOutputSchemaAcceptsSixResultsAndEightAttempts(
	t *testing.T,
) {
	t.Parallel()

	schema := newQueryLegalInformationOutputSchema()
	for name, instance := range querySchemaValidResults() {
		t.Run("result_"+name, func(t *testing.T) {
			assertQuerySchemaAccepts(t, schema, instance)
		})
	}

	for _, fixture := range querySchemaSuccessAttempts() {
		fixture := fixture
		t.Run("attempt_"+fixture.resultKind, func(t *testing.T) {
			result := querySchemaExecutionResult(
				"completed",
				"single",
				[]any{querySchemaInterpretation(
					"available",
					[]any{},
					[]any{fixture.step},
				)},
				[]any{fixture.attempt},
				[]any{},
			)
			assertQuerySchemaAccepts(t, schema, result)
		})
	}

	t.Run("attempt_failed", func(t *testing.T) {
		emptyStep := querySchemaStep(
			"step-empty",
			"search",
			"law",
			"law.search",
		)
		failedStep := querySchemaStep(
			"step-failed",
			"search",
			"law",
			"law.search",
		)
		result := querySchemaExecutionResult(
			"partial",
			"single",
			[]any{querySchemaInterpretation(
				"available",
				[]any{},
				[]any{emptyStep, failedStep},
			)},
			[]any{
				querySchemaEmptyLawSearchAttempt("step-empty"),
				querySchemaFailedAttempt("step-failed"),
			},
			[]any{
				legalquery.LegalQuerySeparateAttemptsNotice,
				legalquery.LegalQueryPartialFailureNotice,
			},
		)
		assertQuerySchemaAccepts(t, schema, result)
	})

	t.Run("ambiguous_location_candidates_have_no_artificial_limit", func(t *testing.T) {
		instance := querySchemaValidResults()["partial"]
		candidates := make([]any, 21)
		for index := range candidates {
			candidates[index] = fmt.Sprintf("候補%d", index+1)
		}
		errorResult, err := model.NewErrorResult(model.ErrorResultValues{
			Code: model.ErrorCodeAmbiguousLocation,
			Details: map[string]any{
				"candidates": candidates,
			},
		})
		if err != nil {
			t.Fatalf("ErrorResult を作成できません: %v", err)
		}
		querySchemaAttemptAt(instance, 1)["error"] = map[string]any{
			"code":      string(errorResult.Code()),
			"message":   errorResult.Message(),
			"retryable": errorResult.Retryable(),
			"details": map[string]any{
				"candidates": candidates,
			},
		}
		assertQuerySchemaAccepts(t, schema, instance)
	})

	t.Run("equal_lower_bound_may_leave_has_more_unknown", func(t *testing.T) {
		instance := querySchemaValidResults()["completed"]
		page := querySchemaAttemptResult(instance, 0)["page"].(map[string]any)
		page["totalRelation"] = "lower_bound"
		delete(page, "hasMore")
		assertQuerySchemaAccepts(t, schema, instance)
	})
}

func TestQueryLegalInformationOutputSchemaRejectsInvalidVariantCombinations(
	t *testing.T,
) {
	t.Parallel()

	schema := newQueryLegalInformationOutputSchema()
	lawSearch := querySchemaSuccessAttempts()[0]
	base := querySchemaExecutionResult(
		"completed",
		"single",
		[]any{querySchemaInterpretation(
			"available",
			[]any{},
			[]any{lawSearch.step},
		)},
		[]any{lawSearch.attempt},
		[]any{},
	)

	tests := map[string]func(map[string]any){
		"collection outcome and items disagree": func(instance map[string]any) {
			querySchemaAttemptAt(instance, 0)["outcome"] = "empty"
		},
		"read attempt cannot be empty": func(instance map[string]any) {
			document := querySchemaSuccessAttempts()[2]
			instance["interpretations"] = []any{querySchemaInterpretation(
				"available",
				[]any{},
				[]any{document.step},
			)}
			instance["attempts"] = []any{document.attempt}
			querySchemaAttemptAt(instance, 0)["outcome"] = "empty"
		},
		"failed attempt cannot have result kind": func(instance map[string]any) {
			querySchemaSetPartialAttempts(instance)
			failed := querySchemaAttemptAt(instance, 1)
			failed["resultKind"] = "law_search"
		},
		"failed attempt cannot have result": func(instance map[string]any) {
			querySchemaSetPartialAttempts(instance)
			failed := querySchemaAttemptAt(instance, 1)
			failed["result"] = map[string]any{
				"page":  querySchemaPage(0),
				"items": []any{},
			}
		},
		"partial requires a successful attempt": func(instance map[string]any) {
			querySchemaSetPartialAttempts(instance)
			instance["attempts"] = []any{
				querySchemaFailedAttempt("step-empty"),
				querySchemaFailedAttempt("step-failed"),
			}
		},
		"partial requires a failed attempt": func(instance map[string]any) {
			querySchemaSetPartialAttempts(instance)
			instance["attempts"] = []any{
				querySchemaEmptyLawSearchAttempt("step-empty"),
				querySchemaEmptyLawSearchAttempt("step-failed"),
			}
		},
		"single decision requires one interpretation": func(instance map[string]any) {
			second := querySchemaCloneMap(
				t,
				instance["interpretations"].([]any)[0].(map[string]any),
			)
			second["interpretationId"] = "interpretation-2"
			instance["interpretations"] = append(
				instance["interpretations"].([]any),
				second,
			)
		},
		"hedged decision requires two interpretations": func(instance map[string]any) {
			instance["decision"] = "hedged"
		},
		"single interpretation starts at one": func(instance map[string]any) {
			querySchemaInterpretationAt(instance, 0)["interpretationId"] =
				"interpretation-2"
			querySchemaAttemptAt(instance, 0)["interpretationId"] =
				"interpretation-2"
		},
		"evidence codes keep public rank order": func(instance map[string]any) {
			querySchemaInterpretationAt(instance, 0)["evidenceCodes"] = []any{
				"general_term",
				"explicit_task",
			}
		},
		"clarification values keep fixed order": func(instance map[string]any) {
			instance["status"] = "needs_clarification"
			instance["decision"] = "no_execution"
			instance["interpretations"] = []any{}
			instance["attempts"] = []any{}
			instance["notices"] = []any{}
			instance["clarification"] = map[string]any{
				"reasonCodes": []any{
					"ambiguous_candidates",
					"below_execution_threshold",
				},
				"questions": []any{
					string(legalquery.LegalQueryQuestionResource),
					string(legalquery.LegalQueryQuestionTask),
				},
			}
		},
		"law item requires law resource type": func(instance map[string]any) {
			querySchemaSourcedItemRefKey(instance, 0)["resourceType"] =
				"judicial-decision"
		},
		"judicial item rejects version": func(instance map[string]any) {
			judicial := querySchemaSuccessAttempts()[5]
			instance["interpretations"] = []any{querySchemaInterpretation(
				"available",
				[]any{},
				[]any{judicial.step},
			)}
			instance["attempts"] = []any{judicial.attempt}
			querySchemaSourcedItemRefKey(instance, 0)["versionId"] =
				"unexpected"
		},
		"law update requires list resource type": func(instance map[string]any) {
			update := querySchemaSuccessAttempts()[4]
			instance["interpretations"] = []any{querySchemaInterpretation(
				"available",
				[]any{},
				[]any{update.step},
			)}
			instance["attempts"] = []any{update.attempt}
			querySchemaSourcedItemRefKey(instance, 0)["resourceType"] =
				"law"
		},
		"page total cannot be below returned count": func(instance map[string]any) {
			page := querySchemaAttemptResult(instance, 0)["page"].(map[string]any)
			page["totalCount"] = 0
		},
		"page exact total determines has more": func(instance map[string]any) {
			page := querySchemaAttemptResult(instance, 0)["page"].(map[string]any)
			page["totalCount"] = 2
			page["hasMore"] = false
		},
		"page lower bound remainder requires has more": func(instance map[string]any) {
			page := querySchemaAttemptResult(instance, 0)["page"].(map[string]any)
			page["totalCount"] = 2
			page["totalRelation"] = "lower_bound"
			delete(page, "hasMore")
		},
		"page lower bound remainder requires first page notice": func(instance map[string]any) {
			page := querySchemaAttemptResult(instance, 0)["page"].(map[string]any)
			page["totalCount"] = 2
			page["totalRelation"] = "lower_bound"
			page["hasMore"] = true
		},
		"page count equals item count": func(instance map[string]any) {
			page := querySchemaAttemptResult(instance, 0)["page"].(map[string]any)
			page["returnedCount"] = 2
			page["totalCount"] = 2
		},
		"has more requires first page notice": func(instance map[string]any) {
			page := querySchemaAttemptResult(instance, 0)["page"].(map[string]any)
			page["hasMore"] = true
			page["totalCount"] = 2
		},
		"single attempt rejects separate attempts notice": func(instance map[string]any) {
			instance["notices"] = []any{
				legalquery.LegalQuerySeparateAttemptsNotice,
			}
		},
		"not found retryable is fixed": func(instance map[string]any) {
			querySchemaSetPartialAttempts(instance)
			querySchemaAttemptAt(instance, 1)["error"] = map[string]any{
				"code":      "not_found",
				"message":   "指定した条件に該当する情報がありません。",
				"retryable": true,
			}
		},
		"internal error rejects details": func(instance map[string]any) {
			querySchemaSetPartialAttempts(instance)
			querySchemaAttemptAt(instance, 1)["error"] = map[string]any{
				"code":      "internal_error",
				"message":   "内部処理を完了できませんでした。",
				"retryable": false,
				"details": map[string]any{
					"providerId": []any{"secret"},
				},
			}
		},
		"page must not expose next token": func(instance map[string]any) {
			querySchemaAttemptResult(instance, 0)["page"].(map[string]any)["nextToken"] = "secret"
		},
		"page must not expose next offset": func(instance map[string]any) {
			querySchemaAttemptResult(instance, 0)["page"].(map[string]any)["nextOffset"] = 1
		},
		"interpretation must not expose score": func(instance map[string]any) {
			querySchemaInterpretationAt(instance, 0)["score"] = 90
		},
		"interpretation must not expose candidate id": func(instance map[string]any) {
			querySchemaInterpretationAt(instance, 0)["candidateId"] = "candidate-1"
		},
		"nested resource must reject unknown field": func(instance map[string]any) {
			item := querySchemaAttemptResult(instance, 0)["items"].([]any)[0].(map[string]any)
			item["providerRoute"] = "internal"
		},
	}

	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			instance := querySchemaCloneMap(t, base)
			mutate(instance)
			assertQuerySchemaRejects(t, schema, instance)
		})
	}
}
