package querypreprocess

import (
	"maps"
	"reflect"
	"slices"
	"testing"
)

func TestRuneTrieHot16SplitsPathsWithIndexedChildren(t *testing.T) {
	t.Parallel()

	patterns := make([]string, 0, 24)
	for index := range 16 {
		patterns = append(patterns, "𠮷共通接頭長文"+string(rune(0x4e00+index))+"末尾")
	}
	// 多分岐になった経路を途中で二度分割し、重複と別経路も追加する。
	patterns = append(patterns,
		"𠮷共通", "𠮷共通接頭", "𠮷共通", "𠮷共通短語", "𠮷共通接頭長文",
	)
	compact, reference := buildComparedTries(t, patterns)
	for _, pattern := range patterns {
		assertTrieHitsEqual(t, compact, reference, rawRunes(pattern+"𠮷共通"))
	}
	assertTrieHitsEqual(t, compact, reference, rawRunes("𠮷共通接頭長文未登録"))
}

func TestRuneTrieHot16PreservesPrefixInsertionOrders(t *testing.T) {
	t.Parallel()

	orders := [][]string{
		{"甲乙丙丁", "甲乙", "甲", "甲乙丙", "甲乙戊", "甲乙丙丁"},
		{"甲", "甲乙", "甲乙丙", "甲乙丙丁", "甲乙丙丁", "甲乙戊"},
		{"甲乙丙丁", "甲乙戊", "甲乙丙", "甲", "甲乙", "甲乙丙丁"},
	}
	for _, patterns := range orders {
		compact, reference := buildComparedTries(t, patterns)
		for _, input := range []string{
			"甲", "甲乙", "甲乙丙", "甲乙丙丁", "甲乙戊", "甲乙丙己", "甲乙丙丁甲乙戊",
		} {
			assertTrieHitsEqual(t, compact, reference, rawRunes(input))
		}
	}
}

func TestRuneTrieHot16SearchPreservesBuiltIndex(t *testing.T) {
	t.Parallel()

	patterns := []string{"𠮷共通接頭長文", "𠮷共通", "𠮷", "𠮷共通", "法", "法令"}
	for index := range 16 {
		patterns = append(patterns, "𠮷共通接頭長文"+string(rune(0x4e00+index)))
	}
	trie, reference := buildComparedTries(t, patterns)
	before := &runeTrie[int]{
		nodes:    slices.Clone(trie.nodes),
		spans:    slices.Clone(trie.spans),
		labels:   slices.Clone(trie.labels),
		roots:    maps.Clone(trie.roots),
		values:   slices.Clone(trie.values),
		branches: maps.Clone(trie.branches),
	}
	for index, values := range before.values {
		before.values[index] = slices.Clone(values)
	}
	for _, pattern := range append(patterns, "未登録の合成照会") {
		input := rawRunes("原文" + pattern + "法令")
		original := slices.Clone(input)
		assertTrieHitsEqual(t, trie, reference, input)
		if !slices.Equal(input, original) {
			t.Fatal("SOT-ARCH-021: 照合で入力の原文位置を変更しました")
		}
	}
	if !reflect.DeepEqual(trie, before) {
		t.Fatal("SOT-ARCH-021: 照合で構築済み索引を変更しました")
	}
}
