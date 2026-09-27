package searchquery

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func TestBoundedDamerauLevenshteinExhaustive(t *testing.T) {
	t.Parallel()

	values := distanceTestStrings([]rune{'a', '法', '🙂'}, 4)
	for _, left := range values {
		for _, right := range values {
			assertBoundedDistance(t, left, right, 3)
		}
	}
}

func TestBoundedDamerauLevenshteinLongInputs(t *testing.T) {
	t.Parallel()

	random := rand.New(rand.NewPCG(21, 30)) //nolint:gosec // SOT-ARCH-021: 暗号用途ではなく、再現可能な合成入力を作るための固定乱数である。
	alphabet := []rune{'a', 'b', '法', '令', '🙂', '\u0301'}
	for range 500 {
		left := make([]rune, random.IntN(100))
		for index := range left {
			left[index] = alphabet[random.IntN(len(alphabet))]
		}
		right := slices.Clone(left)
		for range random.IntN(8) {
			right = distanceTestEdit(random, right, alphabet)
		}
		assertBoundedDistance(t, left, right, 6)
		assertBoundedDistance(t, right, left, 6)
	}
}

func TestBoundedDamerauLevenshteinAdjacentTranspositions(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		left  string
		right string
		want  int
	}{
		{left: "法令", right: "令法", want: 1},
		{left: "法令ab", right: "令法ba", want: 2},
		{left: "CA", right: "ABC", want: 3},
		{left: "a🙂法令b", right: "a法🙂令b", want: 1},
	} {
		got := boundedDamerauLevenshtein([]rune(testCase.left), []rune(testCase.right), 3)
		if got != testCase.want {
			t.Fatalf("SOT-ARCH-021: 隣接転置の距離 (%q, %q) = %d、期待値 %d", testCase.left, testCase.right, got, testCase.want)
		}
	}
}

func assertBoundedDistance(t *testing.T, left, right []rune, largestMaximum int) {
	t.Helper()

	want := fullMatrixAdjacentDistance(left, right)
	for maximum := 0; maximum <= largestMaximum; maximum++ {
		got := boundedDamerauLevenshtein(left, right, maximum)
		boundedWant := min(want, maximum+1)
		if got != boundedWant {
			t.Fatalf("SOT-ARCH-021: 距離 (%q, %q, 上限 %d) = %d、期待値 %d", string(left), string(right), maximum, got, boundedWant)
		}
	}
}

func distanceTestStrings(alphabet []rune, maximumLength int) [][]rune {
	values := [][]rune{nil}
	level := [][]rune{nil}
	for range maximumLength {
		next := make([][]rune, 0, len(level)*len(alphabet))
		for _, prefix := range level {
			for _, character := range alphabet {
				next = append(next, append(slices.Clone(prefix), character))
			}
		}
		values = append(values, next...)
		level = next
	}
	return values
}

func distanceTestEdit(random *rand.Rand, value, alphabet []rune) []rune {
	if len(value) == 0 {
		return []rune{alphabet[random.IntN(len(alphabet))]}
	}
	index := random.IntN(len(value))
	switch random.IntN(4) {
	case 0:
		return slices.Insert(value, index, alphabet[random.IntN(len(alphabet))])
	case 1:
		return slices.Delete(value, index, index+1)
	case 2:
		value[index] = alphabet[random.IntN(len(alphabet))]
	default:
		if index > 0 {
			value[index-1], value[index] = value[index], value[index-1]
		}
	}
	return value
}

// SOT-ARCH-021: 帯幅や行再利用に依存しない全行列で隣接転置の既存動作を検証する。
func fullMatrixAdjacentDistance(left, right []rune) int {
	matrix := make([][]int, len(left)+1)
	for index := range matrix {
		matrix[index] = make([]int, len(right)+1)
		matrix[index][0] = index
	}
	for index := range matrix[0] {
		matrix[0][index] = index
	}
	for row := 1; row <= len(left); row++ {
		for column := 1; column <= len(right); column++ {
			cost := 1
			if left[row-1] == right[column-1] {
				cost = 0
			}
			matrix[row][column] = min(matrix[row-1][column]+1, matrix[row][column-1]+1, matrix[row-1][column-1]+cost)
			if row > 1 && column > 1 && left[row-1] == right[column-2] && left[row-2] == right[column-1] {
				matrix[row][column] = min(matrix[row][column], matrix[row-2][column-2]+1)
			}
		}
	}
	return matrix[len(left)][len(right)]
}

func BenchmarkBoundedDamerauLevenshtein(b *testing.B) {
	for _, testCase := range []struct {
		name    string
		length  int
		maximum int
	}{
		{name: "短い語", length: 4, maximum: 1},
		{name: "中程度の語", length: 16, maximum: 3},
		{name: "長い語", length: 64, maximum: 3},
		{name: "長い入力", length: 256, maximum: 3},
	} {
		left := []rune(strings.Repeat("法令文規", testCase.length/4))
		right := slices.Clone(left)
		right[len(right)/2] = '別'
		b.Run(testCase.name, func(b *testing.B) {
			b.ReportAllocs()
			got := 0
			for b.Loop() {
				got = boundedDamerauLevenshtein(left, right, testCase.maximum)
			}
			if got != 1 {
				b.Fatalf("SOT-ARCH-021: 一文字置換の距離 = %d、期待値 1", got)
			}
		})
	}
}
