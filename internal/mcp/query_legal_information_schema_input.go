package mcp

import (
	"encoding/json"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/google/jsonschema-go/jsonschema"
)

func newQueryLegalInformationInputSchema() *jsonschema.Schema {
	query := queryLegalInformationRequiredStringSchema()
	query.MaxLength = jsonschema.Ptr(legalquery.MaxQueryBytes)
	query.Pattern = "^[^\\x00-\\x1f\\x7f]*" +
		"[^\\x00-\\x20\\x7f\u0085\u00a0\u1680\u2000-\u200a" +
		"\u2028\u2029\u202f\u205f\u3000]" +
		"[^\\x00-\\x1f\\x7f]*$"
	query.Description = "前後の Unicode White_Space を除き、有効な UTF-8 で 1 byte 以上 2048 byte 以下の日本語照会文。"
	query.Extra = map[string]any{
		"x-maxUtf8Bytes":          legalquery.MaxQueryBytes,
		"x-trimUnicodeWhitespace": true,
	}

	limit := queryLegalInformationIntegerSchema(
		1,
		jsonschema.Ptr(legalquery.MaxLimitPerAttempt),
	)
	limit.Default = json.RawMessage("10")

	schema := queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"query":           query,
			"ref":             queryLegalInformationSchemaRef("QueryInputResourceRef"),
			"limitPerAttempt": limit,
		},
		"query",
	)
	schema.Schema = queryLegalInformationSchemaVersion
	schema.Defs = map[string]*jsonschema.Schema{
		"QueryInputResourceRef": queryLegalInformationInputResourceRefSchema(),
		"QueryInputResourceKey": queryLegalInformationInputResourceKeySchema(),
	}
	return schema
}

func queryLegalInformationInputResourceRefSchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"providerId": queryLegalInformationProviderIDSchema(),
			"key":        queryLegalInformationSchemaRef("QueryInputResourceKey"),
		},
		"providerId",
		"key",
	)
}

func queryLegalInformationInputResourceKeySchema() *jsonschema.Schema {
	return queryLegalInformationObjectSchema(
		map[string]*jsonschema.Schema{
			"sourceId": queryLegalInformationRequiredStringSchema(),
			"resourceType": queryLegalInformationStringEnum(
				"law",
				"judicial-decision",
			),
			"resourceId": queryLegalInformationRequiredStringSchema(),
			"versionId":  queryLegalInformationRequiredStringSchema(),
		},
		"sourceId",
		"resourceType",
		"resourceId",
	)
}
