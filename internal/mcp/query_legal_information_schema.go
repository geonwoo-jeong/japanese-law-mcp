package mcp

import "github.com/google/jsonschema-go/jsonschema"

const queryLegalInformationSchemaVersion = "https://json-schema.org/draft/2020-12/schema"

func newQueryLegalInformationOutputSchema() *jsonschema.Schema {
	definitions := queryLegalInformationDefinitions()
	return &jsonschema.Schema{
		Schema: queryLegalInformationSchemaVersion,
		OneOf: []*jsonschema.Schema{
			queryLegalInformationSchemaRef("LegalQueryCompletedResult"),
			queryLegalInformationSchemaRef("LegalQueryEmptyResult"),
			queryLegalInformationSchemaRef("LegalQueryPartialResult"),
			queryLegalInformationSchemaRef("LegalQueryNeedsClarificationResult"),
			queryLegalInformationSchemaRef("LegalQueryCapabilityUnavailableResult"),
			queryLegalInformationSchemaRef("LegalQueryUnsupportedResult"),
		},
		Defs: definitions,
	}
}

func queryLegalInformationDefinitions() map[string]*jsonschema.Schema {
	definitions := make(map[string]*jsonschema.Schema)
	addQueryLegalInformationCoreDefinitions(definitions)
	addQueryLegalInformationResourceDefinitions(definitions)
	addQueryLegalInformationAttemptDefinitions(definitions)
	addQueryLegalInformationResultDefinitions(definitions)
	return definitions
}

func queryLegalInformationSchemaRef(name string) *jsonschema.Schema {
	return &jsonschema.Schema{Ref: "#/$defs/" + name}
}

func queryLegalInformationFalseSchema() *jsonschema.Schema {
	return &jsonschema.Schema{Not: &jsonschema.Schema{}}
}

func queryLegalInformationObjectSchema(
	properties map[string]*jsonschema.Schema,
	required ...string,
) *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:                 "object",
		Properties:           properties,
		Required:             required,
		AdditionalProperties: queryLegalInformationFalseSchema(),
		PropertyOrder:        append([]string{}, required...),
	}
}

func queryLegalInformationArraySchema(
	items *jsonschema.Schema,
	minimum int,
	maximum int,
) *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:     "array",
		Items:    items,
		MinItems: jsonschema.Ptr(minimum),
		MaxItems: jsonschema.Ptr(maximum),
	}
}

func queryLegalInformationArraySchemaAtLeast(
	items *jsonschema.Schema,
	minimum int,
) *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:     "array",
		Items:    items,
		MinItems: jsonschema.Ptr(minimum),
	}
}

func queryLegalInformationOrderedSubsetArraySchema(
	values []string,
	minimum int,
	maximum int,
) *jsonschema.Schema {
	combinations := make([]any, 0)
	for mask := 1; mask < 1<<len(values); mask++ {
		combination := make([]any, 0, len(values))
		for index, value := range values {
			if mask&(1<<index) != 0 {
				combination = append(combination, value)
			}
		}
		if len(combination) >= minimum && len(combination) <= maximum {
			combinations = append(combinations, combination)
		}
	}
	itemSchema := queryLegalInformationStringEnum(values...)
	return &jsonschema.Schema{
		Type:        "array",
		Items:       itemSchema,
		MinItems:    jsonschema.Ptr(minimum),
		MaxItems:    jsonschema.Ptr(maximum),
		UniqueItems: true,
		Enum:        combinations,
	}
}

func queryLegalInformationRequiredStringSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:      "string",
		MinLength: jsonschema.Ptr(1),
	}
}

func queryLegalInformationConstString(value string) *jsonschema.Schema {
	constant := any(value)
	return &jsonschema.Schema{
		Type:  "string",
		Const: &constant,
	}
}

func queryLegalInformationConstBoolean(value bool) *jsonschema.Schema {
	constant := any(value)
	return &jsonschema.Schema{
		Type:  "boolean",
		Const: &constant,
	}
}

func queryLegalInformationStringEnum(values ...string) *jsonschema.Schema {
	enum := make([]any, 0, len(values))
	for _, value := range values {
		enum = append(enum, value)
	}
	return &jsonschema.Schema{
		Type: "string",
		Enum: enum,
	}
}

func queryLegalInformationIntegerSchema(
	minimum int,
	maximum *int,
) *jsonschema.Schema {
	schema := &jsonschema.Schema{
		Type:    "integer",
		Minimum: jsonschema.Ptr(float64(minimum)),
	}
	if maximum != nil {
		schema.Maximum = jsonschema.Ptr(float64(*maximum))
	}
	return schema
}

func queryLegalInformationConstInteger(value int) *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:  "integer",
		Const: queryLegalInformationAnyPointer(value),
	}
}

func queryLegalInformationDateSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:    "string",
		Format:  "date",
		Pattern: `^[0-9]{4}-[0-9]{2}-[0-9]{2}$`,
	}
}

func queryLegalInformationDateTimeSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:   "string",
		Format: "date-time",
	}
}

func queryLegalInformationHTTPSURLSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:    "string",
		Format:  "uri",
		Pattern: `^[hH][tT][tT][pP][sS]://[^/@[:space:]]+(?::[0-9]+)?(?:/.*)?$`,
	}
}

func queryLegalInformationCourtsURLSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:    "string",
		Format:  "uri",
		Pattern: `^[hH][tT][tT][pP][sS]://[wW][wW][wW]\.[cC][oO][uU][rR][tT][sS]\.[gG][oO]\.[jJ][pP]/`,
	}
}
