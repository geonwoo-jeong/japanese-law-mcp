package searchquery

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func comparisonKey(value string) string {
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
