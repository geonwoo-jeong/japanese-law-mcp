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
	repositoryCorpusV16Seed        = 20260806
	repositoryCorpusV16Development = 43
	repositoryCorpusV16Holdout     = 240
	repositoryCorpusV16Execution   = 8

	repositoryCorpusV16FixedLeakageDigests = 187
	repositoryCorpusV16FixedHoldoutDigest  = "e96059d035b7b4d6848b1d007db236f534016f66d2a0b3ae0d558d2a307c0884"
	repositoryCorpusV16FixedManifestSHA256 = "19dac2ac7f9bd474bf1a22fb5f7ff5dcdb36b547d924993bebfad16daeb4bf65"
	repositoryCorpusV16FixedTreeSHA256     = "8db963e0a3e789ea1517ea2cd2e2827bf2b7ecc75fd70f70c80d5b94cb0a3593"
)

func TestRepositoryCorpusV16Manifestを読み込める(t *testing.T) {
	corpus := loadRepositoryCorpusVersion(t, repositoryCorpusV16Root(t), "corpus-v16")
	manifest := corpus.Manifest()
	if manifest.SchemaVersion() != corpusSchemaVersionV2 ||
		manifest.CorpusVersion() != "corpus-v16" ||
		manifest.Seed() != repositoryCorpusV16Seed {
		t.Fatal("SOT-ENG-026: corpus-v16 manifest の識別値が一致しません")
	}
	if !slices.IsSorted(manifest.HoldoutLeakageGroupDigests()) ||
		!slices.IsSorted(repositorySemanticCaseIDs(corpus.Development())) ||
		!slices.IsSorted(repositorySemanticCaseIDs(corpus.Holdout())) ||
		!slices.IsSorted(repositoryExecutionCaseIDs(corpus.Execution())) {
		t.Fatal("SOT-ENG-026: corpus-v16 の manifest または case 順が固定されていません")
	}
}

func TestRepositoryCorpusV16は三集合の件数を固定する(t *testing.T) {
	corpus := loadRepositoryCorpusVersion(t, repositoryCorpusV16Root(t), "corpus-v16")
	manifest := corpus.Manifest()
	if manifest.Development().CaseCount() != repositoryCorpusV16Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV16Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV16Execution ||
		len(corpus.Development()) != repositoryCorpusV16Development ||
		len(corpus.Holdout()) != repositoryCorpusV16Holdout ||
		len(corpus.Execution()) != repositoryCorpusV16Execution {
		t.Fatal("SOT-ENG-026: corpus-v16 の集合件数が固定値と一致しません")
	}
}

func TestRepositoryCorpusV16はDevelopmentAssertionとHoldoutCoverageを満たす(t *testing.T) {
	corpus := loadRepositoryCorpusVersion(t, repositoryCorpusV16Root(t), "corpus-v16")
	t.Run("development assertion", func(t *testing.T) {
		assertRepositoryCorpusV10DevelopmentAssertions(t, corpus.Development())
	})
	t.Run("holdout coverage", func(t *testing.T) {
		assertRepositoryCorpusV10HoldoutCoverage(t, corpus.Holdout())
	})
}

func TestRepositoryCorpusV16は消費済みHoldoutから分離する(t *testing.T) {
	repositoryRoot := repositoryCorpusV16Root(t)
	prepared := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v16")
	consumedVersions := []struct {
		version        string
		manifestSHA256 string
		treeSHA256     string
	}{
		{"corpus-v10", repositoryCorpusV10ManifestSHA256, repositoryCorpusV10TreeSHA256},
		{"corpus-v11", repositoryCorpusV11ManifestSHA256, repositoryCorpusV11TreeSHA256},
		{"corpus-v12", repositoryCorpusV12ManifestSHA256, repositoryCorpusV12TreeSHA256},
		{"corpus-v13", repositoryCorpusV13ManifestSHA256, repositoryCorpusV13TreeSHA256},
		{"corpus-v14", repositoryCorpusV14ManifestSHA256, repositoryCorpusV14TreeSHA256},
	}
	for _, consumedVersion := range consumedVersions {
		t.Run(consumedVersion.version, func(t *testing.T) {
			assertRepositoryCorpusManifestDigest(
				t,
				repositoryRoot,
				filepath.Join("testdata", "legalquery", consumedVersion.version, "manifest.json"),
				consumedVersion.manifestSHA256,
			)
			assertRepositoryCorpusTreeDigest(
				t,
				filepath.Join(repositoryRoot, "testdata", "legalquery", consumedVersion.version),
				consumedVersion.treeSHA256,
			)
			consumed := loadRepositoryCorpusVersion(t, repositoryRoot, consumedVersion.version)
			if prepared.Manifest().HoldoutDigest() == consumed.Manifest().HoldoutDigest() {
				t.Fatal("SOT-ENG-038: 消費済み holdout digest を再利用しています")
			}
			assertNoConsumedLeakageGroupDigest(
				t,
				consumed.Manifest().HoldoutLeakageGroupDigests(),
				prepared.Manifest().HoldoutLeakageGroupDigests(),
			)
			assertRepositoryCorpusV16IndependentHoldout(t, consumed.Holdout(), prepared.Holdout())
		})
	}
}

func TestRepositoryCorpusV16は内部衝突境界と安定LeakageGroupを満たす(t *testing.T) {
	corpus := loadRepositoryCorpusVersion(t, repositoryCorpusV16Root(t), "corpus-v16")
	t.Run("内部衝突境界", func(t *testing.T) {
		assertRepositoryCorpusV16InternalCollisionBoundaries(t, corpus.Holdout())
	})
	t.Run("安定 leakage group", func(t *testing.T) {
		assertRepositoryCorpusV16StableLeakageGroups(t, corpus.Holdout())
	})
}

func TestRepositoryCorpusV16はV14のDevelopmentとExecutionをByte継承する(t *testing.T) {
	repositoryRoot := repositoryCorpusV16Root(t)
	for _, set := range []string{"development", "execution"} {
		t.Run(set, func(t *testing.T) {
			assertRepositoryCorpusV16CarryForward(t, repositoryRoot, set)
		})
	}
}

func TestRepositoryCorpusV16はReview済み最終Digestと不変Byteを固定する(t *testing.T) {
	requireRepositoryCorpusV16FixedDigests(t)
	repositoryRoot := repositoryCorpusV16Root(t)
	prepared := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v16")
	t.Run("holdout digest と leakage digest 件数", func(t *testing.T) {
		manifest := prepared.Manifest()
		if manifest.HoldoutDigest() != repositoryCorpusV16FixedHoldoutDigest ||
			len(manifest.HoldoutLeakageGroupDigests()) != repositoryCorpusV16FixedLeakageDigests {
			t.Fatal("SOT-ENG-026: corpus-v16 の review 済み holdout 固定値が一致しません")
		}
	})
	t.Run("manifest SHA-256", func(t *testing.T) {
		assertRepositoryCorpusManifestDigest(
			t,
			repositoryRoot,
			"testdata/legalquery/corpus-v16/manifest.json",
			repositoryCorpusV16FixedManifestSHA256,
		)
	})
	t.Run("tree SHA-256", func(t *testing.T) {
		assertRepositoryCorpusTreeDigest(
			t,
			filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v16"),
			repositoryCorpusV16FixedTreeSHA256,
		)
	})
	t.Run("再現可能性", func(t *testing.T) {
		assertRepositoryCorpusReproducible(
			t,
			repositoryRoot,
			"testdata/legalquery/corpus-v16",
			prepared,
		)
	})
	t.Run("不変 getter", func(t *testing.T) {
		assertRepositoryCorpusImmutable(t, prepared)
	})
}

func repositoryCorpusV16Root(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
}

func requireRepositoryCorpusV16FixedDigests(t *testing.T) {
	t.Helper()
	if repositoryCorpusV16FixedLeakageDigests < 1 ||
		repositoryCorpusV16FixedHoldoutDigest == "" ||
		repositoryCorpusV16FixedManifestSHA256 == "" ||
		repositoryCorpusV16FixedTreeSHA256 == "" {
		t.Fatal("SOT-ENG-026: manifest 生成と独立 review の完了後に最終 digest を固定する必要があります")
	}
}

func assertRepositoryCorpusV16IndependentHoldout(
	t *testing.T,
	consumed []SemanticCase,
	prepared []SemanticCase,
) {
	t.Helper()
	consumedCaseIDs := make(map[string]struct{}, len(consumed))
	consumedRequests := make(map[rawRequestIdentity]struct{}, len(consumed))
	consumedComparisonKeys := make(map[string]struct{}, len(consumed))
	consumedLeakageGroups := make(map[string]struct{}, len(consumed))
	for _, semanticCase := range consumed {
		consumedCaseIDs[semanticCase.CaseID()] = struct{}{}
		consumedRequests[rawRequestIdentityOf(semanticCase.Request())] = struct{}{}
		comparisonKey := semanticComparisonKey(semanticCase)
		if comparisonKey != "" {
			consumedComparisonKeys[comparisonKey] = struct{}{}
		}
		consumedLeakageGroups[semanticCase.LeakageGroupID()] = struct{}{}
	}

	for _, semanticCase := range prepared {
		if _, exists := consumedCaseIDs[semanticCase.CaseID()]; exists {
			t.Fatal("SOT-ENG-026: 消費済み holdout の caseId を再利用しています")
		}
		if _, exists := consumedRequests[rawRequestIdentityOf(semanticCase.Request())]; exists {
			t.Fatal("SOT-ENG-026: 消費済み holdout の完全 request を再利用しています")
		}
		comparisonKey := semanticComparisonKey(semanticCase)
		if comparisonKey != "" {
			if _, exists := consumedComparisonKeys[comparisonKey]; exists {
				t.Fatal("SOT-ENG-026: 消費済み holdout の ComparisonKey を再利用しています")
			}
		}
		if _, exists := consumedLeakageGroups[semanticCase.LeakageGroupID()]; exists {
			t.Fatal("SOT-ENG-026: 消費済み holdout の leakageGroupId を再利用しています")
		}
	}
}

func assertRepositoryCorpusV16InternalCollisionBoundaries(t *testing.T, holdout []SemanticCase) {
	t.Helper()
	caseIDs := make(map[string]struct{}, len(holdout))
	requests := make(map[rawRequestIdentity]struct{}, len(holdout))
	comparisonKeys := make(map[string]string, len(holdout))
	for _, semanticCase := range holdout {
		groupID := semanticCase.LeakageGroupID()
		if _, exists := caseIDs[semanticCase.CaseID()]; exists {
			t.Fatal("SOT-ENG-026: corpus-v16 内で caseId が重複しています")
		}
		caseIDs[semanticCase.CaseID()] = struct{}{}
		request := rawRequestIdentityOf(semanticCase.Request())
		if _, exists := requests[request]; exists {
			t.Fatal("SOT-ENG-026: corpus-v16 内で完全 request が重複しています")
		}
		requests[request] = struct{}{}
		comparisonKey := semanticComparisonKey(semanticCase)
		if comparisonKey != "" {
			assertRepositoryCorpusV16CollisionGroup(
				t,
				comparisonKeys,
				comparisonKey,
				groupID,
				"ComparisonKey",
			)
		}
	}
}

func assertRepositoryCorpusV16CollisionGroup[T comparable](
	t *testing.T,
	seen map[T]string,
	value T,
	groupID string,
	axis string,
) {
	t.Helper()
	if previousGroup, exists := seen[value]; exists && previousGroup != groupID {
		t.Fatalf("SOT-ENG-026: corpus-v16 内の %s が異なる意味家族で重複しています", axis)
	}
	seen[value] = groupID
}

func assertRepositoryCorpusV16StableLeakageGroups(t *testing.T, holdout []SemanticCase) {
	t.Helper()
	for _, semanticCase := range holdout {
		groupID := semanticCase.LeakageGroupID()
		if strings.Contains(groupID, "v16") ||
			strings.Contains(groupID, "corpus") ||
			!strings.HasPrefix(groupID, "lqg-") {
			t.Fatal("SOT-ENG-026: leakageGroupId が安定分類ではありません")
		}
	}
}

func assertRepositoryCorpusV16CarryForward(t *testing.T, repositoryRoot string, set string) {
	t.Helper()
	previousRoot := filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v14", set)
	preparedRoot := filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v16", set)
	previousEntries, err := os.ReadDir(previousRoot)
	if err != nil {
		t.Fatalf("%s: corpus-v14 の集合を列挙できません: %v", corpusImmutableVersionVerificationID, err)
	}
	preparedEntries, err := os.ReadDir(preparedRoot)
	if err != nil {
		t.Fatalf("%s: corpus-v16 の集合を列挙できません: %v", corpusImmutableVersionVerificationID, err)
	}
	if len(previousEntries) != len(preparedEntries) {
		t.Fatalf("%s: 継承件数が一致しません", corpusImmutableVersionVerificationID)
	}
	for index, previousEntry := range previousEntries {
		preparedEntry := preparedEntries[index]
		if previousEntry.Name() != preparedEntry.Name() ||
			previousEntry.Type() != preparedEntry.Type() ||
			!bytes.Equal(
				readRepositoryCorpusFile(t, filepath.Join(previousRoot, previousEntry.Name())),
				readRepositoryCorpusFile(t, filepath.Join(preparedRoot, preparedEntry.Name())),
			) {
			t.Fatalf("%s: corpus-v14 の集合が byte 単位で継承されていません", corpusImmutableVersionVerificationID)
		}
	}
}
