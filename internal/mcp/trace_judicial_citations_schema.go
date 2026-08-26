package mcp

import (
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationtrace"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/google/jsonschema-go/jsonschema"
)

const (
	traceJudicialCitationMaximumDecisionEdges = 64
	traceJudicialCitationMaximumLawEdges      = 32
	traceJudicialCitationMaximumEvidenceItems = 256
)

func newTraceJudicialCitationsOutputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Schema: queryLegalInformationSchemaVersion,
		Type:   "object",
		Properties: map[string]*jsonschema.Schema{
			"status": queryLegalInformationStringEnum(
				string(model.JudicialCitationResultStatusComplete),
				string(model.JudicialCitationResultStatusPartial),
			),
			"coverageNotice": queryLegalInformationConstString(
				judicialcitationtrace.CoverageNotice,
			),
			"graph": traceJudicialCitationSchemaRef("JudicialCitationGraph"),
			"issues": queryLegalInformationArraySchemaAtLeast(
				traceJudicialCitationSchemaRef("JudicialCitationIssue"),
				0,
			),
		},
		Required:             []string{"status", "coverageNotice", "graph", "issues"},
		AdditionalProperties: queryLegalInformationFalseSchema(),
		Defs:                 traceJudicialCitationDefinitions(),
	}
}

func traceJudicialCitationDefinitions() map[string]*jsonschema.Schema {
	return map[string]*jsonschema.Schema{
		"SourceResourceKey":                 queryLegalInformationResourceKeySchema(),
		"SourceResourceRef":                 queryLegalInformationResourceRefSchema(),
		"InformationSource":                 queryLegalInformationSourceSchema(),
		"Provenance":                        traceJudicialCitationProvenanceSchema(),
		"JudicialDocumentLink":              queryLegalInformationJudicialDocumentLinkSchema(),
		"JudicialDecisionSummary":           queryLegalInformationJudicialSummarySchema(),
		"LawArticleLocation":                queryLegalInformationLawArticleLocationSchema(),
		"JudicialCitationLawReference":      traceJudicialCitationLawReferenceSchema(),
		"JudicialCitationNode":              traceJudicialCitationNodeSchema(),
		"JudicialCitationEvidence":          traceJudicialCitationEvidenceSchema(),
		"JudicialCitationEdge":              traceJudicialCitationEdgeSchema(),
		"JudicialCitationUnresolvedMention": traceJudicialCitationUnresolvedMentionSchema(),
		"JudicialCitationYearBucket":        traceJudicialCitationYearBucketSchema(),
		"JudicialCitationCategoryBucket":    traceJudicialCitationCategoryBucketSchema(),
		"JudicialCitationSummary":           traceJudicialCitationSummarySchema(),
		"JudicialCitationDirectionCoverage": traceJudicialCitationDirectionCoverageSchema(),
		"JudicialCitationCoverage":          traceJudicialCitationCoverageSchema(),
		"JudicialCitationIssue":             traceJudicialCitationIssueSchema(),
		"JudicialCitationGraph":             traceJudicialCitationGraphSchema(),
	}
}

func traceJudicialCitationProvenanceSchema() *jsonschema.Schema {
	return &jsonschema.Schema{OneOf: []*jsonschema.Schema{
		traceJudicialCitationProvenanceVariant(
			model.ProvenanceTransformationUnchanged,
			false,
			false,
		),
		traceJudicialCitationProvenanceVariant(
			model.ProvenanceTransformationExtracted,
			true,
			false,
		),
		traceJudicialCitationProvenanceVariant(
			model.ProvenanceTransformationNormalized,
			true,
			false,
		),
		traceJudicialCitationProvenanceVariant(
			model.ProvenanceTransformationDerived,
			true,
			true,
		),
	}}
}

func traceJudicialCitationProvenanceVariant(
	transformation model.ProvenanceTransformation,
	requireMethod bool,
	requireInputKeys bool,
) *jsonschema.Schema {
	properties := map[string]*jsonschema.Schema{
		"source":      traceJudicialCitationSchemaRef("InformationSource"),
		"resourceKey": traceJudicialCitationSchemaRef("SourceResourceKey"),
		"url":         queryLegalInformationHTTPSURLSchema(),
		"retrievedAt": queryLegalInformationDateTimeSchema(),
		"sourceUpdatedAt": {
			AnyOf: []*jsonschema.Schema{
				queryLegalInformationDateSchema(),
				queryLegalInformationDateTimeSchema(),
			},
		},
		"mediaType":      queryLegalInformationRequiredStringSchema(),
		"location":       queryLegalInformationRequiredStringSchema(),
		"transformation": queryLegalInformationConstString(string(transformation)),
		"methodId":       queryLegalInformationRequiredStringSchema(),
		"inputKeys": queryLegalInformationArraySchemaAtLeast(
			traceJudicialCitationSchemaRef("SourceResourceKey"),
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
	}
	if requireInputKeys {
		required = append(required, "inputKeys")
	}
	return queryLegalInformationObjectSchema(properties, required...)
}

func traceJudicialCitationSchemaRef(name string) *jsonschema.Schema {
	return &jsonschema.Schema{Ref: "#/$defs/" + name}
}

func traceJudicialCitationLawReferenceSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"lawId":    queryLegalInformationRequiredStringSchema(),
			"lawTitle": queryLegalInformationRequiredStringSchema(),
			"location": traceJudicialCitationSchemaRef("LawArticleLocation"),
		},
		"lawId",
		"lawTitle",
		"location",
	)
}

func traceJudicialCitationNodeSchema() *jsonschema.Schema {
	base := func(nodeType model.JudicialCitationNodeType) map[string]*jsonschema.Schema {
		return map[string]*jsonschema.Schema{
			"nodeId":   queryLegalInformationRequiredStringSchema(),
			"nodeType": queryLegalInformationConstString(string(nodeType)),
			"label":    queryLegalInformationRequiredStringSchema(),
		}
	}
	decision := base(model.JudicialCitationNodeTypeDecision)
	decision["ref"] = traceJudicialCitationSchemaRef("SourceResourceRef")
	decision["decisionSummary"] = traceJudicialCitationSchemaRef("JudicialDecisionSummary")
	law := base(model.JudicialCitationNodeTypeLawProvision)
	law["lawReference"] = traceJudicialCitationSchemaRef("JudicialCitationLawReference")
	reference := base(model.JudicialCitationNodeTypeDecisionReference)
	reference["referenceText"] = queryLegalInformationRequiredStringSchema()
	return &jsonschema.Schema{OneOf: []*jsonschema.Schema{
		queryLegalInformationObjectSchema(
			decision,
			"nodeId",
			"nodeType",
			"label",
			"ref",
			"decisionSummary",
		),
		queryLegalInformationObjectSchema(
			law,
			"nodeId",
			"nodeType",
			"label",
			"lawReference",
		),
		queryLegalInformationObjectSchema(
			reference,
			"nodeId",
			"nodeType",
			"label",
			"referenceText",
		),
	}}
}

func traceJudicialCitationEvidenceSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"evidenceLevel": queryLegalInformationStringEnum(
				string(model.JudicialCitationEvidenceLevelOfficialMetadata),
				string(model.JudicialCitationEvidenceLevelExactTextMatch),
				string(model.JudicialCitationEvidenceLevelOfficialSearchCandidate),
			),
			"provenance": traceJudicialCitationSchemaRef("Provenance"),
			"excerpt": {
				Type:      "string",
				MinLength: jsonschema.Ptr(1),
				MaxLength: jsonschema.Ptr(256),
				Extra: map[string]any{
					"x-maxUtf8Bytes": 256,
				},
			},
		},
		"evidenceLevel",
		"provenance",
	)
}

func traceJudicialCitationEdgeSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"edgeId":     queryLegalInformationRequiredStringSchema(),
			"fromNodeId": queryLegalInformationRequiredStringSchema(),
			"toNodeId":   queryLegalInformationRequiredStringSchema(),
			"relationType": queryLegalInformationStringEnum(
				string(model.JudicialCitationRelationTypeCitesDecision),
				string(model.JudicialCitationRelationTypePossibleCitesDecision),
				string(model.JudicialCitationRelationTypeReferencesLawProvision),
				string(model.JudicialCitationRelationTypeHasLowerCourtDecision),
			),
			"evidence": queryLegalInformationArraySchema(
				traceJudicialCitationSchemaRef("JudicialCitationEvidence"),
				1,
				traceJudicialCitationMaximumEvidenceItems,
			),
		},
		"edgeId",
		"fromNodeId",
		"toNodeId",
		"relationType",
		"evidence",
	)
}

func traceJudicialCitationUnresolvedMentionSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"mentionType": queryLegalInformationStringEnum(
				string(model.JudicialCitationMentionTypeDecision),
				string(model.JudicialCitationMentionTypeLawProvision),
			),
			"mentionText": queryLegalInformationRequiredStringSchema(),
			"reason": queryLegalInformationStringEnum(
				string(model.JudicialCitationUnresolvedReasonAmbiguousTarget),
				string(model.JudicialCitationUnresolvedReasonNoPublishedTargetMatch),
				string(model.JudicialCitationUnresolvedReasonInsufficientIdentity),
				string(model.JudicialCitationUnresolvedReasonUnsupportedReference),
				string(model.JudicialCitationUnresolvedReasonUnregisteredLawName),
				string(model.JudicialCitationUnresolvedReasonAmbiguousLawLocation),
				string(model.JudicialCitationUnresolvedReasonFuzzyMatchOnly),
			),
			"provenance": traceJudicialCitationSchemaRef("Provenance"),
		},
		"mentionType",
		"mentionText",
		"reason",
		"provenance",
	)
}

func traceJudicialCitationYearBucketSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"year":  queryLegalInformationIntegerSchema(1, nil),
			"count": queryLegalInformationIntegerSchema(1, nil),
		},
		"year",
		"count",
	)
}

func traceJudicialCitationCategoryBucketSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"publicationCategory": queryLegalInformationStringEnum(
				string(model.JudicialPublicationCategorySupremeCourt),
				string(model.JudicialPublicationCategoryHighCourt),
				string(model.JudicialPublicationCategoryLowerCourt),
				string(model.JudicialPublicationCategoryAdministrative),
				string(model.JudicialPublicationCategoryLabor),
				string(model.JudicialPublicationCategoryIntellectualProperty),
			),
			"count": queryLegalInformationIntegerSchema(1, nil),
		},
		"publicationCategory",
		"count",
	)
}

func traceJudicialCitationSummarySchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"confirmedOutgoingDecisionCount": queryLegalInformationIntegerSchema(
				0,
				jsonschema.Ptr(traceJudicialCitationMaximumDecisionEdges),
			),
			"incomingCandidateCount": queryLegalInformationIntegerSchema(
				0,
				jsonschema.Ptr(traceJudicialCitationMaximumDecisionEdges),
			),
			"referencedProvisionCount": queryLegalInformationIntegerSchema(
				0,
				jsonschema.Ptr(traceJudicialCitationMaximumLawEdges),
			),
			"lowerCourtRelationCount": queryLegalInformationIntegerSchema(
				0,
				jsonschema.Ptr(traceJudicialCitationMaximumDecisionEdges),
			),
			"unresolvedMentionCount": queryLegalInformationIntegerSchema(
				0,
				nil,
			),
			"incomingObservedYearBuckets": queryLegalInformationArraySchemaAtLeast(
				traceJudicialCitationSchemaRef("JudicialCitationYearBucket"),
				0,
			),
			"incomingObservedCategoryBuckets": queryLegalInformationArraySchema(
				traceJudicialCitationSchemaRef("JudicialCitationCategoryBucket"),
				0,
				6,
			),
		},
		"confirmedOutgoingDecisionCount",
		"incomingCandidateCount",
		"referencedProvisionCount",
		"lowerCourtRelationCount",
		"unresolvedMentionCount",
		"incomingObservedYearBuckets",
		"incomingObservedCategoryBuckets",
	)
}

func traceJudicialCitationDirectionCoverageSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"status": queryLegalInformationStringEnum(
				string(model.JudicialCitationDirectionStatusComplete),
				string(model.JudicialCitationDirectionStatusPartial),
				string(model.JudicialCitationDirectionStatusUnavailable),
				string(model.JudicialCitationDirectionStatusNotRequested),
			),
			"methods": {
				Type: "array",
				Items: queryLegalInformationStringEnum(
					string(model.JudicialCitationMethodOfficialDetailMetadata),
					string(model.JudicialCitationMethodOfficialPDFText),
					string(model.JudicialCitationMethodOfficialCaseSearch),
				),
				MinItems:    jsonschema.Ptr(0),
				MaxItems:    jsonschema.Ptr(3),
				UniqueItems: true,
			},
			"truncated":         {Type: "boolean"},
			"limit":             queryLegalInformationIntegerSchema(1, jsonschema.Ptr(10)),
			"attemptedSearches": queryLegalInformationIntegerSchema(0, jsonschema.Ptr(2)),
			"completedSearches": queryLegalInformationIntegerSchema(0, jsonschema.Ptr(2)),
		},
		"status",
		"methods",
		"truncated",
	)
}

func traceJudicialCitationCoverageSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"requestedDirection": queryLegalInformationStringEnum(
				string(model.JudicialCitationRequestedDirectionOutgoing),
				string(model.JudicialCitationRequestedDirectionIncoming),
				string(model.JudicialCitationRequestedDirectionBoth),
			),
			"hopDepth": queryLegalInformationConstInteger(1),
			"outgoing": traceJudicialCitationSchemaRef("JudicialCitationDirectionCoverage"),
			"incoming": traceJudicialCitationSchemaRef("JudicialCitationDirectionCoverage"),
		},
		"requestedDirection",
		"hopDepth",
		"outgoing",
		"incoming",
	)
}

func traceJudicialCitationIssueSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"direction": queryLegalInformationStringEnum(
				string(model.JudicialCitationIssueDirectionOutgoing),
				string(model.JudicialCitationIssueDirectionIncoming),
				string(model.JudicialCitationIssueDirectionShared),
			),
			"stage": queryLegalInformationStringEnum(
				string(model.JudicialCitationIssueStageOfficialDetailMetadata),
				string(model.JudicialCitationIssueStageOfficialPDFText),
				string(model.JudicialCitationIssueStageOfficialCaseSearch),
				string(model.JudicialCitationIssueStageLawReferenceResolution),
			),
			"code":      queryLegalInformationRequiredStringSchema(),
			"message":   queryLegalInformationRequiredStringSchema(),
			"retryable": {Type: "boolean"},
		},
		"direction",
		"stage",
		"code",
		"message",
		"retryable",
	)
}

func traceJudicialCitationGraphSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"rootNodeId": queryLegalInformationRequiredStringSchema(),
			"nodes": queryLegalInformationArraySchema(
				traceJudicialCitationSchemaRef("JudicialCitationNode"),
				1,
				1+traceJudicialCitationMaximumDecisionEdges+traceJudicialCitationMaximumLawEdges,
			),
			"edges": queryLegalInformationArraySchema(
				traceJudicialCitationSchemaRef("JudicialCitationEdge"),
				0,
				traceJudicialCitationMaximumDecisionEdges+traceJudicialCitationMaximumLawEdges,
			),
			"unresolvedMentions": {
				Type:     "array",
				Items:    traceJudicialCitationSchemaRef("JudicialCitationUnresolvedMention"),
				MinItems: jsonschema.Ptr(0),
			},
			"summary":  traceJudicialCitationSchemaRef("JudicialCitationSummary"),
			"coverage": traceJudicialCitationSchemaRef("JudicialCitationCoverage"),
		},
		"rootNodeId",
		"nodes",
		"edges",
		"unresolvedMentions",
		"summary",
		"coverage",
	)
}
