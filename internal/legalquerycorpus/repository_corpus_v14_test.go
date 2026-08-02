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
	repositoryCorpusV14Seed           = 20260804
	repositoryCorpusV14Development    = 43
	repositoryCorpusV14Holdout        = 255
	repositoryCorpusV14Execution      = 8
	repositoryCorpusV14LeakageDigests = 228
	repositoryCorpusV14HoldoutDigest  = "33cb786fc26ecefc08c3b80ec1219f13de51ec112186f91237dfc7cba95f2489"
	repositoryCorpusV14ManifestSHA256 = "7bdb610ba3f95f4d291b16d4c7e73f89686fcd331ba046f9a3bb460c737be49e"
	repositoryCorpusV14TreeSHA256     = "bc6ec2c3184fdce073fa19ae2b9e090ac6bb3fe0947bef269fa6791cc2d6286d"
)

func TestRepositoryCorpusV14は独立HoldoutとV13継承Byteを固定する(t *testing.T) {
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

	prepared := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v14")
	assertRepositoryCorpusV14Manifest(t, prepared)

	consumedHoldout := make([]SemanticCase, 0, repositoryCorpusV14Holdout*4)
	for _, consumedVersion := range consumedVersions {
		consumed := loadRepositoryCorpusVersion(
			t,
			repositoryRoot,
			consumedVersion.version,
		)
		if prepared.Manifest().HoldoutDigest() == consumed.Manifest().HoldoutDigest() {
			t.Fatal("SOT-ENG-038: 消費済み holdout digest を再利用しています")
		}
		assertNoConsumedLeakageGroupDigest(
			t,
			consumed.Manifest().HoldoutLeakageGroupDigests(),
			prepared.Manifest().HoldoutLeakageGroupDigests(),
		)
		consumedHoldout = append(consumedHoldout, consumed.Holdout()...)
	}

	assertRepositoryCorpusV14IndependentHoldout(
		t,
		consumedHoldout,
		prepared.Holdout(),
	)
	assertRepositoryCorpusV14InternalCollisionBoundaries(t, prepared.Holdout())
	assertRepositoryCorpusV10DevelopmentAssertions(t, prepared.Development())
	assertRepositoryCorpusV10HoldoutCoverage(t, prepared.Holdout())
	assertRepositoryCorpusV14StableLeakageGroups(t, prepared.Holdout())
	assertRepositoryCorpusV14CarryForward(t, repositoryRoot, "development")
	assertRepositoryCorpusV14CarryForward(t, repositoryRoot, "execution")
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v14/manifest.json",
		repositoryCorpusV14ManifestSHA256,
	)
	assertRepositoryCorpusTreeDigest(
		t,
		filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v14"),
		repositoryCorpusV14TreeSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v14",
		prepared,
	)
	assertRepositoryCorpusImmutable(t, prepared)
}

func assertRepositoryCorpusV14Manifest(t *testing.T, corpus Corpus) {
	t.Helper()
	manifest := corpus.Manifest()
	digests := manifest.HoldoutLeakageGroupDigests()
	if manifest.SchemaVersion() != corpusSchemaVersionV2 ||
		manifest.CorpusVersion() != "corpus-v14" ||
		manifest.Seed() != repositoryCorpusV14Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV14HoldoutDigest ||
		manifest.Development().CaseCount() != repositoryCorpusV14Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV14Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV14Execution ||
		len(corpus.Development()) != repositoryCorpusV14Development ||
		len(corpus.Holdout()) != repositoryCorpusV14Holdout ||
		len(corpus.Execution()) != repositoryCorpusV14Execution ||
		len(digests) != repositoryCorpusV14LeakageDigests ||
		!slices.IsSorted(digests) {
		t.Fatal("SOT-ENG-026: corpus-v14 manifest が固定値と一致しません")
	}
	if !slices.IsSorted(repositorySemanticCaseIDs(corpus.Development())) ||
		!slices.IsSorted(repositorySemanticCaseIDs(corpus.Holdout())) ||
		!slices.IsSorted(repositoryExecutionCaseIDs(corpus.Execution())) {
		t.Fatal("SOT-ENG-026: corpus-v14 の case 順が固定されていません")
	}
}

func assertRepositoryCorpusV14IndependentHoldout(
	t *testing.T,
	consumed []SemanticCase,
	prepared []SemanticCase,
) {
	t.Helper()
	consumedCaseIDs := make(map[string]struct{}, len(consumed))
	consumedRequests := make(map[rawRequestIdentity]struct{}, len(consumed))
	consumedComparisonKeys := make(map[string]struct{}, len(consumed))
	consumedLeakageGroups := make(map[string]struct{}, len(consumed))
	consumedMeaningSignatures := make(map[string]struct{})
	for _, semanticCase := range consumed {
		consumedCaseIDs[semanticCase.CaseID()] = struct{}{}
		consumedRequests[rawRequestIdentityOf(semanticCase.Request())] = struct{}{}
		comparisonKey := semanticComparisonKey(semanticCase)
		if comparisonKey != "" {
			consumedComparisonKeys[comparisonKey] = struct{}{}
		}
		consumedLeakageGroups[semanticCase.LeakageGroupID()] = struct{}{}
		for _, signature := range repositoryCorpusV13MeaningSignatures(semanticCase) {
			consumedMeaningSignatures[signature] = struct{}{}
		}
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
		for _, signature := range repositoryCorpusV13MeaningSignatures(semanticCase) {
			if _, exists := consumedMeaningSignatures[signature]; exists {
				t.Fatal("SOT-ENG-026: 消費済み holdout の期待意味署名を再利用しています")
			}
		}
	}
}

func assertRepositoryCorpusV14InternalCollisionBoundaries(
	t *testing.T,
	holdout []SemanticCase,
) {
	t.Helper()
	caseIDs := make(map[string]struct{}, len(holdout))
	requests := make(map[rawRequestIdentity]struct{}, len(holdout))
	comparisonKeys := make(map[string]string, len(holdout))
	meaningSignatures := make(map[string]string)
	for _, semanticCase := range holdout {
		groupID := semanticCase.LeakageGroupID()
		if _, exists := caseIDs[semanticCase.CaseID()]; exists {
			t.Fatal("SOT-ENG-026: corpus-v14 内で caseId が重複しています")
		}
		caseIDs[semanticCase.CaseID()] = struct{}{}
		request := rawRequestIdentityOf(semanticCase.Request())
		if _, exists := requests[request]; exists {
			t.Fatal("SOT-ENG-026: corpus-v14 内で完全 request が重複しています")
		}
		requests[request] = struct{}{}
		comparisonKey := semanticComparisonKey(semanticCase)
		if comparisonKey != "" {
			assertRepositoryCorpusV14CollisionGroup(
				t,
				comparisonKeys,
				comparisonKey,
				groupID,
				"ComparisonKey",
			)
		}
		for _, signature := range repositoryCorpusV13MeaningSignatures(semanticCase) {
			assertRepositoryCorpusV14CollisionGroup(
				t,
				meaningSignatures,
				signature,
				groupID,
				"期待意味署名",
			)
		}
	}
}

func assertRepositoryCorpusV14CollisionGroup[T comparable](
	t *testing.T,
	seen map[T]string,
	value T,
	groupID string,
	axis string,
) {
	t.Helper()
	if previousGroup, exists := seen[value]; exists && previousGroup != groupID {
		t.Fatalf("SOT-ENG-026: corpus-v14 内の %s が異なる意味家族で重複しています", axis)
	}
	seen[value] = groupID
}

func assertRepositoryCorpusV14StableLeakageGroups(
	t *testing.T,
	holdout []SemanticCase,
) {
	t.Helper()
	for _, semanticCase := range holdout {
		groupID := semanticCase.LeakageGroupID()
		if strings.Contains(groupID, "v14") ||
			strings.Contains(groupID, "corpus") ||
			!strings.HasPrefix(groupID, "lqg-") {
			t.Fatal("SOT-ENG-026: leakageGroupId が安定分類ではありません")
		}
	}
}

func assertRepositoryCorpusV14CarryForward(
	t *testing.T,
	repositoryRoot string,
	set string,
) {
	t.Helper()
	previousRoot := filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v13", set)
	preparedRoot := filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v14", set)
	previousEntries, err := os.ReadDir(previousRoot)
	if err != nil {
		t.Fatalf("%s: corpus-v13 の集合を列挙できません: %v", corpusImmutableVersionVerificationID, err)
	}
	preparedEntries, err := os.ReadDir(preparedRoot)
	if err != nil {
		t.Fatalf("%s: corpus-v14 の集合を列挙できません: %v", corpusImmutableVersionVerificationID, err)
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
			t.Fatalf("%s: corpus-v13 の集合が byte 単位で継承されていません", corpusImmutableVersionVerificationID)
		}
	}
}
