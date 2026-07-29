package core

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestProfileは同じ検索語の包含と除外を位置結合後に拒否する(
	t *testing.T,
) {
	t.Parallel()

	profile := mustProfile(t)
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("preprocessor を構築できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: "法令本文から「匿名加工情報」を除いて「匿名加工情報」と「委託契約」を検索してください。",
	})
	if err != nil {
		t.Fatalf("request を構築できません: %v", err)
	}
	preprocessed, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("前処理に失敗しました: %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf("profile 入力を構築できません: %v", err)
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を構築できません: %v", err)
	}
	if _, err := profile.Generate(input, scope); err == nil {
		t.Fatal("SOT-MODEL-025: 異なる span の包含語と除外語を縮約しました")
	}
}
