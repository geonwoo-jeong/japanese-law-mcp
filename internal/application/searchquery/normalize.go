package searchquery

import "github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"

func comparisonKey(value string) string {
	return querynormalization.ComparisonKey(value)
}
