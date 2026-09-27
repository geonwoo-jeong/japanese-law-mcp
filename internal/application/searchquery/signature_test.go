package searchquery

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func TestFuzzySignaturePreservesExhaustiveBoundedMatches(t *testing.T) {
	t.Parallel()

	values := distanceTestStrings([]rune{'a', '法', '🙂'}, 4)
	for _, left := range values {
		for _, right := range values {
			assertSignaturePreservesDistance(t, left, right)
		}
	}
}

func TestFuzzySignaturePreservesLongUnicodeEdits(t *testing.T) {
	t.Parallel()

	random := rand.New(rand.NewPCG(21, 25)) //nolint:gosec // SOT-ARCH-021: 暗号用途ではなく、合成編集の再現性を保つ固定乱数である。
	alphabet := []rune{'a', 'z', '法', '令', '権', '🙂', '𠮷', 'カ', '\u0301', '\ufffd'}
	for range 500 {
		left := make([]rune, random.IntN(96))
		for index := range left {
			left[index] = alphabet[random.IntN(len(alphabet))]
		}
		right := slices.Clone(left)
		for range random.IntN(7) {
			right = distanceTestEdit(random, right, alphabet)
		}
		assertSignaturePreservesDistance(t, left, right)
		assertSignaturePreservesDistance(t, right, left)
	}
}

func assertSignaturePreservesDistance(t *testing.T, left, right []rune) {
	t.Helper()

	distance := fullMatrixAdjacentDistance(left, right)
	for maximum := 0; maximum <= 3; maximum++ {
		if distance <= maximum && !fuzzySignatureMatches(
			fuzzyRuneSignature(string(left)),
			fuzzyRuneSignature(string(right)),
			maximum,
		) {
			t.Fatalf("SOT-ARCH-030: 閾値%d以内の候補を除外しました: %q / %q", maximum, string(left), string(right))
		}
	}
}

func TestFuzzySignatureCollisionStillRequiresExactDistance(t *testing.T) {
	t.Parallel()

	left, right := signatureCollision(t)
	query := strings.Repeat(string(left), 4)
	term := strings.Repeat(string(right), 4)
	if !fuzzySignatureMatches(fuzzyRuneSignature(query), fuzzyRuneSignature(term), 0) {
		t.Fatal("SOT-ARCH-021: 衝突した署名だけで候補を除外しました")
	}
	resolver := mustResolver(t, []EntryValues{{ResourceID: "衝突", Canonical: term}}, analyzerStub{})
	matches, err := resolver.ResolveUniqueTypoMatches(context.Background(), query)
	if err != nil || len(matches) != 0 {
		t.Fatalf("SOT-ARCH-030: 署名の衝突だけで解決しました: %#v、エラー %v", matches, err)
	}
}

func signatureCollision(t *testing.T) (rune, rune) {
	t.Helper()

	seen := make(map[[2]uint64]rune)
	for character := rune(0x4e00); character <= 0x4e80; character++ {
		signature := fuzzyRuneSignature(string(character))
		if previous, exists := seen[signature]; exists {
			return previous, character
		}
		seen[signature] = character
	}
	t.Fatal("SOT-ARCH-021: 合成入力で署名の衝突を構成できません")
	return 0, 0
}

func TestFuzzySignatureResolverMatchesUnfilteredReference(t *testing.T) {
	t.Parallel()

	entries := []EntryValues{
		{ResourceID: "共有乙", Canonical: "共有対象乙", Terms: []string{"共有法規"}},
		{ResourceID: "共有甲", Canonical: "共有対象甲", Terms: []string{"共有法規", "共有法規"}},
		{ResourceID: "同率甲", Canonical: "甲乙丙"},
		{ResourceID: "同率乙", Canonical: "甲乙丁"},
		{ResourceID: "同一対象", Canonical: "試験法", Terms: []string{"試験令"}},
		{ResourceID: "正規化", Canonical: "法例テスト"},
		{ResourceID: "転置", Canonical: "法規試験"},
	}
	queries := []string{"共有法規", "共有法則", "甲乙戊", "試験則", "法例ﾃｽﾄ", "規法試験", "法", strings.Repeat("q", 80)}
	random := rand.New(rand.NewPCG(30, 25)) //nolint:gosec // SOT-MODEL-025: 辞書と検索語の合成入力を同じ seed で再現する。
	alphabet := []rune{'法', '令', '語', '権', 'a', 'z', '🙂', '𠮷'}
	for index := range 40 {
		term := make([]rune, 3+random.IntN(40))
		for runeIndex := range term {
			term[runeIndex] = alphabet[random.IntN(len(alphabet))]
		}
		entries = append(entries, EntryValues{ResourceID: fmt.Sprintf("合成%d", index), Canonical: string(term)})
		queries = append(queries, string(term))
		for range 5 {
			edited := slices.Clone(term)
			for range 1 + random.IntN(4) {
				edited = distanceTestEdit(random, edited, alphabet)
			}
			if len(edited) > 0 {
				queries = append(queries, string(edited))
			}
		}
	}
	resolver := mustResolver(t, entries, analyzerStub{})
	for _, query := range queries {
		assertResolverMatchesReference(t, resolver, query)
	}
}

func assertResolverMatchesReference(t *testing.T, resolver *Resolver, query string) {
	t.Helper()

	for _, typoOnly := range []bool{false, true} {
		var matches []Match
		var err error
		if typoOnly {
			matches, err = resolver.ResolveUniqueTypoMatches(context.Background(), query)
		} else {
			matches, err = resolver.ResolveDirectMatches(context.Background(), query)
		}
		if err != nil {
			t.Fatalf("SOT-ARCH-021: 合成入力%qの照合エラー: %v", query, err)
		}
		want := unfilteredReferenceMatches(resolver, query, typoOnly)
		if !slices.Equal(matches, want) {
			t.Fatalf("SOT-MODEL-025: %qの対象または順序が全候補照合と異なります: %#v、期待値 %#v", query, matches, want)
		}
	}
}

func unfilteredReferenceMatches(resolver *Resolver, query string, typoOnly bool) []Match {
	if exact := resolver.exact[query]; len(exact) > 0 {
		if typoOnly {
			return nil
		}
		return matchesFromTargets(exact, MatchKindExact)
	}
	key := comparisonKey(query)
	if normalized := resolver.normalized[key]; len(normalized) > 0 {
		if typoOnly {
			return nil
		}
		return matchesFromTargets(normalized, MatchKindComparisonNormalized)
	}
	queryRunes := []rune(key)
	if len(queryRunes) < 3 {
		return nil
	}
	bestDistance := 4
	var closest []fuzzyTerm
	for length, terms := range resolver.fuzzy {
		for _, term := range terms {
			distance := fullMatrixAdjacentDistance(queryRunes, []rune(term.value))
			if distance > fuzzyMaximum(length) || distance > bestDistance {
				continue
			}
			if distance < bestDistance {
				bestDistance = distance
				closest = nil
			}
			closest = append(closest, term)
		}
	}
	if len(closest) != 1 {
		return nil
	}
	return matchesFromTargets(closest[0].targets, MatchKindUniqueTypoCorrection)
}
