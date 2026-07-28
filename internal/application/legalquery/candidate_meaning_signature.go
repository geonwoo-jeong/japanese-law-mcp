package legalquery

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func legalQueryCandidateMeaningSignature(
	candidate LegalQueryCandidate,
) (string, error) {
	parts := make([]string, 0, len(candidate.Steps())+1)
	packs := candidate.RequiredPacks()
	slices.Sort(packs)
	parts = append(parts, strings.Join(packs, "\x00"))
	for _, step := range candidate.Steps() {
		signature, err := legalQueryLogicalInputSignature(step.LogicalInput())
		if err != nil {
			return "", err
		}
		parts = append(parts, signature)
	}
	return strings.Join(parts, "\x1f"), nil
}

func legalQueryLogicalInputSignature(value LogicalInput) (string, error) {
	switch input := value.(type) {
	case LawSearchIntentV1:
		return "law-search|" + input.Query() + "|" +
			optionalDateMeaningSignature(input.AsOf()), nil
	case LawContentSearchIntentV1:
		return lawContentMeaningSignature(input), nil
	case LawReadIntentV1:
		return lawReadMeaningSignature(input), nil
	case LawArticleReadIntentV1:
		return lawArticleMeaningSignature(input), nil
	case LawUpdateListIntentV1:
		return "law-updates|" + input.Date().String(), nil
	case JudicialDecisionSearchIntentV1:
		return "judicial-search|" + input.Query(), nil
	case JudicialDecisionReadIntentV1:
		return "judicial-read|" + resourceRefMeaningSignature(input.Ref()), nil
	default:
		return "", fmt.Errorf(
			"意味署名を作成できない logical input %T です",
			value,
		)
	}
}

func lawContentMeaningSignature(input LawContentSearchIntentV1) string {
	return "law-content|" +
		strings.Join(input.AllTerms(), "\x00") + "|" +
		strings.Join(input.AnyTerms(), "\x00") + "|" +
		strings.Join(input.ExcludeTerms(), "\x00") + "|" +
		optionalDateMeaningSignature(input.AsOf())
}

func lawReadMeaningSignature(input LawReadIntentV1) string {
	if ref, exists := input.Ref(); exists {
		return "law-read|ref|" + resourceRefMeaningSignature(ref)
	}
	lawID, _ := input.LawID()
	revisionID, _ := input.RevisionID()
	return "law-read|id|" + lawID + "|" + revisionID + "|" +
		optionalDateMeaningSignature(input.AsOf())
}

func lawArticleMeaningSignature(input LawArticleReadIntentV1) string {
	target := ""
	if ref, exists := input.Ref(); exists {
		target = "ref|" + resourceRefMeaningSignature(ref)
	} else {
		lawID, _ := input.LawID()
		target = "id|" + lawID
	}
	location := input.Location()
	paragraph := ""
	if value, exists := location.ParagraphNumber(); exists {
		paragraph = strconv.Itoa(value)
	}
	return "law-article|" + target + "|" +
		string(location.Provision()) + "|" +
		location.ArticleNumber() + "|" +
		paragraph + "|" +
		optionalDateMeaningSignature(input.AsOf())
}

func optionalDateMeaningSignature(date model.Date, exists bool) string {
	if !exists {
		return ""
	}
	return date.String()
}

func resourceRefMeaningSignature(ref model.SourceResourceRef) string {
	key := ref.Key()
	version, _ := key.VersionID()
	return ref.ProviderID() + "|" +
		key.SourceID() + "|" +
		key.ResourceType() + "|" +
		key.ResourceID() + "|" +
		version
}
