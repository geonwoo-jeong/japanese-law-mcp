package searchquery

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestResolveDirectMatchesSharesIndexesWithoutCallingAnalyzer(t *testing.T) {
	t.Parallel()
	entries := []EntryValues{
		{ResourceID: "civil", Canonical: "民事訴訟法", Terms: []string{"民訴法"}},
		{ResourceID: "cost", Canonical: "民事訴訟費用法", Terms: []string{"民訴法"}},
		{ResourceID: "labor", Canonical: "労働契約法", Terms: []string{"労契法"}},
	}
	wantError := errors.New("共有しても既存の解析器を変更してはなりません")
	shared := mustResolver(t, entries, analyzerStub{err: wantError})
	originalDirect := mustResolver(t, entries, analyzerStub{})
	for _, query := range []string{
		"労働契約法", "労 契 法", "労契去", "民訴法", "民訴去", "未知語", "法",
	} {
		got, err := shared.ResolveDirectMatches(context.Background(), query)
		want, wantErr := originalDirect.ResolveMatches(context.Background(), query)
		if err != nil || wantErr != nil || !slices.Equal(got, want) {
			t.Fatalf("SOT-ARCH-030: %q の直接照合が変化しました: %#v / %#v (%v, %v)", query, got, want, err, wantErr)
		}
	}
	if _, err := shared.ResolveMatches(context.Background(), "辞書外の入力"); !errors.Is(err, wantError) {
		t.Fatalf("SOT-ARCH-021: 元の解析器を変更しました: %v", err)
	}
}

func TestResolveDirectMatchesKeepsValidationAndCancellation(t *testing.T) {
	t.Parallel()
	resolver := mustResolver(t, []EntryValues{{ResourceID: "law", Canonical: "試験法"}}, analyzerStub{})
	var nilContext context.Context
	if _, err := resolver.ResolveDirectMatches(nilContext, "試験法"); err == nil {
		t.Fatal("SOT-ARCH-030: nil context を受理しました")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := resolver.ResolveDirectMatches(ctx, "試験法"); !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-ARCH-030: 取消結果が異なります: %v", err)
	}
	for _, query := range []string{"", string([]byte{0xff})} {
		if _, err := resolver.ResolveDirectMatches(context.Background(), query); err == nil {
			t.Fatal("SOT-ARCH-030: 不正な入力を受理しました")
		}
	}
	for _, invalid := range []*Resolver{nil, {}} {
		if _, err := invalid.ResolveDirectMatches(context.Background(), "試験法"); err == nil {
			t.Fatal("SOT-ARCH-030: 未初期化の索引を受理しました")
		}
	}
}
