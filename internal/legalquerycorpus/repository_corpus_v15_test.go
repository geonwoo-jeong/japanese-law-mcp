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
	repositoryCorpusV15Seed           = 20260805
	repositoryCorpusV15Development    = 43
	repositoryCorpusV15Holdout        = 255
	repositoryCorpusV15Execution      = 8
	repositoryCorpusV15LeakageDigests = 255
	repositoryCorpusV15HoldoutDigest  = "009c5c54ea8ddb5154e4b8bcbb1c6fbd8cb1d6c23e088dbf9dffdb6807918c0c"
	repositoryCorpusV15ManifestSHA256 = "6553b64916df11b32aac391ccd0a2aa678d173bc8f7ed488ad4e4d45750ffb64"
	repositoryCorpusV15TreeSHA256     = "5ed58092fd65f2c0a7c353379e069408ca480c03af8e4dc5d3815e0a710a9ebc"
)

func TestRepositoryCorpusV15は独立HoldoutとV14継承Byteを固定する(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
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
		assertRepositoryCorpusManifestDigest(
			t,
			repositoryRoot,
			filepath.Join(
				"testdata",
				"legalquery",
				consumedVersion.version,
				"manifest.json",
			),
			consumedVersion.manifestSHA256,
		)
		assertRepositoryCorpusTreeDigest(
			t,
			filepath.Join(
				repositoryRoot,
				"testdata",
				"legalquery",
				consumedVersion.version,
			),
			consumedVersion.treeSHA256,
		)
	}

	prepared := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v15")
	assertRepositoryCorpusV15Manifest(t, prepared)

	consumedHoldout := make([]SemanticCase, 0, repositoryCorpusV15Holdout*5)
	for _, consumedVersion := range consumedVersions {
		consumed := loadRepositoryCorpusVersion(
			t,
			repositoryRoot,
			consumedVersion.version,
		)
		if prepared.Manifest().HoldoutDigest() == consumed.Manifest().HoldoutDigest() {
			t.Fatal("SOT-ENG-038, SOT-ENG-042: 予約済み holdout digest を再利用しています")
		}
		assertNoConsumedLeakageGroupDigest(
			t,
			consumed.Manifest().HoldoutLeakageGroupDigests(),
			prepared.Manifest().HoldoutLeakageGroupDigests(),
		)
		consumedHoldout = append(consumedHoldout, consumed.Holdout()...)
	}

	assertRepositoryCorpusV15IndependentHoldout(
		t,
		consumedHoldout,
		prepared.Holdout(),
	)
	assertRepositoryCorpusV15InternalCollisionBoundaries(t, prepared.Holdout())
	assertRepositoryCorpusV10DevelopmentAssertions(t, prepared.Development())
	assertRepositoryCorpusV10HoldoutCoverage(t, prepared.Holdout())
	assertRepositoryCorpusV15StableLeakageGroups(t, prepared.Holdout())
	assertRepositoryCorpusV15CarryForward(t, repositoryRoot, "development")
	assertRepositoryCorpusV15CarryForward(t, repositoryRoot, "execution")
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v15/manifest.json",
		repositoryCorpusV15ManifestSHA256,
	)
	assertRepositoryCorpusTreeDigest(
		t,
		filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v15"),
		repositoryCorpusV15TreeSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v15",
		prepared,
	)
	assertRepositoryCorpusImmutable(t, prepared)
}

func assertRepositoryCorpusV15Manifest(t *testing.T, corpus Corpus) {
	t.Helper()
	manifest := corpus.Manifest()
	digests := manifest.HoldoutLeakageGroupDigests()
	if manifest.SchemaVersion() != corpusSchemaVersionV2 ||
		manifest.CorpusVersion() != "corpus-v15" ||
		manifest.Seed() != repositoryCorpusV15Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV15HoldoutDigest ||
		manifest.Development().CaseCount() != repositoryCorpusV15Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV15Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV15Execution ||
		len(corpus.Development()) != repositoryCorpusV15Development ||
		len(corpus.Holdout()) != repositoryCorpusV15Holdout ||
		len(corpus.Execution()) != repositoryCorpusV15Execution ||
		len(digests) != repositoryCorpusV15LeakageDigests ||
		!slices.IsSorted(digests) {
		t.Fatal("SOT-ENG-026: corpus-v15 manifest が固定値と一致しません")
	}
	if !slices.IsSorted(repositorySemanticCaseIDs(corpus.Development())) ||
		!slices.IsSorted(repositorySemanticCaseIDs(corpus.Holdout())) ||
		!slices.IsSorted(repositoryExecutionCaseIDs(corpus.Execution())) {
		t.Fatal("SOT-ENG-026: corpus-v15 の case 順が固定されていません")
	}
}

func assertRepositoryCorpusV15IndependentHoldout(
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
			t.Fatal("SOT-ENG-026, SOT-ENG-042: 予約済み holdout の caseId を再利用しています")
		}
		if _, exists := consumedRequests[rawRequestIdentityOf(semanticCase.Request())]; exists {
			t.Fatal("SOT-ENG-026, SOT-ENG-042: 予約済み holdout の完全 request を再利用しています")
		}
		comparisonKey := semanticComparisonKey(semanticCase)
		if comparisonKey != "" {
			if _, exists := consumedComparisonKeys[comparisonKey]; exists {
				t.Fatal("SOT-ENG-026, SOT-ENG-042: 予約済み holdout の ComparisonKey を再利用しています")
			}
		}
		if _, exists := consumedLeakageGroups[semanticCase.LeakageGroupID()]; exists {
			t.Fatal("SOT-ENG-026, SOT-ENG-042: 予約済み holdout の leakageGroupId を再利用しています")
		}
	}
}

func assertRepositoryCorpusV15InternalCollisionBoundaries(
	t *testing.T,
	holdout []SemanticCase,
) {
	t.Helper()
	caseIDs := make(map[string]struct{}, len(holdout))
	requests := make(map[rawRequestIdentity]struct{}, len(holdout))
	comparisonKeys := make(map[string]string, len(holdout))
	for _, semanticCase := range holdout {
		groupID := semanticCase.LeakageGroupID()
		if _, exists := caseIDs[semanticCase.CaseID()]; exists {
			t.Fatal("SOT-ENG-026: corpus-v15 内で caseId が重複しています")
		}
		caseIDs[semanticCase.CaseID()] = struct{}{}
		request := rawRequestIdentityOf(semanticCase.Request())
		if _, exists := requests[request]; exists {
			t.Fatal("SOT-ENG-026: corpus-v15 内で完全 request が重複しています")
		}
		requests[request] = struct{}{}
		comparisonKey := semanticComparisonKey(semanticCase)
		if comparisonKey != "" {
			assertRepositoryCorpusV15CollisionGroup(
				t,
				comparisonKeys,
				comparisonKey,
				groupID,
				"ComparisonKey",
			)
		}
	}
}

func assertRepositoryCorpusV15CollisionGroup[T comparable](
	t *testing.T,
	seen map[T]string,
	value T,
	groupID string,
	axis string,
) {
	t.Helper()
	if previousGroup, exists := seen[value]; exists && previousGroup != groupID {
		t.Fatalf("SOT-ENG-026: corpus-v15 内の %s が異なる意味家族で重複しています", axis)
	}
	seen[value] = groupID
}

func assertRepositoryCorpusV15StableLeakageGroups(
	t *testing.T,
	holdout []SemanticCase,
) {
	t.Helper()
	for _, semanticCase := range holdout {
		groupID := semanticCase.LeakageGroupID()
		if strings.Contains(groupID, "v15") ||
			strings.Contains(groupID, "corpus") ||
			!strings.HasPrefix(groupID, "lqg-") {
			t.Fatal("SOT-ENG-026: leakageGroupId が安定分類ではありません")
		}
	}
}

func assertRepositoryCorpusV15CarryForward(
	t *testing.T,
	repositoryRoot string,
	set string,
) {
	t.Helper()
	previousRoot := filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v14", set)
	preparedRoot := filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v15", set)
	previousEntries, err := os.ReadDir(previousRoot)
	if err != nil {
		t.Fatalf("%s: corpus-v14 の集合を列挙できません: %v", corpusImmutableVersionVerificationID, err)
	}
	preparedEntries, err := os.ReadDir(preparedRoot)
	if err != nil {
		t.Fatalf("%s: corpus-v15 の集合を列挙できません: %v", corpusImmutableVersionVerificationID, err)
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
