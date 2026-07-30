package legalquery

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
)

const profileSetVersionPrefix = "profile-set-sha256-"

func queryProfileSetVersion(
	metadata []QueryProfileMetadata,
	compositionVersion string,
) string {
	var canonical strings.Builder
	canonicalSchema := "schema-v2"
	if len(metadata) > 0 && metadata[0].SchemaVersion() == 2 {
		canonicalSchema = "schema-v3"
	}
	appendCanonicalPart(&canonical, canonicalSchema)
	appendCanonicalPart(&canonical, compositionVersion)
	for _, value := range metadata {
		if value.SchemaVersion() == 1 {
			appendLegacyMetadataCanonicalParts(&canonical, value)
			continue
		}
		appendMetadataCanonicalParts(&canonical, value)
	}
	sum := sha256.Sum256([]byte(canonical.String()))
	return profileSetVersionPrefix + hex.EncodeToString(sum[:])
}

func queryProfileMetadataSignature(
	metadata QueryProfileMetadata,
) string {
	var canonical strings.Builder
	appendMetadataCanonicalParts(&canonical, metadata)
	return canonical.String()
}

func rankingCalibrationSignature(
	metadata QueryProfileMetadata,
) string {
	var canonical strings.Builder
	appendCanonicalPart(
		&canonical,
		strconv.Itoa(metadata.SchemaVersion()),
	)
	appendCanonicalPart(&canonical, metadata.RankingVersion())
	appendScoreCanonicalParts(&canonical, metadata.Score())
	appendSelectionCanonicalParts(&canonical, metadata.Selection())
	for _, value := range metadata.TieBreak() {
		appendCanonicalPart(&canonical, string(value))
	}
	return canonical.String()
}

func appendLegacyMetadataCanonicalParts(
	target *strings.Builder,
	metadata QueryProfileMetadata,
) {
	appendMetadataIdentityCanonicalParts(target, metadata)
	appendScoreCanonicalParts(target, metadata.Score())
	appendLegacySelectionCanonicalParts(target, metadata.Selection())
	for _, value := range metadata.TieBreak() {
		appendCanonicalPart(target, string(value))
	}
}

func appendMetadataCanonicalParts(
	target *strings.Builder,
	metadata QueryProfileMetadata,
) {
	appendMetadataIdentityCanonicalParts(target, metadata)
	appendScoreCanonicalParts(target, metadata.Score())
	appendSelectionCanonicalParts(target, metadata.Selection())
	for _, value := range metadata.TieBreak() {
		appendCanonicalPart(target, string(value))
	}
	appendConditionalTieBreakCanonicalParts(target, metadata)
}

func appendMetadataIdentityCanonicalParts(
	target *strings.Builder,
	metadata QueryProfileMetadata,
) {
	appendCanonicalPart(target, strconv.Itoa(metadata.SchemaVersion()))
	appendCanonicalPart(target, metadata.ProfileID())
	appendCanonicalPart(target, metadata.ProfileVersion())
	appendCanonicalPart(target, metadata.RankingVersion())
	appendCanonicalPart(target, metadata.CueSetVersion())
	appendCanonicalPart(target, metadata.LawNameLexiconVersion())
	appendCanonicalPart(target, metadata.LegalConceptLexiconVersion())
	for _, value := range metadata.Targets() {
		appendCanonicalPart(target, string(value.Task()))
		appendCanonicalPart(target, string(value.Resource()))
		appendCanonicalPart(target, string(value.InputKind()))
	}
}

func appendConditionalTieBreakCanonicalParts(
	target *strings.Builder,
	metadata QueryProfileMetadata,
) {
	names := make([]string, 0, len(metadata.ConditionalTieBreaks()))
	for name := range metadata.ConditionalTieBreaks() {
		names = append(names, string(name))
	}
	sort.Strings(names)
	conditional := metadata.ConditionalTieBreaks()
	for _, name := range names {
		appendCanonicalPart(target, name)
		for _, value := range conditional[ConditionalTieBreakName(name)] {
			appendCanonicalPart(target, string(value))
		}
	}
}

func appendLegacySelectionCanonicalParts(
	target *strings.Builder,
	selection QuerySelectionPolicy,
) {
	appendCanonicalPart(target, strconv.Itoa(selection.SingleThreshold()))
	appendCanonicalPart(
		target,
		strconv.Itoa(selection.MinimumExecutionThreshold()),
	)
	appendCanonicalPart(target, strconv.Itoa(selection.SingleMargin()))
	appendCanonicalPart(target, strconv.Itoa(selection.HedgeMargin()))
}

func appendScoreCanonicalParts(
	target *strings.Builder,
	score QueryScorePolicy,
) {
	appendCanonicalPart(target, strconv.Itoa(score.Minimum()))
	appendCanonicalPart(target, strconv.Itoa(score.Maximum()))
	appendCanonicalPart(target, strconv.Itoa(score.HighConfidenceAt()))
	appendCanonicalPart(target, strconv.Itoa(score.MediumConfidenceAt()))
	for _, value := range score.EvidenceWeights() {
		appendCanonicalPart(target, string(value.Code()))
		appendCanonicalPart(target, strconv.Itoa(value.Weight()))
	}
}

func appendSelectionCanonicalParts(
	target *strings.Builder,
	selection QuerySelectionPolicy,
) {
	appendLegacySelectionCanonicalParts(target, selection)
	branchRetentionMargin, branchRetentionPresent := selection.BranchRetentionMargin()
	if branchRetentionPresent {
		appendCanonicalPart(target, "1")
		appendCanonicalPart(target, strconv.Itoa(branchRetentionMargin))
		return
	}
	appendCanonicalPart(target, "0")
	appendCanonicalPart(target, "")
}

func appendCanonicalPart(target *strings.Builder, value string) {
	target.WriteString(strconv.Itoa(len(value)))
	target.WriteByte(':')
	target.WriteString(value)
	target.WriteByte(';')
}
