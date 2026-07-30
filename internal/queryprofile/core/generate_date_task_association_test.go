package core

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestProfileは複数日付を検索基準日と更新日へ位置結合する(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name       string
		query      string
		searchDate string
		updateDate string
	}{
		{
			name:       "同じ日付を二つの明示taskへ保持する",
			query:      "2024年4月1日時点の民法を検索し、2024年4月1日の法令更新一覧も取得して",
			searchDate: "2024-04-01",
			updateDate: "2024-04-01",
		},
		{
			name:       "異なる日付を直後の明示taskへ結合する",
			query:      "2024年4月1日時点の民法を検索し、2024年4月2日の法令更新一覧も取得して",
			searchDate: "2024-04-01",
			updateDate: "2024-04-02",
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
			if len(steps) != 2 ||
				steps[0].InputKind() != legalquery.InputKindLawSearch ||
				steps[1].InputKind() != legalquery.InputKindLawUpdates {
				t.Fatalf("SOT-MODEL-025: step 順 = %#v", steps)
			}

			search, ok := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
			if !ok {
				t.Fatalf("法令検索入力 = %T", steps[0].LogicalInput())
			}
			asOf, exists := search.AsOf()
			if !exists || asOf.String() != test.searchDate {
				t.Fatalf(
					"法令検索基準日 = %q, exists=%t, want %q",
					asOf.String(),
					exists,
					test.searchDate,
				)
			}

			update, ok :=
				steps[1].LogicalInput().(legalquery.LawUpdateListIntentV1)
			if !ok || update.Date().String() != test.updateDate {
				t.Fatalf(
					"法令更新日 = %#v, want %q",
					steps[1].LogicalInput(),
					test.updateDate,
				)
			}
		})
	}
}
