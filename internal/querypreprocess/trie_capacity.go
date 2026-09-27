package querypreprocess

import (
	"math"
	"slices"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

type runeTrieCapacity struct {
	nodes  int
	values int
	labels int
}

func newRuneTrieWithCapacity[T any](capacity runeTrieCapacity) *runeTrie[T] {
	return &runeTrie[T]{
		nodes:  make([]trieNode, 1, max(1, capacity.nodes)),
		spans:  make([]uint64, 1, max(1, capacity.nodes)),
		labels: make([]rune, 0, max(0, capacity.labels)),
		roots:  make(map[rune]uint32),
		values: make([][]T, 1, max(1, capacity.values)),
	}
}

// SOT-ARCH-021: 登録順序を変えず、終端と分岐の節だけを radix の容量として数える。
// UTF-8 の整列で隣接語の共通接頭辞を求め、既存の祖先は深さの stack で再利用する。
func measureRuneTrieCapacity(patterns []string) (runeTrieCapacity, bool) {
	ordered := slices.Clone(patterns)
	for _, pattern := range ordered {
		if pattern == "" || !utf8.ValidString(pattern) {
			return runeTrieCapacity{}, false
		}
	}
	slices.Sort(ordered)
	capacity := runeTrieCapacity{nodes: 1, values: 1}
	depths := []int{0}
	previous := ""
	for _, pattern := range ordered {
		if pattern == previous {
			continue
		}
		length := utf8.RuneCountInString(pattern)
		common := commonTriePrefixRunes(previous, pattern)
		for depths[len(depths)-1] > common {
			depths = depths[:len(depths)-1]
		}
		addedNodes := 1
		if depths[len(depths)-1] < common {
			addedNodes++
			depths = append(depths, common)
		}
		addedLabels := length - common
		if addedNodes > math.MaxInt32-capacity.nodes ||
			capacity.values >= math.MaxInt32 ||
			addedLabels > math.MaxInt32-capacity.labels {
			return runeTrieCapacity{}, false
		}
		capacity.nodes += addedNodes
		capacity.values++
		capacity.labels += addedLabels
		depths = append(depths, length)
		previous = pattern
	}
	return capacity, true
}

func commonTriePrefixRunes(left, right string) int {
	count := 0
	for offset := 0; offset < len(left) && offset < len(right); {
		leftRune, width := utf8.DecodeRuneInString(left[offset:])
		rightRune, _ := utf8.DecodeRuneInString(right[offset:])
		if leftRune != rightRune {
			break
		}
		count++
		offset += width
	}
	return count
}

func preprocessorTrieCapacities(values Values) (runeTrieCapacity, runeTrieCapacity) {
	if !validPreprocessorTrieEntries(values) {
		return runeTrieCapacity{}, runeTrieCapacity{}
	}
	patterns, ok := normalizedPreprocessorTriePatterns(values)
	if !ok {
		return runeTrieCapacity{}, runeTrieCapacity{}
	}
	normalized, ok := measureRuneTrieCapacity(patterns)
	if !ok {
		return runeTrieCapacity{}, runeTrieCapacity{}
	}
	var identifiers []string
	for _, entry := range values.LawNames {
		for _, pattern := range []string{entry.ResourceID, entry.RevisionID, entry.LawNumber} {
			if len(pattern) > maxPreprocessTermBytes {
				return runeTrieCapacity{}, runeTrieCapacity{}
			}
			identifiers = append(identifiers, pattern)
		}
	}
	raw, ok := measureRuneTrieCapacity(identifiers)
	if !ok {
		return runeTrieCapacity{}, runeTrieCapacity{}
	}
	return normalized, raw
}

func normalizedPreprocessorTriePatterns(values Values) ([]string, bool) {
	var patterns []string
	for _, entry := range values.LawNames {
		patterns = append(patterns, entry.Canonical)
		patterns = append(patterns, entry.Terms...)
	}
	for _, entry := range values.LegalConcepts {
		patterns = append(patterns, entry.Terms...)
	}
	for _, entry := range values.Cues {
		if len(entry.Terms) == 0 || len(entry.Terms) > maxCueTermsPerEntry {
			return nil, false
		}
		patterns = append(patterns, entry.Terms...)
	}
	for index, pattern := range patterns {
		if len(pattern) > maxPreprocessTermBytes || !utf8.ValidString(pattern) {
			return nil, false
		}
		patterns[index] = querynormalization.ComparisonKey(pattern)
		if patterns[index] == "" {
			return nil, false
		}
	}
	return patterns, true
}

// 不正設定では予約用の語を集めず、既存 builder に元の順序でエラーを返させる。
func validPreprocessorTrieEntries(values Values) bool {
	if isNilAnalyzer(values.Analyzer) || len(values.Cues) > maxCueEntries {
		return false
	}
	lawIDs := make(map[string]struct{})
	for _, entry := range values.LawNames {
		if err := validateLawEntry(entry); err != nil {
			return false
		}
		if _, exists := lawIDs[entry.ResourceID]; exists {
			return false
		}
		lawIDs[entry.ResourceID] = struct{}{}
	}
	conceptIDs := make(map[string]struct{})
	for _, entry := range values.LegalConcepts {
		if err := validateConceptEntry(entry); err != nil {
			return false
		}
		if _, exists := conceptIDs[entry.ConceptID]; exists {
			return false
		}
		conceptIDs[entry.ConceptID] = struct{}{}
	}
	cueKeys := make(map[string]struct{})
	for _, entry := range values.Cues {
		if err := validateCueEntry(entry); err != nil {
			return false
		}
		key := cueKey(entry.ProfileID, entry.CueID)
		if _, exists := cueKeys[key]; exists {
			return false
		}
		cueKeys[key] = struct{}{}
	}
	return true
}
