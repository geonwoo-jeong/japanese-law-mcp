package core

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestProfileは正規短縮形の公式略称を法令検索語に保つ(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"個人情報保護法を検索し、「個人データ」を含む条文も検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("候補数 = %d, want 1", len(candidates))
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 {
		t.Fatalf("step 数 = %d, want 2", len(steps))
	}
	search, ok := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok {
		t.Fatalf("先頭の logical input = %T", steps[0].LogicalInput())
	}
	if search.Query() != "個人情報保護法" {
		t.Fatalf(
			"SOT-ARCH-021: 法令検索語 = %q, want %q",
			search.Query(),
			"個人情報保護法",
		)
	}
	content, ok := steps[1].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok {
		t.Fatalf("二番目の logical input = %T", steps[1].LogicalInput())
	}
	if terms := content.AllTerms(); len(terms) != 1 || terms[0] != "個人データ" {
		t.Fatalf("条文検索語 = %#v", terms)
	}
}

func TestProfileは不透明な略称と誤記を正式名称へ解決する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "不透明な公式略称",
			query: "入管法を探してください。",
			want:  "出入国管理及び難民認定法",
		},
		{
			name:  "比較用正規化した公式略称",
			query: "JAS法という法令略称を検索してください。",
			want:  "日本農林規格等に関する法律",
		},
		{
			name:  "一意な誤記補正",
			query: "建築規準法を検索してください。",
			want:  "建築基準法",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			generation := generateQuery(t, test.query, nil)
			candidates := generation.Candidates()
			if len(candidates) != 1 {
				t.Fatalf("候補数 = %d, want 1", len(candidates))
			}
			steps := candidates[0].Steps()
			if len(steps) != 1 {
				t.Fatalf("step 数 = %d, want 1", len(steps))
			}
			search, ok := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
			if !ok {
				t.Fatalf("logical input = %T", steps[0].LogicalInput())
			}
			if search.Query() != test.want {
				t.Fatalf(
					"SOT-ARCH-021: 法令検索語 = %q, want %q",
					search.Query(),
					test.want,
				)
			}
		})
	}
}

func TestProfileは空白だけが異なる法令名を正式名称へ正規化する(t *testing.T) {
	t.Parallel()

	generation := generateQuery(t, "消費者 契約法を探してください。", nil)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("候補数 = %d, want 1", len(candidates))
	}
	steps := candidates[0].Steps()
	if len(steps) != 1 {
		t.Fatalf("step 数 = %d, want 1", len(steps))
	}
	search, ok := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok {
		t.Fatalf("logical input = %T", steps[0].LogicalInput())
	}
	if search.Query() != "消費者契約法" {
		t.Fatalf(
			"SOT-ARCH-021: 法令検索語 = %q, want %q",
			search.Query(),
			"消費者契約法",
		)
	}
}
