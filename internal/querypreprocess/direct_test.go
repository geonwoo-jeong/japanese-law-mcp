package querypreprocess

import (
	"context"
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawtarget"
)

func TestPreprocessorSharesLawIndexWithoutRepeatingAnalysis(t *testing.T) {
	t.Parallel()
	preprocessor, err := New(Values{
		Analyzer: errorOccurrenceAnalyzer{err: errors.New("形態素解析を再実行してはなりません")},
		LawNames: validInternalLawEntries(),
	})
	if err != nil {
		t.Fatal(err)
	}
	shared, err := lawtarget.NewPreprocessResolverWithDirectMatcher(preprocessor, preprocessor)
	if err != nil {
		t.Fatal(err)
	}
	original, err := lawtarget.NewPreprocessResolver(preprocessor, validInternalLawEntries())
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"民法", "みんぽう", "ミンポウ", "みんほう", "未知語"} {
		got, resolved, resolveErr := shared.ResolveLogicalInput(context.Background(), query)
		want, wantResolved, wantErr := original.ResolveLogicalInput(context.Background(), query)
		if resolveErr != nil || wantErr != nil || resolved != wantResolved || got != want {
			t.Fatalf("SOT-ARCH-030: %q の共有前後が一致しません: %#v / %#v (%v, %v)", query, got, want, resolveErr, wantErr)
		}
	}
}

func TestPreprocessorDirectMatcherRejectsMissingDependency(t *testing.T) {
	t.Parallel()
	var missing *Preprocessor
	for _, value := range []*Preprocessor{missing, {}} {
		if _, err := value.ResolveDirectMatches(context.Background(), "民法"); err == nil {
			t.Fatal("SOT-ARCH-030: 法令名索引のない前処理器を受理しました")
		}
	}
	if _, err := lawtarget.NewPreprocessResolverWithDirectMatcher(missing, missing); err == nil {
		t.Fatal("SOT-ARCH-030: nil 依存を受理しました")
	}
	preprocessor, err := New(Values{Analyzer: emptyOccurrenceAnalyzer{}, LawNames: validInternalLawEntries()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lawtarget.NewPreprocessResolverWithDirectMatcher(preprocessor, missing); err == nil {
		t.Fatal("SOT-ARCH-030: 型付き nil の直接照合器を受理しました")
	}
}
