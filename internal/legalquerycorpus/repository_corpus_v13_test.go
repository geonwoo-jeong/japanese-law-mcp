package legalquerycorpus

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	repositoryCorpusV13Seed           = 20260803
	repositoryCorpusV13Development    = 43
	repositoryCorpusV13Holdout        = 251
	repositoryCorpusV13Execution      = 8
	repositoryCorpusV13LeakageDigests = 204
	repositoryCorpusV13HoldoutDigest  = "7b36df431a2ebc1bfa1a2ade1357754aa723c6919079574a99add9f4eaadaadc"
	repositoryCorpusV13ManifestSHA256 = "495e8c40ff9873d097fa45365a7ca6c811740ddfc512ece912dab06699ab63c6"
	repositoryCorpusV13TreeSHA256     = "8a1a9c174f0f65d3645a90b5219209237d3efdefe8486a6f715a816b2b40a396"
)

func TestRepositoryCorpusV13は独立HoldoutとV12継承Byteを固定する(t *testing.T) {
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
	prepared := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v13")
	assertRepositoryCorpusV13Manifest(t, prepared)

	consumedHoldout := make([]SemanticCase, 0, repositoryCorpusV13Holdout*3)
	var previousHoldout []SemanticCase
	for _, consumedVersion := range consumedVersions {
		version := consumedVersion.version
		consumed := loadRepositoryCorpusVersion(t, repositoryRoot, version)
		if prepared.Manifest().HoldoutDigest() == consumed.Manifest().HoldoutDigest() {
			t.Fatal("SOT-ENG-038: 消費済み holdout digest を再利用しています")
		}
		assertNoConsumedLeakageGroupDigest(
			t,
			consumed.Manifest().HoldoutLeakageGroupDigests(),
			prepared.Manifest().HoldoutLeakageGroupDigests(),
		)
		consumedHoldout = append(consumedHoldout, consumed.Holdout()...)
		if version == "corpus-v12" {
			previousHoldout = consumed.Holdout()
		}
	}

	assertRepositoryCorpusV13IndependentHoldout(
		t,
		consumedHoldout,
		prepared.Holdout(),
	)
	assertRepositoryCorpusV13LeakagePartition(t, previousHoldout, prepared.Holdout())
	assertRepositoryCorpusV10DevelopmentAssertions(t, prepared.Development())
	assertRepositoryCorpusV10HoldoutCoverage(t, prepared.Holdout())
	assertRepositoryCorpusV13StableLeakageGroups(t, prepared.Holdout())
	assertRepositoryCorpusV13CarryForward(t, repositoryRoot, "development")
	assertRepositoryCorpusV13CarryForward(t, repositoryRoot, "execution")
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v13/manifest.json",
		repositoryCorpusV13ManifestSHA256,
	)
	assertRepositoryCorpusTreeDigest(
		t,
		filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v13"),
		repositoryCorpusV13TreeSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v13",
		prepared,
	)
	assertRepositoryCorpusImmutable(t, prepared)
}

func assertRepositoryCorpusV13Manifest(t *testing.T, corpus Corpus) {
	t.Helper()
	manifest := corpus.Manifest()
	digests := manifest.HoldoutLeakageGroupDigests()
	if manifest.SchemaVersion() != corpusSchemaVersionV2 ||
		manifest.CorpusVersion() != "corpus-v13" ||
		manifest.Seed() != repositoryCorpusV13Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV13HoldoutDigest ||
		manifest.Development().CaseCount() != repositoryCorpusV13Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV13Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV13Execution ||
		len(corpus.Development()) != repositoryCorpusV13Development ||
		len(corpus.Holdout()) != repositoryCorpusV13Holdout ||
		len(corpus.Execution()) != repositoryCorpusV13Execution ||
		len(digests) != repositoryCorpusV13LeakageDigests ||
		!slices.IsSorted(digests) {
		t.Fatal("SOT-ENG-026: corpus-v13 manifest が固定値と一致しません")
	}
	if !slices.IsSorted(repositorySemanticCaseIDs(corpus.Development())) ||
		!slices.IsSorted(repositorySemanticCaseIDs(corpus.Holdout())) ||
		!slices.IsSorted(repositoryExecutionCaseIDs(corpus.Execution())) {
		t.Fatal("SOT-ENG-026: corpus-v13 の case 順が固定されていません")
	}
}

func assertRepositoryCorpusV13LeakagePartition(
	t *testing.T,
	previous []SemanticCase,
	prepared []SemanticCase,
) {
	t.Helper()
	if len(previous) != len(prepared) {
		t.Fatal("SOT-ENG-026: corpus-v12 と corpus-v13 の holdout 件数が一致しません")
	}
	previousBySuffix := make(map[string]SemanticCase, len(previous))
	for _, semanticCase := range previous {
		suffix := strings.TrimPrefix(semanticCase.CaseID(), "holdout-v12-")
		if suffix == semanticCase.CaseID() || suffix == "" {
			t.Fatal("SOT-ENG-026: corpus-v12 の holdout caseId が対応規則に従いません")
		}
		previousBySuffix[suffix] = semanticCase
	}

	previousToPrepared := make(map[string]string)
	preparedToPrevious := make(map[string]string)
	for _, semanticCase := range prepared {
		suffix := strings.TrimPrefix(semanticCase.CaseID(), "holdout-v13-")
		previousCase, exists := previousBySuffix[suffix]
		if !exists || suffix == "" {
			t.Fatal("SOT-ENG-026: corpus-v13 の holdout caseId に対応する corpus-v12 case がありません")
		}
		previousGroup := previousCase.LeakageGroupID()
		preparedGroup := semanticCase.LeakageGroupID()
		if mapped, exists := previousToPrepared[previousGroup]; exists && mapped != preparedGroup {
			t.Fatal("SOT-ENG-026: corpus-v12 の leakage partition が corpus-v13 で分割されています")
		}
		if mapped, exists := preparedToPrevious[preparedGroup]; exists && mapped != previousGroup {
			t.Fatal("SOT-ENG-026: corpus-v12 の異なる leakage partition が corpus-v13 で統合されています")
		}
		previousToPrepared[previousGroup] = preparedGroup
		preparedToPrevious[preparedGroup] = previousGroup
		delete(previousBySuffix, suffix)
	}
	if len(previousBySuffix) != 0 {
		t.Fatal("SOT-ENG-026: corpus-v12 の holdout case に対応する corpus-v13 case がありません")
	}
}

func assertRepositoryCorpusV13IndependentHoldout(
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

func repositoryCorpusV13MeaningSignatures(semanticCase SemanticCase) []string {
	plan, ok := semanticCase.Expected().(ExpectedPlan)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(plan.Meanings()))
	for _, meaning := range plan.Meanings() {
		var signature strings.Builder
		repositoryCorpusV13AppendStrings(&signature, meaning.RequiredPacks())
		for _, step := range meaning.Steps() {
			repositoryCorpusV13AppendString(&signature, string(step.Task()))
			repositoryCorpusV13AppendString(&signature, string(step.Resource()))
			repositoryCorpusV13AppendString(&signature, string(step.InputKind()))
			repositoryCorpusV13AppendString(
				&signature,
				repositoryCorpusV13LogicalInputSignature(step.LogicalInput()),
			)
		}
		result = append(result, signature.String())
	}
	return result
}

func repositoryCorpusV13LogicalInputSignature(input legalquery.LogicalInput) string {
	var signature strings.Builder
	switch value := input.(type) {
	case legalquery.LawSearchIntentV1:
		repositoryCorpusV13AppendString(&signature, value.Query())
		date, exists := value.AsOf()
		repositoryCorpusV13AppendOptionalString(&signature, date.String(), exists)
	case legalquery.LawContentSearchIntentV1:
		repositoryCorpusV13AppendStrings(&signature, value.AllTerms())
		repositoryCorpusV13AppendStrings(&signature, value.AnyTerms())
		repositoryCorpusV13AppendStrings(&signature, value.ExcludeTerms())
		date, exists := value.AsOf()
		repositoryCorpusV13AppendOptionalString(&signature, date.String(), exists)
	case legalquery.LawReadIntentV1:
		lawID, lawIDExists := value.LawID()
		revisionID, revisionIDExists := value.RevisionID()
		date, dateExists := value.AsOf()
		ref, refExists := value.Ref()
		repositoryCorpusV13AppendOptionalString(&signature, lawID, lawIDExists)
		repositoryCorpusV13AppendOptionalString(&signature, revisionID, revisionIDExists)
		repositoryCorpusV13AppendOptionalString(&signature, date.String(), dateExists)
		repositoryCorpusV13AppendOptionalRef(&signature, ref, refExists)
	case legalquery.LawArticleReadIntentV1:
		lawID, lawIDExists := value.LawID()
		ref, refExists := value.Ref()
		location := value.Location()
		paragraph, paragraphExists := location.ParagraphNumber()
		date, dateExists := value.AsOf()
		repositoryCorpusV13AppendOptionalString(&signature, lawID, lawIDExists)
		repositoryCorpusV13AppendOptionalRef(&signature, ref, refExists)
		repositoryCorpusV13AppendString(&signature, string(location.Provision()))
		repositoryCorpusV13AppendString(&signature, location.ArticleNumber())
		repositoryCorpusV13AppendOptionalString(
			&signature,
			fmt.Sprintf("%d", paragraph),
			paragraphExists,
		)
		repositoryCorpusV13AppendOptionalString(&signature, date.String(), dateExists)
	case legalquery.LawUpdateListIntentV1:
		repositoryCorpusV13AppendString(&signature, value.Date().String())
	case legalquery.JudicialDecisionSearchIntentV1:
		repositoryCorpusV13AppendString(&signature, value.Query())
	case legalquery.JudicialDecisionReadIntentV1:
		repositoryCorpusV13AppendRef(&signature, value.Ref())
	default:
		panic("SOT-ENG-026: 未定義の logical input です")
	}
	return signature.String()
}

func repositoryCorpusV13AppendStrings(signature *strings.Builder, values []string) {
	repositoryCorpusV13AppendString(signature, fmt.Sprintf("%d", len(values)))
	for _, value := range values {
		repositoryCorpusV13AppendString(signature, value)
	}
}

func repositoryCorpusV13AppendOptionalString(
	signature *strings.Builder,
	value string,
	exists bool,
) {
	if !exists {
		repositoryCorpusV13AppendString(signature, "absent")
		return
	}
	repositoryCorpusV13AppendString(signature, "present")
	repositoryCorpusV13AppendString(signature, value)
}

func repositoryCorpusV13AppendOptionalRef(
	signature *strings.Builder,
	ref model.SourceResourceRef,
	exists bool,
) {
	if !exists {
		repositoryCorpusV13AppendString(signature, "absent")
		return
	}
	repositoryCorpusV13AppendString(signature, "present")
	repositoryCorpusV13AppendRef(signature, ref)
}

func repositoryCorpusV13AppendRef(
	signature *strings.Builder,
	ref model.SourceResourceRef,
) {
	key := ref.Key()
	versionID, versionExists := key.VersionID()
	repositoryCorpusV13AppendString(signature, ref.ProviderID())
	repositoryCorpusV13AppendString(signature, key.SourceID())
	repositoryCorpusV13AppendString(signature, key.ResourceType())
	repositoryCorpusV13AppendString(signature, key.ResourceID())
	repositoryCorpusV13AppendOptionalString(signature, versionID, versionExists)
}

func repositoryCorpusV13AppendString(signature *strings.Builder, value string) {
	fmt.Fprintf(signature, "%d:%s", len(value), value)
}

func assertRepositoryCorpusV13StableLeakageGroups(
	t *testing.T,
	holdout []SemanticCase,
) {
	t.Helper()
	for _, semanticCase := range holdout {
		groupID := semanticCase.LeakageGroupID()
		if strings.Contains(groupID, "v13") ||
			strings.Contains(groupID, "corpus") ||
			!strings.HasPrefix(groupID, "lqg-") {
			t.Fatal("SOT-ENG-026: leakageGroupId が安定分類ではありません")
		}
	}
}

func assertRepositoryCorpusV13CarryForward(
	t *testing.T,
	repositoryRoot string,
	set string,
) {
	t.Helper()
	previousRoot := filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v12", set)
	preparedRoot := filepath.Join(repositoryRoot, "testdata/legalquery/corpus-v13", set)
	previousEntries, err := os.ReadDir(previousRoot)
	if err != nil {
		t.Fatalf("%s: corpus-v12 の集合を列挙できません: %v", corpusImmutableVersionVerificationID, err)
	}
	preparedEntries, err := os.ReadDir(preparedRoot)
	if err != nil {
		t.Fatalf("%s: corpus-v13 の集合を列挙できません: %v", corpusImmutableVersionVerificationID, err)
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
			t.Fatalf("%s: corpus-v12 の集合が byte 単位で継承されていません", corpusImmutableVersionVerificationID)
		}
	}
}
