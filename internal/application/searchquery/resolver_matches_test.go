package searchquery

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestResolveMatchesPreservesAmbiguousExactTargetsInStableOrder(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t, []EntryValues{
		{
			ResourceID: "agency",
			Canonical:  "独立行政法人等の保有する情報の公開に関する法律",
			Terms:      []string{"開示法"},
		},
		{
			ResourceID: "administrative",
			Canonical:  "行政機関の保有する情報の公開に関する法律",
			Terms:      []string{"開示法"},
		},
	}, analyzerStub{})

	want := []Match{
		{
			resourceID: "administrative",
			canonical:  "行政機関の保有する情報の公開に関する法律",
			kind:       MatchKindExact,
		},
		{
			resourceID: "agency",
			canonical:  "独立行政法人等の保有する情報の公開に関する法律",
			kind:       MatchKindExact,
		},
	}
	got, err := resolver.ResolveMatches(context.Background(), "開示法")
	if err != nil {
		t.Fatalf("SOT-ARCH-021: ResolveMatches() のエラー = %v", err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("SOT-ENG-022: matches = %#v, want %#v", got, want)
	}
	if got[0].ResourceID() != "administrative" ||
		got[0].Canonical() != "行政機関の保有する情報の公開に関する法律" ||
		got[0].Kind() != MatchKindExact {
		t.Fatalf("SOT-ARCH-021: match getters = %#v", got[0])
	}

	got[0] = Match{}
	reloaded, err := resolver.ResolveMatches(context.Background(), "開示法")
	if err != nil {
		t.Fatalf("SOT-ARCH-021: ResolveMatches() の再読込エラー = %v", err)
	}
	if !slices.Equal(reloaded, want) {
		t.Fatalf("SOT-ARCH-021: resolver が変更されました: %#v", reloaded)
	}
}

func TestResolveMatchesDistinguishesProviderIndependentMatchKinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entries  []EntryValues
		analyzer analyzerStub
		query    string
		want     Match
	}{
		{
			name: "comparison normalized",
			entries: []EntryValues{{
				ResourceID: "privacy",
				Canonical:  "個人情報の保護に関する法律",
				Terms:      []string{"APPI"},
			}},
			query: "ＡＰＰＩ",
			want: Match{
				resourceID: "privacy",
				canonical:  "個人情報の保護に関する法律",
				kind:       MatchKindComparisonNormalized,
			},
		},
		{
			name: "kagome registered term",
			entries: []EntryValues{{
				ResourceID: "antimonopoly",
				Canonical:  "私的独占の禁止及び公正取引の確保に関する法律",
				Terms:      []string{"独禁法"},
			}},
			analyzer: analyzerStub{terms: []string{"独禁法"}},
			query:    "独禁法の正式な法令を検索してください。",
			want: Match{
				resourceID: "antimonopoly",
				canonical:  "私的独占の禁止及び公正取引の確保に関する法律",
				kind:       MatchKindRegisteredTerm,
			},
		},
		{
			name: "unique typo correction",
			entries: []EntryValues{{
				ResourceID: "labor-contract",
				Canonical:  "労働契約法",
			}},
			query: "労契約法",
			want: Match{
				resourceID: "labor-contract",
				canonical:  "労働契約法",
				kind:       MatchKindUniqueTypoCorrection,
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			resolver := mustResolver(t, testCase.entries, testCase.analyzer)
			got, err := resolver.ResolveMatches(
				context.Background(),
				testCase.query,
			)
			if err != nil {
				t.Fatalf("SOT-ARCH-021: ResolveMatches() のエラー = %v", err)
			}
			if !slices.Equal(got, []Match{testCase.want}) {
				t.Fatalf("SOT-ARCH-021: matches = %#v", got)
			}
		})
	}
}

func TestResolveMatchesDoesNotExposeAmbiguousTypoCorrection(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t, []EntryValues{
		{ResourceID: "first", Canonical: "情報保護法"},
		{ResourceID: "second", Canonical: "情報保障法"},
	}, analyzerStub{})

	got, err := resolver.ResolveMatches(context.Background(), "情報保法")
	if err != nil {
		t.Fatalf("SOT-ARCH-021: ResolveMatches() のエラー = %v", err)
	}
	if got != nil {
		t.Fatalf("SOT-ARCH-021: 曖昧な誤記候補 = %#v", got)
	}
}

func TestResolveMatchesRejectsInvalidInputAndPropagatesContext(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t, []EntryValues{{
		ResourceID: "privacy",
		Canonical:  "個人情報保護法",
	}}, analyzerStub{})

	var nilContext context.Context
	if _, err := resolver.ResolveMatches(
		nilContext,
		"個人情報保護法",
	); err == nil {
		t.Fatal("SOT-ARCH-021: nil context を受理しました")
	}
	if _, err := resolver.ResolveMatches(context.Background(), ""); err == nil {
		t.Fatal("SOT-ARCH-021: 空の検索語を受理しました")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := resolver.ResolveMatches(ctx, "個人情報保護法")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-ARCH-021: context error = %v", err)
	}
}
