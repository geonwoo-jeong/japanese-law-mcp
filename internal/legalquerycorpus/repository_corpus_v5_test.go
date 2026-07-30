package legalquerycorpus

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
)

const (
	repositoryCorpusV5Seed           = 20260727
	repositoryCorpusV5Development    = 32
	repositoryCorpusV5Holdout        = 240
	repositoryCorpusV5Execution      = 8
	repositoryCorpusV5HoldoutDigest  = "e8e5b4bb024ec8866ec5fc4ce138c1e2f37833e2c3b51e4023b3dd2d8eb3547e"
	repositoryCorpusV5ManifestSHA256 = "a66755e9a04fd12116552da6aafb41fb60b43b1433114c2f8754f329c90fa4ab"
)

func TestRepositoryCorpusV5は曖昧な略称誤記の選択期待値を訂正する(
	t *testing.T,
) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
	previous := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v4")
	corrected := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v5")
	assertRepositoryCorpusV5Manifest(t, corrected.Manifest())
	assertRepositoryCorpusV5Correction(t, previous, corrected)
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v5/manifest.json",
		repositoryCorpusV5ManifestSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v5",
		corrected,
	)
	assertRepositoryCorpusImmutable(t, corrected)
}

func loadRepositoryCorpusVersion(
	t *testing.T,
	repositoryRoot string,
	version string,
) Corpus {
	t.Helper()
	corpus, err := Load(
		context.Background(),
		repositoryRoot,
		filepath.Join("testdata", "legalquery", version),
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024, SOT-ENG-026: %s Load() error = %v", version, err)
	}
	return corpus
}

func assertRepositoryCorpusV5Manifest(t *testing.T, manifest Manifest) {
	t.Helper()
	if manifest.CorpusVersion() != "corpus-v5" ||
		manifest.Seed() != repositoryCorpusV5Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV5HoldoutDigest ||
		manifest.Development().CaseCount() != repositoryCorpusV5Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV5Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV5Execution {
		t.Fatalf("SOT-ENG-024: corpus-v5 の固定値が一致しません")
	}
}

func assertRepositoryCorpusV5Correction(
	t *testing.T,
	previous Corpus,
	corrected Corpus,
) {
	t.Helper()
	previousPlan := repositoryCorpusV5ExpectedPlan(t, previous)
	correctedPlan := repositoryCorpusV5ExpectedPlan(t, corrected)
	if len(previousPlan.SelectedMeaningIDs()) != 0 {
		t.Fatalf("SOT-ENG-024: corpus-v4 の訂正前 selection が空ではありません")
	}
	want := []string{"current-act", "amendment-enforcement-act"}
	if got := correctedPlan.SelectedMeaningIDs(); !slices.Equal(got, want) {
		t.Fatalf("SOT-ENG-024: corpus-v5 の訂正後 selection = %#v", got)
	}
}

func repositoryCorpusV5ExpectedPlan(t *testing.T, corpus Corpus) ExpectedPlan {
	t.Helper()
	for _, semanticCase := range corpus.Holdout() {
		if semanticCase.CaseID() != "holdout-typo-15" {
			continue
		}
		plan, ok := semanticCase.Expected().(ExpectedPlan)
		if !ok {
			t.Fatalf("SOT-ENG-026: holdout-typo-15 が plan ではありません")
		}
		return plan
	}
	t.Fatalf("SOT-ENG-026: holdout-typo-15 がありません")
	return ExpectedPlan{}
}
