package mcp

import (
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type querySchemaAttemptFixture struct {
	resultKind string
	step       map[string]any
	attempt    map[string]any
}

func querySchemaValidResults() map[string]map[string]any {
	lawSearch := querySchemaSuccessAttempts()[0]
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
	return map[string]map[string]any{
		"completed": querySchemaExecutionResult(
			"completed",
			"single",
			[]any{querySchemaInterpretation(
				"available",
				[]any{},
				[]any{lawSearch.step},
			)},
			[]any{lawSearch.attempt},
			[]any{},
		),
		"empty": querySchemaExecutionResult(
			"empty",
			"single",
			[]any{querySchemaInterpretation(
				"available",
				[]any{},
				[]any{emptyStep},
			)},
			[]any{querySchemaEmptyLawSearchAttempt("step-empty")},
			[]any{},
		),
		"partial": querySchemaExecutionResult(
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
		),
		"needs clarification": {
			"status":          "needs_clarification",
			"decision":        "no_execution",
			"language":        "ja",
			"interpretations": []any{},
			"attempts":        []any{},
			"notices":         []any{},
			"clarification": map[string]any{
				"reasonCodes": []any{"below_execution_threshold"},
				"questions": []any{
					string(legalquery.LegalQueryQuestionTask),
				},
			},
		},
		"capability unavailable": {
			"status":   "capability_unavailable",
			"decision": "no_execution",
			"language": "ja",
			"interpretations": []any{querySchemaInterpretation(
				"pack_disabled",
				[]any{"judicial-cases"},
				[]any{querySchemaStep(
					"step-judicial-search",
					"search",
					"judicial_decision",
					"judicial-decision.search",
				)},
			)},
			"attempts": []any{},
			"notices": []any{
				legalquery.LegalQueryPackDisabledNotice,
			},
		},
		"unsupported": {
			"status":          "unsupported",
			"decision":        "no_execution",
			"language":        "ja",
			"interpretations": []any{},
			"attempts":        []any{},
			"notices": []any{
				legalquery.LegalQueryNonJapaneseNotice,
			},
		},
	}
}

func querySchemaSuccessAttempts() []querySchemaAttemptFixture {
	return []querySchemaAttemptFixture{
		querySchemaCollectionAttempt(
			"step-law-search",
			"search",
			"law",
			"law.search",
			"law_search",
			querySchemaSourcedLawSummary(),
			false,
		),
		querySchemaCollectionAttempt(
			"step-law-content",
			"search",
			"law_provision",
			"law.content.search",
			"law_content_search",
			querySchemaSourcedLawContent(),
			false,
		),
		querySchemaReadAttempt(
			"step-law-document",
			"read",
			"law",
			"law.document.read",
			"law_document",
			querySchemaSourcedLawDocument(),
			false,
		),
		querySchemaReadAttempt(
			"step-law-article",
			"read",
			"law_provision",
			"law.article.read",
			"law_article",
			querySchemaSourcedLawArticle(),
			false,
		),
		querySchemaCollectionAttempt(
			"step-law-updates",
			"list_updates",
			"law",
			"law.update.list",
			"law_updates",
			querySchemaSourcedLawUpdate(),
			false,
		),
		querySchemaCollectionAttempt(
			"step-judicial-search",
			"search",
			"judicial_decision",
			"judicial-decision.search",
			"judicial_decision_search",
			querySchemaSourcedJudicialSummary(),
			true,
		),
		querySchemaReadAttempt(
			"step-judicial-read",
			"read",
			"judicial_decision",
			"judicial-decision.read",
			"judicial_decision",
			querySchemaSourcedJudicialDetails(),
			true,
		),
	}
}

func querySchemaCollectionAttempt(
	stepID string,
	task string,
	resource string,
	capabilityID string,
	resultKind string,
	item map[string]any,
	coverage bool,
) querySchemaAttemptFixture {
	result := map[string]any{
		"page":  querySchemaPage(1),
		"items": []any{item},
	}
	if coverage {
		result["coverageNotice"] = legalquery.JudicialCasesCoverageNotice
	}
	return querySchemaAttemptFixture{
		resultKind: resultKind,
		step: querySchemaStep(
			stepID,
			task,
			resource,
			capabilityID,
		),
		attempt: querySchemaAttemptHeader(
			stepID,
			capabilityID,
			"completed",
			resultKind,
			result,
		),
	}
}

func querySchemaReadAttempt(
	stepID string,
	task string,
	resource string,
	capabilityID string,
	resultKind string,
	item map[string]any,
	coverage bool,
) querySchemaAttemptFixture {
	result := map[string]any{"item": item}
	if coverage {
		result["coverageNotice"] = legalquery.JudicialCasesCoverageNotice
	}
	return querySchemaAttemptFixture{
		resultKind: resultKind,
		step:       querySchemaStep(stepID, task, resource, capabilityID),
		attempt: querySchemaAttemptHeader(
			stepID,
			capabilityID,
			"completed",
			resultKind,
			result,
		),
	}
}

func querySchemaAttemptHeader(
	stepID string,
	capabilityID string,
	outcome string,
	resultKind string,
	result map[string]any,
) map[string]any {
	return map[string]any{
		"interpretationId":       "interpretation-1",
		"stepId":                 stepID,
		"capabilityId":           capabilityID,
		"capabilityMajorVersion": 1,
		"outcome":                outcome,
		"resultKind":             resultKind,
		"result":                 result,
	}
}

func querySchemaEmptyLawSearchAttempt(stepID string) map[string]any {
	return querySchemaAttemptHeader(
		stepID,
		"law.search",
		"empty",
		"law_search",
		map[string]any{
			"page":  querySchemaPage(0),
			"items": []any{},
		},
	)
}

func querySchemaFailedAttempt(stepID string) map[string]any {
	errorResult, err := model.NewErrorResult(model.ErrorResultValues{
		Code: model.ErrorCodeSourceTimeout,
	})
	if err != nil {
		panic("試験用 ErrorResult を作成できません: " + err.Error())
	}
	return map[string]any{
		"interpretationId":       "interpretation-1",
		"stepId":                 stepID,
		"capabilityId":           "law.search",
		"capabilityMajorVersion": 1,
		"outcome":                "failed",
		"error": map[string]any{
			"code":      string(errorResult.Code()),
			"message":   errorResult.Message(),
			"retryable": errorResult.Retryable(),
		},
	}
}

func querySchemaExecutionResult(
	status string,
	decision string,
	interpretations []any,
	attempts []any,
	notices []any,
) map[string]any {
	return map[string]any{
		"status":          status,
		"decision":        decision,
		"language":        "ja",
		"interpretations": interpretations,
		"attempts":        attempts,
		"notices":         notices,
	}
}

func querySchemaInterpretation(
	availability string,
	requiredPacks []any,
	steps []any,
) map[string]any {
	return map[string]any{
		"interpretationId": "interpretation-1",
		"confidence":       "high",
		"evidenceCodes":    []any{"explicit_task"},
		"conceptSources":   []any{},
		"availability":     availability,
		"requiredPacks":    requiredPacks,
		"steps":            steps,
	}
}

func querySchemaStep(
	stepID string,
	task string,
	resource string,
	capabilityID string,
) map[string]any {
	return map[string]any{
		"stepId":                 stepID,
		"task":                   task,
		"resource":               resource,
		"capabilityId":           capabilityID,
		"capabilityMajorVersion": 1,
	}
}

func querySchemaPage(returned int) map[string]any {
	return map[string]any{
		"returnedCount": returned,
		"hasMore":       false,
		"totalCount":    returned,
		"totalRelation": "exact",
	}
}
