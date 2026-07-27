// Package querynormalization は、能力とプロバイダーに依存しない照会文の比較用正規化を提供する。
package querynormalization

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// ComparisonKey は、表示値を変更せずに辞書照合と重複判定だけで使う比較キーを返す。
func ComparisonKey(value string) string {
	normalized := norm.NFKC.String(value)
	var builder strings.Builder
	builder.Grow(len(normalized))
	for _, current := range normalized {
		if unicode.IsSpace(current) || unicode.IsPunct(current) {
			continue
		}
		switch {
		case current >= 'A' && current <= 'Z':
			current += 'a' - 'A'
		case current >= '\u30a1' && current <= '\u30f6':
			current -= '\u0060'
		case current == '\u30fd':
			current = '\u309d'
		case current == '\u30fe':
			current = '\u309e'
		}
		builder.WriteRune(current)
	}
	return builder.String()
}
