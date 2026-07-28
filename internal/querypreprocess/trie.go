package querypreprocess

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
	"golang.org/x/text/unicode/norm"
)

type positionedRune struct {
	value     rune
	startByte int
	endByte   int
}

type trieNode[T any] struct {
	children map[rune]*trieNode[T]
	values   []T
}

type runeTrie[T any] struct {
	root *trieNode[T]
}

type trieHit[T any] struct {
	startByte int
	endByte   int
	value     T
}

func newRuneTrie[T any]() *runeTrie[T] {
	return &runeTrie[T]{root: &trieNode[T]{children: make(map[rune]*trieNode[T])}}
}

func (t *runeTrie[T]) add(pattern string, value T) error {
	runes := []rune(pattern)
	if len(runes) == 0 {
		return fmt.Errorf("照合 pattern は一文字以上必要です")
	}
	node := t.root
	for _, current := range runes {
		next := node.children[current]
		if next == nil {
			next = &trieNode[T]{children: make(map[rune]*trieNode[T])}
			node.children[current] = next
		}
		node = next
	}
	node.values = append(node.values, value)
	return nil
}

func (t *runeTrie[T]) find(input []positionedRune) []trieHit[T] {
	hits := make([]trieHit[T], 0)
	for start := range input {
		node := t.root
		for current := start; current < len(input); current++ {
			node = node.children[input[current].value]
			if node == nil {
				break
			}
			for _, value := range node.values {
				hits = append(hits, trieHit[T]{
					startByte: input[start].startByte,
					endByte:   input[current].endByte,
					value:     value,
				})
			}
		}
	}
	return hits
}

func rawRunes(value string) []positionedRune {
	runes := make([]positionedRune, 0, len(value))
	for startByte, current := range value {
		endByte := startByte + len(string(current))
		runes = append(runes, positionedRune{
			value:     current,
			startByte: startByte,
			endByte:   endByte,
		})
	}
	return runes
}

func normalizedRunes(value string) ([]positionedRune, string, error) {
	var iterator norm.Iter
	iterator.InitString(norm.NFKC, value)

	runes := make([]positionedRune, 0, len(value))
	var keyBuilder strings.Builder
	for !iterator.Done() {
		startByte := iterator.Pos()
		segment := iterator.Next()
		endByte := iterator.Pos()
		for _, current := range string(segment) {
			if unicode.IsSpace(current) || unicode.IsPunct(current) {
				continue
			}
			current = normalizeComparisonRune(current)
			keyBuilder.WriteRune(current)
			runes = append(runes, positionedRune{
				value:     current,
				startByte: startByte,
				endByte:   endByte,
			})
		}
	}
	key := keyBuilder.String()
	if key != querynormalization.ComparisonKey(value) {
		return nil, "", fmt.Errorf("比較用正規化と span 対応が一致しません")
	}
	return runes, key, nil
}

func normalizeComparisonRune(current rune) rune {
	switch {
	case current >= 'A' && current <= 'Z':
		return current + 'a' - 'A'
	case current >= '\u30a1' && current <= '\u30f6':
		return current - '\u0060'
	case current == '\u30fd':
		return '\u309d'
	case current == '\u30fe':
		return '\u309e'
	default:
		return current
	}
}
