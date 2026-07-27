package main

import (
	"context"
	"testing"
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
		query string
		want  string
	}{
		{
			query: "個情法",
			want:  "個人情報の保護に関する法律",
		},
		{
			query: "個情法について教えてください",
			want:  "個人情報の保護に関する法律",
		},
		{
			query: "道路交法通",
			want:  "道路交通法",
		},
	} {
		got, resolved, resolveErr := resolver.Resolve(
			context.Background(),
			testCase.query,
		)
		if resolveErr != nil {
			t.Fatalf(
				"SOT-IF-049: Resolve(%q) のエラー = %v",
				testCase.query,
				resolveErr,
			)
		}
		if !resolved || got != testCase.want {
			t.Fatalf(
				"SOT-IF-049: Resolve(%q) = %q, %t",
				testCase.query,
				got,
				resolved,
			)
		}
	}
}
