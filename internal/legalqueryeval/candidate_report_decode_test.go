package legalqueryeval

import (
	"context"
	"os"
	"testing"
)

func TestDecodeStandardReportはBaselinePointerなしでCanonicalReportを復元する(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../testdata/legalquery/baselines/default.json")
	if err != nil {
		t.Fatalf("標準 report fixture を読めません: %v", err)
	}
	report, err := DecodeStandardReport(context.Background(), "../..", raw)
	if err != nil {
		t.Fatalf("canonical report を復元できません: %v", err)
	}
	if err := report.Validate(); err != nil {
		t.Fatalf("復元した report が不正です: %v", err)
	}
	if _, err := DecodeStandardReport(context.Background(), "../..", append([]byte{' '}, raw...)); err == nil {
		t.Fatal("non-canonical report を受理しました")
	}
}
