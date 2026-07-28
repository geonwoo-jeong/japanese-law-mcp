package mcp

import (
	"strconv"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/google/jsonschema-go/jsonschema"
)

func addQueryLegalInformationResultDefinitions(
	definitions map[string]*jsonschema.Schema,
) {
	definitions["LegalQueryCompletedResult"] =
		queryLegalInformationExecutionResultSchema(
			legalquery.LegalQueryResultStatusCompleted,
			queryLegalInformationCompletedAttemptsSchema(),
			queryLegalInformationCompletedNoticesSchema(),
		)
	definitions["LegalQueryEmptyResult"] =
		queryLegalInformationExecutionResultSchema(
			legalquery.LegalQueryResultStatusEmpty,
			queryLegalInformationEmptyAttemptsSchema(),
			queryLegalInformationCompletedNoticesSchema(),
		)
	definitions["LegalQueryPartialResult"] =
		queryLegalInformationExecutionResultSchema(
			legalquery.LegalQueryResultStatusPartial,
			queryLegalInformationPartialAttemptsSchema(),
			queryLegalInformationPartialNoticesSchema(),
		)
	definitions["LegalQueryNeedsClarificationResult"] =
		queryLegalInformationNeedsClarificationResultSchema()
	definitions["LegalQueryCapabilityUnavailableResult"] =
		queryLegalInformationCapabilityUnavailableResultSchema()
	definitions["LegalQueryUnsupportedResult"] =
		queryLegalInformationUnsupportedResultSchema()
}

func queryLegalInformationExecutionResultSchema(
	status legalquery.LegalQueryResultStatus,
	attempts *jsonschema.Schema,
	notices *jsonschema.Schema,
) *jsonschema.Schema {
	schema := queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"status": queryLegalInformationConstString(string(status)),
			"decision": queryLegalInformationStringEnum(
				string(legalquery.LegalQueryResultDecisionSingle),
				string(legalquery.LegalQueryResultDecisionHedged),
			),
			"language": queryLegalInformationConstString("ja"),
			"interpretations": queryLegalInformationOrderedInterpretationsSchema(
				"LegalQueryAvailableInterpretation",
				1,
				legalquery.MaxSelectedCandidates,
			),
			"attempts": attempts,
			"notices":  notices,
		},
		"status",
		"decision",
		"language",
		"interpretations",
		"attempts",
		"notices",
	)
	schema.AllOf = append(
		queryLegalInformationDecisionCountConstraints(),
		queryLegalInformationNoticeConstraints()...,
	)
	return schema
}

func queryLegalInformationNeedsClarificationResultSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"status": queryLegalInformationConstString(
				string(
					legalquery.LegalQueryResultStatusNeedsClarification,
				),
			),
			"decision": queryLegalInformationConstString(
				string(legalquery.LegalQueryResultDecisionNoExecution),
			),
			"language": queryLegalInformationConstString("ja"),
			"interpretations": queryLegalInformationOrderedInterpretationsSchema(
				"LegalQueryAvailableInterpretation",
				0,
				legalquery.MaxSelectedCandidates,
			),
			"attempts": queryLegalInformationArraySchema(
				queryLegalInformationSchemaRef("LegalQueryAttempt"),
				0,
				0,
			),
			"notices": queryLegalInformationArrayEnum([]any{}),
			"clarification": queryLegalInformationSchemaRef(
				"LegalQueryClarification",
			),
		},
		"status",
		"decision",
		"language",
		"interpretations",
		"attempts",
		"notices",
		"clarification",
	)
}

func queryLegalInformationCapabilityUnavailableResultSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"status": queryLegalInformationConstString(
				string(
					legalquery.LegalQueryResultStatusCapabilityUnavailable,
				),
			),
			"decision": queryLegalInformationConstString(
				string(legalquery.LegalQueryResultDecisionNoExecution),
			),
			"language": queryLegalInformationConstString("ja"),
			"interpretations": queryLegalInformationOrderedInterpretationsSchema(
				"LegalQueryPackDisabledInterpretation",
				1,
				legalquery.MaxSelectedCandidates,
			),
			"attempts": queryLegalInformationArraySchema(
				queryLegalInformationSchemaRef("LegalQueryAttempt"),
				0,
				0,
			),
			"notices": queryLegalInformationArrayEnum([]any{
				legalquery.LegalQueryPackDisabledNotice,
			}),
		},
		"status",
		"decision",
		"language",
		"interpretations",
		"attempts",
		"notices",
	)
}

func queryLegalInformationUnsupportedResultSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"status": queryLegalInformationConstString(
				string(legalquery.LegalQueryResultStatusUnsupported),
			),
			"decision": queryLegalInformationConstString(
				string(legalquery.LegalQueryResultDecisionNoExecution),
			),
			"language": queryLegalInformationConstString("ja"),
			"interpretations": queryLegalInformationArraySchema(
				queryLegalInformationSchemaRef(
					"LegalQueryAvailableInterpretation",
				),
				0,
				0,
			),
			"attempts": queryLegalInformationArraySchema(
				queryLegalInformationSchemaRef("LegalQueryAttempt"),
				0,
				0,
			),
			"notices": queryLegalInformationUnsupportedNoticesSchema(),
		},
		"status",
		"decision",
		"language",
		"interpretations",
		"attempts",
		"notices",
	)
}

func queryLegalInformationDecisionCountConstraints() []*jsonschema.Schema {
	return []*jsonschema.Schema{
		{
			If: queryLegalInformationDecisionCondition(
				legalquery.LegalQueryResultDecisionSingle,
			),
			Then: queryLegalInformationSingleDecisionShapeSchema(),
		},
		{
			If: queryLegalInformationDecisionCondition(
				legalquery.LegalQueryResultDecisionHedged,
			),
			Then: queryLegalInformationInterpretationCountSchema(2),
		},
	}
}

func queryLegalInformationDecisionCondition(
	decision legalquery.LegalQueryResultDecision,
) *jsonschema.Schema {
	return &jsonschema.Schema{
		Properties: map[string]*jsonschema.Schema{
			"decision": queryLegalInformationConstString(string(decision)),
		},
		Required: []string{"decision"},
	}
}

func queryLegalInformationInterpretationCountSchema(
	count int,
) *jsonschema.Schema {
	return &jsonschema.Schema{
		Properties: map[string]*jsonschema.Schema{
			"interpretations": queryLegalInformationOrderedInterpretationsSchema(
				"LegalQueryAvailableInterpretation",
				count,
				count,
			),
		},
	}
}

func queryLegalInformationSingleDecisionShapeSchema() *jsonschema.Schema {
	schema := queryLegalInformationInterpretationCountSchema(1)
	schema.Properties["attempts"] = &jsonschema.Schema{
		Items: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"interpretationId": queryLegalInformationConstString(
					"interpretation-1",
				),
			},
			Required: []string{"interpretationId"},
		},
	}
	return schema
}

func queryLegalInformationOrderedInterpretationsSchema(
	definition string,
	minimum int,
	maximum int,
) *jsonschema.Schema {
	prefixItems := make([]*jsonschema.Schema, 0, maximum)
	for index := 1; index <= maximum; index++ {
		prefixItems = append(prefixItems, &jsonschema.Schema{
			AllOf: []*jsonschema.Schema{
				queryLegalInformationSchemaRef(definition),
				{
					Properties: map[string]*jsonschema.Schema{
						"interpretationId": queryLegalInformationConstString(
							"interpretation-" + strconv.Itoa(index),
						),
					},
					Required: []string{"interpretationId"},
				},
			},
		})
	}
	return &jsonschema.Schema{
		Type:        "array",
		PrefixItems: prefixItems,
		Items:       queryLegalInformationFalseSchema(),
		MinItems:    jsonschema.Ptr(minimum),
		MaxItems:    jsonschema.Ptr(maximum),
	}
}

func queryLegalInformationNoticeConstraints() []*jsonschema.Schema {
	return []*jsonschema.Schema{
		queryLegalInformationNoticeConstraint(
			&jsonschema.Schema{
				Properties: map[string]*jsonschema.Schema{
					"attempts": {
						MinItems: jsonschema.Ptr(2),
					},
				},
				Required: []string{"attempts"},
			},
			legalquery.LegalQuerySeparateAttemptsNotice,
		),
		queryLegalInformationNoticeConstraint(
			queryLegalInformationHasMoreCondition(),
			legalquery.LegalQueryFirstPageNotice,
		),
	}
}

func queryLegalInformationHasMoreCondition() *jsonschema.Schema {
	return &jsonschema.Schema{
		Properties: map[string]*jsonschema.Schema{
			"attempts": {
				Contains: &jsonschema.Schema{
					Properties: map[string]*jsonschema.Schema{
						"result": {
							Properties: map[string]*jsonschema.Schema{
								"page": {
									Properties: map[string]*jsonschema.Schema{
										"hasMore": queryLegalInformationConstBoolean(
											true,
										),
									},
									Required: []string{"hasMore"},
								},
							},
							Required: []string{"page"},
						},
					},
					Required: []string{"result"},
				},
				MinContains: jsonschema.Ptr(1),
			},
		},
		Required: []string{"attempts"},
	}
}

func queryLegalInformationNoticeConstraint(
	condition *jsonschema.Schema,
	notice string,
) *jsonschema.Schema {
	containsNotice := &jsonschema.Schema{
		Contains: queryLegalInformationConstString(notice),
	}
	return &jsonschema.Schema{
		If: condition,
		Then: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"notices": containsNotice,
			},
		},
		Else: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"notices": {
					Not: containsNotice.CloneSchemas(),
				},
			},
		},
	}
}

func queryLegalInformationCompletedAttemptsSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:     "array",
		Items:    queryLegalInformationSchemaRef("LegalQuerySuccessfulAttempt"),
		MinItems: jsonschema.Ptr(1),
		MaxItems: jsonschema.Ptr(legalquery.MaxCapabilityCalls),
		Contains: queryLegalInformationSchemaRef(
			"LegalQueryCompletedSuccessfulAttempt",
		),
		MinContains: jsonschema.Ptr(1),
	}
}

func queryLegalInformationEmptyAttemptsSchema() *jsonschema.Schema {
	return queryLegalInformationArraySchema(
		queryLegalInformationSchemaRef("LegalQueryEmptyCollectionAttempt"),
		1,
		legalquery.MaxCapabilityCalls,
	)
}

func queryLegalInformationPartialAttemptsSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "array",
		Items:       queryLegalInformationSchemaRef("LegalQueryAttempt"),
		MinItems:    jsonschema.Ptr(2),
		MaxItems:    jsonschema.Ptr(legalquery.MaxCapabilityCalls),
		Contains:    queryLegalInformationSchemaRef("LegalQueryFailedAttempt"),
		MinContains: jsonschema.Ptr(1),
		AllOf: []*jsonschema.Schema{
			{
				Contains: queryLegalInformationSchemaRef(
					"LegalQuerySuccessfulAttempt",
				),
				MinContains: jsonschema.Ptr(1),
			},
		},
	}
}

func queryLegalInformationCompletedNoticesSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "array",
		Enum: []any{
			[]any{},
			[]any{legalquery.LegalQueryFirstPageNotice},
			[]any{legalquery.LegalQuerySeparateAttemptsNotice},
			[]any{
				legalquery.LegalQueryFirstPageNotice,
				legalquery.LegalQuerySeparateAttemptsNotice,
			},
		},
	}
}

func queryLegalInformationPartialNoticesSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "array",
		Enum: []any{
			[]any{legalquery.LegalQueryPartialFailureNotice},
			[]any{
				legalquery.LegalQueryFirstPageNotice,
				legalquery.LegalQueryPartialFailureNotice,
			},
			[]any{
				legalquery.LegalQuerySeparateAttemptsNotice,
				legalquery.LegalQueryPartialFailureNotice,
			},
			[]any{
				legalquery.LegalQueryFirstPageNotice,
				legalquery.LegalQuerySeparateAttemptsNotice,
				legalquery.LegalQueryPartialFailureNotice,
			},
		},
	}
}

func queryLegalInformationUnsupportedNoticesSchema() *jsonschema.Schema {
	values := []string{
		legalquery.LegalQueryNonJapaneseNotice,
		legalquery.LegalQueryMixedUnsupportedNotice,
		legalquery.LegalQueryUnsupportedScopeNotice,
	}
	enums := make([]any, 0, 8)
	enums = append(enums, []any{
		legalquery.LegalQueryStandaloneStructuredNotice,
	})
	for mask := 1; mask < 1<<len(values); mask++ {
		noticeSet := make([]any, 0, len(values))
		for index, value := range values {
			if mask&(1<<index) != 0 {
				noticeSet = append(noticeSet, value)
			}
		}
		enums = append(enums, noticeSet)
	}
	return &jsonschema.Schema{
		Type: "array",
		Enum: enums,
	}
}

func queryLegalInformationArrayEnum(values []any) *jsonschema.Schema {
	constant := any(values)
	return &jsonschema.Schema{
		Type:  "array",
		Const: &constant,
	}
}
