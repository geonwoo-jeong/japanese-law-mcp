package mcp

import (
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/google/jsonschema-go/jsonschema"
)

func addQueryLegalInformationResourceDefinitions(
	definitions map[string]*jsonschema.Schema,
) {
	definitions["LawResourceRef"] =
		queryLegalInformationTypedResourceRefSchema("law", true)
	definitions["LawUpdateResourceRef"] =
		queryLegalInformationTypedResourceRefSchema("law-update-list", false)
	definitions["JudicialDecisionResourceRef"] =
		queryLegalInformationTypedResourceRefSchema(
			"judicial-decision",
			false,
		)
	definitions["LawSummary"] = queryLegalInformationLawSummarySchema()
	definitions["LawContentMatch"] = queryLegalInformationLawContentMatchSchema()
	definitions["LawDocumentRepresentation"] =
		queryLegalInformationLawDocumentSchema()
	definitions["LawArticleFragment"] =
		queryLegalInformationLawArticleSchema()
	definitions["LawUpdate"] = queryLegalInformationLawUpdateSchema()
	definitions["JudicialDocumentLink"] =
		queryLegalInformationJudicialDocumentLinkSchema()
	definitions["JudicialDecisionSummary"] =
		queryLegalInformationJudicialSummarySchema()
	definitions["JudicialDecisionDetails"] =
		queryLegalInformationJudicialDetailsSchema()

	type sourcedDefinition struct {
		data string
		ref  string
	}
	for name, definition := range map[string]sourcedDefinition{
		"SourcedLawSummary": {
			data: "LawSummary",
			ref:  "LawResourceRef",
		},
		"SourcedLawContentMatch": {
			data: "LawContentMatch",
			ref:  "LawResourceRef",
		},
		"SourcedLawDocument": {
			data: "LawDocumentRepresentation",
			ref:  "LawResourceRef",
		},
		"SourcedLawArticle": {
			data: "LawArticleFragment",
			ref:  "LawResourceRef",
		},
		"SourcedLawUpdate": {
			data: "LawUpdate",
			ref:  "LawUpdateResourceRef",
		},
		"SourcedJudicialDecisionSummary": {
			data: "JudicialDecisionSummary",
			ref:  "JudicialDecisionResourceRef",
		},
		"SourcedJudicialDecisionDetails": {
			data: "JudicialDecisionDetails",
			ref:  "JudicialDecisionResourceRef",
		},
	} {
		definitions[name] =
			queryLegalInformationSourcedResourceSchema(
				definition.data,
				definition.ref,
			)
	}
}

func queryLegalInformationSourcedResourceSchema(
	dataDefinition string,
	refDefinition string,
) *jsonschema.Schema {
	provenance := &jsonschema.Schema{
		Type:     "array",
		Items:    queryLegalInformationSchemaRef("Provenance"),
		MinItems: jsonschema.Ptr(1),
	}
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"ref":        queryLegalInformationSchemaRef(refDefinition),
			"provenance": provenance,
			"data":       queryLegalInformationSchemaRef(dataDefinition),
		},
		"ref",
		"provenance",
		"data",
	)
}

func queryLegalInformationTypedResourceRefSchema(
	resourceType string,
	versionRequired bool,
) *jsonschema.Schema {
	keyProperties := map[string]*jsonschema.Schema{
		"sourceId": queryLegalInformationRequiredStringSchema(),
		"resourceType": queryLegalInformationConstString(
			resourceType,
		),
		"resourceId": queryLegalInformationRequiredStringSchema(),
	}
	keyRequired := []string{"sourceId", "resourceType", "resourceId"}
	if versionRequired {
		keyProperties["versionId"] =
			queryLegalInformationRequiredStringSchema()
		keyRequired = append(keyRequired, "versionId")
	}
	key := queryLegalInformationObjectSchema(
		keyProperties,
		keyRequired...,
	)
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"providerId": queryLegalInformationProviderIDSchema(),
			"key":        key,
		},
		"providerId",
		"key",
	)
}

func queryLegalInformationLawSummarySchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"lawId":                 queryLegalInformationRequiredStringSchema(),
			"revisionId":            queryLegalInformationRequiredStringSchema(),
			"title":                 queryLegalInformationRequiredStringSchema(),
			"lawNumber":             queryLegalInformationRequiredStringSchema(),
			"promulgationDate":      queryLegalInformationDateSchema(),
			"revisionEffectiveDate": queryLegalInformationDateSchema(),
			"source":                queryLegalInformationSchemaRef("LegalSource"),
		},
		"lawId",
		"revisionId",
		"title",
		"source",
	)
}

func queryLegalInformationLawContentMatchSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"law":      queryLegalInformationSchemaRef("LawSummary"),
			"location": queryLegalInformationRequiredStringSchema(),
			"text":     queryLegalInformationRequiredStringSchema(),
			"citation": queryLegalInformationSchemaRef("Citation"),
		},
		"law",
		"location",
		"text",
		"citation",
	)
}

func queryLegalInformationLawDocumentSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"law":  queryLegalInformationSchemaRef("LawSummary"),
			"asOf": queryLegalInformationDateSchema(),
			"format": queryLegalInformationStringEnum(
				model.LawDocumentFormatXML,
				model.LawDocumentFormatHTML,
				model.LawDocumentFormatText,
			),
			"content":  queryLegalInformationRequiredStringSchema(),
			"citation": queryLegalInformationSchemaRef("Citation"),
		},
		"law",
		"format",
		"content",
		"citation",
	)
}

func queryLegalInformationLawArticleSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"law":      queryLegalInformationSchemaRef("LawSummary"),
			"location": queryLegalInformationSchemaRef("LawArticleLocation"),
			"format": queryLegalInformationStringEnum(
				model.LawArticleFormatXML,
				model.LawArticleFormatHTML,
				model.LawArticleFormatText,
			),
			"content":  queryLegalInformationRequiredStringSchema(),
			"citation": queryLegalInformationSchemaRef("Citation"),
		},
		"law",
		"location",
		"format",
		"content",
		"citation",
	)
}

func queryLegalInformationLawUpdateSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"updatedOn":                 queryLegalInformationDateSchema(),
			"lawId":                     queryLegalInformationRequiredStringSchema(),
			"title":                     queryLegalInformationRequiredStringSchema(),
			"lawType":                   queryLegalInformationRequiredStringSchema(),
			"lawNumber":                 queryLegalInformationRequiredStringSchema(),
			"titleKana":                 queryLegalInformationRequiredStringSchema(),
			"previousTitle":             queryLegalInformationRequiredStringSchema(),
			"promulgationDate":          queryLegalInformationDateSchema(),
			"amendmentTitle":            queryLegalInformationRequiredStringSchema(),
			"amendmentLawNumber":        queryLegalInformationRequiredStringSchema(),
			"amendmentPromulgationDate": queryLegalInformationDateSchema(),
			"effectiveDate":             queryLegalInformationDateSchema(),
			"effectiveDateNote":         queryLegalInformationRequiredStringSchema(),
			"documentUrl":               queryLegalInformationHTTPSURLSchema(),
			"enforcementPending":        {Type: "boolean"},
			"authorityReviewPending":    {Type: "boolean"},
			"source":                    queryLegalInformationSchemaRef("LegalSource"),
		},
		"updatedOn",
		"lawId",
		"title",
		"source",
	)
}

func queryLegalInformationJudicialDocumentLinkSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"kind": queryLegalInformationStringEnum(
				string(model.JudicialDocumentKindFullText),
				string(model.JudicialDocumentKindSummary),
				string(model.JudicialDocumentKindAttachment),
			),
			"label": queryLegalInformationRequiredStringSchema(),
			"mediaType": queryLegalInformationConstString(
				model.JudicialDocumentMediaTypePDF,
			),
			"url": queryLegalInformationCourtsURLSchema(),
		},
		"kind",
		"label",
		"mediaType",
		"url",
	)
}

func queryLegalInformationJudicialSummarySchema() *jsonschema.Schema {
	documents := &jsonschema.Schema{
		Type:  "array",
		Items: queryLegalInformationSchemaRef("JudicialDocumentLink"),
	}
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"decisionId": queryLegalInformationRequiredStringSchema(),
			"publicationCategory": queryLegalInformationStringEnum(
				string(model.JudicialPublicationCategorySupremeCourt),
				string(model.JudicialPublicationCategoryHighCourt),
				string(model.JudicialPublicationCategoryLowerCourt),
				string(model.JudicialPublicationCategoryAdministrative),
				string(model.JudicialPublicationCategoryLabor),
				string(model.JudicialPublicationCategoryIntellectualProperty),
			),
			"sourceCategoryLabel": queryLegalInformationRequiredStringSchema(),
			"caseNumber":          queryLegalInformationRequiredStringSchema(),
			"caseName":            queryLegalInformationRequiredStringSchema(),
			"decisionDate":        queryLegalInformationDateSchema(),
			"courtName":           queryLegalInformationRequiredStringSchema(),
			"branchName":          queryLegalInformationRequiredStringSchema(),
			"divisionName":        queryLegalInformationRequiredStringSchema(),
			"decisionType":        queryLegalInformationRequiredStringSchema(),
			"outcome":             queryLegalInformationRequiredStringSchema(),
			"detailUrl":           queryLegalInformationCourtsURLSchema(),
			"documents":           documents,
			"source":              queryLegalInformationSchemaRef("InformationSource"),
		},
		"decisionId",
		"publicationCategory",
		"sourceCategoryLabel",
		"caseNumber",
		"decisionDate",
		"courtName",
		"detailUrl",
		"documents",
		"source",
	)
}

func queryLegalInformationJudicialDetailsSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"summary":                  queryLegalInformationSchemaRef("JudicialDecisionSummary"),
			"reporterCitation":         queryLegalInformationRequiredStringSchema(),
			"lowerCourtName":           queryLegalInformationRequiredStringSchema(),
			"lowerCourtCaseNumber":     queryLegalInformationRequiredStringSchema(),
			"lowerCourtDecisionDate":   queryLegalInformationDateSchema(),
			"holdingText":              queryLegalInformationRequiredStringSchema(),
			"summaryText":              queryLegalInformationRequiredStringSchema(),
			"referencedProvisionsText": queryLegalInformationRequiredStringSchema(),
		},
		"summary",
	)
}
