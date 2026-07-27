package legalquery

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLegalQueryPagePreviewAcceptsOptionalFieldsAndIsImmutable(t *testing.T) {
	t.Parallel()

	hasMore := true
	totalCount := 3
	page, err := NewLegalQueryPagePreview(LegalQueryPagePreviewValues{
		ReturnedCount: 1,
		HasMore:       &hasMore,
		TotalCount:    &totalCount,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-024: page preview を作成できません: %v", err)
	}
	hasMore = false
	totalCount = 1
	gotHasMore, hasHasMore := page.HasMore()
	gotTotal, hasTotal := page.TotalCount()
	gotRelation, hasRelation := page.TotalRelation()
	if page.ReturnedCount() != 1 ||
		!hasHasMore || !gotHasMore ||
		!hasTotal || gotTotal != 3 ||
		!hasRelation || gotRelation != model.TotalRelationExact {
		t.Fatalf("SOT-MODEL-024: page preview = %#v", page)
	}

	unknown, err := NewLegalQueryPagePreview(LegalQueryPagePreviewValues{
		ReturnedCount: 0,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-024: optional field 省略を拒否しました: %v", err)
	}
	if _, exists := unknown.HasMore(); exists {
		t.Fatal("SOT-MODEL-024: 判定不能な hasMore を公開しました")
	}
	if _, exists := unknown.TotalCount(); exists {
		t.Fatal("SOT-MODEL-024: 未指定の totalCount を公開しました")
	}

	lowerBoundCount := 1
	equalLowerBound, err := NewLegalQueryPagePreview(
		LegalQueryPagePreviewValues{
			ReturnedCount: 1,
			TotalCount:    &lowerBoundCount,
			TotalRelation: model.TotalRelationLowerBound,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: 同数の lower_bound を拒否しました: %v", err)
	}
	if _, exists := equalLowerBound.HasMore(); exists {
		t.Fatal("SOT-MODEL-024: 判定不能な lower_bound の hasMore を公開しました")
	}
}

func TestLegalQueryPagePreviewEnforcesCrossConstraints(t *testing.T) {
	t.Parallel()

	trueValue := true
	falseValue := false
	one := 1
	two := 2
	tests := map[string]LegalQueryPagePreviewValues{
		"returnedCount が負": {
			ReturnedCount: -1,
		},
		"totalCount が負": {
			ReturnedCount: 0,
			TotalCount:    intPointer(-1),
			TotalRelation: model.TotalRelationLowerBound,
		},
		"totalCount が小さい": {
			ReturnedCount: 2,
			TotalCount:    &one,
			TotalRelation: model.TotalRelationLowerBound,
		},
		"relation だけ": {
			ReturnedCount: 0,
			TotalRelation: model.TotalRelationExact,
		},
		"totalCount に relation なし": {
			ReturnedCount: 1,
			TotalCount:    &two,
		},
		"未知の relation": {
			ReturnedCount: 1,
			TotalCount:    &two,
			TotalRelation: model.TotalRelation("estimated"),
		},
		"exact に hasMore なし": {
			ReturnedCount: 1,
			TotalCount:    &two,
			TotalRelation: model.TotalRelationExact,
		},
		"exact の残件と false": {
			ReturnedCount: 1,
			HasMore:       &falseValue,
			TotalCount:    &two,
			TotalRelation: model.TotalRelationExact,
		},
		"exact の完了と true": {
			ReturnedCount: 1,
			HasMore:       &trueValue,
			TotalCount:    &one,
			TotalRelation: model.TotalRelationExact,
		},
		"lower_bound の残件に hasMore なし": {
			ReturnedCount: 1,
			TotalCount:    &two,
			TotalRelation: model.TotalRelationLowerBound,
		},
		"lower_bound の残件と false": {
			ReturnedCount: 1,
			HasMore:       &falseValue,
			TotalCount:    &two,
			TotalRelation: model.TotalRelationLowerBound,
		},
	}
	for name, values := range tests {
		if _, err := NewLegalQueryPagePreview(values); err == nil {
			t.Fatalf("SOT-MODEL-024: %sを受理しました", name)
		}
	}
}

func TestLegalQueryPagePreviewJSONNeverExposesContinuationPosition(t *testing.T) {
	t.Parallel()

	page := resultTestPage(t, 1, true, 2, model.TotalRelationExact)
	encoded, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: JSON に変換できません: %v", err)
	}
	for _, forbidden := range []string{"nextToken", "nextOffset", "continuationToken"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("SOT-MODEL-024: %s を公開しました: %s", forbidden, encoded)
		}
	}
	if string(encoded) != `{"returnedCount":1,"hasMore":true,"totalCount":2,"totalRelation":"exact"}` {
		t.Fatalf("SOT-MODEL-024: JSON = %s", encoded)
	}
	if err := json.Unmarshal(encoded, &LegalQueryPagePreview{}); err == nil {
		t.Fatal("SOT-MODEL-024: page preview を JSON から直接復元できました")
	}
}

func resultTestPage(
	t *testing.T,
	returnedCount int,
	hasMore bool,
	totalCount int,
	relation model.TotalRelation,
) LegalQueryPagePreview {
	t.Helper()
	page, err := NewLegalQueryPagePreview(LegalQueryPagePreviewValues{
		ReturnedCount: returnedCount,
		HasMore:       &hasMore,
		TotalCount:    &totalCount,
		TotalRelation: relation,
	})
	if err != nil {
		t.Fatalf("試験用 page preview を作成できません: %v", err)
	}
	return page
}

func intPointer(value int) *int {
	return &value
}
