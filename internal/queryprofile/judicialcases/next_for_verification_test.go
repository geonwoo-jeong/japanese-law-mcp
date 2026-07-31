package judicialcases

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateartifact"
)

func TestLoadNextForVerificationはDevelopment校正CueをActiveから分離する(
	t *testing.T,
) {
	t.Parallel()

	next, err := loadCandidateForTest()
	if err != nil {
		t.Fatalf("SOT-ENG-024/SOT-ENG-039: 校正版を読み込めません: %v", err)
	}
	active, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ARCH-033: active 版を読み込めません: %v", err)
	}
	if next.Metadata().ProfileVersion() !=
		"judicial-cases-2026-07-31-12" ||
		next.Metadata().CueSetVersion() !=
			"judicial-cases-cues-2026-07-31-5" ||
		active.Metadata().ProfileVersion() !=
			"judicial-cases-2026-07-30-9" ||
		active.Metadata().CueSetVersion() !=
			"judicial-cases-cues-2026-07-30-4" {
		t.Fatal("SOT-ARCH-033: 校正版と active 版の identity が分離されていません")
	}

	nextSearch := judicialTaskSearchTerms(t, next.CueVocabulary())
	activeSearch := judicialTaskSearchTerms(t, active.CueVocabulary())
	for _, term := range []string{"教えて", "教えてください"} {
		if slices.Contains(nextSearch, term) ||
			!slices.Contains(activeSearch, term) {
			t.Fatalf(
				"SOT-ARCH-025/SOT-ARCH-033: term %q の校正境界が不正です",
				term,
			)
		}
	}
	for _, term := range []string{"検索", "検索してください"} {
		if !slices.Contains(nextSearch, term) ||
			!slices.Contains(activeSearch, term) {
			t.Fatalf(
				"SOT-ARCH-038: 明示検索 term %q を失いました",
				term,
			)
		}
	}
}

func TestLoadNextForVerificationは明示的な裁判例検索を保持する(
	t *testing.T,
) {
	t.Parallel()

	const verificationID = "next-judicial-explicit-search-regression"
	profile, err := loadCandidateForTest()
	if err != nil {
		t.Fatalf("%s: 校正版を読み込めません: %v", verificationID, err)
	}
	generation := generateJudicialEvidenceQuery(
		t,
		profile,
		"医療過誤の裁判例を検索してください。",
		nil,
		verificationID,
	)
	candidate := mustSingleJudicialEvidenceCandidate(
		t,
		generation,
		verificationID,
	)
	if got := judicialEvidenceSearchQueries(
		t,
		candidate,
		verificationID,
	); !slices.Equal(got, []string{"医療過誤"}) {
		t.Fatalf("%s: search queries = %#v", verificationID, got)
	}
}

func loadCandidateForTest() (*Profile, error) {
	lawNames, err := lawnamelexicon.LoadEmbedded()
	if err != nil {
		return nil, err
	}
	concepts, err := legalconceptlexicon.LoadEmbedded()
	if err != nil {
		return nil, err
	}
	artifact := legalquerycandidateartifact.JudicialCases()
	return Load(artifact.Metadata(), artifact.Cues(), lawNames, concepts)
}

func judicialTaskSearchTerms(
	t *testing.T,
	entries []legalquery.CueVocabularyEntry,
) []string {
	t.Helper()

	for _, entry := range entries {
		if entry.CueID == "task-search" {
			return slices.Clone(entry.Terms)
		}
	}
	t.Fatal("SOT-ENG-024: task-search cue がありません")
	return nil
}
