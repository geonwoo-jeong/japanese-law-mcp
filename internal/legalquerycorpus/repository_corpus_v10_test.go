package legalquerycorpus

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

const (
	repositoryCorpusV10Seed           = 20260727
	repositoryCorpusV10Development    = 43
	repositoryCorpusV10Holdout        = 251
	repositoryCorpusV10Execution      = 8
	repositoryCorpusV10LeakageDigests = 228
	repositoryCorpusV10HoldoutDigest  = "7d85f11321883cf13d1369cd1f93d77709b8524ca5854a08818385faad041213"
	repositoryCorpusV10ManifestSHA256 = "1f27583f9d63571ee4804dac7cb009534a560df3c3e8b40d4b82eb4267702044"
	repositoryCorpusV9TreeSHA256      = "ffb6dc8d4bd264e8120bd280aefbae80a0fe240dcc5ac96f7d727ea4dece6b31"
	repositoryCorpusV10TreeSHA256     = "ce255dcdb6fa08491671fdbd8fc6f00f284d9ed31cf4f1acee6bfefbb60ca2b6"
	repositoryCorpusV1SchemaSHA256    = "37c89587f6bd93b3f57e6ee0530f24d21c6ddcd0a9ac852753775f84827dc78c"
	repositoryCorpusV2SchemaSHA256    = "269c1e5c74029243cea62c9138d73ebbc7b8b987ff93e26eb74f81d2c5db0add"
)

func TestRepositoryCorpusV10はV2契約と不変Byteを固定する(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
	corpus := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v10")
	manifest := corpus.Manifest()
	assertRepositoryCorpusV10Manifest(t, manifest)
	assertRepositoryCorpusV10DevelopmentAssertions(t, corpus.Development())
	assertRepositoryCorpusV10HoldoutCoverage(t, corpus.Holdout())
	assertRepositoryCorpusV10CarryForward(t, repositoryRoot)
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v10/manifest.json",
		repositoryCorpusV10ManifestSHA256,
	)
	assertRepositoryCorpusTreeDigest(
		t,
		filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v9"),
		repositoryCorpusV9TreeSHA256,
	)
	assertRepositoryCorpusTreeDigest(
		t,
		filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v10"),
		repositoryCorpusV10TreeSHA256,
	)
	assertRepositoryCorpusFileDigest(
		t,
		filepath.Join(repositoryRoot, "testdata/legalquery/schemas", corpusSchemaV1Filename),
		repositoryCorpusV1SchemaSHA256,
	)
	assertRepositoryCorpusFileDigest(
		t,
		filepath.Join(repositoryRoot, "testdata/legalquery/schemas", corpusSchemaV2Filename),
		repositoryCorpusV2SchemaSHA256,
	)
	assertRepositoryCorpusImmutable(t, corpus)
}

func assertRepositoryCorpusV10Manifest(t *testing.T, manifest Manifest) {
	t.Helper()
	digests := manifest.HoldoutLeakageGroupDigests()
	if manifest.SchemaVersion() != corpusSchemaVersionV2 ||
		manifest.CorpusVersion() != "corpus-v10" ||
		manifest.Seed() != repositoryCorpusV10Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV10HoldoutDigest ||
		manifest.Development().CaseCount() != repositoryCorpusV10Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV10Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV10Execution ||
		len(digests) != repositoryCorpusV10LeakageDigests ||
		!slices.IsSorted(digests) {
		t.Fatalf("%s: corpus-v10 manifest が固定値と一致しません", corpusImmutableVersionVerificationID)
	}
}

func assertRepositoryCorpusV10DevelopmentAssertions(
	t *testing.T,
	development []SemanticCase,
) {
	t.Helper()
	present := make(map[string]int)
	for _, semanticCase := range development {
		for _, assertionID := range semanticCase.DevelopmentAssertionIDs() {
			present[assertionID]++
		}
	}
	for _, required := range manifestRequiredDevelopmentAssertionIDs() {
		if present[required] != 1 {
			t.Fatalf(
				"%s: assertion %q の件数 = %d",
				corpusV2DevelopmentAssertionsVerificationID,
				required,
				present[required],
			)
		}
	}
}

func assertRepositoryCorpusV10HoldoutCoverage(
	t *testing.T,
	holdout []SemanticCase,
) {
	t.Helper()
	counts := collectHoldoutRequirementCounts(holdout)
	for _, categoryID := range semanticCategoryIDs() {
		if counts.categories[categoryID] < minimumHoldoutCategoryCaseCount {
			t.Fatalf(
				"%s: category %q の件数 = %d",
				corpusV2HoldoutCoverageVerificationID,
				categoryID,
				counts.categories[categoryID],
			)
		}
	}
	for _, definition := range semanticCoverageDefinitionsForSchemaVersion(
		corpusSchemaVersionV2,
	) {
		if counts.coverages[definition.id] < definition.minimumHoldoutCount {
			t.Fatalf(
				"%s: coverage %q の件数 = %d",
				corpusV2HoldoutCoverageVerificationID,
				definition.id,
				counts.coverages[definition.id],
			)
		}
		if !definition.requiresSafetyVariantPair {
			continue
		}
		for _, variant := range []SafetyVariant{
			SafetyVariantOrdinary,
			SafetyVariantAdversarial,
		} {
			if counts.safetyVariants[holdoutSafetyVariantKey{
				coverageID: definition.id,
				variant:    variant,
			}] < 1 {
				t.Fatalf(
					"%s: safety coverage %q の variant %q がありません",
					corpusV2HoldoutCoverageVerificationID,
					definition.id,
					variant,
				)
			}
		}
	}
}

func assertRepositoryCorpusV10CarryForward(t *testing.T, repositoryRoot string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(
		repositoryRoot,
		"testdata/legalquery/corpus-v9/holdout/*.json",
	))
	if err != nil || len(paths) != repositoryCorpusV9Holdout {
		t.Fatalf("%s: corpus-v9 holdout を列挙できません", corpusImmutableVersionVerificationID)
	}
	for _, previousPath := range paths {
		currentPath := filepath.Join(
			repositoryRoot,
			"testdata/legalquery/corpus-v10/holdout",
			filepath.Base(previousPath),
		)
		previous := repositoryCorpusJSONWithoutVersion(t, previousPath)
		current := repositoryCorpusJSONWithoutVersion(t, currentPath)
		if !reflect.DeepEqual(previous, current) {
			t.Fatalf(
				"%s: 既存 holdout %q の意味内容が変更されました",
				corpusImmutableVersionVerificationID,
				filepath.Base(previousPath),
			)
		}
	}
}

func repositoryCorpusJSONWithoutVersion(
	t *testing.T,
	path string,
) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("SOT-ENG-026: fixture を読めません: %v", err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("SOT-ENG-026: fixture JSON を解釈できません: %v", err)
	}
	delete(value, "schemaVersion")
	return value
}

func assertRepositoryCorpusFileDigest(
	t *testing.T,
	path string,
	expected string,
) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: 固定 file を読めません: %v", corpusImmutableVersionVerificationID, err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != expected {
		t.Fatalf("%s: SHA-256 = %q", corpusImmutableVersionVerificationID, got)
	}
}

func assertRepositoryCorpusTreeDigest(
	t *testing.T,
	root string,
	expected string,
) {
	t.Helper()
	if got := repositoryCorpusTreeDigest(t, root); got != expected {
		t.Fatalf("%s: corpus tree SHA-256 = %q", corpusImmutableVersionVerificationID, got)
	}
}

func repositoryCorpusTreeDigest(t *testing.T, root string) string {
	t.Helper()
	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type().IsRegular() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("%s: corpus tree を列挙できません: %v", corpusImmutableVersionVerificationID, err)
	}
	slices.Sort(paths)
	input := make([]byte, 0)
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatalf("%s: corpus path を相対化できません: %v", corpusImmutableVersionVerificationID, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: corpus file を読めません: %v", corpusImmutableVersionVerificationID, err)
		}
		input = append(input, filepath.ToSlash(relative)...)
		input = append(input, 0)
		input = append(input, data...)
		input = append(input, 0)
	}
	return fmt.Sprintf("%x", sha256.Sum256(input))
}
