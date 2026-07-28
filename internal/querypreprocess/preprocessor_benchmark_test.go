package querypreprocess_test

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func BenchmarkPreprocessEmbedded(b *testing.B) {
	preprocessor, err := querypreprocess.NewEmbedded(
		[]legalquery.CueVocabularyEntry{{
			ProfileID: "benchmark-core",
			CueID:     "task-search",
			Terms:     []string{"検索"},
		}},
	)
	if err != nil {
		b.Fatalf("SOT-ARCH-021: 前処理器を作成できません: %v", err)
	}
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "登録済み法令名",
			query: "民法第709条を検索",
		},
		{
			name:  "一意な誤記",
			query: "労契去を検索",
		},
	}
	for _, test := range tests {
		request, requestErr := legalquery.NewRequest(legalquery.RequestValues{
			Query: test.query,
		})
		if requestErr != nil {
			b.Fatalf("SOT-MODEL-025: 照会を作成できません: %v", requestErr)
		}
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, preprocessErr := preprocessor.Preprocess(
					context.Background(),
					request,
				); preprocessErr != nil {
					b.Fatalf(
						"SOT-MODEL-025: Preprocess() のエラー = %v",
						preprocessErr,
					)
				}
			}
		})
	}
}
