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
	// SOT-ARCH-021、SOT-ARCH-030: 比較規則を保ち、変更がない不変文字列を共有する。
	// Map は、不正 UTF-8 を range と同じ U+FFFD に置換する。
	return strings.Map(func(current rune) rune {
		if unicode.IsSpace(current) || unicode.IsPunct(current) {
			return -1
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
		return current
	}, normalized)
}
