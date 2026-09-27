package querypreprocess

import (
	"fmt"
	"math"
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

// SOT-ARCH-021: 照合で多用する節は従来と同じ 16 byte に保つ。
// 圧縮経路の位置は別配列へ置き、構築後の照合では共有状態を変更しない。
const (
	trieIndexedChildren uint32 = 1 << 31
	trieChildIndexMask         = trieIndexedChildren - 1
	trieCompressedPath  uint32 = 1 << 31
	trieValueIndexMask         = trieCompressedPath - 1
	trieLinearChildren         = 8
)

type trieNode struct {
	character rune
	child     uint32
	sibling   uint32
	values    uint32
}

type trieEdge struct {
	parent    uint32
	character rune
}

type runeTrie[T any] struct {
	nodes    []trieNode
	spans    []uint64
	labels   []rune
	roots    map[rune]uint32
	values   [][]T
	branches map[trieEdge]uint32
}

type trieHit[T any] struct {
	startByte int
	endByte   int
	value     T
}

func newRuneTrie[T any]() *runeTrie[T] {
	return newRuneTrieWithCapacity[T](runeTrieCapacity{})
}

func (t *runeTrie[T]) add(pattern string, value T) error {
	remaining := []rune(pattern)
	if len(remaining) == 0 {
		return fmt.Errorf("照合 pattern は一文字以上必要です")
	}
	var parent uint32
	for len(remaining) > 0 {
		var next uint32
		if parent == 0 {
			next = t.roots[remaining[0]]
		} else {
			next = t.child(parent, remaining[0])
		}
		if next == 0 {
			var err error
			parent, err = t.addPath(parent, remaining)
			if err != nil {
				return err
			}
			break
		}
		start, end := trieSpanBounds(t.spans[next])
		label := t.labels[start:end]
		common := 0
		for common < len(remaining) && common < len(label) && remaining[common] == label[common] {
			common++
		}
		if common < len(label) {
			if err := t.splitPath(next, common); err != nil {
				return err
			}
		}
		parent = next
		remaining = remaining[common:]
	}
	return t.addValue(parent, value)
}

func (t *runeTrie[T]) addValue(parent uint32, value T) error {
	index := t.nodes[parent].values & trieValueIndexMask
	if index == 0 {
		valueCount := len(t.values)
		if valueCount >= math.MaxInt32 {
			return fmt.Errorf("照合索引の語数が上限を超えています")
		}
		t.nodes[parent].values = uint32(valueCount) | (t.nodes[parent].values & trieCompressedPath)
		t.values = append(t.values, []T{value})
	} else {
		t.values[index] = append(t.values[index], value)
	}
	return nil
}

func (t *runeTrie[T]) child(parent uint32, character rune) uint32 {
	first := t.nodes[parent].child
	if first&trieIndexedChildren != 0 {
		return t.branches[trieEdge{parent: parent, character: character}]
	}
	for next := first; next != 0; next = t.nodes[next].sibling {
		if t.nodes[next].character == character {
			return next
		}
	}
	return 0
}

func (t *runeTrie[T]) addPath(parent uint32, label []rune) (uint32, error) {
	count, labelCount, labelSize := len(t.nodes), len(t.labels), len(label)
	if count >= math.MaxInt32 || labelCount > math.MaxInt32 ||
		labelSize > math.MaxInt32 || labelSize > math.MaxInt32-labelCount {
		return 0, fmt.Errorf("照合索引の文字数が上限を超えています")
	}
	index, start := uint32(count), uint32(labelCount)
	end := start + uint32(labelSize)
	t.labels = append(t.labels, label...)
	t.nodes = append(t.nodes, trieNode{
		character: label[0],
		values:    triePathValues(0, end-start),
	})
	t.spans = append(t.spans, trieSpan(start, end))
	t.attachChild(parent, label[0], index)
	return index, nil
}

func (t *runeTrie[T]) splitPath(index uint32, common int) error {
	count := len(t.nodes)
	if count >= math.MaxInt32 || common <= 0 || common > math.MaxInt32 {
		return fmt.Errorf("照合経路の分割位置または節数が範囲外です")
	}
	start, end := trieSpanBounds(t.spans[index])
	suffixStart := start + uint32(common)
	if suffixStart >= end {
		return fmt.Errorf("照合経路の分割位置が終端を超えています")
	}
	original, suffix := t.nodes[index], uint32(count)
	t.nodes = append(t.nodes, trieNode{
		character: t.labels[suffixStart],
		child:     original.child,
		values:    triePathValues(original.values&trieValueIndexMask, end-suffixStart),
	})
	t.spans = append(t.spans, trieSpan(suffixStart, end))
	if original.child&trieIndexedChildren != 0 {
		t.moveIndexedChildren(index, suffix, original.child&trieChildIndexMask)
	}
	t.nodes[index] = trieNode{
		character: original.character,
		child:     suffix,
		sibling:   original.sibling,
		values:    triePathValues(0, suffixStart-start),
	}
	t.spans[index] = trieSpan(start, suffixStart)
	return nil
}

// 親の節番号を含む既存の分岐 key だけを、構築中に接尾経路へ付け替える。
func (t *runeTrie[T]) moveIndexedChildren(from uint32, to uint32, first uint32) {
	for next := first; next != 0; next = t.nodes[next].sibling {
		character := t.nodes[next].character
		t.branches[trieEdge{parent: to, character: character}] = next
		delete(t.branches, trieEdge{parent: from, character: character})
	}
}

func (t *runeTrie[T]) attachChild(parent uint32, character rune, index uint32) {
	if parent == 0 {
		t.roots[character] = index
		return
	}
	first := t.nodes[parent].child
	t.nodes[index].sibling = first & trieChildIndexMask
	t.nodes[parent].child = index | (first & trieIndexedChildren)
	if first&trieIndexedChildren != 0 {
		t.branches[trieEdge{parent: parent, character: character}] = index
		return
	}
	count := 0
	for next := index; next != 0; next = t.nodes[next].sibling {
		count++
	}
	if count <= trieLinearChildren {
		return
	}
	if t.branches == nil {
		t.branches = make(map[trieEdge]uint32)
	}
	for next := index; next != 0; next = t.nodes[next].sibling {
		t.branches[trieEdge{parent: parent, character: t.nodes[next].character}] = next
	}
	t.nodes[parent].child |= trieIndexedChildren
}

func trieSpan(start uint32, end uint32) uint64 {
	return uint64(start)<<32 | uint64(end)
}

func trieSpanBounds(span uint64) (uint32, uint32) {
	return uint32((span >> 32) & math.MaxUint32), uint32(span & math.MaxUint32)
}

func triePathValues(index uint32, length uint32) uint32 {
	if length > 1 {
		return index | trieCompressedPath
	}
	return index
}

func (t *runeTrie[T]) find(input []positionedRune) []trieHit[T] {
	hits := make([]trieHit[T], 0)
	for start := range input {
		parent := t.roots[input[start].value]
		for current := start; parent != 0; current++ {
			values := t.nodes[parent].values
			if values&trieCompressedPath != 0 {
				labelStart, labelEnd := trieSpanBounds(t.spans[parent])
				label := t.labels[labelStart:labelEnd]
				if len(label) > len(input)-current {
					break
				}
				offset := 1
				for offset < len(label) && input[current+offset].value == label[offset] {
					offset++
				}
				if offset != len(label) {
					break
				}
				current += len(label) - 1
				values &= trieValueIndexMask
			}
			for _, value := range t.values[values] {
				hits = append(hits, trieHit[T]{
					startByte: input[start].startByte,
					endByte:   input[current].endByte,
					value:     value,
				})
			}
			if current+1 == len(input) {
				break
			}
			parent = t.child(parent, input[current+1].value)
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
