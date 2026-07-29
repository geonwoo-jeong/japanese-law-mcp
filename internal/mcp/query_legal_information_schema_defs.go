package mcp

import (
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/google/jsonschema-go/jsonschema"
)

const queryLegalInformationIdentifierPattern = `^[a-z0-9]+(?:-[a-z0-9]+)*$`

func addQueryLegalInformationCoreDefinitions(
	definitions map[string]*jsonschema.Schema,
) {
	definitions["SourceResourceKey"] = queryLegalInformationResourceKeySchema()
	definitions["SourceResourceRef"] = queryLegalInformationResourceRefSchema()
	definitions["LegalSource"] = queryLegalInformationLegalSourceSchema()
	definitions["InformationSource"] = queryLegalInformationSourceSchema()
	definitions["Provenance"] = queryLegalInformationProvenanceSchema()
	definitions["Citation"] = queryLegalInformationCitationSchema()
	definitions["LawArticleLocation"] = queryLegalInformationLawArticleLocationSchema()
	definitions["LegalConceptSource"] = queryLegalInformationConceptSourceSchema()
	definitions["LegalQueryStepSummary"] = queryLegalInformationStepSummarySchema()
	definitions["LegalQueryAvailableInterpretation"] =
		queryLegalInformationInterpretationSchema(
			legalquery.SelectionAvailabilityAvailable,
			0,
		)
	definitions["LegalQueryPackDisabledInterpretation"] =
		queryLegalInformationInterpretationSchema(
			legalquery.SelectionAvailabilityPackDisabled,
			1,
		)
	definitions["LegalQueryClarification"] =
		queryLegalInformationClarificationSchema()
	definitions["LegalQueryPagePreview"] =
		queryLegalInformationPagePreviewSchema()
	definitions["ErrorResult"] = queryLegalInformationErrorResultSchema()
}

func queryLegalInformationIdentifierSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:      "string",
		MinLength: jsonschema.Ptr(1),
		MaxLength: jsonschema.Ptr(64),
		Pattern:   queryLegalInformationIdentifierPattern,
		Extra: map[string]any{
			"x-maxUtf8Bytes": 64,
		},
	}
}

func queryLegalInformationProviderIDSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:      "string",
		MinLength: jsonschema.Ptr(1),
		Pattern:   queryLegalInformationIdentifierPattern,
	}
}

func queryLegalInformationResourceKeySchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"sourceId":     queryLegalInformationRequiredStringSchema(),
			"resourceType": queryLegalInformationRequiredStringSchema(),
			"resourceId":   queryLegalInformationRequiredStringSchema(),
			"versionId":    queryLegalInformationRequiredStringSchema(),
		},
		"sourceId",
		"resourceType",
		"resourceId",
	)
}

func queryLegalInformationResourceRefSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"providerId": queryLegalInformationProviderIDSchema(),
			"key":        queryLegalInformationSchemaRef("SourceResourceKey"),
		},
		"providerId",
		"key",
	)
}

func queryLegalInformationLegalSourceSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"id":   queryLegalInformationRequiredStringSchema(),
			"name": queryLegalInformationRequiredStringSchema(),
			"authority": queryLegalInformationStringEnum(
				string(model.AuthorityOfficial),
				string(model.AuthoritySupplementary),
			),
			"serviceUrl": queryLegalInformationHTTPSURLSchema(),
		},
		"id",
		"name",
		"authority",
		"serviceUrl",
	)
}

func queryLegalInformationSourceSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"id":        queryLegalInformationRequiredStringSchema(),
			"name":      queryLegalInformationRequiredStringSchema(),
			"publisher": queryLegalInformationRequiredStringSchema(),
			"authority": queryLegalInformationStringEnum(
				string(model.AuthorityOfficial),
				string(model.AuthoritySupplementary),
			),
			"serviceUrl": queryLegalInformationHTTPSURLSchema(),
		},
		"id",
		"name",
		"publisher",
		"authority",
		"serviceUrl",
	)
}

func queryLegalInformationProvenanceSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			queryLegalInformationProvenanceVariant(
				model.ProvenanceTransformationUnchanged,
				false,
				false,
			),
			queryLegalInformationProvenanceVariant(
				model.ProvenanceTransformationExtracted,
				true,
				false,
			),
			queryLegalInformationProvenanceVariant(
				model.ProvenanceTransformationNormalized,
				true,
				false,
			),
			queryLegalInformationProvenanceVariant(
				model.ProvenanceTransformationDerived,
				true,
				true,
			),
		},
	}
}

func queryLegalInformationProvenanceVariant(
	transformation model.ProvenanceTransformation,
	requireMethod bool,
	requireInputKeys bool,
) *jsonschema.Schema {
	properties := map[string]*jsonschema.Schema{
		"source":      queryLegalInformationSchemaRef("InformationSource"),
		"resourceKey": queryLegalInformationSchemaRef("SourceResourceKey"),
		"url":         queryLegalInformationHTTPSURLSchema(),
		"retrievedAt": queryLegalInformationDateTimeSchema(),
		"sourceUpdatedAt": {
			OneOf: []*jsonschema.Schema{
				queryLegalInformationDateSchema(),
				queryLegalInformationDateTimeSchema(),
			},
		},
		"mediaType":      queryLegalInformationRequiredStringSchema(),
		"location":       queryLegalInformationRequiredStringSchema(),
		"transformation": queryLegalInformationConstString(string(transformation)),
		"methodId":       queryLegalInformationRequiredStringSchema(),
		"inputKeys": queryLegalInformationArraySchemaAtLeast(
			queryLegalInformationSchemaRef("SourceResourceKey"),
			0,
		),
		"contentDigest": {
			Type:    "string",
			Pattern: `^sha256:[0-9A-Fa-f]{64}$`,
		},
	}
	required := []string{
		"source",
		"resourceKey",
		"url",
		"retrievedAt",
		"mediaType",
		"transformation",
	}
	if requireMethod {
		required = append(required, "methodId")
	} else {
		delete(properties, "methodId")
	}
	if requireInputKeys {
		required = append(required, "inputKeys")
	} else {
		delete(properties, "inputKeys")
	}
	return queryLegalInformationObjectSchema(properties, required...)
}

func queryLegalInformationCitationSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"source":     queryLegalInformationSchemaRef("LegalSource"),
			"lawId":      queryLegalInformationRequiredStringSchema(),
			"revisionId": queryLegalInformationRequiredStringSchema(),
			"location":   queryLegalInformationRequiredStringSchema(),
			"url":        queryLegalInformationHTTPSURLSchema(),
		},
		"source",
		"lawId",
		"revisionId",
		"url",
	)
}

func queryLegalInformationLawArticleLocationSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"provision": queryLegalInformationStringEnum(
				string(model.LawArticleProvisionMain),
				string(model.LawArticleProvisionSupplementary),
			),
			"articleNumber": {
				Type:      "string",
				MinLength: jsonschema.Ptr(1),
				MaxLength: jsonschema.Ptr(64),
				Pattern:   `^[1-9][0-9]*(?:_[1-9][0-9]*)*$`,
				Extra: map[string]any{
					"x-maxUtf8Bytes": 64,
				},
			},
			"paragraphNumber": queryLegalInformationIntegerSchema(1, nil),
		},
		"provision",
		"articleNumber",
	)
}

func queryLegalInformationConceptSourceSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"conceptId":   queryLegalInformationRequiredStringSchema(),
			"title":       queryLegalInformationRequiredStringSchema(),
			"url":         queryLegalInformationHTTPSURLSchema(),
			"confirmedOn": queryLegalInformationDateSchema(),
		},
		"conceptId",
		"title",
		"url",
		"confirmedOn",
	)
}

func queryLegalInformationStepSummarySchema() *jsonschema.Schema {
	type stepVariant struct {
		task         legalquery.Task
		resource     legalquery.Resource
		capabilityID string
	}
	variants := []stepVariant{
		{legalquery.TaskSearch, legalquery.ResourceLaw, lawsearch.CapabilityID},
		{
			legalquery.TaskSearch,
			legalquery.ResourceLawProvision,
			lawcontentsearch.CapabilityID,
		},
		{legalquery.TaskRead, legalquery.ResourceLaw, lawdocumentread.CapabilityID},
		{
			legalquery.TaskRead,
			legalquery.ResourceLawProvision,
			lawarticleread.CapabilityID,
		},
		{
			legalquery.TaskListUpdates,
			legalquery.ResourceLaw,
			lawupdatelist.CapabilityID,
		},
		{
			legalquery.TaskSearch,
			legalquery.ResourceJudicialDecision,
			judicialdecisionsearch.CapabilityID,
		},
		{
			legalquery.TaskRead,
			legalquery.ResourceJudicialDecision,
			judicialdecisionread.CapabilityID,
		},
	}
	oneOf := make([]*jsonschema.Schema, 0, len(variants))
	for _, variant := range variants {
		oneOf = append(oneOf, queryLegalInformationObjectSchema(
			map[string]*jsonschema.Schema{
				"stepId": queryLegalInformationIdentifierSchema(),
				"task": queryLegalInformationConstString(
					string(variant.task),
				),
				"resource": queryLegalInformationConstString(
					string(variant.resource),
				),
				"capabilityId": queryLegalInformationConstString(
					variant.capabilityID,
				),
				"capabilityMajorVersion": {
					Type:  "integer",
					Const: queryLegalInformationAnyPointer(1),
				},
			},
			"stepId",
			"task",
			"resource",
			"capabilityId",
			"capabilityMajorVersion",
		))
	}
	return &jsonschema.Schema{OneOf: oneOf}
}

func queryLegalInformationInterpretationSchema(
	availability legalquery.SelectionAvailability,
	minimumPacks int,
) *jsonschema.Schema {
	evidenceCodes := queryLegalInformationOrderedSubsetArraySchema(
		[]string{
			string(legalquery.EvidenceOfficialIdentifier),
			string(legalquery.EvidenceStructuredReference),
			string(legalquery.EvidenceExplicitTask),
			string(legalquery.EvidenceExplicitResource),
			string(legalquery.EvidenceOfficialAlias),
			string(legalquery.EvidenceLegalConcept),
			string(legalquery.EvidenceMorphologicalContext),
			string(legalquery.EvidenceUniqueTypoCorrection),
			string(legalquery.EvidenceGeneralTerm),
		},
		1,
		9,
	)

	conceptSources := queryLegalInformationArraySchemaAtLeast(
		queryLegalInformationSchemaRef("LegalConceptSource"),
		0,
	)
	requiredPacks := queryLegalInformationArraySchemaAtLeast(
		queryLegalInformationIdentifierSchema(),
		minimumPacks,
	)
	requiredPacks.UniqueItems = true
	steps := queryLegalInformationArraySchema(
		queryLegalInformationSchemaRef("LegalQueryStepSummary"),
		1,
		legalquery.MaxCapabilityCalls,
	)

	schema := queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"interpretationId": queryLegalInformationStringEnum(
				"interpretation-1",
				"interpretation-2",
			),
			"confidence": queryLegalInformationStringEnum(
				string(legalquery.ConfidenceHigh),
				string(legalquery.ConfidenceMedium),
				string(legalquery.ConfidenceLow),
			),
			"evidenceCodes":  evidenceCodes,
			"conceptSources": conceptSources,
			"availability": queryLegalInformationConstString(
				string(availability),
			),
			"requiredPacks": requiredPacks,
			"steps":         steps,
		},
		"interpretationId",
		"confidence",
		"evidenceCodes",
		"conceptSources",
		"availability",
		"requiredPacks",
		"steps",
	)
	schema.AllOf = []*jsonschema.Schema{
		{
			If: &jsonschema.Schema{
				Properties: map[string]*jsonschema.Schema{
					"evidenceCodes": {
						Contains: queryLegalInformationConstString(
							string(legalquery.EvidenceLegalConcept),
						),
					},
				},
				Required: []string{"evidenceCodes"},
			},
			Then: &jsonschema.Schema{
				Properties: map[string]*jsonschema.Schema{
					"conceptSources": {
						MinItems: jsonschema.Ptr(1),
					},
				},
			},
			Else: &jsonschema.Schema{
				Properties: map[string]*jsonschema.Schema{
					"conceptSources": {
						MaxItems: jsonschema.Ptr(0),
					},
				},
			},
		},
	}
	return schema
}

func queryLegalInformationClarificationSchema() *jsonschema.Schema {
	ordinary := queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"reasonCodes": queryLegalInformationOrderedSubsetArraySchema(
				[]string{
					string(legalquery.ReasonCodeBelowExecutionThreshold),
					string(legalquery.ReasonCodeAmbiguousCandidates),
				},
				1,
				2,
			),
			"questions": queryLegalInformationOrderedSubsetArraySchema(
				[]string{
					string(legalquery.LegalQueryQuestionTask),
					string(legalquery.LegalQueryQuestionResource),
					string(legalquery.LegalQueryQuestionLaw),
					string(
						legalquery.LegalQueryQuestionJudicialDecision,
					),
				},
				1,
				2,
			),
		},
		"reasonCodes",
		"questions",
	)
	stepLimitExceeded := queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"reasonCodes": queryLegalInformationArrayEnum([]any{
				string(legalquery.ReasonCodeStepLimitExceeded),
			}),
			"questions": queryLegalInformationArrayEnum([]any{
				string(
					legalquery.LegalQueryQuestionStepLimitExceeded,
				),
			}),
		},
		"reasonCodes",
		"questions",
	)
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			ordinary,
			stepLimitExceeded,
		},
	}
}

func queryLegalInformationPagePreviewSchema() *jsonschema.Schema {
	schema := queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"returnedCount": queryLegalInformationIntegerSchema(
				0,
				jsonschema.Ptr(legalquery.MaxItemsPerCollectionStep),
			),
			"hasMore": {Type: "boolean"},
			"totalCount": queryLegalInformationIntegerSchema(
				0,
				nil,
			),
			"totalRelation": queryLegalInformationStringEnum(
				string(model.TotalRelationExact),
				string(model.TotalRelationLowerBound),
			),
		},
		"returnedCount",
	)
	schema.DependentRequired = map[string][]string{
		"totalCount":    {"totalRelation"},
		"totalRelation": {"totalCount"},
	}
	schema.AllOf = []*jsonschema.Schema{
		{
			OneOf: queryLegalInformationPageCountVariants(),
		},
	}
	return schema
}

func queryLegalInformationPageCountVariants() []*jsonschema.Schema {
	variants := make(
		[]*jsonschema.Schema,
		0,
		legalquery.MaxItemsPerCollectionStep+1,
	)
	for returnedCount := 0; returnedCount <= legalquery.MaxItemsPerCollectionStep; returnedCount++ {
		variant := &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"returnedCount": queryLegalInformationConstInteger(
					returnedCount,
				),
			},
			Required: []string{"returnedCount"},
			AllOf: []*jsonschema.Schema{
				{
					If: &jsonschema.Schema{
						Required: []string{"totalCount"},
					},
					Then: &jsonschema.Schema{
						Properties: map[string]*jsonschema.Schema{
							"totalCount": queryLegalInformationIntegerSchema(
								returnedCount,
								nil,
							),
						},
					},
				},
				queryLegalInformationExactPageConstraint(returnedCount),
				queryLegalInformationLowerBoundPageConstraint(returnedCount),
			},
		}
		variants = append(variants, variant)
	}
	return variants
}

func queryLegalInformationExactPageConstraint(
	returnedCount int,
) *jsonschema.Schema {
	return &jsonschema.Schema{
		If: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"totalRelation": queryLegalInformationConstString(
					string(model.TotalRelationExact),
				),
			},
			Required: []string{"totalRelation"},
		},
		Then: &jsonschema.Schema{
			OneOf: []*jsonschema.Schema{
				{
					Properties: map[string]*jsonschema.Schema{
						"totalCount": queryLegalInformationConstInteger(
							returnedCount,
						),
						"hasMore": queryLegalInformationConstBoolean(false),
					},
					Required: []string{"totalCount", "hasMore"},
				},
				{
					Properties: map[string]*jsonschema.Schema{
						"totalCount": queryLegalInformationIntegerSchema(
							returnedCount+1,
							nil,
						),
						"hasMore": queryLegalInformationConstBoolean(true),
					},
					Required: []string{"totalCount", "hasMore"},
				},
			},
		},
	}
}

func queryLegalInformationLowerBoundPageConstraint(
	returnedCount int,
) *jsonschema.Schema {
	return &jsonschema.Schema{
		If: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"totalRelation": queryLegalInformationConstString(
					string(model.TotalRelationLowerBound),
				),
				"totalCount": queryLegalInformationIntegerSchema(
					returnedCount+1,
					nil,
				),
			},
			Required: []string{"totalRelation", "totalCount"},
		},
		Then: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"hasMore": queryLegalInformationConstBoolean(true),
			},
			Required: []string{"hasMore"},
		},
	}
}

func queryLegalInformationErrorResultSchema() *jsonschema.Schema {
	type errorVariant struct {
		code       model.ErrorCode
		detailKeys []string
	}
	variants := []errorVariant{
		{code: model.ErrorCodeNotFound},
		{
			code:       model.ErrorCodeAmbiguousLocation,
			detailKeys: []string{"candidates"},
		},
		{
			code: model.ErrorCodeUnsupportedQuery,
			detailKeys: []string{
				"providerId",
				"sourceId",
				"capabilityId",
				"field",
				"constraint",
			},
		},
		{
			code: model.ErrorCodeSourceAuthFailed,
			detailKeys: []string{
				"providerId", "sourceId", "capabilityId", "operation",
			},
		},
		{
			code: model.ErrorCodeRateLimited,
			detailKeys: []string{
				"providerId", "sourceId", "capabilityId", "operation",
				"retryAfter",
			},
		},
		{
			code: model.ErrorCodeSourceTimeout,
			detailKeys: []string{
				"providerId", "sourceId", "capabilityId", "operation",
			},
		},
		{
			code: model.ErrorCodeSourceUnavailable,
			detailKeys: []string{
				"providerId", "sourceId", "capabilityId", "operation",
				"retryAfter",
			},
		},
		{
			code: model.ErrorCodeSourceBusy,
			detailKeys: []string{
				"providerId", "sourceId", "capabilityId", "operation",
			},
		},
		{
			code: model.ErrorCodeSourceContractChanged,
			detailKeys: []string{
				"providerId", "sourceId", "capabilityId", "operation",
			},
		},
		{
			code: model.ErrorCodeInvalidSourceResponse,
			detailKeys: []string{
				"providerId", "sourceId", "capabilityId", "operation",
			},
		},
		{
			code: model.ErrorCodeSourceResponseTooLarge,
			detailKeys: []string{
				"providerId", "sourceId", "capabilityId", "operation",
				"limitName",
			},
		},
		{
			code: model.ErrorCodeSourceProcessingLimit,
			detailKeys: []string{
				"providerId", "sourceId", "capabilityId", "operation",
				"limitName",
			},
		},
		{
			code: model.ErrorCodeUnsafeSourceContent,
			detailKeys: []string{
				"providerId", "sourceId", "capabilityId", "operation",
			},
		},
		{code: model.ErrorCodeInternalError},
	}
	oneOf := make([]*jsonschema.Schema, 0, len(variants))
	for _, variant := range variants {
		oneOf = append(
			oneOf,
			queryLegalInformationErrorResultVariantSchema(
				variant.code,
				variant.detailKeys,
			),
		)
	}
	return &jsonschema.Schema{OneOf: oneOf}
}

func queryLegalInformationErrorResultVariantSchema(
	code model.ErrorCode,
	detailKeys []string,
) *jsonschema.Schema {
	result, err := model.NewErrorResult(
		model.ErrorResultValues{Code: code},
	)
	if err != nil {
		panic("固定 ErrorResult の schema を作成できません: " + err.Error())
	}
	properties := map[string]*jsonschema.Schema{
		"code":      queryLegalInformationConstString(string(code)),
		"message":   queryLegalInformationConstString(result.Message()),
		"retryable": queryLegalInformationConstBoolean(result.Retryable()),
	}
	if len(detailKeys) > 0 {
		detailProperties := make(map[string]*jsonschema.Schema, len(detailKeys))
		for _, key := range detailKeys {
			detailProperties[key] =
				queryLegalInformationErrorDetailSchema(key)
		}
		properties["details"] = queryLegalInformationObjectSchema(
			detailProperties,
		)
	}
	return queryLegalInformationObjectSchema(
		properties,
		"code",
		"message",
		"retryable",
	)
}

func queryLegalInformationErrorDetailSchema(
	key string,
) *jsonschema.Schema {
	switch key {
	case "providerId":
		return queryLegalInformationProviderIDSchema()
	case "candidates":
		return queryLegalInformationArraySchemaAtLeast(
			queryLegalInformationRequiredStringSchema(),
			1,
		)
	default:
		return queryLegalInformationRequiredStringSchema()
	}
}

func queryLegalInformationAnyPointer(value any) *any {
	return &value
}
