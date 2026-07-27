package searchquery

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type analyzerStub struct {
	terms []string
	err   error
}

func (a analyzerStub) RegisteredTerms(
	ctx context.Context,
	_ string,
) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]string(nil), a.terms...), a.err
}

func TestResolverUsesExactAndComparisonNormalizedTerms(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t, []EntryValues{
		{
			ResourceID: "privacy",
			Canonical:  "個人情報の保護に関する法律",
			Terms:      []string{"個人情報保護法", "個情法"},
		},
		{
			ResourceID: "number",
			Canonical:  "行政手続における特定の個人を識別するための番号の利用等に関する法律",
			Terms:      []string{"マイナンバー法", "まいなんばーほう"},
		},
	}, analyzerStub{})

	for _, testCase := range []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "完全一致",
			query: "個情法",
			want:  "個人情報の保護に関する法律",
		},
		{
			name:  "NFKCと句読点",
			query: "「 ﾏｲﾅﾝﾊﾞｰ法 」",
			want:  "行政手続における特定の個人を識別するための番号の利用等に関する法律",
		},
		{
			name:  "かな",
			query: "マイナンバーホウ",
			want:  "行政手続における特定の個人を識別するための番号の利用等に関する法律",
		},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, resolved, err := resolver.Resolve(
				context.Background(),
				testCase.query,
			)
			if err != nil {
				t.Fatalf("SOT-IF-049: Resolve() のエラー = %v", err)
			}
			if !resolved || got != testCase.want {
				t.Fatalf(
					"SOT-IF-049: Resolve(%q) = %q, %t",
					testCase.query,
					got,
					resolved,
				)
			}
		})
	}
}

func TestResolverUsesKagomeRegisteredTermFromNaturalLanguage(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t, []EntryValues{{
		ResourceID: "privacy",
		Canonical:  "個人情報の保護に関する法律",
		Terms:      []string{"個情法"},
	}}, analyzerStub{terms: []string{"個情法"}})

	got, resolved, err := resolver.Resolve(
		context.Background(),
		"個情法について教えてください",
	)
	if err != nil {
		t.Fatalf("SOT-IF-049: Resolve() のエラー = %v", err)
	}
	if !resolved || got != "個人情報の保護に関する法律" {
		t.Fatalf("SOT-IF-049: Resolve() = %q, %t", got, resolved)
	}
}

func TestResolverCorrectsBoundedDamerauLevenshteinTypos(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t, []EntryValues{{
		ResourceID: "privacy",
		Canonical:  "個人情報保護法",
	}}, analyzerStub{})

	for _, query := range []string{
		"個情報保護法",
		"個人人情報保護法",
		"個人情報保護方",
		"個人情報保法護",
	} {
		query := query
		t.Run(query, func(t *testing.T) {
			t.Parallel()
			got, resolved, err := resolver.Resolve(context.Background(), query)
			if err != nil {
				t.Fatalf("SOT-IF-049: Resolve() のエラー = %v", err)
			}
			if !resolved || got != "個人情報保護法" {
				t.Fatalf(
					"SOT-IF-049: Resolve(%q) = %q, %t",
					query,
					got,
					resolved,
				)
			}
		})
	}
}

func TestResolverDoesNotGuessAmbiguousOrUnsafeTerms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entries  []EntryValues
		analyzer analyzerStub
		query    string
	}{
		{
			name: "短い誤記",
			entries: []EntryValues{
				{ResourceID: "civil", Canonical: "民法"},
			},
			query: "眠法",
		},
		{
			name: "同じ略称が複数法令",
			entries: []EntryValues{
				{
					ResourceID: "housing",
					Canonical:  "住宅の品質確保の促進等に関する法律",
					Terms:      []string{"品確法"},
				},
				{
					ResourceID: "fuel",
					Canonical:  "揮発油等の品質の確保等に関する法律",
					Terms:      []string{"品確法"},
				},
			},
			query: "品確法",
		},
		{
			name: "自然文に複数法令",
			entries: []EntryValues{
				{ResourceID: "civil", Canonical: "民法"},
				{ResourceID: "penal", Canonical: "刑法"},
			},
			analyzer: analyzerStub{terms: []string{"民法", "刑法"}},
			query:    "民法と刑法の違い",
		},
		{
			name: "最小距離が異なる法令で同率",
			entries: []EntryValues{
				{ResourceID: "first", Canonical: "情報保護法"},
				{ResourceID: "second", Canonical: "情報保障法"},
			},
			query: "情報保法",
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			resolver := mustResolver(t, testCase.entries, testCase.analyzer)
			got, resolved, err := resolver.Resolve(
				context.Background(),
				testCase.query,
			)
			if err != nil {
				t.Fatalf("SOT-IF-049: Resolve() のエラー = %v", err)
			}
			if resolved || got != "" {
				t.Fatalf(
					"SOT-IF-049: 曖昧な Resolve() = %q, %t",
					got,
					resolved,
				)
			}
		})
	}
}

func TestResolverPreservesImmutableInputAndSupportsConcurrentUse(t *testing.T) {
	t.Parallel()

	values := []EntryValues{{
		ResourceID: "traffic",
		Canonical:  "道路交通法",
		Terms:      []string{"道交法"},
	}}
	resolver := mustResolver(t, values, analyzerStub{})
	values[0].Canonical = "変更後"
	values[0].Terms[0] = "変更後"

	const goroutines = 32
	var waitGroup sync.WaitGroup
	errorsChannel := make(chan error, goroutines)
	for range goroutines {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			got, resolved, err := resolver.Resolve(
				context.Background(),
				"道交法",
			)
			if err != nil {
				errorsChannel <- err
				return
			}
			if !resolved || got != "道路交通法" {
				errorsChannel <- errors.New("不変な辞書から解決されませんでした")
			}
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Error(err)
	}
}

func TestResolverRejectsInvalidInputAndPropagatesAnalyzerFailure(t *testing.T) {
	t.Parallel()

	if _, err := NewResolver(nil, analyzerStub{}); err == nil {
		t.Fatal("SOT-ARCH-021: 空の entry を受理しました")
	}
	if _, err := NewResolver([]EntryValues{{
		ResourceID: "id",
		Canonical:  "法令",
	}}, nil); err == nil {
		t.Fatal("SOT-ARCH-021: nil analyzer を受理しました")
	}
	if _, err := NewResolver([]EntryValues{{
		ResourceID: "id",
		Canonical:  "「 」",
	}}, analyzerStub{}); err == nil {
		t.Fatal("SOT-ARCH-021: 比較できない canonical を受理しました")
	}

	wantErr := errors.New("形態素解析失敗")
	resolver := mustResolver(t, []EntryValues{{
		ResourceID: "privacy",
		Canonical:  "個人情報保護法",
	}}, analyzerStub{err: wantErr})
	_, _, err := resolver.Resolve(context.Background(), "未知の自然文です")
	if !errors.Is(err, wantErr) {
		t.Fatalf("SOT-ARCH-021: Resolve() のエラー = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = resolver.Resolve(ctx, "個人情報保護法")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-ENG-010: Resolve() のエラー = %v", err)
	}
}

func mustResolver(
	t *testing.T,
	entries []EntryValues,
	analyzer Analyzer,
) *Resolver {
	t.Helper()
	resolver, err := NewResolver(entries, analyzer)
	if err != nil {
		t.Fatalf("NewResolver() のエラー = %v", err)
	}
	return resolver
}
