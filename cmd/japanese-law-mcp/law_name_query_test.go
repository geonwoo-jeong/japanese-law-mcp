package main

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawtarget"
)

func TestDefaultLawNameQueryResolverCombinesEmbeddedLexiconAndKagome(
	t *testing.T,
) {
	t.Parallel()

	resolver, err := newLawNameQueryResolver()
	if err != nil {
		t.Fatalf("SOT-ARCH-021: resolver の初期化エラー = %v", err)
	}
	for _, testCase := range []struct {
		query     string
		wantTitle string
		wantKind  lawtarget.MatchKind
	}{
		{
			query:     "個情法",
			wantTitle: "個人情報の保護に関する法律",
			wantKind:  lawtarget.MatchKindExact,
		},
		{
			query:     "個情法について教えてください",
			wantTitle: "個人情報の保護に関する法律",
			wantKind:  lawtarget.MatchKindRegisteredTerm,
		},
		{
			query:     "著権作法",
			wantTitle: "著作権法",
			wantKind:  lawtarget.MatchKindUniqueTypoCorrection,
		},
	} {
		got, resolved, resolveErr := resolver.Resolve(
			context.Background(),
			testCase.query,
		)
		if resolveErr != nil {
			t.Fatalf(
				"SOT-ARCH-030/SOT-IF-053: Resolve(%q) のエラー = %v",
				testCase.query,
				resolveErr,
			)
		}
		if !resolved ||
			got.OfficialTitle() != testCase.wantTitle ||
			got.MatchKind() != testCase.wantKind {
			t.Fatalf(
				"SOT-ARCH-030/SOT-IF-053: Resolve(%q) = title:%q kind:%q resolved:%t",
				testCase.query,
				got.OfficialTitle(),
				got.MatchKind(),
				resolved,
			)
		}
	}
}
