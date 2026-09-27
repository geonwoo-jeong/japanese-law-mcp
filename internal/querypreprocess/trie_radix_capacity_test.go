package querypreprocess

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

// SOT-ARCH-021: flat の全接頭辞数を圧縮節数として予約しない。
func TestRuneTrieRadixCapacityCountsBranchesAndTerminalPrefixes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		patterns []string
		want     runeTrieCapacity
	}{
		{nil, runeTrieCapacity{nodes: 1, values: 1}},
		{[]string{"甲乙丙丁戊己"}, runeTrieCapacity{nodes: 2, values: 2, labels: 6}},
		{[]string{"甲", "甲乙", "甲乙丙", "甲"}, runeTrieCapacity{nodes: 4, values: 4, labels: 3}},
		{[]string{"abc", "abd", "abe"}, runeTrieCapacity{nodes: 5, values: 4, labels: 5}},
		{[]string{"aaa", "aab", "aba", "abb", "aca"}, runeTrieCapacity{nodes: 9, values: 6, labels: 9}},
		{[]string{"𠮷甲乙", "𠮷甲丙", "𠮷丁乙", "𠮷丁丙", "𠮷甲"}, runeTrieCapacity{nodes: 8, values: 6, labels: 7}},
	}
	for _, current := range cases {
		got, ok := measureRuneTrieCapacity(current.patterns)
		if !ok || got != current.want {
			t.Fatalf("SOT-ARCH-021: radix 容量 = %#v、期待 = %#v、語 = %#v",
				got, current.want, current.patterns)
		}
		assertReservedRadixCapacity(t, current.patterns)
		reversed := slices.Clone(current.patterns)
		slices.Reverse(reversed)
		assertReservedRadixCapacity(t, reversed)
	}
}

func TestRuneTrieRadixCapacityPreservesDenseSplitRegistrations(t *testing.T) {
	t.Parallel()

	var patterns []string
	for index := range 32 {
		patterns = append(patterns, "𠮷共通接頭長文"+string(rune(0x4e00+index))+"末尾")
	}
	patterns = append(patterns, "𠮷共通", "𠮷", "𠮷共通接頭", "𠮷共通接頭長文", "𠮷共通", "\x00")
	assertReservedRadixCapacity(t, patterns)
	slices.Reverse(patterns)
	assertReservedRadixCapacity(t, patterns)
}

func assertReservedRadixCapacity(t *testing.T, patterns []string) {
	t.Helper()

	capacity, ok := measureRuneTrieCapacity(patterns)
	if !ok {
		t.Fatal("SOT-ARCH-021: 有効な合成語の radix 容量を計算できません")
	}
	trie := newRuneTrieWithCapacity[int](capacity)
	reference := newReferenceTrie[int]()
	for index, pattern := range patterns {
		if err := trie.add(pattern, index); err != nil {
			t.Fatalf("SOT-ARCH-021: radix の合成語を登録できません: %v", err)
		}
		reference.add(pattern, index)
	}
	if len(trie.nodes) != cap(trie.nodes) ||
		len(trie.spans) != cap(trie.spans) ||
		len(trie.labels) != cap(trie.labels) ||
		len(trie.values) != cap(trie.values) {
		t.Fatalf("SOT-ARCH-021: radix 容量に過不足があります: nodes=%d/%d spans=%d/%d labels=%d/%d values=%d/%d",
			len(trie.nodes), cap(trie.nodes), len(trie.spans), cap(trie.spans),
			len(trie.labels), cap(trie.labels), len(trie.values), cap(trie.values))
	}
	input := rawRunes(strings.Join(patterns, "未登録"))
	if !reflect.DeepEqual(trie.find(input), reference.find(input)) {
		t.Fatal("SOT-ARCH-021: 容量予約を含む radix の照合結果が参照実装と一致しません")
	}
}
