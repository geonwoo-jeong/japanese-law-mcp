package listlawupdates

import (
	"encoding/json"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestRequestKeepsValidatedDate(t *testing.T) {
	t.Parallel()

	date := mustListLawUpdatesDate(t, "2026-07-26")
	request, err := NewRequest(RequestValues{Date: date})
	if err != nil {
		t.Fatalf("SOT-IF-038: NewRequest() のエラー = %v", err)
	}
	if request.Date() != date {
		t.Fatalf("SOT-IF-038: Date() = %q", request.Date().String())
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("SOT-IF-038: Validate() のエラー = %v", err)
	}
}

func TestRequestRejectsZeroDateAndDirectJSONDecode(t *testing.T) {
	t.Parallel()

	if _, err := NewRequest(RequestValues{}); err == nil {
		t.Fatal("SOT-IF-038: date のゼロ値を受理した")
	}

	var request Request
	if err := json.Unmarshal([]byte(`{"date":"2026-07-26"}`), &request); err == nil {
		t.Fatal("SOT-IF-038: Request を JSON から直接復元できた")
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
