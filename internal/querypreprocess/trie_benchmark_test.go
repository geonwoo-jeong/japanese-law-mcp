package querypreprocess

import (
	"fmt"
	"testing"
)

// SOT-ARCH-021: 実辞書の本文を固定せず、接頭辞共有と長い末尾を持つ合成語で比較する。
func syntheticTriePatterns() []string {
	patterns := make([]string, 6000)
	for index := range patterns {
		patterns[index] = fmt.Sprintf(
			"%c試験第%04d号の手続及び施行に関する法律",
			rune(0x4e00+index%1024),
			index,
		)
	}
	return patterns
}

func BenchmarkRuneTrieConstruction(b *testing.B) {
	patterns := syntheticTriePatterns()
	b.Run("参照実装", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			trie := newReferenceTrie[dictionaryTarget]()
			for _, pattern := range patterns {
				trie.add(pattern, dictionaryTarget{term: pattern})
			}
		}
	})
	b.Run("配列索引", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			trie := newRuneTrie[dictionaryTarget]()
			for _, pattern := range patterns {
				if err := trie.add(pattern, dictionaryTarget{term: pattern}); err != nil {
					b.Fatalf("SOT-ARCH-021: 語の登録エラー = %v", err)
				}
			}
		}
	})
}

func BenchmarkRuneTrieFind(b *testing.B) {
	patterns := syntheticTriePatterns()
	compact := newRuneTrie[dictionaryTarget]()
	reference := newReferenceTrie[dictionaryTarget]()
	for _, pattern := range patterns {
		target := dictionaryTarget{term: pattern}
		if err := compact.add(pattern, target); err != nil {
			b.Fatalf("SOT-ARCH-021: 語の登録エラー = %v", err)
		}
		reference.add(pattern, target)
	}
	tests := []struct {
		name  string
		input string
	}{
		{name: "一致", input: patterns[2000]},
		{name: "途中不一致", input: "両試験第0000号の未登録語"},
		{name: "不一致", input: "この仮の単語では照合しません"},
		{name: "複数一致", input: patterns[11] + "と" + patterns[4020]},
	}
	for _, test := range tests {
		input := rawRunes(test.input)
		b.Run(test.name+"/参照実装", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = reference.find(input)
			}
		})
		b.Run(test.name+"/配列索引", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = compact.find(input)
			}
		})
	}
}
