package listlawupdates

import (
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestRequestKeepsValidatedDateAndAppliesDefaultLimit(t *testing.T) {
	t.Parallel()

	date := mustListLawUpdatesDate(t, "2026-07-26")
	request, err := NewRequest(RequestValues{Date: date})
	if err != nil {
		t.Fatalf("SOT-IF-076: NewRequest() のエラー = %v", err)
	}
	if request.Date() != date || request.Limit() != DefaultLimit {
		t.Fatalf(
			"SOT-IF-076: Date() = %q, Limit() = %d",
			request.Date().String(),
			request.Limit(),
		)
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("SOT-IF-076: Validate() のエラー = %v", err)
	}
}

func TestRequestAcceptsLimitBounds(t *testing.T) {
	t.Parallel()

	date := mustListLawUpdatesDate(t, "2026-07-26")
	for _, limit := range []int{1, 208, MaxLimit} {
		limit := limit
		request, err := NewRequest(RequestValues{Date: date, Limit: &limit})
		if err != nil {
			t.Fatalf("SOT-IF-076: limit %d のエラー = %v", limit, err)
		}
		if request.Limit() != limit {
			t.Fatalf("SOT-IF-076: Limit() = %d, want %d", request.Limit(), limit)
		}
	}
}

func TestRequestRejectsLimitOutsideBounds(t *testing.T) {
	t.Parallel()

	date := mustListLawUpdatesDate(t, "2026-07-26")
	for _, limit := range []int{0, -1, MaxLimit + 1} {
		limit := limit
		if _, err := NewRequest(RequestValues{Date: date, Limit: &limit}); err == nil {
			t.Fatalf("SOT-IF-076: limit %d を受理した", limit)
		}
	}
}

func TestRequestRejectsZeroDateAndDirectJSONDecode(t *testing.T) {
	t.Parallel()

	if _, err := NewRequest(RequestValues{}); err == nil {
		t.Fatal("SOT-IF-076: date のゼロ値を受理した")
	}

	var request Request
	if err := json.Unmarshal([]byte(`{"date":"2026-07-26"}`), &request); err == nil {
		t.Fatal("SOT-IF-076: Request を JSON から直接復元できた")
	}
}

func mustListLawUpdatesDate(t *testing.T, value string) model.Date {
	t.Helper()

	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("試験用日付を作成できません: %v", err)
	}
	return date
}
