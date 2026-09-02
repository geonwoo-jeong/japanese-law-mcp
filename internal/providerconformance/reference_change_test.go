package providerconformance

import (
	"slices"
	"strings"
	"testing"
)

func TestClassifyInterfaceSOTReferenceOnlyChangeは同位置の置換を返す(t *testing.T) {
	t.Parallel()

	before := validProviderYAML(
		"test-provider",
		validRowYAML("test-provider", "law.search", "GET /laws", "laws-json"),
	)
	after := []byte(strings.NewReplacer(
		"      - SOT-IF-004", "      - SOT-IF-077",
		"      - SOT-IF-022", "      - SOT-IF-099",
	).Replace(string(before)))

	change, onlyReferences, err := ClassifyInterfaceSOTReferenceOnlyChange(
		testRepositoryRoot(t),
		"test-provider",
		before,
		after,
	)
	if err != nil {
		t.Fatalf("SOT 参照だけの変更を判定できません: %v", err)
	}
	if !onlyReferences {
		t.Fatal("SOT 参照だけの変更を false と判定しました")
	}

	want := []InterfaceSOTReferenceReplacement{
		{
			RowIndex:       0,
			ReferenceIndex: 0,
			CapabilityID:   "law.search",
			MajorVersion:   1,
			Operation:      "GET /laws",
			PreviousSOTID:  "SOT-IF-004",
			CurrentSOTID:   "SOT-IF-077",
		},
		{
			RowIndex:       0,
			ReferenceIndex: 2,
			CapabilityID:   "law.search",
			MajorVersion:   1,
			Operation:      "GET /laws",
			PreviousSOTID:  "SOT-IF-022",
			CurrentSOTID:   "SOT-IF-099",
		},
	}
	if !slices.Equal(change.Replacements, want) {
		t.Fatalf("置換 = %#v、期待値は %#v です", change.Replacements, want)
	}
}

func TestClassifyInterfaceSOTReferenceOnlyChangeは参照以外の差分を除外する(t *testing.T) {
	t.Parallel()

	baseRow := validRowYAML("test-provider", "law.search", "GET /laws", "laws-json")
	base := validProviderYAML("test-provider", baseRow)
	rowWithFourReferences := strings.Replace(
		baseRow,
		"      - SOT-IF-022",
		"      - SOT-IF-022\n      - SOT-IF-099",
		1,
	)
	articleRow := validRowYAML("test-provider", "law.article.read", "GET /law", "law-data-xml")
	twoRows := validProviderYAML("test-provider", articleRow, baseRow)

	tests := map[string]struct {
		before []byte
		after  []byte
	}{
		"formatだけ": {
			before: base,
			after:  []byte(strings.Replace(string(base), "schemaVersion: 1", "schemaVersion: 1\n# 説明", 1)),
		},
		"他field": {
			before: base,
			after:  []byte(strings.Replace(string(base), "budgetKey: laws-json", "budgetKey: changed-json", 1)),
		},
		"row追加": {
			before: base,
			after:  twoRows,
		},
		"row削除": {
			before: twoRows,
			after:  base,
		},
		"参照配列の長さ変更": {
			before: validProviderYAML("test-provider", rowWithFourReferences),
			after:  base,
		},
		"参照配列の並べ替え": {
			before: base,
			after: []byte(strings.Replace(
				string(base),
				"      - SOT-IF-004\n      - SOT-IF-050",
				"      - SOT-IF-050\n      - SOT-IF-004",
				1,
			)),
		},
		"参照配列の一部移動と置換": {
			before: base,
			after: []byte(strings.Replace(
				string(base),
				"      - SOT-IF-004\n      - SOT-IF-050",
				"      - SOT-IF-050\n      - SOT-IF-077",
				1,
			)),
		},
	}

	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			change, onlyReferences, err := ClassifyInterfaceSOTReferenceOnlyChange(
				testRepositoryRoot(t),
				"test-provider",
				test.before,
				test.after,
			)
			if err != nil {
				t.Fatalf("有効な matrix を比較できません: %v", err)
			}
			if onlyReferences {
				t.Fatalf("SOT 参照だけではない変更を受理しました: %#v", change)
			}
			if len(change.Replacements) != 0 {
				t.Fatalf("除外した変更から置換が返されました: %#v", change.Replacements)
			}
		})
	}
}

func TestClassifyInterfaceSOTReferenceOnlyChangeは不正なsnapshotを拒否する(t *testing.T) {
	t.Parallel()

	baseRow := validRowYAML("test-provider", "law.search", "GET /laws", "laws-json")
	base := validProviderYAML("test-provider", baseRow)
	articleRow := validRowYAML("test-provider", "law.article.read", "GET /law", "law-data-xml")
	duplicateTuple := validProviderYAML(
		"test-provider",
		validRowYAML("test-provider", "law.search", "GET /laws", "a-budget"),
		validRowYAML("test-provider", "law.search", "GET /laws", "b-budget"),
	)
	reorderedRows := validProviderYAML("test-provider", baseRow, articleRow)

	tests := map[string]struct {
		before []byte
		after  []byte
	}{
		"UTF-8不正": {
			before: append(append([]byte(nil), base...), 0xff),
			after:  base,
		},
		"key重複": {
			before: base,
			after: []byte(strings.Replace(
				string(base),
				"    capabilityId: law.search",
				"    capabilityId: law.search\n    capabilityId: law.search",
				1,
			)),
		},
		"tuple重複": {
			before: base,
			after:  duplicateTuple,
		},
		"row未整列": {
			before: base,
			after:  reorderedRows,
		},
		"providerId不一致": {
			before: base,
			after:  []byte(strings.Replace(string(base), "providerId: test-provider", "providerId: other-provider", 1)),
		},
	}

	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, _, err := ClassifyInterfaceSOTReferenceOnlyChange(
				testRepositoryRoot(t),
				"test-provider",
				test.before,
				test.after,
			); err == nil {
				t.Fatal("不正な snapshot を受理しました")
			}
		})
	}
}

func TestClassifyInterfaceSOTReferenceOnlyChangeは入力境界を検証する(t *testing.T) {
	t.Parallel()

	base := validProviderYAML(
		"test-provider",
		validRowYAML("test-provider", "law.search", "GET /laws", "laws-json"),
	)
	tests := map[string]struct {
		repository string
		providerID string
	}{
		"repository未指定": {
			repository: "",
			providerID: "test-provider",
		},
		"providerId形式不正": {
			repository: testRepositoryRoot(t),
			providerID: "Test_Provider",
		},
	}

	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, _, err := ClassifyInterfaceSOTReferenceOnlyChange(
				test.repository,
				test.providerID,
				base,
				base,
			); err == nil {
				t.Fatal("不正な入力境界を受理しました")
			}
		})
	}
}
