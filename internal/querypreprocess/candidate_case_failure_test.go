package querypreprocess_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

type candidateCaseFailure interface {
	CandidateCaseFailure()
}

func TestPreprocessは誤記候補予算超過を入力別処理失敗として分類する(
	t *testing.T,
) {
	t.Parallel()

	preprocessor, err := querypreprocess.NewEmbedded(nil)
	if err != nil {
		t.Fatalf("SOT-ENG-041: 前処理器を構築できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: strings.Repeat("東京", 100),
	})
	if err != nil {
		t.Fatalf("SOT-ENG-041: 有効な境界試験 request を構築できません: %v", err)
	}

	_, err = preprocessor.Preprocess(context.Background(), request)
	if err == nil {
		t.Fatal("SOT-ENG-041: 誤記候補予算を超えた入力を受理しました")
	}
	var failure candidateCaseFailure
	if !errors.As(err, &failure) {
		t.Fatalf("SOT-ENG-041: 誤記候補予算超過が入力別処理失敗ではありません: %v", err)
	}
}
