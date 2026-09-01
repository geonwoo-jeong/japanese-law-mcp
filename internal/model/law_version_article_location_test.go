package model_test

import (
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLawVersionArticleLocationAcceptsSingleArticleAndOfficialRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		articleNumber string
	}{
		{name: "単一条", articleNumber: "38_3_2"},
		{name: "単純な条範囲", articleNumber: "38:84"},
		{name: "桁数が増える条範囲", articleNumber: "9:10"},
		{name: "枝番号を含む条範囲", articleNumber: "38_2:38_4"},
		{name: "枝番号の桁数が増える条範囲", articleNumber: "38_2:38_10"},
		{name: "単一条から枝番号への範囲", articleNumber: "38:38_2"},
		{name: "枝番号から次の単一条への範囲", articleNumber: "38_2:39"},
		{
			name:          "64 byte の条範囲",
			articleNumber: strings.Repeat("1", 31) + ":" + strings.Repeat("2", 32),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			location, err := model.NewLawVersionArticleLocation(
				model.LawVersionArticleLocationValues{
					Provision:     model.LawArticleProvisionMain,
					ArticleNumber: test.articleNumber,
				},
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-033: 条番号 %q を受理できません: %v", test.articleNumber, err)
			}
			if location.ArticleNumber() != test.articleNumber {
				t.Fatalf("ArticleNumber() = %q、期待値 = %q", location.ArticleNumber(), test.articleNumber)
			}
		})
	}
}

func TestLawVersionArticleLocationRejectsInvalidOfficialRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		articleNumber string
	}{
		{name: "開始条なし", articleNumber: ":84"},
		{name: "終了条なし", articleNumber: "38:"},
		{name: "複数の区切り", articleNumber: "38:84:90"},
		{name: "開始条が不正", articleNumber: "038:84"},
		{name: "終了条が不正", articleNumber: "38:084"},
		{name: "同一条", articleNumber: "38:38"},
		{name: "逆順", articleNumber: "84:38"},
		{name: "桁数が減る逆順", articleNumber: "10:9"},
		{name: "枝番号の逆順", articleNumber: "38_4:38_2"},
		{name: "枝番号の桁数が減る逆順", articleNumber: "38_10:38_2"},
		{name: "65 byte", articleNumber: strings.Repeat("1", 32) + ":" + strings.Repeat("2", 32)},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := model.NewLawVersionArticleLocation(
				model.LawVersionArticleLocationValues{
					Provision:     model.LawArticleProvisionMain,
					ArticleNumber: test.articleNumber,
				},
			)
			if err == nil {
				t.Fatalf("SOT-MODEL-033: 不正な条範囲 %q を受理しました", test.articleNumber)
			}
		})
	}
}

func TestLawArticleLocationStillRejectsOfficialRange(t *testing.T) {
	t.Parallel()

	_, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:     model.LawArticleProvisionMain,
		ArticleNumber: "38:84",
	})
	if err == nil {
		t.Fatal("SOT-MODEL-018: 個別条位置が条範囲を受理しました")
	}
}
