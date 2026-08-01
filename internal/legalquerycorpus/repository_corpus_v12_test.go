package legalquerycorpus

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	repositoryCorpusV12Seed           = 20260802
	repositoryCorpusV12Development    = 43
	repositoryCorpusV12Holdout        = 251
	repositoryCorpusV12Execution      = 8
	repositoryCorpusV12LeakageDigests = 204
	repositoryCorpusV12HoldoutDigest  = "6cd334e801499b0fe7de55532afb4c32254af4e66cc69e4922b71f38124fbfc0"
	repositoryCorpusV12ManifestSHA256 = "3457c5d8b512b19e3178447134522499f47f118c75c34c0b152acfa07b094c8b"
	repositoryCorpusV12TreeSHA256     = "05df4cfad052d5156de230cb91cc76df6b44f8525c21964ec70623f3beb2cd3a"
)

func TestRepositoryCorpusV12は過去のV2Holdoutから分離し不変Byteを固定する(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
	prepared := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v12")
	manifest := prepared.Manifest()
	if manifest.SchemaVersion() != corpusSchemaVersionV2 ||
		manifest.CorpusVersion() != "corpus-v12" ||
		manifest.Seed() != repositoryCorpusV12Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV12HoldoutDigest ||
		manifest.Development().CaseCount() != repositoryCorpusV12Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV12Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV12Execution ||
		len(manifest.HoldoutLeakageGroupDigests()) != repositoryCorpusV12LeakageDigests ||
		!slices.IsSorted(manifest.HoldoutLeakageGroupDigests()) {
		t.Fatalf("%s: corpus-v12 manifest が固定値と一致しません", corpusImmutableVersionVerificationID)
	}

	for _, version := range []string{"corpus-v10", "corpus-v11"} {
		consumedArtifact, err := LoadManifest(
			t.Context(),
			repositoryRoot,
			filepath.Join("testdata", "legalquery", version),
		)
		if err != nil {
			t.Fatalf("SOT-ENG-038: %s manifest を限定読取りできません: %v", version, err)
		}
		consumed := consumedArtifact.Manifest()
		if manifest.HoldoutDigest() == consumed.HoldoutDigest() {
			t.Fatalf("SOT-ENG-038: %s の消費済み holdout digest を再利用しています", version)
		}
		assertNoConsumedLeakageGroupDigest(
			t,
			consumed.HoldoutLeakageGroupDigests(),
			manifest.HoldoutLeakageGroupDigests(),
		)
	}

	assertRepositoryCorpusV10DevelopmentAssertions(t, prepared.Development())
	assertRepositoryCorpusV10HoldoutCoverage(t, prepared.Holdout())
	assertCorpusV12StableLeakageGroups(t, prepared.Holdout())
	assertCorpusV12CarryForward(t, repositoryRoot, "development")
	assertCorpusV12CarryForward(t, repositoryRoot, "execution")
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v12/manifest.json",
		repositoryCorpusV12ManifestSHA256,
	)
	assertRepositoryCorpusTreeDigest(
		t,
		filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v12"),
		repositoryCorpusV12TreeSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v12",
		prepared,
	)
	assertRepositoryCorpusImmutable(t, prepared)
}

func assertCorpusV12StableLeakageGroups(t *testing.T, holdout []SemanticCase) {
	t.Helper()
	for _, fixture := range holdout {
		groupID := fixture.LeakageGroupID()
		if strings.Contains(groupID, "v12") || strings.Contains(groupID, "corpus") {
			t.Fatalf("SOT-ENG-026: leakageGroupId が corpus version に依存しています: %q", groupID)
		}
		if !strings.HasPrefix(groupID, "lqg-") {
			t.Fatalf("SOT-ENG-026: leakageGroupId の安定分類が不明です: %q", groupID)
		}
	}
}

func assertCorpusV12CarryForward(t *testing.T, repositoryRoot string, set string) {
	t.Helper()
	previousRoot := filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v11", set)
	preparedRoot := filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v12", set)
	previousEntries, err := os.ReadDir(previousRoot)
	if err != nil {
		t.Fatalf("%s: corpus-v11 %s を列挙できません: %v", corpusImmutableVersionVerificationID, set, err)
	}
	preparedEntries, err := os.ReadDir(preparedRoot)
	if err != nil {
		t.Fatalf("%s: corpus-v12 %s を列挙できません: %v", corpusImmutableVersionVerificationID, set, err)
	}
	if len(previousEntries) != len(preparedEntries) {
		t.Fatalf("%s: %s の継承件数が一致しません", corpusImmutableVersionVerificationID, set)
	}
	for index, previousEntry := range previousEntries {
		preparedEntry := preparedEntries[index]
		if previousEntry.Name() != preparedEntry.Name() ||
			previousEntry.Type() != preparedEntry.Type() ||
			!bytes.Equal(
				readRepositoryCorpusFile(t, filepath.Join(previousRoot, previousEntry.Name())),
				readRepositoryCorpusFile(t, filepath.Join(preparedRoot, preparedEntry.Name())),
			) {
			t.Fatalf("%s: corpus-v11 の %s が byte 単位で継承されていません", corpusImmutableVersionVerificationID, set)
		}
	}
}
