package querypreprocess

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

// SOT-ARCH-021: 容量予約だけを変え、節番号、登録順序と照合結果を保つ。
func TestRuneTrieCapacityMatchesIncrementalLayout(t *testing.T) {
	t.Parallel()

	cases := [][]string{
		nil,
		{"abc", "ab", "a", "abc", "b", "ab"},
		{"法令", "法", "法律", "法令", "令", "𠮷法", "𠮷", "\x00", "\x00法", "�"},
		{"一", "丁", "丂", "七", "一丁", "一丂", "一", "𠮷", "𠮸"},
	}
	var dense []string
	for index := range 128 {
		character := string(rune(0x4e00 + index))
		dense = append(dense, "共通"+character+"末尾", character+"先頭", "共通"+character)
	}
	cases = append(cases, dense)
	for _, patterns := range cases {
		assertReservedTrieLayout(t, patterns)
		reversed := slices.Clone(patterns)
		slices.Reverse(reversed)
		assertReservedTrieLayout(t, reversed)
	}
}

func assertReservedTrieLayout(t *testing.T, patterns []string) {
	t.Helper()

	original := slices.Clone(patterns)
	capacity, ok := measureRuneTrieCapacity(patterns)
	if !ok {
		t.Fatalf("SOT-ARCH-021: 有効な照合語の容量を計算できません: %#v", patterns)
	}
	if !slices.Equal(patterns, original) {
		t.Fatal("SOT-ARCH-021: 容量計算で入力の登録順序を変更しました")
	}
	reserved := newRuneTrieWithCapacity[int](capacity)
	incremental := newRuneTrie[int]()
	firstNode, firstValues := &reserved.nodes[0], &reserved.values[0]
	for index, pattern := range patterns {
		if err := reserved.add(pattern, index); err != nil {
			t.Fatalf("SOT-ARCH-021: 容量予約した索引に登録できません: %v", err)
		}
		if err := incremental.add(pattern, index); err != nil {
			t.Fatalf("SOT-ARCH-021: 逐次拡張する索引に登録できません: %v", err)
		}
	}
	if len(reserved.nodes) != cap(reserved.nodes) ||
		len(reserved.values) != cap(reserved.values) ||
		firstNode != &reserved.nodes[0] || firstValues != &reserved.values[0] {
		t.Fatalf("SOT-ARCH-021: 容量予約が過不足です: nodes=%d/%d values=%d/%d",
			len(reserved.nodes), cap(reserved.nodes), len(reserved.values), cap(reserved.values))
	}
	if !reflect.DeepEqual(reserved, incremental) {
		t.Fatal("SOT-ARCH-021: 容量予約で索引の内容または登録順序が変わりました")
	}
	for _, input := range []string{"未登録", strings.Join(patterns, "区切"), "法令𠮷法"} {
		if !reflect.DeepEqual(reserved.find(rawRunes(input)), incremental.find(rawRunes(input))) {
			t.Fatalf("SOT-ARCH-021: 容量予約で照合結果が変わりました: %q", input)
		}
	}
}

func TestRuneTrieCapacityFallsBackForInvalidPatterns(t *testing.T) {
	t.Parallel()

	for _, patterns := range [][]string{{""}, {"abc", ""}, {"\xff"}, {"法\xe6\xb3"}} {
		capacity, ok := measureRuneTrieCapacity(patterns)
		if ok || capacity != (runeTrieCapacity{}) {
			t.Fatalf("SOT-ARCH-021: 無効な語の容量を予約しました: %#v", patterns)
		}
	}
}

func TestCommonTriePrefixUsesWholeRunes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		left, right string
		want        int
	}{
		{"", "法", 0},
		{"一", "丁", 0},
		{"𠮷", "𠮸", 0},
		{"法一", "法丁", 1},
		{"a𠮷法", "a𠮷律", 2},
		{"法令", "法令解説", 2},
		{"\x00法", "\x00令", 1},
	}
	for _, current := range cases {
		if got := commonTriePrefixRunes(current.left, current.right); got != current.want {
			t.Errorf("SOT-ARCH-021: %q と %q の共通 rune 数 = %d、期待 = %d",
				current.left, current.right, got, current.want)
		}
	}
}

func TestPreprocessorReservesBothTrieCapacities(t *testing.T) {
	t.Parallel()

	laws := validInternalLawEntries()
	laws[0].Terms = []string{"みんぽう", "民 法", "ミンポウ", "ＡＢＣ", "abc"}
	concepts := validInternalConceptEntries()
	concepts[0].Canonical = "未登録の概念名"
	concepts[0].Terms = []string{"永住権", "abc", "㍿"}
	preprocessor, err := New(Values{
		Analyzer:      emptyOccurrenceAnalyzer{},
		LawNames:      laws,
		LegalConcepts: concepts,
		Cues: []legalquery.CueVocabularyEntry{{
			ProfileID:  "synthetic-profile",
			SyntaxRole: legalquery.CueSyntaxRoleNone,
			CueID:      "synthetic-cue",
			Terms:      []string{"ＡＢＣ", "民法", "合成検索", "株式会社"},
		}},
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 合成前処理器を構築できません: %v", err)
	}
	assertExactTrieCapacity(t, preprocessor.normalizedTerms)
	assertExactTrieCapacity(t, preprocessor.identifiers)
}

func assertExactTrieCapacity[T any](t *testing.T, trie *runeTrie[T]) {
	t.Helper()
	if len(trie.nodes) != cap(trie.nodes) || len(trie.values) != cap(trie.values) {
		t.Fatalf("SOT-ARCH-021: 前処理器の照合索引に未使用容量または再確保があります: nodes=%d/%d values=%d/%d",
			len(trie.nodes), cap(trie.nodes), len(trie.values), cap(trie.values))
	}
}

func TestPreprocessorTrieCapacityFallbackPreservesValidation(t *testing.T) {
	t.Parallel()

	for _, invalid := range []string{"", "---", "\xff", strings.Repeat("a", maxPreprocessTermBytes+1)} {
		laws := validInternalLawEntries()
		laws[0].Canonical = invalid
		values := Values{Analyzer: emptyOccurrenceAnalyzer{}, LawNames: laws}
		normalized, raw := preprocessorTrieCapacities(values)
		if normalized != (runeTrieCapacity{}) || raw != (runeTrieCapacity{}) {
			t.Fatalf("SOT-ARCH-021: 不正な入力の容量を予約しました: %q", invalid)
		}
		if _, err := New(values); err == nil {
			t.Fatalf("SOT-ARCH-021: 容量予約の省略で入力検証を省略しました: %q", invalid)
		}
	}
}

func TestPreprocessorTrieCapacitySkipsOversizedCueVocabulary(t *testing.T) {
	t.Parallel()

	terms := make([]string, maxCueTermsPerEntry+1)
	for index := range terms {
		terms[index] = "合成語"
	}
	values := Values{
		Analyzer: emptyOccurrenceAnalyzer{},
		Cues: []legalquery.CueVocabularyEntry{{
			ProfileID:  "synthetic-profile",
			SyntaxRole: legalquery.CueSyntaxRoleNone,
			CueID:      "synthetic-cue",
			Terms:      terms,
		}},
	}
	if patterns, ok := normalizedPreprocessorTriePatterns(values); ok || patterns != nil {
		t.Fatal("SOT-ARCH-021: 件数上限を超えた cue の照合語をコピーしました")
	}
	if _, err := New(values); err == nil {
		t.Fatal("SOT-ARCH-021: 件数上限を超えた cue を受理しました")
	}
}

func TestPreprocessorTrieCapacitySkipsInvalidEntriesAndDuplicates(t *testing.T) {
	t.Parallel()

	valid := func() Values {
		return Values{
			Analyzer:      emptyOccurrenceAnalyzer{},
			LawNames:      validInternalLawEntries(),
			LegalConcepts: validInternalConceptEntries(),
			Cues: []legalquery.CueVocabularyEntry{{
				ProfileID:  "synthetic-profile",
				SyntaxRole: legalquery.CueSyntaxRoleNone,
				CueID:      "synthetic-cue",
				Terms:      []string{"合成検索"},
			}},
		}
	}
	cases := []struct {
		name   string
		modify func(*Values)
		prefix string
	}{
		{"法令の不正", func(values *Values) {
			values.LawNames[0].Canonical = ""
		}, "lawNames[0]:"},
		{"法令の重複を後続概念の不正より優先", func(values *Values) {
			values.LawNames = append(values.LawNames, values.LawNames[0])
			values.LegalConcepts[0].ConceptID = "INVALID"
		}, "lawId "},
		{"概念 ID の不正", func(values *Values) {
			values.LegalConcepts[0].ConceptID = "INVALID"
		}, "legalConcepts[0]: conceptId"},
		{"概念 canonical の不正", func(values *Values) {
			values.LegalConcepts[0].Canonical = ""
		}, "legalConcepts[0]: canonical"},
		{"概念の重複を後続 cue の不正より優先", func(values *Values) {
			values.LegalConcepts = append(values.LegalConcepts, values.LegalConcepts[0])
			values.Cues[0].CueID = "INVALID"
		}, "conceptId "},
		{"cue ID の不正", func(values *Values) {
			values.Cues[0].CueID = "INVALID"
		}, "cues[0]: cueId"},
		{"cue の重複", func(values *Values) {
			values.Cues = append(values.Cues, values.Cues[0])
		}, "profileId="},
	}
	for _, current := range cases {
		t.Run(current.name, func(t *testing.T) {
			values := valid()
			current.modify(&values)
			normalized, raw := preprocessorTrieCapacities(values)
			if normalized != (runeTrieCapacity{}) || raw != (runeTrieCapacity{}) {
				t.Fatal("SOT-ARCH-021: 不正 entry または重複 entry の容量を予約しました")
			}
			if _, err := New(values); err == nil || !strings.HasPrefix(err.Error(), current.prefix) {
				t.Fatalf("SOT-ARCH-021: 既存 builder の入力検証順序が変わりました: %v", err)
			}
		})
	}
}
