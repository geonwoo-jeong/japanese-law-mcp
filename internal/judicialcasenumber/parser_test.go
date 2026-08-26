package judicialcasenumber_test

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/judicialcasenumber"
)

func TestParsePrefixは公的表記の事件番号を解析する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input      string
		era        string
		year       int
		caseCode   string
		serial     int
		searchText string
		endByte    int
	}{
		{
			input:      "令和7(わ)第207号",
			era:        "令和",
			year:       7,
			caseCode:   "(わ)",
			serial:     207,
			searchText: "令和7(わ)第207号",
			endByte:    len("令和7(わ)第207号"),
		},
		{
			input:      "平成26年特（わ）第914号 以下省略",
			era:        "平成",
			year:       26,
			caseCode:   "特(わ)",
			serial:     914,
			searchText: "平成26年特(わ)第914号",
			endByte:    len("平成26年特（わ）第914号"),
		},
		{
			input:      "令和０１年（受）第０１０５５号",
			era:        "令和",
			year:       1,
			caseCode:   "(受)",
			serial:     1055,
			searchText: "令和元年(受)第1055号",
			endByte:    len("令和０１年（受）第０１０５５号"),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			got, err := judicialcasenumber.ParsePrefix(test.input)
			if err != nil {
				t.Fatalf("ParsePrefix() のエラー = %v", err)
			}
			if got.Era() != test.era ||
				got.Year() != test.year ||
				got.CaseCode() != test.caseCode ||
				got.SerialNumber() != test.serial ||
				got.SearchText() != test.searchText ||
				got.EndByte() != test.endByte {
				t.Fatalf("ParsePrefix() = %#v", got)
			}
		})
	}
}

func TestParsePrefixは不正な事件番号を拒否する(t *testing.T) {
	t.Parallel()

	for _, input := range []string{
		"明治1(オ)1",
		"平成0(オ)1",
		"令和100(オ)1",
		"令和1(A)1",
		"令和1(オ)0",
		"令和1\t(オ)1",
	} {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if _, err := judicialcasenumber.ParsePrefix(input); err == nil {
				t.Fatalf("不正な入力 %q を受理しました", input)
			}
		})
	}
}
