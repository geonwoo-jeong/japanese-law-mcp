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

func TestProfileは後続法令resourceで引用内の裁判例誤記抑止を解除しない(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"公表裁判例で「著作権 演奏権」を検索し、民法の法律も検索する",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("SOT-ARCH-021: candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindLawSearch {
		t.Fatalf("SOT-ARCH-021: steps = %#v", steps)
	}
	law, ok := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok || law.Query() != "民法" {
		t.Fatalf("SOT-ARCH-021: law input = %#v", steps[0].LogicalInput())
	}
}
