package lawnamelexicon

import "testing"

func TestLexiconVersionは公式辞書と補足辞書の版を固定する(t *testing.T) {
	t.Parallel()

	lexicon, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ENG-022: embedded lexicon のエラー = %v", err)
	}
	const expected = "e-gov-law-api-v2-laws-2026-07-27+ndl-common-abbreviations-2026-07-27"
	if lexicon.Version() != expected {
		t.Fatalf("SOT-ENG-022: version = %q, want %q", lexicon.Version(), expected)
	}
	var nilLexicon *Lexicon
	if nilLexicon.Version() != "" {
		t.Fatalf("nil lexicon version = %q", nilLexicon.Version())
	}
}
