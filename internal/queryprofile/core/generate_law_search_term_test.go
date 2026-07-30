package core

import (
	"slices"
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
		{
			name:  "一文字を挿入した誤記補正",
			query: "労働基礎準法を法令名で検索してください。",
			want:  "労働基準法",
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

func TestProfileは長い法令名の誤記に含まれる短い誤記候補を検索しない(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"行政事件等訴訟法の法令情報を見つけたいです。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("候補数 = %d, want 1: %#v", len(candidates), candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 1 {
		t.Fatalf("step 数 = %d, want 1", len(steps))
	}
	search, ok := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok {
		t.Fatalf("logical input = %T", steps[0].LogicalInput())
	}
	if search.Query() != "行政事件訴訟法" {
		t.Fatalf(
			"SOT-ARCH-021: 法令検索語 = %q, want %q",
			search.Query(),
			"行政事件訴訟法",
		)
	}
	if !slices.Equal(
		candidates[0].EvidenceCodes(),
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
			legalquery.EvidenceOfficialAlias,
			legalquery.EvidenceUniqueTypoCorrection,
		},
	) {
		t.Fatalf("根拠 = %#v", candidates[0].EvidenceCodes())
	}
}

func Test包含誤記候補の抑制は同一Spanの衝突と別位置の候補を保つ(
	t *testing.T,
) {
	t.Parallel()

	values := []lawTarget{
		{
			startByte: 0,
			endByte:   24,
			lawID:     "outer-a",
			evidence:  legalquery.EvidenceOfficialAlias,
			typo:      true,
		},
		{
			startByte: 0,
			endByte:   24,
			lawID:     "outer-b",
			evidence:  legalquery.EvidenceOfficialAlias,
			typo:      true,
		},
		{
			startByte: 15,
			endByte:   24,
			lawID:     "contained-typo",
			evidence:  legalquery.EvidenceOfficialAlias,
			typo:      true,
		},
		{
			startByte: 15,
			endByte:   24,
			lawID:     "contained-exact",
			evidence:  legalquery.EvidenceOfficialAlias,
		},
		{
			startByte: 30,
			endByte:   42,
			lawID:     "separate",
			evidence:  legalquery.EvidenceOfficialAlias,
			typo:      true,
		},
	}

	got := suppressContainedTypoLawTargets(values)
	gotIDs := make([]string, 0, len(got))
	for _, target := range got {
		gotIDs = append(gotIDs, target.lawID)
	}
	wantIDs := []string{
		"outer-a",
		"outer-b",
		"contained-exact",
		"separate",
	}
	if !slices.Equal(gotIDs, wantIDs) {
		t.Fatalf("抑制後の law IDs = %#v, want %#v", gotIDs, wantIDs)
	}
}

func Test包含誤記候補の抑制は内側の複数法令候補を消さない(
	t *testing.T,
) {
	t.Parallel()

	values := []lawTarget{
		{
			startByte: 0,
			endByte:   30,
			lawID:     "outer",
			evidence:  legalquery.EvidenceOfficialAlias,
			typo:      true,
		},
		{
			startByte: 0,
			endByte:   9,
			lawID:     "child-a",
			evidence:  legalquery.EvidenceOfficialAlias,
			typo:      true,
		},
		{
			startByte: 18,
			endByte:   30,
			lawID:     "child-b",
			evidence:  legalquery.EvidenceOfficialAlias,
			typo:      true,
		},
		{
			startByte: 27,
			endByte:   36,
			lawID:     "partial-overlap",
			evidence:  legalquery.EvidenceOfficialAlias,
			typo:      true,
		},
	}

	got := suppressContainedTypoLawTargets(values)
	gotIDs := make([]string, 0, len(got))
	for _, target := range got {
		gotIDs = append(gotIDs, target.lawID)
	}
	wantIDs := []string{
		"outer",
		"child-a",
		"child-b",
		"partial-overlap",
	}
	if !slices.Equal(gotIDs, wantIDs) {
		t.Fatalf("抑制後の law IDs = %#v, want %#v", gotIDs, wantIDs)
	}
}
