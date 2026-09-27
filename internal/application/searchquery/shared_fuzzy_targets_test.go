package searchquery

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// SOT-ARCH-021: 正規化の公開順序と fuzzy の登録順序を区別し、共有しても変えない。
func TestSharedFuzzyTargetsPreserveNormalizedAndInsertionOrders(t *testing.T) {
	t.Parallel()

	for _, ordered := range []bool{false, true} {
		entries := []EntryValues{
			{ResourceID: "z-resource", Canonical: "後順位の法律", Terms: []string{"ＡＢＣ法", "abc法"}},
			{ResourceID: "a-resource", Canonical: "先順位の法律", Terms: []string{"abc法"}},
		}
		if ordered {
			slices.Reverse(entries)
		}
		resolver := mustResolver(t, entries, analyzerStub{})
		term := sharedTestFuzzyTerm(t, resolver, "abc法")
		normalized := resolver.normalized["abc法"]
		if shared := &term.targets[0] == &normalized[0]; shared != ordered {
			t.Fatalf("SOT-ARCH-021: fuzzy 行の共有条件 = %t、期待 = %t", shared, ordered)
		}
		wantNormalized := []string{"a-resource", "z-resource"}
		wantFuzzy := []string{entries[0].ResourceID, entries[1].ResourceID}
		assertSharedResolverIDs(t, resolver, "Ａｂｃ法", wantNormalized)
		assertSharedResolverIDs(t, resolver, "abd法", wantFuzzy)
		for _, entry := range entries {
			key := comparisonKey(entry.Canonical)
			canonical := sharedTestFuzzyTerm(t, resolver, key)
			if &canonical.targets[0] != &resolver.normalized[key][0] {
				t.Fatal("SOT-ARCH-021: 単一 target の非公開配列を共有していません")
			}
		}
	}
}

func assertSharedResolverIDs(t *testing.T, resolver *Resolver, query string, want []string) {
	t.Helper()
	ctx := context.Background()
	methods := []func(context.Context, string) ([]Match, error){resolver.ResolveMatches, resolver.ResolveDirectMatches}
	if query == "abd法" {
		methods = append(methods, resolver.ResolveUniqueTypoMatches)
	}
	for _, method := range methods {
		for attempt := range 2 {
			matches, err := method(ctx, query)
			if err != nil {
				t.Fatalf("SOT-ARCH-021: 共有索引の照合に失敗しました: %v", err)
			}
			ids := make([]string, len(matches))
			for index, match := range matches {
				ids[index] = match.ResourceID()
			}
			if !slices.Equal(ids, want) {
				t.Fatalf("SOT-ARCH-021: %q の target 順序 = %#v、期待 = %#v", query, ids, want)
			}
			if attempt == 0 && len(matches) > 0 {
				matches[0] = Match{}
			}
		}
	}
}

func sharedTestFuzzyTerm(t *testing.T, resolver *Resolver, key string) fuzzyTerm {
	t.Helper()
	for _, term := range resolver.fuzzy[len([]rune(key))] {
		if term.value == key {
			return term
		}
	}
	t.Fatalf("SOT-ARCH-021: fuzzy 行 %q がありません", key)
	return fuzzyTerm{}
}

func TestSharedFuzzyTargetsMatchIndependentIndexAndStayImmutable(t *testing.T) {
	t.Parallel()

	entries := []EntryValues{
		{ResourceID: "z", Canonical: "後順位法", Terms: []string{"ＡＢＣ法", "abc法", "仮名表記法", "カナ表記法"}},
		{ResourceID: "a", Canonical: "先順位法", Terms: []string{"abc法", "かな表記法"}},
		{ResourceID: "middle", Canonical: "情報保護法", Terms: []string{"情報保障法"}},
	}
	resolver := mustResolver(t, entries, analyzerStub{})
	reference := *resolver
	reference.fuzzy = independentFuzzyIndex(entries)
	beforeNormalized := cloneSharedTargetMap(resolver.normalized)
	beforeFuzzy := independentFuzzyIndex(entries)
	for _, query := range []string{"Ａｂｃ法", "abd法", "カナ表記去", "情報保法", "全く未登録の照会", "abc法"} {
		got, err := resolver.ResolveMatches(context.Background(), query)
		want, wantErr := reference.ResolveMatches(context.Background(), query)
		if err != nil || wantErr != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("SOT-ARCH-021: %q の結果が独立 fuzzy 索引と一致しません: %#v / %#v / %v / %v", query, got, want, err, wantErr)
		}
		if len(got) > 0 {
			got[0] = Match{}
		}
	}
	if !reflect.DeepEqual(resolver.normalized, beforeNormalized) || !reflect.DeepEqual(resolver.fuzzy, beforeFuzzy) {
		t.Fatal("SOT-ARCH-021: 検索または返却値の変更で内部共有索引が変わりました")
	}
}

// 登録順から独立配列を作る従来方式を、共有候補との比較に用いる。
func independentFuzzyIndex(entries []EntryValues) map[int][]fuzzyTerm {
	values := make(map[string][]target)
	for _, entry := range entries {
		for _, term := range append([]string{entry.Canonical}, entry.Terms...) {
			key := comparisonKey(term)
			values[key] = appendUniqueTarget(values[key], target{resourceID: entry.ResourceID, canonical: entry.Canonical})
		}
	}
	index := make(map[int][]fuzzyTerm)
	for key, targets := range values {
		length := len([]rune(key))
		index[length] = append(index[length], fuzzyTerm{
			value: key, targets: slices.Clone(targets), signature: fuzzyRuneSignature(key),
		})
	}
	for length := range index {
		slices.SortFunc(index[length], func(left, right fuzzyTerm) int {
			return strings.Compare(left.value, right.value)
		})
	}
	return index
}

func cloneSharedTargetMap(input map[string][]target) map[string][]target {
	output := make(map[string][]target, len(input))
	for key, values := range input {
		output[key] = slices.Clone(values)
	}
	return output
}
