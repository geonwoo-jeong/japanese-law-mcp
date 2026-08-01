package legalquerycorpus

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	repositoryCorpusV11Seed           = 20260801
	repositoryCorpusV11Development    = 43
	repositoryCorpusV11Holdout        = 251
	repositoryCorpusV11Execution      = 8
	repositoryCorpusV11LeakageDigests = 139
	repositoryCorpusV11HoldoutDigest  = "91b9c85930dcebd8bd5f58e414ac3289de535e913270c0821aa6ac92432fff93"
	repositoryCorpusV11ManifestSHA256 = "28b937b561effee8674f40981208698fc3287e85ffcd1caa1167f7c7636ec8b7"
	repositoryCorpusV11TreeSHA256     = "4fb2ac82ed97fc81920e567c9c687bb471c943f4c6b6d5e04c6c0090f89950ff"
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
	assertCorpusV11StableLeakageGroups(t, prepared.Holdout())
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

func assertCorpusV11StableLeakageGroups(
	t *testing.T,
	holdout []SemanticCase,
) {
	t.Helper()
	allowedPrefixes := []string{"lqg-case-courts-", "lqg-concept-", "lqg-law-", "lqg-ls-", "lqg-topic-"}
	for _, fixture := range holdout {
		groupID := fixture.LeakageGroupID()
		if strings.Contains(groupID, "v11") {
			t.Fatalf("SOT-ENG-026: leakageGroupId が corpus version に依存しています: %q", groupID)
		}
		if !slices.ContainsFunc(allowedPrefixes, func(prefix string) bool {
			return strings.HasPrefix(groupID, prefix)
		}) {
			t.Fatalf("SOT-ENG-026: leakageGroupId の安定分類が不明です: %q", groupID)
		}
	}
}
