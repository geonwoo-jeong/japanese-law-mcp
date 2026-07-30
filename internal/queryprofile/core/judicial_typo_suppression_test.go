package core

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestProfileは引用内の法令誤記補正を裁判例resource文脈で候補化しない(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"公表裁判例で「著作権 演奏権」を検索する",
		nil,
	)
	if len(generation.Candidates()) != 0 {
		t.Fatalf(
			"SOT-ARCH-021/SOT-ARCH-025: candidates = %#v",
			generation.Candidates(),
		)
	}
	if !slices.Contains(
		generation.Signals(),
		legalquery.CandidateSignalReservedPackRequest,
	) {
		t.Fatalf("signals = %#v", generation.Signals())
	}
}
