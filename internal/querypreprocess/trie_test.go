package querypreprocess

import (
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

// SOT-ARCH-021: 単純な参照実装との比較で索引形式から独立した照合結果を確認する。
type referenceTrieNode[T any] struct {
	children map[rune]*referenceTrieNode[T]
	values   []T
}

func newReferenceTrie[T any]() *referenceTrieNode[T] {
	return &referenceTrieNode[T]{children: make(map[rune]*referenceTrieNode[T])}
}

func (t *referenceTrieNode[T]) add(pattern string, value T) {
	current := t
	for _, character := range pattern {
		next := current.children[character]
		if next == nil {
			next = newReferenceTrie[T]()
			current.children[character] = next
		}
		current = next
	}
	current.values = append(current.values, value)
}

func (t *referenceTrieNode[T]) find(input []positionedRune) []trieHit[T] {
	hits := make([]trieHit[T], 0)
	for start := range input {
		current := t
		for end := start; end < len(input); end++ {
			current = current.children[input[end].value]
			if current == nil {
				break
			}
			for _, value := range current.values {
				hits = append(hits, trieHit[T]{
					startByte: input[start].startByte,
					endByte:   input[end].endByte,
					value:     value,
				})
			}
		}
	}
	return hits
}

func TestRuneTriePreservesOverlapsValuesAndByteSpans(t *testing.T) {
	t.Parallel()

	trie := newRuneTrie[int]()
	for index, pattern := range []string{"法", "法令", "令", "法", "𠮷法令", "法令"} {
		if err := trie.add(pattern, index); err != nil {
			t.Fatalf("SOT-ARCH-021: 語の登録エラー = %v", err)
		}
	}
	got := trie.find(rawRunes("𠮷法令"))
	want := []trieHit[int]{
		{startByte: 0, endByte: 10, value: 4},
		{startByte: 4, endByte: 7, value: 0},
		{startByte: 4, endByte: 7, value: 3},
		{startByte: 4, endByte: 10, value: 1},
		{startByte: 4, endByte: 10, value: 5},
		{startByte: 7, endByte: 10, value: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SOT-ARCH-021: 重複、接頭辞、原文位置 = %#v、期待 = %#v", got, want)
	}
}

func TestRuneTrieRejectsEmptyPattern(t *testing.T) {
	t.Parallel()

	trie := newRuneTrie[int]()
	if err := trie.add("", 1); err == nil {
		t.Fatal("SOT-ARCH-021: 空の照合語を登録しました")
	}
	if hits := trie.find(rawRunes("照合語")); len(hits) != 0 {
		t.Fatalf("SOT-ARCH-021: 未登録の語に一致しました: %#v", hits)
	}
}

func TestRuneTrieMatchesReferenceForDenseBranches(t *testing.T) {
	t.Parallel()

	patterns := []string{"接頭", "接頭語", "接頭", "\x00", "𠮷", strings.Repeat("a", 256)}
	for index := range 256 {
		character := string(rune(0x4e00 + index))
		patterns = append(patterns,
			character+"接頭語",
			"接頭語"+character,
			"共通"+character+"末尾",
		)
	}
	compact, reference := buildComparedTries(t, patterns)
	for _, pattern := range patterns {
		assertTrieHitsEqual(t, compact, reference, rawRunes("𠮷"+pattern+"接頭"))
	}
	assertTrieHitsEqual(t, compact, reference, nil)
	assertTrieHitsEqual(t, compact, reference, rawRunes("未登録のかな語"))
}

func TestRuneTrieMatchesReferenceForGeneratedPatterns(t *testing.T) {
	t.Parallel()

	generator := rand.New(rand.NewPCG(21, 28)) //nolint:gosec // SOT-ARCH-021: 暗号用途ではなく、照合結果を再現可能な合成入力で比較するための固定乱数である。
	alphabet := []rune("abc012民法令条労働ー𠮷")
	patterns := make([]string, 512)
	for index := range patterns {
		characters := make([]rune, 1+generator.IntN(40))
		for current := range characters {
			characters[current] = alphabet[generator.IntN(len(alphabet))]
		}
		patterns[index] = string(characters)
	}
	compact, reference := buildComparedTries(t, patterns)
	for range 512 {
		input := patterns[generator.IntN(len(patterns))] +
			patterns[generator.IntN(len(patterns))] + "未登録"
		assertTrieHitsEqual(t, compact, reference, rawRunes(input))
	}
}

func TestRuneTriePreservesNormalizedSegmentSpans(t *testing.T) {
	t.Parallel()

	patterns := []string{"あ", "あが", "が", "a", "ab", "法", "株式会社"}
	compact, reference := buildComparedTries(t, patterns)
	input, _, err := normalizedRunes("Ａ Ｂ、ｱｶﾞ・法㍿")
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 比較用正規化エラー = %v", err)
	}
	assertTrieHitsEqual(t, compact, reference, input)
}

func TestRuneTrieSupportsConcurrentReads(t *testing.T) {
	t.Parallel()

	compact, reference := buildComparedTries(t, []string{"語", "合成語", "合成語", "成語"})
	for range 16 {
		t.Run("並行参照", func(t *testing.T) {
			t.Parallel()
			for range 100 {
				assertTrieHitsEqual(t, compact, reference, rawRunes("合成語の語"))
			}
		})
	}
}

func buildComparedTries(
	t *testing.T,
	patterns []string,
) (*runeTrie[int], *referenceTrieNode[int]) {
	t.Helper()
	compact := newRuneTrie[int]()
	reference := newReferenceTrie[int]()
	for index, pattern := range patterns {
		if err := compact.add(pattern, index); err != nil {
			t.Fatalf("SOT-ARCH-021: 語の登録エラー = %v", err)
		}
		reference.add(pattern, index)
	}
	return compact, reference
}

func assertTrieHitsEqual(
	t *testing.T,
	compact *runeTrie[int],
	reference *referenceTrieNode[int],
	input []positionedRune,
) {
	t.Helper()
	got := compact.find(input)
	want := reference.find(input)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SOT-ARCH-021: 配列索引の照合 = %#v、参照実装 = %#v", got, want)
	}
}
