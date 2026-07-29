package legalquery

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

const profileSetVersionPrefix = "profile-set-sha256-"

func queryProfileSetVersion(
	metadata []QueryProfileMetadata,
	compositionVersion string,
) string {
	var canonical strings.Builder
	appendCanonicalPart(&canonical, "schema-v2")
	appendCanonicalPart(&canonical, compositionVersion)
	for _, value := range metadata {
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
	appendCanonicalPart(&canonical, metadata.RankingVersion())
	appendScoreCanonicalParts(&canonical, metadata.Score())
	appendSelectionCanonicalParts(&canonical, metadata.Selection())
	for _, value := range metadata.TieBreak() {
		appendCanonicalPart(&canonical, string(value))
	}
	return canonical.String()
}

func appendMetadataCanonicalParts(
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
	appendScoreCanonicalParts(target, metadata.Score())
	appendSelectionCanonicalParts(target, metadata.Selection())
	for _, value := range metadata.TieBreak() {
		appendCanonicalPart(target, string(value))
	}
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
	appendCanonicalPart(target, strconv.Itoa(selection.SingleThreshold()))
	appendCanonicalPart(
		target,
		strconv.Itoa(selection.MinimumExecutionThreshold()),
	)
	appendCanonicalPart(target, strconv.Itoa(selection.SingleMargin()))
	appendCanonicalPart(target, strconv.Itoa(selection.HedgeMargin()))
}

func appendCanonicalPart(target *strings.Builder, value string) {
	target.WriteString(strconv.Itoa(len(value)))
	target.WriteByte(':')
	target.WriteString(value)
	target.WriteByte(';')
}
