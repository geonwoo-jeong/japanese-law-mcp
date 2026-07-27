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
	"github.com/google/jsonschema-go/jsonschema"
)

type queryLegalInformationCollectionAttemptDefinition struct {
	name           string
	resultKind     legalquery.LegalQueryAttemptResultKind
	capabilityID   string
	itemDefinition string
	coverageNotice bool
}

func addQueryLegalInformationAttemptDefinitions(
	definitions map[string]*jsonschema.Schema,
) {
	collections := []queryLegalInformationCollectionAttemptDefinition{
		{
			name:           "LegalQueryLawSearchAttempt",
			resultKind:     legalquery.LegalQueryResultKindLawSearch,
			capabilityID:   lawsearch.CapabilityID,
			itemDefinition: "SourcedLawSummary",
		},
		{
			name:           "LegalQueryLawContentSearchAttempt",
			resultKind:     legalquery.LegalQueryResultKindLawContentSearch,
			capabilityID:   lawcontentsearch.CapabilityID,
			itemDefinition: "SourcedLawContentMatch",
		},
		{
			name:           "LegalQueryLawUpdatesAttempt",
			resultKind:     legalquery.LegalQueryResultKindLawUpdates,
			capabilityID:   lawupdatelist.CapabilityID,
			itemDefinition: "SourcedLawUpdate",
		},
		{
			name:           "LegalQueryJudicialSearchAttempt",
			resultKind:     legalquery.LegalQueryResultKindJudicialSearch,
			capabilityID:   judicialdecisionsearch.CapabilityID,
			itemDefinition: "SourcedJudicialDecisionSummary",
			coverageNotice: true,
		},
	}
	for _, definition := range collections {
		completedName := definition.name + "Completed"
		emptyName := definition.name + "Empty"
		definitions[completedName] =
			queryLegalInformationCollectionAttemptSchema(
				definition,
				legalquery.LegalQueryAttemptOutcomeCompleted,
			)
		definitions[emptyName] =
			queryLegalInformationCollectionAttemptSchema(
				definition,
				legalquery.LegalQueryAttemptOutcomeEmpty,
			)
		definitions[definition.name] = &jsonschema.Schema{
			OneOf: []*jsonschema.Schema{
				queryLegalInformationSchemaRef(completedName),
				queryLegalInformationSchemaRef(emptyName),
			},
		}
	}

	addQueryLegalInformationReadAttemptDefinitions(definitions)
	definitions["LegalQueryFailedAttempt"] =
		queryLegalInformationFailedAttemptSchema()

	definitions["LegalQuerySuccessfulAttempt"] = &jsonschema.Schema{
		OneOf: queryLegalInformationAttemptRefs(
			"LegalQueryLawSearchAttempt",
			"LegalQueryLawContentSearchAttempt",
			"LegalQueryLawDocumentAttempt",
			"LegalQueryLawArticleAttempt",
			"LegalQueryLawUpdatesAttempt",
			"LegalQueryJudicialSearchAttempt",
			"LegalQueryJudicialDecisionAttempt",
		),
	}
	definitions["LegalQueryCompletedSuccessfulAttempt"] = &jsonschema.Schema{
		OneOf: queryLegalInformationAttemptRefs(
			"LegalQueryLawSearchAttemptCompleted",
			"LegalQueryLawContentSearchAttemptCompleted",
			"LegalQueryLawDocumentAttempt",
			"LegalQueryLawArticleAttempt",
			"LegalQueryLawUpdatesAttemptCompleted",
			"LegalQueryJudicialSearchAttemptCompleted",
			"LegalQueryJudicialDecisionAttempt",
		),
	}
	definitions["LegalQueryEmptyCollectionAttempt"] = &jsonschema.Schema{
		OneOf: queryLegalInformationAttemptRefs(
			"LegalQueryLawSearchAttemptEmpty",
			"LegalQueryLawContentSearchAttemptEmpty",
			"LegalQueryLawUpdatesAttemptEmpty",
			"LegalQueryJudicialSearchAttemptEmpty",
		),
	}
	definitions["LegalQueryAttempt"] = &jsonschema.Schema{
		OneOf: queryLegalInformationAttemptRefs(
			"LegalQueryLawSearchAttempt",
			"LegalQueryLawContentSearchAttempt",
			"LegalQueryLawDocumentAttempt",
			"LegalQueryLawArticleAttempt",
			"LegalQueryLawUpdatesAttempt",
			"LegalQueryJudicialSearchAttempt",
			"LegalQueryJudicialDecisionAttempt",
			"LegalQueryFailedAttempt",
		),
	}
}

func addQueryLegalInformationReadAttemptDefinitions(
	definitions map[string]*jsonschema.Schema,
) {
	definitions["LegalQueryLawDocumentAttempt"] =
		queryLegalInformationReadAttemptSchema(
			legalquery.LegalQueryResultKindLawDocument,
			lawdocumentread.CapabilityID,
			"SourcedLawDocument",
			false,
		)
	definitions["LegalQueryLawArticleAttempt"] =
		queryLegalInformationReadAttemptSchema(
			legalquery.LegalQueryResultKindLawArticle,
			lawarticleread.CapabilityID,
			"SourcedLawArticle",
			false,
		)
	definitions["LegalQueryJudicialDecisionAttempt"] =
		queryLegalInformationReadAttemptSchema(
			legalquery.LegalQueryResultKindJudicialDecision,
			judicialdecisionread.CapabilityID,
			"SourcedJudicialDecisionDetails",
			true,
		)
}

func queryLegalInformationCollectionAttemptSchema(
	definition queryLegalInformationCollectionAttemptDefinition,
	outcome legalquery.LegalQueryAttemptOutcome,
) *jsonschema.Schema {
	minimumItems := 1
	maximumItems := legalquery.MaxItemsPerCollectionStep
	pageSchema := queryLegalInformationCollectionPageSchema(1, maximumItems)
	if outcome == legalquery.LegalQueryAttemptOutcomeEmpty {
		minimumItems = 0
		maximumItems = 0
		pageSchema = queryLegalInformationCollectionPageSchema(0, 0)
	}
	resultProperties := map[string]*jsonschema.Schema{
		"page": pageSchema,
		"items": queryLegalInformationArraySchema(
			queryLegalInformationSchemaRef(definition.itemDefinition),
			minimumItems,
			maximumItems,
		),
	}
	required := []string{"page", "items"}
	if definition.coverageNotice {
		resultProperties["coverageNotice"] =
			queryLegalInformationConstString(
				legalquery.JudicialCasesCoverageNotice,
			)
		required = append([]string{"coverageNotice"}, required...)
	}
	result := queryLegalInformationObjectSchema(resultProperties, required...)
	result.AllOf = []*jsonschema.Schema{
		{
			OneOf: queryLegalInformationCollectionCountVariants(
				minimumItems,
				maximumItems,
			),
		},
	}

	properties := queryLegalInformationSuccessfulAttemptProperties(
		definition.capabilityID,
		outcome,
		definition.resultKind,
		result,
	)
	return queryLegalInformationObjectSchema(
		properties,
		"interpretationId",
		"stepId",
		"capabilityId",
		"capabilityMajorVersion",
		"outcome",
		"resultKind",
		"result",
	)
}

func queryLegalInformationCollectionCountVariants(
	minimumItems int,
	maximumItems int,
) []*jsonschema.Schema {
	variants := make(
		[]*jsonschema.Schema,
		0,
		maximumItems-minimumItems+1,
	)
	for count := minimumItems; count <= maximumItems; count++ {
		variants = append(variants, &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"page": {
					Properties: map[string]*jsonschema.Schema{
						"returnedCount": queryLegalInformationConstInteger(
							count,
						),
					},
					Required: []string{"returnedCount"},
				},
				"items": {
					MinItems: jsonschema.Ptr(count),
					MaxItems: jsonschema.Ptr(count),
				},
			},
			Required: []string{"page", "items"},
		})
	}
	return variants
}

func queryLegalInformationCollectionPageSchema(
	minimumReturned int,
	maximumReturned int,
) *jsonschema.Schema {
	schema := queryLegalInformationSchemaRef("LegalQueryPagePreview")
	schema.AllOf = []*jsonschema.Schema{
		{
			Properties: map[string]*jsonschema.Schema{
				"returnedCount": queryLegalInformationIntegerSchema(
					minimumReturned,
					jsonschema.Ptr(maximumReturned),
				),
			},
		},
	}
	return schema
}

func queryLegalInformationReadAttemptSchema(
	resultKind legalquery.LegalQueryAttemptResultKind,
	capabilityID string,
	itemDefinition string,
	coverageNotice bool,
) *jsonschema.Schema {
	resultProperties := map[string]*jsonschema.Schema{
		"item": queryLegalInformationSchemaRef(itemDefinition),
	}
	required := []string{"item"}
	if coverageNotice {
		resultProperties["coverageNotice"] =
			queryLegalInformationConstString(
				legalquery.JudicialCasesCoverageNotice,
			)
		required = append([]string{"coverageNotice"}, required...)
	}
	result := queryLegalInformationObjectSchema(resultProperties, required...)
	properties := queryLegalInformationSuccessfulAttemptProperties(
		capabilityID,
		legalquery.LegalQueryAttemptOutcomeCompleted,
		resultKind,
		result,
	)
	return queryLegalInformationObjectSchema(
		properties,
		"interpretationId",
		"stepId",
		"capabilityId",
		"capabilityMajorVersion",
		"outcome",
		"resultKind",
		"result",
	)
}

func queryLegalInformationSuccessfulAttemptProperties(
	capabilityID string,
	outcome legalquery.LegalQueryAttemptOutcome,
	resultKind legalquery.LegalQueryAttemptResultKind,
	result *jsonschema.Schema,
) map[string]*jsonschema.Schema {
	return map[string]*jsonschema.Schema{
		"interpretationId": queryLegalInformationStringEnum(
			"interpretation-1",
			"interpretation-2",
		),
		"stepId":       queryLegalInformationIdentifierSchema(),
		"capabilityId": queryLegalInformationConstString(capabilityID),
		"capabilityMajorVersion": {
			Type:  "integer",
			Const: queryLegalInformationAnyPointer(1),
		},
		"outcome":    queryLegalInformationConstString(string(outcome)),
		"resultKind": queryLegalInformationConstString(string(resultKind)),
		"result":     result,
	}
}

func queryLegalInformationFailedAttemptSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"interpretationId": queryLegalInformationStringEnum(
				"interpretation-1",
				"interpretation-2",
			),
			"stepId": queryLegalInformationIdentifierSchema(),
			"capabilityId": queryLegalInformationStringEnum(
				lawsearch.CapabilityID,
				lawcontentsearch.CapabilityID,
				lawdocumentread.CapabilityID,
				lawarticleread.CapabilityID,
				lawupdatelist.CapabilityID,
				judicialdecisionsearch.CapabilityID,
				judicialdecisionread.CapabilityID,
			),
			"capabilityMajorVersion": {
				Type:  "integer",
				Const: queryLegalInformationAnyPointer(1),
			},
			"outcome": queryLegalInformationConstString(
				string(legalquery.LegalQueryAttemptOutcomeFailed),
			),
			"error": queryLegalInformationSchemaRef("ErrorResult"),
		},
		"interpretationId",
		"stepId",
		"capabilityId",
		"capabilityMajorVersion",
		"outcome",
		"error",
	)
}

func queryLegalInformationAttemptRefs(
	names ...string,
) []*jsonschema.Schema {
	references := make([]*jsonschema.Schema, 0, len(names))
	for _, name := range names {
		references = append(
			references,
			queryLegalInformationSchemaRef(name),
		)
	}
	return references
}
