package core

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func draftMeaningSignature(value candidateDraft) (string, error) {
	parts := make([]string, 0, len(value.steps)+1)
	packs := append([]string(nil), value.requiredPacks...)
	slices.Sort(packs)
	parts = append(parts, strings.Join(slices.Compact(packs), ","))
	for _, step := range value.steps {
		signature, err := logicalInputSignature(step.input)
		if err != nil {
			return "", err
		}
		parts = append(parts, signature)
	}
	return strings.Join(parts, "\x1f"), nil
}

func draftMeaningRankingSignature(value candidateDraft) (string, error) {
	parts := make([]string, 0, len(value.steps)+1)
	packs := append([]string(nil), value.requiredPacks...)
	slices.Sort(packs)
	parts = append(parts, strings.Join(slices.Compact(packs), ","))
	for _, step := range value.steps {
		signature, err := logicalInputRankingSignature(step.input)
		if err != nil {
			return "", err
		}
		parts = append(parts, signature)
	}
	return strings.Join(parts, "\x1f"), nil
}

func logicalInputSignature(value legalquery.LogicalInput) (string, error) {
	switch input := value.(type) {
	case legalquery.LawSearchIntentV1:
		return lengthPrefixedSignature(
			"01-law-search",
			input.Query(),
			optionalDateSignature(input.AsOf()),
		), nil
	case legalquery.LawContentSearchIntentV1:
		return lengthPrefixedSignature(
			"02-law-content",
			stringListSignature(input.AllTerms()),
			stringListSignature(input.AnyTerms()),
			stringListSignature(input.ExcludeTerms()),
			optionalDateSignature(input.AsOf()),
		), nil
	case legalquery.LawReadIntentV1:
		if ref, exists := input.Ref(); exists {
			return lengthPrefixedSignature(
				"03-law-read",
				"ref",
				resourceRefSignature(ref),
			), nil
		}
		lawID, _ := input.LawID()
		revisionID, _ := input.RevisionID()
		return lengthPrefixedSignature(
			"03-law-read",
			"id",
			lawID,
			revisionID,
			optionalDateSignature(input.AsOf()),
		), nil
	case legalquery.LawArticleReadIntentV1:
		targetKind := "id"
		target := ""
		if ref, exists := input.Ref(); exists {
			targetKind = "ref"
			target = resourceRefSignature(ref)
		} else {
			target, _ = input.LawID()
		}
		location := input.Location()
		paragraph, paragraphExists := location.ParagraphNumber()
		return lengthPrefixedSignature(
			"04-law-article",
			targetKind,
			target,
			string(location.Provision()),
			location.ArticleNumber(),
			optionalIntegerSignature(paragraph, paragraphExists),
			optionalDateSignature(input.AsOf()),
		), nil
	case legalquery.LawUpdateListIntentV1:
		return lengthPrefixedSignature(
			"05-law-updates",
			input.Date().String(),
		), nil
	default:
		return "", fmt.Errorf("core profile が未対応の logical input を生成しました")
	}
}

func logicalInputRankingSignature(value legalquery.LogicalInput) (string, error) {
	switch input := value.(type) {
	case legalquery.LawSearchIntentV1:
		return "01-law-search|" + input.Query() + "|" +
			optionalDateRankingSignature(input.AsOf()), nil
	case legalquery.LawContentSearchIntentV1:
		return "02-law-content|" +
			strings.Join(input.AllTerms(), "\x00") + "|" +
			strings.Join(input.AnyTerms(), "\x00") + "|" +
			strings.Join(input.ExcludeTerms(), "\x00") + "|" +
			optionalDateRankingSignature(input.AsOf()), nil
	case legalquery.LawReadIntentV1:
		if ref, exists := input.Ref(); exists {
			return "03-law-read|ref|" + resourceRefRankingSignature(ref), nil
		}
		lawID, _ := input.LawID()
		revisionID, _ := input.RevisionID()
		return "03-law-read|" + lawID + "|" + revisionID + "|" +
			optionalDateRankingSignature(input.AsOf()), nil
	case legalquery.LawArticleReadIntentV1:
		target := ""
		if ref, exists := input.Ref(); exists {
			target = resourceRefRankingSignature(ref)
		} else {
			target, _ = input.LawID()
		}
		location := input.Location()
		paragraph, _ := location.ParagraphNumber()
		return "04-law-article|" + target + "|" +
			string(location.Provision()) + "|" +
			location.ArticleNumber() + "|" +
			strconv.Itoa(paragraph) + "|" +
			optionalDateRankingSignature(input.AsOf()), nil
	case legalquery.LawUpdateListIntentV1:
		return "05-law-updates|" + input.Date().String(), nil
	default:
		return "", fmt.Errorf("core profile が未対応の logical input を生成しました")
	}
}

func optionalDateRankingSignature(date model.Date, exists bool) string {
	if !exists {
		return ""
	}
	return date.String()
}

func optionalDateSignature(date model.Date, exists bool) string {
	if !exists {
		return lengthPrefixedSignature("0")
	}
	return lengthPrefixedSignature("1", date.String())
}

func optionalIntegerSignature(value int, exists bool) string {
	if !exists {
		return lengthPrefixedSignature("0")
	}
	return lengthPrefixedSignature("1", strconv.Itoa(value))
}

func stringListSignature(values []string) string {
	fields := make([]string, 0, len(values)+1)
	fields = append(fields, strconv.Itoa(len(values)))
	fields = append(fields, values...)
	return lengthPrefixedSignature(fields...)
}

func lengthPrefixedSignature(fields ...string) string {
	var signature strings.Builder
	for _, field := range fields {
		signature.WriteString(strconv.Itoa(len(field)))
		signature.WriteByte(':')
		signature.WriteString(field)
	}
	return signature.String()
}

func resourceRefSignature(ref model.SourceResourceRef) string {
	key := ref.Key()
	version, versionExists := key.VersionID()
	return lengthPrefixedSignature(
		ref.ProviderID(),
		key.SourceID(),
		key.ResourceType(),
		key.ResourceID(),
		lengthPrefixedSignature(
			strconv.FormatBool(versionExists),
			version,
		),
	)
}

func resourceRefRankingSignature(ref model.SourceResourceRef) string {
	key := ref.Key()
	version, _ := key.VersionID()
	return ref.ProviderID() + "|" +
		key.SourceID() + "|" +
		key.ResourceType() + "|" +
		key.ResourceID() + "|" +
		version
}
