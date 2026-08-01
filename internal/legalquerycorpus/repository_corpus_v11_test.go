package legalquerycorpus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const (
	repositoryCorpusV11Seed           = 20260801
	repositoryCorpusV11Development    = 43
	repositoryCorpusV11Holdout        = 251
	repositoryCorpusV11Execution      = 8
	repositoryCorpusV11LeakageDigests = 203
	repositoryCorpusV11HoldoutDigest  = "b51f973307a4bc5da8b4be8ac3577ff0a718db6108f6dc861e9a469519f7c401"
	repositoryCorpusV11ManifestSHA256 = "70007b6c837423d861af4d18af2a9802c357ed2cac5850942f3ce9659b90b365"
	repositoryCorpusV11TreeSHA256     = "d9293176ca43c33080da8ee047d722bb0ed799a0674ddeb9a534e06d67b05ab1"
)

func TestRepositoryCorpusV11は消費済みHoldoutから分離する(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
	consumedArtifact, err := LoadManifest(
		t.Context(),
		repositoryRoot,
		"testdata/legalquery/corpus-v10",
	)
	if err != nil {
		t.Fatalf("SOT-ENG-038: 消費済み manifest を限定読取りできません: %v", err)
	}

	prepared := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v11")
	manifest := prepared.Manifest()
	if manifest.SchemaVersion() != corpusSchemaVersionV2 ||
		manifest.CorpusVersion() != "corpus-v11" ||
		manifest.Seed() != repositoryCorpusV11Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV11HoldoutDigest ||
		manifest.Development().CaseCount() != repositoryCorpusV11Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV11Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV11Execution ||
		len(manifest.HoldoutLeakageGroupDigests()) != repositoryCorpusV11LeakageDigests {
		t.Fatalf("SOT-ENG-026: corpus-v11 の版境界が一致しません")
	}

	consumed := consumedArtifact.Manifest()
	if manifest.HoldoutDigest() == consumed.HoldoutDigest() {
		t.Fatal("SOT-ENG-038: 消費済み holdout digest を再利用しています")
	}
	assertNoConsumedLeakageGroupDigest(
		t,
		consumed.HoldoutLeakageGroupDigests(),
		manifest.HoldoutLeakageGroupDigests(),
	)
	assertNoConsumedEquivalentHoldout(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v10/holdout",
		"testdata/legalquery/corpus-v11/holdout",
	)
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v11/manifest.json",
		repositoryCorpusV11ManifestSHA256,
	)
	assertRepositoryCorpusTreeDigest(
		t,
		filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v11"),
		repositoryCorpusV11TreeSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v11",
		prepared,
	)
	assertRepositoryCorpusImmutable(t, prepared)
}

func assertNoConsumedLeakageGroupDigest(
	t *testing.T,
	consumed []string,
	prepared []string,
) {
	t.Helper()
	consumedSet := make(map[string]struct{}, len(consumed))
	for _, digest := range consumed {
		consumedSet[digest] = struct{}{}
	}
	for _, digest := range prepared {
		if _, exists := consumedSet[digest]; exists {
			t.Fatalf("SOT-ENG-038: 消費済み leakage group digest %q を再利用しています", digest)
		}
	}
}

func assertNoConsumedEquivalentHoldout(
	t *testing.T,
	repositoryRoot string,
	consumedRelative string,
	preparedRelative string,
) {
	t.Helper()
	consumedPatterns := loadEquivalentHoldoutPatterns(
		t,
		filepath.Join(repositoryRoot, consumedRelative),
	)
	preparedPatterns := loadEquivalentHoldoutPatterns(
		t,
		filepath.Join(repositoryRoot, preparedRelative),
	)
	for pattern, consumedNames := range consumedPatterns {
		preparedNames, exists := preparedPatterns[pattern]
		if !exists {
			continue
		}
		t.Fatalf(
			"SOT-ENG-038: 消費済み holdout と同値な fixture を再利用しています: consumed=%v prepared=%v",
			consumedNames,
			preparedNames,
		)
	}
}

func loadEquivalentHoldoutPatterns(
	t *testing.T,
	directory string,
) map[string][]string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(directory, "*.json"))
	if err != nil {
		t.Fatalf("SOT-ENG-038: holdout fixture を列挙できません: %v", err)
	}
	slices.Sort(paths)
	patterns := make(map[string][]string, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("SOT-ENG-038: holdout fixture を読めません: %v", err)
		}
		var document map[string]any
		if err := json.Unmarshal(raw, &document); err != nil {
			t.Fatalf("SOT-ENG-038: holdout fixture JSON を解釈できません: %v", err)
		}
		delete(document, "schemaVersion")
		delete(document, "caseId")
		delete(document, "leakageGroupId")
		normalized, err := json.Marshal(document)
		if err != nil {
			t.Fatalf("SOT-ENG-038: holdout fixture を正規化できません: %v", err)
		}
		key := string(normalized)
		patterns[key] = append(patterns[key], filepath.Base(path))
	}
	return patterns
}
