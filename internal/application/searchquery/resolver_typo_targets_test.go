package searchquery

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestResolveMatchesPreservesTargetsOfOneUniqueCorrectedTerm(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t, []EntryValues{
		{
			ResourceID: "civil-procedure",
			Canonical:  "民事訴訟法",
			Terms:      []string{"民訴法"},
		},
		{
			ResourceID: "civil-procedure-costs",
			Canonical:  "民事訴訟費用等に関する法律",
			Terms:      []string{"民訴法"},
		},
	}, analyzerStub{})

	got, err := resolver.ResolveMatches(context.Background(), "民訴去")
	if err != nil {
		t.Fatalf("SOT-MODEL-025: ResolveMatches() のエラー = %v", err)
	}
	want := []Match{
		{
			resourceID: "civil-procedure",
			canonical:  "民事訴訟法",
			kind:       MatchKindUniqueTypoCorrection,
		},
		{
			resourceID: "civil-procedure-costs",
			canonical:  "民事訴訟費用等に関する法律",
			kind:       MatchKindUniqueTypoCorrection,
		},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("SOT-MODEL-025: matches = %#v, want %#v", got, want)
	}

	canonical, resolved, err := resolver.Resolve(
		context.Background(),
		"民訴去",
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Resolve() のエラー = %v", err)
	}
	if resolved || canonical != "" {
		t.Fatalf(
			"SOT-MODEL-025: 複数対象を一件へ縮約しました: %q, %t",
			canonical,
			resolved,
		)
	}
}

func TestResolveUniqueTypoMatchesDoesNotInvokeAnalyzer(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t, []EntryValues{{
		ResourceID: "labor-contract",
		Canonical:  "労働契約法",
	}}, analyzerStub{err: errors.New("呼び出してはならない解析器")})

	got, err := resolver.ResolveUniqueTypoMatches(
		context.Background(),
		"労契約法",
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: ResolveUniqueTypoMatches() のエラー = %v", err)
	}
	want := []Match{{
		resourceID: "labor-contract",
		canonical:  "労働契約法",
		kind:       MatchKindUniqueTypoCorrection,
	}}
	if !slices.Equal(got, want) {
		t.Fatalf("SOT-ARCH-021: typo matches = %#v, want %#v", got, want)
	}

	exact, err := resolver.ResolveUniqueTypoMatches(
		context.Background(),
		"労働契約法",
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 完全一致の判定エラー = %v", err)
	}
	if exact != nil {
		t.Fatalf("SOT-ARCH-021: 完全一致を誤記として返しました: %#v", exact)
	}
}
