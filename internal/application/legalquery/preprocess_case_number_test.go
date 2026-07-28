package legalquery_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

func TestJudicialCaseNumberMentionは検証済み構成要素を保持する(t *testing.T) {
	t.Parallel()

	const surface = "令和4年（ネ）第１００３９号"
	mention, err := legalquery.NewJudicialCaseNumberMention(
		legalquery.JudicialCaseNumberMentionValues{
			Span:         mustQuerySpan(t, 0, len(surface)),
			Surface:      surface,
			Era:          "令和",
			Year:         4,
			CaseCode:     "(ネ)",
			SerialNumber: 10039,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-027: mention を構築できません: %v", err)
	}
	if mention.Span() != mustQuerySpan(t, 0, len(surface)) ||
		mention.Surface() != surface ||
		mention.Era() != "令和" ||
		mention.Year() != 4 ||
		mention.CaseCode() != "(ネ)" ||
		mention.SerialNumber() != 10039 ||
		mention.SearchText() != "令和4年(ネ)第10039号" {
		t.Fatalf("SOT-MODEL-027: mention = %#v", mention)
	}
	if err := mention.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-027: Validate() = %v", err)
	}
}

func TestJudicialCaseNumberMentionは元年と非制御Unicode空白を受理する(
	t *testing.T,
) {
	t.Parallel()

	const surface = "令和　元　（　行ツ　）　１６４"
	mention, err := legalquery.NewJudicialCaseNumberMention(
		legalquery.JudicialCaseNumberMentionValues{
			Span:         mustQuerySpan(t, 0, len(surface)),
			Surface:      surface,
			Era:          "令和",
			Year:         1,
			CaseCode:     "(行ツ)",
			SerialNumber: 164,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-027: Unicode 空白を拒否しました: %v", err)
	}
	if mention.Year() != 1 {
		t.Fatalf("SOT-MODEL-027: 元年 = %d", mention.Year())
	}
	if mention.SearchText() != "令和1(行ツ)164" {
		t.Fatalf("SOT-MODEL-027: SearchText() = %q", mention.SearchText())
	}
}

func TestJudicialCaseNumberMentionは公的表記の事件符号構造を保持する(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		surface    string
		era        string
		year       int
		code       string
		serial     int
		searchText string
	}{
		{
			surface:    "令和7(わ)第207号",
			era:        "令和",
			year:       7,
			code:       "(わ)",
			serial:     207,
			searchText: "令和7(わ)第207号",
		},
		{
			surface:    "平成26年特（わ）第914号",
			era:        "平成",
			year:       26,
			code:       "特(わ)",
			serial:     914,
			searchText: "平成26年特(わ)第914号",
		},
		{
			surface:    "平成26特(わ)914",
			era:        "平成",
			year:       26,
			code:       "特(わ)",
			serial:     914,
			searchText: "平成26特(わ)914",
		},
		{
			surface:    "令和2（を）新102",
			era:        "令和",
			year:       2,
			code:       "(を)新",
			serial:     102,
			searchText: "令和2(を)新102",
		},
		{
			surface:    "令和０１年（受）第０１０５５号",
			era:        "令和",
			year:       1,
			code:       "(受)",
			serial:     1055,
			searchText: "令和元年(受)第1055号",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.surface, func(t *testing.T) {
			t.Parallel()

			mention, err := legalquery.NewJudicialCaseNumberMention(
				legalquery.JudicialCaseNumberMentionValues{
					Span:         mustQuerySpan(t, 0, len(test.surface)),
					Surface:      test.surface,
					Era:          test.era,
					Year:         test.year,
					CaseCode:     test.code,
					SerialNumber: test.serial,
				},
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-027: mention を構築できません: %v", err)
			}
			if mention.CaseCode() != test.code ||
				mention.SearchText() != test.searchText {
				t.Fatalf("SOT-MODEL-027: mention = %#v", mention)
			}
		})
	}
}

func TestJudicialCaseNumberMentionは不正な構造と値の不一致を拒否する(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name    string
		surface string
		era     string
		year    int
		code    string
		serial  int
	}{
		{name: "未知の元号", surface: "明治1(オ)1", era: "明治", year: 1, code: "(オ)", serial: 1},
		{name: "昭和の上限超過", surface: "昭和65(オ)1", era: "昭和", year: 65, code: "(オ)", serial: 1},
		{name: "平成の零年", surface: "平成0(オ)1", era: "平成", year: 0, code: "(オ)", serial: 1},
		{name: "令和の上限超過", surface: "令和100(オ)1", era: "令和", year: 100, code: "(オ)", serial: 1},
		{name: "事件符号の文字種", surface: "令和1(A)1", era: "令和", year: 1, code: "(A)", serial: 1},
		{name: "事件符号の長さ", surface: "令和1(あいうえおかきくけ)1", era: "令和", year: 1, code: "(あいうえおかきくけ)", serial: 1},
		{name: "番号の零", surface: "令和1(オ)0", era: "令和", year: 1, code: "(オ)", serial: 0},
		{name: "番号の上限超過", surface: "令和1(オ)100000000", era: "令和", year: 1, code: "(オ)", serial: 100000000},
		{name: "構成要素の不一致", surface: "令和1(オ)1", era: "令和", year: 2, code: "(オ)", serial: 1},
		{name: "ASCII tab", surface: "令和1\t(オ)1", era: "令和", year: 1, code: "(オ)", serial: 1},
		{name: "ASCII 改行", surface: "令和1\n(オ)1", era: "令和", year: 1, code: "(オ)", serial: 1},
		{name: "next line", surface: "令和1\u0085(オ)1", era: "令和", year: 1, code: "(オ)", serial: 1},
		{name: "line separator", surface: "令和1\u2028(オ)1", era: "令和", year: 1, code: "(オ)", serial: 1},
		{name: "paragraph separator", surface: "令和1\u2029(オ)1", era: "令和", year: 1, code: "(オ)", serial: 1},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := legalquery.NewJudicialCaseNumberMention(
				legalquery.JudicialCaseNumberMentionValues{
					Span:         mustQuerySpan(t, 0, len(test.surface)),
					Surface:      test.surface,
					Era:          test.era,
					Year:         test.year,
					CaseCode:     test.code,
					SerialNumber: test.serial,
				},
			)
			if err == nil {
				t.Fatalf("SOT-MODEL-027: %q を受理しました", test.surface)
			}
		})
	}
}

func TestPreprocessResultは事件番号を複製し正規順を検証する(t *testing.T) {
	t.Parallel()

	const query = "平成25(オ)1079と令和4年(ネ)第10039号"
	first := mustJudicialCaseNumberMention(
		t,
		query,
		"平成25(オ)1079",
		"平成",
		25,
		"オ",
		1079,
		"平成25(オ)1079",
	)
	second := mustJudicialCaseNumberMention(
		t,
		query,
		"令和4年(ネ)第10039号",
		"令和",
		4,
		"ネ",
		10039,
		"令和4年(ネ)第10039号",
	)
	values := []legalquery.JudicialCaseNumberMention{first, second}
	result, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:              query,
			ComparisonKey:      querynormalization.ComparisonKey(query),
			CaseNumberMentions: values,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025/027: result を構築できません: %v", err)
	}
	values[0] = legalquery.JudicialCaseNumberMention{}
	got := result.CaseNumberMentions()
	if len(got) != 2 || got[0].SerialNumber() != 1079 ||
		got[1].SerialNumber() != 10039 {
		t.Fatalf("SOT-MODEL-027: mentions = %#v", got)
	}
	got[0] = legalquery.JudicialCaseNumberMention{}
	if result.CaseNumberMentions()[0].SerialNumber() != 1079 {
		t.Fatal("SOT-MODEL-027: getter から result を変更できました")
	}

	_, err = legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:              query,
			ComparisonKey:      querynormalization.ComparisonKey(query),
			CaseNumberMentions: []legalquery.JudicialCaseNumberMention{second, first},
		},
	)
	if err == nil {
		t.Fatal("SOT-MODEL-027: 逆順の mention を受理しました")
	}
}

func TestCandidateGenerationInputは原文なしで事件番号を複製する(t *testing.T) {
	t.Parallel()

	const query = "平成25(オ)1079"
	mention := mustJudicialCaseNumberMention(
		t,
		query,
		query,
		"平成",
		25,
		"オ",
		1079,
		"平成25(オ)1079",
	)
	result, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:              query,
			ComparisonKey:      querynormalization.ComparisonKey(query),
			CaseNumberMentions: []legalquery.JudicialCaseNumberMention{mention},
		},
	)
	if err != nil {
		t.Fatalf("result を構築できません: %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(result)
	if err != nil {
		t.Fatalf("input を構築できません: %v", err)
	}
	got := input.CaseNumberMentions()
	if len(got) != 1 || got[0].Surface() != query {
		t.Fatalf("SOT-MODEL-027: input mentions = %#v", got)
	}
	got[0] = legalquery.JudicialCaseNumberMention{}
	if input.CaseNumberMentions()[0].Surface() != query {
		t.Fatal("SOT-MODEL-027: input getter から変更できました")
	}
	if err := input.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-027: input.Validate() = %v", err)
	}
}

func TestPreprocessResultは事件番号と全出現の上限超過を拒否する(
	t *testing.T,
) {
	t.Parallel()

	var queryBuilder strings.Builder
	caseNumbers := make([]legalquery.JudicialCaseNumberMention, 0, 65)
	for serial := 1; serial <= 65; serial++ {
		if serial > 1 {
			queryBuilder.WriteString("、")
		}
		fmt.Fprintf(&queryBuilder, "令和1(オ)%d", serial)
	}
	queryBuilder.WriteString("第1項")
	query := queryBuilder.String()
	for serial := 1; serial <= 65; serial++ {
		surface := fmt.Sprintf("令和1(オ)%d", serial)
		caseNumbers = append(
			caseNumbers,
			mustJudicialCaseNumberMention(
				t,
				query,
				surface,
				"令和",
				1,
				"オ",
				serial,
				fmt.Sprintf("令和1(オ)%d", serial),
			),
		)
	}
	if _, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:              query,
			ComparisonKey:      querynormalization.ComparisonKey(query),
			CaseNumberMentions: caseNumbers,
		},
	); err == nil {
		t.Fatal("SOT-MODEL-027: 六十五件の事件番号を受理しました")
	}

	lawSpan := mustQuerySpan(t, 0, len("令"))
	laws := make([]legalquery.LawNameMention, 0, 64)
	cues := make([]legalquery.CueMention, 0, 128)
	for index := 0; index < 64; index++ {
		laws = append(
			laws,
			mustLawNameMention(
				t,
				lawSpan,
				"令",
				fmt.Sprintf("law-%03d", index),
			),
		)
	}
	for index := 0; index < 128; index++ {
		cues = append(
			cues,
			mustCueMention(
				t,
				lawSpan,
				"令",
				"profile",
				fmt.Sprintf("cue-%03d", index),
			),
		)
	}
	if _, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:              query,
			ComparisonKey:      querynormalization.ComparisonKey(query),
			LawNameMentions:    laws,
			CueMentions:        cues,
			CaseNumberMentions: caseNumbers[:64],
			ParagraphMentions: []legalquery.ParagraphMention{
				mustParagraphMention(
					t,
					spanForSurface(t, query, "第1項"),
					"第1項",
					1,
				),
			},
		},
	); err == nil {
		t.Fatal("SOT-MODEL-025: 全出現二百五十六件超過を受理しました")
	}
}

func mustJudicialCaseNumberMention(
	t *testing.T,
	query string,
	surface string,
	era string,
	year int,
	code string,
	serial int,
	searchText string,
) legalquery.JudicialCaseNumberMention {
	t.Helper()

	mention, err := legalquery.NewJudicialCaseNumberMention(
		legalquery.JudicialCaseNumberMentionValues{
			Span:         spanForSurface(t, query, surface),
			Surface:      surface,
			Era:          era,
			Year:         year,
			CaseCode:     "(" + code + ")",
			SerialNumber: serial,
		},
	)
	if err != nil {
		t.Fatalf("事件番号 mention を構築できません: %v", err)
	}
	if mention.SearchText() != searchText {
		t.Fatalf(
			"事件番号 mention の searchText = %q, want %q",
			mention.SearchText(),
			searchText,
		)
	}
	return mention
}
