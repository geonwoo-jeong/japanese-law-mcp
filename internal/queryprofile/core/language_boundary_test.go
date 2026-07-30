package core

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestProfileは中国語混在を非日本語として閉じる(t *testing.T) {
	t.Parallel()

	for _, query := range []string{
		"请民法を検索",
		"请阅读で検索",
	} {
		generation := generateQuery(t, query, nil)
		if !slices.Contains(
			generation.Signals(),
			legalquery.CandidateSignalNonJapaneseQuery,
		) {
			t.Fatalf(
				"SOT-MODEL-026: query=%q の non_japanese_query signal がありません: %#v",
				query,
				generation.Signals(),
			)
		}
	}
}
