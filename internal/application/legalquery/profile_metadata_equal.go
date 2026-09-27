package legalquery

import "slices"

// queryProfileMetadataEqual は、SOT-MODEL-026/SOT-ENG-035 の照合を既存署名と同値に保つ。
// 全 field が同じ場合だけ直接受理し、異なる場合は従来の canonical 比較へ戻す。
func queryProfileMetadataEqual(left, right QueryProfileMetadata) bool {
	if queryProfileMetadataFieldsEqual(left, right) {
		return true
	}
	return queryProfileMetadataSignature(left) == queryProfileMetadataSignature(right)
}

func queryProfileMetadataFieldsEqual(left, right QueryProfileMetadata) bool {
	if left.schemaVersion != right.schemaVersion ||
		left.profileID != right.profileID ||
		left.profileVersion != right.profileVersion ||
		left.rankingVersion != right.rankingVersion ||
		left.cueSetVersion != right.cueSetVersion ||
		left.lawNameLexiconVersion != right.lawNameLexiconVersion ||
		left.legalConceptLexiconVersion != right.legalConceptLexiconVersion ||
		left.selection != right.selection {
		return false
	}
	if !queryProfileMetadataSliceEqual(left.targets, right.targets) ||
		!queryProfileScoreFieldsEqual(left.score, right.score) ||
		!queryProfileMetadataSliceEqual(left.tieBreak, right.tieBreak) {
		return false
	}
	if (left.conditionalTieBreaks == nil) != (right.conditionalTieBreaks == nil) ||
		len(left.conditionalTieBreaks) != len(right.conditionalTieBreaks) {
		return false
	}
	for name, order := range left.conditionalTieBreaks {
		other, exists := right.conditionalTieBreaks[name]
		if !exists || !queryProfileMetadataSliceEqual(order, other) {
			return false
		}
	}
	return true
}

func queryProfileScoreFieldsEqual(left, right QueryScorePolicy) bool {
	return left.minimum == right.minimum &&
		left.maximum == right.maximum &&
		left.highConfidenceAt == right.highConfidenceAt &&
		left.mediumConfidenceAt == right.mediumConfidenceAt &&
		queryProfileMetadataSliceEqual(left.evidenceWeights, right.evidenceWeights)
}

func queryProfileMetadataSliceEqual[T comparable](left, right []T) bool {
	return (left == nil) == (right == nil) && slices.Equal(left, right)
}
