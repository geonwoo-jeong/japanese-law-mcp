package judicialcases

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func judicialLogicalInputSignature(
	value legalquery.LogicalInput,
) (string, error) {
	switch input := value.(type) {
	case legalquery.JudicialDecisionSearchIntentV1:
		return judicialSignatureParts("judicial-search", input.Query()), nil
	case legalquery.JudicialDecisionReadIntentV1:
		return judicialSignatureParts(
			"judicial-read",
			judicialResourceRefSignature(input.Ref()),
		), nil
	default:
		return "", fmt.Errorf("judicial 意味署名を作成できない logical input %T です", value)
	}
}

func judicialDraftMeaningSignature(
	value candidateDraft,
) (string, error) {
	parts := make([]string, 0, len(value.steps))
	for _, step := range value.steps {
		signature, err := judicialLogicalInputSignature(step.input)
		if err != nil {
			return "", err
		}
		parts = append(parts, signature)
	}
	return judicialSignatureParts(parts...), nil
}

func judicialResourceRefSignature(ref model.SourceResourceRef) string {
	key := ref.Key()
	version, hasVersion := key.VersionID()
	versionPart := ""
	if hasVersion {
		versionPart = version
	}
	return judicialSignatureParts(
		ref.ProviderID(),
		key.SourceID(),
		key.ResourceType(),
		key.ResourceID(),
		strconv.FormatBool(hasVersion),
		versionPart,
	)
}

func judicialSignatureParts(parts ...string) string {
	built := make([]string, 0, len(parts))
	for _, part := range parts {
		built = append(built, strconv.Itoa(len(part))+":"+part)
	}
	return strings.Join(built, "|")
}
