package core

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

type lawAliasRankingFact struct {
	groupKey    string
	prefixCount int
}

// withLawAliasCollisionRanks は、SOT-ARCH-028 の衝突群内順位を付与する。
func withLawAliasCollisionRanks(values []lawTarget) []lawTarget {
	result := append([]lawTarget(nil), values...)
	groups := make(map[string][]int)
	for index, value := range result {
		if value.evidence != legalquery.EvidenceOfficialAlias ||
			value.surface == "" ||
			value.canonical == "" {
			continue
		}
		key := lawAliasCollisionGroupKey(value)
		groups[key] = append(groups[key], index)
	}
	for key, indexes := range groups {
		if len(indexes) < 2 {
			continue
		}
		prefixCounts := make(map[int]int, len(indexes))
		for _, index := range indexes {
			prefixCounts[index] = canonicalPrefixCount(
				result,
				indexes,
				index,
			)
		}
		for _, index := range indexes {
			result[index].aliasRank = lawAliasRankingFact{
				groupKey:    key,
				prefixCount: prefixCounts[index],
			}
		}
	}
	return result
}

func lawAliasCollisionGroupKey(value lawTarget) string {
	return strconv.Itoa(value.startByte) + ":" +
		strconv.Itoa(value.endByte) + ":" +
		string(querynormalization.ComparisonKey(value.surface))
}

func canonicalPrefixCount(
	values []lawTarget,
	groupIndexes []int,
	targetIndex int,
) int {
	target := string(querynormalization.ComparisonKey(
		values[targetIndex].canonical,
	))
	if target == "" {
		return 0
	}
	count := 0
	for _, otherIndex := range groupIndexes {
		if otherIndex == targetIndex {
			continue
		}
		other := string(querynormalization.ComparisonKey(
			values[otherIndex].canonical,
		))
		if target != other && strings.HasPrefix(other, target) {
			count++
		}
	}
	return count
}

func mergeLawAliasRankingFacts(
	left []lawAliasRankingFact,
	right []lawAliasRankingFact,
) []lawAliasRankingFact {
	result := append([]lawAliasRankingFact(nil), left...)
	for _, value := range right {
		if slices.Contains(result, value) {
			continue
		}
		result = append(result, value)
	}
	return result
}

func withLawAliasCollisionRankingSignatures(
	values []preparedDraft,
) []preparedDraft {
	result := append([]preparedDraft(nil), values...)
	type rankingBucket struct {
		groupKey          string
		score             int
		evidenceSignature string
		stepCount         int
	}
	groups := make(map[rankingBucket][]int)
	for index, value := range result {
		if len(value.draft.aliasRankings) != 1 {
			continue
		}
		fact := value.draft.aliasRankings[0]
		key := rankingBucket{
			groupKey:          fact.groupKey,
			score:             value.score,
			evidenceSignature: evidenceSignature(value.evidence),
			stepCount:         len(value.draft.steps),
		}
		groups[key] = append(groups[key], index)
	}
	for _, indexes := range groups {
		if len(indexes) < 2 {
			continue
		}
		signatureSlots := make([]string, 0, len(indexes))
		for _, index := range indexes {
			signatureSlots = append(
				signatureSlots,
				result[index].rankingSignature,
			)
		}
		slices.Sort(signatureSlots)

		ordered := append([]int(nil), indexes...)
		sort.SliceStable(ordered, func(left, right int) bool {
			leftFact := result[ordered[left]].draft.aliasRankings[0]
			rightFact := result[ordered[right]].draft.aliasRankings[0]
			if leftFact.prefixCount != rightFact.prefixCount {
				return leftFact.prefixCount > rightFact.prefixCount
			}
			if result[ordered[left]].rankingSignature !=
				result[ordered[right]].rankingSignature {
				return result[ordered[left]].rankingSignature <
					result[ordered[right]].rankingSignature
			}
			return result[ordered[left]].signature < result[ordered[right]].signature
		})
		for order, index := range ordered {
			result[index].rankingSignature = signatureSlots[order]
		}
	}
	return result
}

func compareAliasCollisionGroupPositions(
	left preparedDraft,
	right preparedDraft,
) int {
	if len(left.draft.aliasRankings) != 1 ||
		len(right.draft.aliasRankings) != 1 {
		return 0
	}
	leftGroup := left.draft.aliasRankings[0].groupKey
	rightGroup := right.draft.aliasRankings[0].groupKey
	if leftGroup == "" || rightGroup == "" || leftGroup == rightGroup {
		return 0
	}
	leftPosition := sourcePosition(left.draft)
	rightPosition := sourcePosition(right.draft)
	if leftPosition != rightPosition {
		return leftPosition - rightPosition
	}
	return 0
}

func hasMultipleLawAliasCollisionGroups(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) bool {
	collisionGroupCount := 0
	for _, group := range groupLawTargets(buildLawTargets(input, cues)) {
		if len(group) < 2 {
			continue
		}
		collisionGroupCount++
		if collisionGroupCount == 2 {
			return true
		}
	}
	return false
}
