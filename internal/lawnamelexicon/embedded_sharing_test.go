package lawnamelexicon

import (
	"reflect"
	"testing"
)

func TestLoadEmbeddedSharesValidatedImmutableDictionary(t *testing.T) {
	t.Parallel()

	first, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 組込み辞書を読み込めません: %v", err)
	}
	type result struct {
		lexicon *Lexicon
		err     error
	}
	const callers = 8
	results := make(chan result, callers)
	for range callers {
		go func() {
			lexicon, loadErr := LoadEmbedded()
			results <- result{lexicon: lexicon, err: loadErr}
		}()
	}
	for range callers {
		got := <-results
		if got.err != nil || got.lexicon != first {
			t.Fatalf("SOT-ARCH-021: 検証済み辞書が共有されていません: %v", got.err)
		}
	}

	entries := first.Entries()
	terms := first.Terms()
	if len(entries) == 0 || len(terms) == 0 {
		t.Fatal("SOT-ENG-022: 組込み辞書の登録語がありません")
	}
	wantEntries, wantTerms := first.Entries(), first.Terms()
	entries[0].Canonical = "一時的な試験値"
	if len(entries[0].Terms) > 0 {
		entries[0].Terms[0] = "一時的な試験値"
	}
	terms[0] = "一時的な試験値"
	again, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 共有辞書を再取得できません: %v", err)
	}
	if !reflect.DeepEqual(again.Entries(), wantEntries) || !reflect.DeepEqual(again.Terms(), wantTerms) {
		t.Fatal("SOT-ARCH-021: getter の複製への変更が共有辞書へ伝わりました")
	}
}
