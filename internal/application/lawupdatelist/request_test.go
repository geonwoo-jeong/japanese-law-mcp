package lawupdatelist_test

import (
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestRequestConstants(t *testing.T) {
	t.Parallel()

	if lawupdatelist.CapabilityID != "law.update.list" ||
		lawupdatelist.MajorVersion != 1 {
		t.Fatal("SOT-IF-034: capability 定数が契約と一致しない")
	}
}

func TestNewRequestAcceptsAnyValidCommonDate(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"2020-11-23", "2020-11-24", "2999-12-31"} {
		value := value
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			date, err := model.NewDate(value)
			if err != nil {
				t.Fatalf("試験用 Date を作成できない: %v", err)
			}
			request, err := lawupdatelist.NewRequest(lawupdatelist.RequestValues{
				Date: date,
			})
			if err != nil {
				t.Fatalf("SOT-IF-034: 共通 Date %s を拒否した: %v", value, err)
			}
			if request.Date() != date {
				t.Fatalf("SOT-IF-034: Date() = %q", request.Date().String())
			}
			if err := request.Validate(); err != nil {
				t.Fatalf("SOT-IF-034: Validate() のエラー = %v", err)
			}
		})
	}
}

func TestNewRequestRejectsInvalidDate(t *testing.T) {
	t.Parallel()

	if _, err := lawupdatelist.NewRequest(lawupdatelist.RequestValues{}); err == nil {
		t.Fatal("SOT-IF-034: Date のゼロ値を受理した")
	}
}

func TestRequestRejectsDirectJSONDecode(t *testing.T) {
	t.Parallel()

	var request lawupdatelist.Request
	if err := json.Unmarshal([]byte(`{"date":"2026-07-26"}`), &request); err == nil {
		t.Fatal("SOT-IF-034: Request を JSON から直接復元できた")
	}
}
