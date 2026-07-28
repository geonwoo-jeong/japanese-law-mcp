package querypreprocess_test

import (
	"context"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestPreprocessは完全な事件番号を原文順の型付きfactへ変換する(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustQueryTermPreprocessor(t)
	const query = "平成25(オ)1079、令和4年（ネ）第１００３９号、令和　元年　（行ツ）　１６４、" +
		"令和7(わ)第207号、平成26年特（わ）第914号、令和2（を）新102の裁判例を検索"
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-027: Preprocess() = %v", err)
	}
	mentions := result.CaseNumberMentions()
	if len(mentions) != 6 {
		t.Fatalf("SOT-MODEL-027: mentions = %#v", mentions)
	}
	gotSurface := make([]string, 0, len(mentions))
	gotEra := make([]string, 0, len(mentions))
	gotYear := make([]int, 0, len(mentions))
	gotCode := make([]string, 0, len(mentions))
	gotSerial := make([]int, 0, len(mentions))
	gotSearchText := make([]string, 0, len(mentions))
	for _, mention := range mentions {
		gotSurface = append(gotSurface, mention.Surface())
		gotEra = append(gotEra, mention.Era())
		gotYear = append(gotYear, mention.Year())
		gotCode = append(gotCode, mention.CaseCode())
		gotSerial = append(gotSerial, mention.SerialNumber())
		gotSearchText = append(gotSearchText, mention.SearchText())
		assertSpan(
			t,
			query,
			mention.Span(),
			mention.Surface(),
			indexOfSurface(t, query, mention.Surface()),
		)
	}
	if !slices.Equal(gotSurface, []string{
		"平成25(オ)1079",
		"令和4年（ネ）第１００３９号",
		"令和　元年　（行ツ）　１６４",
		"令和7(わ)第207号",
		"平成26年特（わ）第914号",
		"令和2（を）新102",
	}) ||
		!slices.Equal(gotEra, []string{
			"平成",
			"令和",
			"令和",
			"令和",
			"平成",
			"令和",
		}) ||
		!slices.Equal(gotYear, []int{25, 4, 1, 7, 26, 2}) ||
		!slices.Equal(gotCode, []string{
			"(オ)",
			"(ネ)",
			"(行ツ)",
			"(わ)",
			"特(わ)",
			"(を)新",
		}) ||
		!slices.Equal(gotSerial, []int{1079, 10039, 164, 207, 914, 102}) ||
		!slices.Equal(gotSearchText, []string{
			"平成25(オ)1079",
			"令和4年(ネ)第10039号",
			"令和元年(行ツ)164",
			"令和7(わ)第207号",
			"平成26年特(わ)第914号",
			"令和2(を)新102",
		}) {
		t.Fatalf(
			"SOT-MODEL-027: surface=%#v era=%#v year=%#v code=%#v serial=%#v search=%#v",
			gotSurface,
			gotEra,
			gotYear,
			gotCode,
			gotSerial,
			gotSearchText,
		)
	}
}

func TestPreprocessは事件番号を形態素句へ重複させず引用句とは併存する(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustQueryTermPreprocessor(t)
	const query = "「令和5年（受）第123号」と平成19(行ツ)164の裁判例を個別に検索"
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-027: Preprocess() = %v", err)
	}
	if len(result.CaseNumberMentions()) != 2 {
		t.Fatalf("SOT-MODEL-027: case numbers = %#v", result.CaseNumberMentions())
	}
	terms := result.QueryTermMentions()
	if len(terms) != 1 ||
		terms[0].Kind() != legalquery.QueryTermMentionQuotedPhrase ||
		terms[0].Surface() != "令和5年（受）第123号" {
		t.Fatalf("SOT-MODEL-027: query terms = %#v", terms)
	}
}

func TestPreprocessは不完全または範囲外の事件番号を採用しない(t *testing.T) {
	t.Parallel()

	preprocessor := mustQueryTermPreprocessor(t)
	tests := []string{
		"平成25(オ)の裁判例を検索",
		"平成25オ1079の裁判例を検索",
		"平成25(オ)0の裁判例を検索",
		"平成32(オ)1の裁判例を検索",
		"昭和65(オ)1の裁判例を検索",
		"令和100(オ)1の裁判例を検索",
		"令和1(A)1の裁判例を検索",
		"令和1(あいうえおかきくけ)1の裁判例を検索",
		"令和1(オ)100000000の裁判例を検索",
		"令和1オ1の裁判例を検索",
		"令和1((オ))1の裁判例を検索",
		"令和1(オ)(ア)1の裁判例を検索",
		"令和1(オ第1号の裁判例を検索",
		"9令和1(オ)1の裁判例を検索",
		"令和1(オ)12号の裁判例を検索",
		"令和1(オ)+1の裁判例を検索",
		"令和1(オ)1.2の裁判例を検索",
	}
	for _, query := range tests {
		query := query
		t.Run(query, func(t *testing.T) {
			t.Parallel()

			result, err := preprocessor.Preprocess(
				context.Background(),
				mustRequest(t, query),
			)
			if err != nil {
				t.Fatalf("Preprocess() = %v", err)
			}
			if mentions := result.CaseNumberMentions(); len(mentions) != 0 {
				t.Fatalf("SOT-MODEL-027: %q => %#v", query, mentions)
			}
		})
	}
}

func TestCandidateGenerationInputは構造だけの照会境界を一度だけ確定する(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustQueryTermPreprocessor(t)
	tests := []struct {
		name       string
		query      string
		standalone bool
	}{
		{
			name:       "事件番号だけ",
			query:      "平成25(オ)1079",
			standalone: true,
		},
		{
			name:       "引用された事件番号だけ",
			query:      "「平成25(オ)1079」",
			standalone: true,
		},
		{
			name:       "句読点で列挙",
			query:      "平成25(オ)1079、令和7(わ)第207号。",
			standalone: true,
		},
		{
			name:  "task と resource がある",
			query: "平成25(オ)1079の裁判例を検索",
		},
		{
			name:  "異なる一般検索語がある",
			query: "平成25(オ)1079と医療過誤",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := preprocessor.Preprocess(
				context.Background(),
				mustRequest(t, test.query),
			)
			if err != nil {
				t.Fatalf("Preprocess() = %v", err)
			}
			input, err := legalquery.NewCandidateGenerationInput(result)
			if err != nil {
				t.Fatalf("generation input = %v", err)
			}
			if input.StandaloneStructuredQuery() != test.standalone {
				t.Fatalf(
					"SOT-MODEL-026: StandaloneStructuredQuery() = %t, want %t; result=%#v",
					input.StandaloneStructuredQuery(),
					test.standalone,
					snapshotResult(result),
				)
			}
		})
	}
}

func indexOfSurface(t *testing.T, query string, surface string) int {
	t.Helper()

	for start := 0; start+len(surface) <= len(query); start++ {
		if query[start:start+len(surface)] == surface {
			return start
		}
	}
	t.Fatalf("surface %q が query にありません", surface)
	return -1
}
