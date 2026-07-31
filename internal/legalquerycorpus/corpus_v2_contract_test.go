package legalquerycorpus

import (
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const (
	corpusV2DevelopmentAssertionsVerificationID = "legal-query-corpus-v2-development-assertions"
	corpusV2HoldoutCoverageVerificationID       = "legal-query-corpus-v2-holdout-coverage"
	corpusV2LeakageDigestsVerificationID        = "legal-query-corpus-v2-leakage-digests"
	corpusImmutableVersionVerificationID        = "legal-query-corpus-immutable-version"
)

func TestCorpusV2DevelopmentAssertionsは固定十一件を版別に保持する(
	t *testing.T,
) {
	want := []string{
		"shared-terminal-five-distinct-meanings",
		"shared-terminal-five-with-repeated-meaning",
		"shared-terminal-four-distinct-meanings",
		"shared-terminal-nonunique-maximal-sequence",
		"shared-terminal-repeated-meaning-different-span",
		"shared-terminal-repeated-separator",
		"shared-terminal-same-span-multiple-meanings",
		"shared-terminal-tail-connector-chain",
		"shared-terminal-unknown-separator",
		"shared-terminal-valid-maximal-no-subsequence",
		"shared-terminal-valid-separator",
	}
	if got := manifestRequiredDevelopmentAssertionIDs(); !slices.Equal(got, want) {
		t.Fatalf(
			"%s: requiredDevelopmentAssertionIds = %#v",
			corpusV2DevelopmentAssertionsVerificationID,
			got,
		)
	}

	values := validSemanticCaseValues(t)
	values.SchemaVersion = corpusSchemaVersionV2
	values.DevelopmentAssertionIDs = []string{want[0]}
	semanticCase, err := NewSemanticCase(values)
	if err != nil {
		t.Fatalf(
			"%s: v2 development case を作成できません: %v",
			corpusV2DevelopmentAssertionsVerificationID,
			err,
		)
	}
	got := semanticCase.DevelopmentAssertionIDs()
	got[0] = "changed"
	if !slices.Equal(semanticCase.DevelopmentAssertionIDs(), want[:1]) {
		t.Fatalf(
			"%s: getter から assertion ID が変更されました",
			corpusV2DevelopmentAssertionsVerificationID,
		)
	}

	values.SchemaVersion = corpusSchemaVersionV1
	if _, err := NewSemanticCase(values); err == nil {
		t.Fatalf(
			"%s: v1 case が developmentAssertionIds を受理しました",
			corpusV2DevelopmentAssertionsVerificationID,
		)
	}
	values.SchemaVersion = corpusSchemaVersionV2
	values.CaseID = "holdout-development-assertion"
	if _, err := NewSemanticCase(values); err == nil {
		t.Fatalf(
			"%s: holdout case が developmentAssertionIds を受理しました",
			corpusV2DevelopmentAssertionsVerificationID,
		)
	}
}

func TestCorpusV2HoldoutCoverageは追加七件を版別に保持する(t *testing.T) {
	want := map[string]semanticCoverageDefinition{
		"boundary-no-unbounded-fanout": {
			id:                        "boundary-no-unbounded-fanout",
			categoryID:                semanticCategorySafetyExecutionBoundary,
			minimumHoldoutCount:       2,
			requiresSafetyVariantPair: true,
		},
		"boundary-unmarked-enumeration": {
			id:                        "boundary-unmarked-enumeration",
			categoryID:                semanticCategorySafetyExecutionBoundary,
			minimumHoldoutCount:       2,
			requiresSafetyVariantPair: true,
		},
		"boundary-unsupported-candidate-scope": {
			id:                        "boundary-unsupported-candidate-scope",
			categoryID:                semanticCategorySafetyExecutionBoundary,
			minimumHoldoutCount:       2,
			requiresSafetyVariantPair: true,
		},
		"boundary-unsupported-cue-context": {
			id:                        "boundary-unsupported-cue-context",
			categoryID:                semanticCategorySafetyExecutionBoundary,
			minimumHoldoutCount:       2,
			requiresSafetyVariantPair: true,
		},
		"structure-shared-terminal-cue": {
			id:                  "structure-shared-terminal-cue",
			categoryID:          semanticCategoryStructuredLocationAndDate,
			minimumHoldoutCount: 1,
		},
		"unsupported-relationship-analysis": {
			id:                  "unsupported-relationship-analysis",
			categoryID:          semanticCategoryUnsupportedScope,
			minimumHoldoutCount: 1,
		},
		"unsupported-version-comparison": {
			id:                  "unsupported-version-comparison",
			categoryID:          semanticCategoryUnsupportedScope,
			minimumHoldoutCount: 1,
		},
	}

	v1 := semanticCoverageDefinitionsForSchemaVersion(corpusSchemaVersionV1)
	v2 := semanticCoverageDefinitionsForSchemaVersion(corpusSchemaVersionV2)
	if len(v2) != len(v1)+len(want) {
		t.Fatalf(
			"%s: coverage 件数 = (%d, %d)",
			corpusV2HoldoutCoverageVerificationID,
			len(v1),
			len(v2),
		)
	}
	for coverageID, expected := range want {
		if slices.ContainsFunc(v1, func(current semanticCoverageDefinition) bool {
			return current.id == coverageID
		}) {
			t.Fatalf(
				"%s: v1 が v2 coverage %q を保持しています",
				corpusV2HoldoutCoverageVerificationID,
				coverageID,
			)
		}
		index := slices.IndexFunc(v2, func(current semanticCoverageDefinition) bool {
			return current.id == coverageID
		})
		if index < 0 || !reflect.DeepEqual(v2[index], expected) {
			t.Fatalf(
				"%s: v2 coverage %q = %#v",
				corpusV2HoldoutCoverageVerificationID,
				coverageID,
				v2[index],
			)
		}
	}
}

func TestCorpusV2LeakageDigestsはIDを重複排除してByte順に投影する(
	t *testing.T,
) {
	cases := []SemanticCase{
		corpusV2LeakageDigestCase(t, 1, "leakage-z"),
		corpusV2LeakageDigestCase(t, 2, "leakage-a"),
		corpusV2LeakageDigestCase(t, 3, "leakage-z"),
	}
	want := []string{
		sha256Hex([]byte("legal-query-leakage-group-v1\x00leakage-z")),
		sha256Hex([]byte("legal-query-leakage-group-v1\x00leakage-a")),
	}
	slices.Sort(want)
	got := computeHoldoutLeakageGroupDigests(cases)
	if !slices.Equal(got, want) {
		t.Fatalf(
			"%s: digest 集合 = %#v",
			corpusV2LeakageDigestsVerificationID,
			got,
		)
	}
}

func TestCorpusV2DevelopmentAssertionsは全必須IDの存在を検証する(
	t *testing.T,
) {
	required := manifestRequiredDevelopmentAssertionIDs()
	development := make([]SemanticCase, 0, len(required))
	for index, assertionID := range required {
		values := validSemanticCaseValues(t)
		values.SchemaVersion = corpusSchemaVersionV2
		values.CaseID = fmt.Sprintf("development-v2-assertion-%02d", index)
		values.LeakageGroupID = fmt.Sprintf("development-v2-assertion-%02d", index)
		values.DevelopmentAssertionIDs = []string{assertionID}
		semanticCase, err := NewSemanticCase(values)
		if err != nil {
			t.Fatalf("%s: assertion fixture error = %v", corpusV2DevelopmentAssertionsVerificationID, err)
		}
		development = append(development, semanticCase)
	}
	checked := integrityCheckedCorpus{
		manifest:    corpusV2ContractManifest(t, []string{strings.Repeat("0", 64)}),
		development: development,
	}
	if _, err := validateIntegrityDevelopmentAssertions(checked); err != nil {
		t.Fatalf("%s: 全 assertion ID を拒否しました: %v", corpusV2DevelopmentAssertionsVerificationID, err)
	}
	checked.development = checked.development[:len(checked.development)-1]
	if _, err := validateIntegrityDevelopmentAssertions(checked); err == nil {
		t.Fatalf("%s: assertion ID の不足を受理しました", corpusV2DevelopmentAssertionsVerificationID)
	}
}

func TestCorpusV2StepLimitだけが空Meaningの明確化を許す(t *testing.T) {
	if _, err := NewExpectedPlan(ExpectedPlanValues{
		Decision:    legalquery.PlanDecisionNeedsClarification,
		ReasonCodes: []legalquery.ReasonCode{legalquery.ReasonCodeBelowExecutionThreshold},
	}); err == nil {
		t.Fatalf("%s: 通常の明確化が空 meanings を受理しました", corpusV2DevelopmentAssertionsVerificationID)
	}
	if _, err := NewExpectedPlan(ExpectedPlanValues{
		Decision:    legalquery.PlanDecisionNeedsClarification,
		ReasonCodes: []legalquery.ReasonCode{legalquery.ReasonCodeStepLimitExceeded},
	}); err != nil {
		t.Fatalf("%s: step_limit_exceeded の空 meanings を拒否しました: %v", corpusV2DevelopmentAssertionsVerificationID, err)
	}
}

func TestCorpusV2HoldoutCoverageは最小十一件の追加を要求する(t *testing.T) {
	holdout := corpusV2ValidHoldout(t)
	if err := validateHoldoutRequirementsForSchemaVersion(
		corpusSchemaVersionV2,
		holdout,
	); err != nil {
		t.Fatalf("%s: v2 最小 holdout を拒否しました: %v", corpusV2HoldoutCoverageVerificationID, err)
	}
	if err := validateHoldoutRequirementsForSchemaVersion(
		corpusSchemaVersionV2,
		holdout[:len(holdout)-1],
	); err == nil {
		t.Fatalf("%s: 追加 coverage の不足を受理しました", corpusV2HoldoutCoverageVerificationID)
	}
}

func TestCorpusV2HoldoutCoverageはCategory最小件数とSafety対を個別に要求する(
	t *testing.T,
) {
	counts := collectHoldoutRequirementCounts(corpusV2ValidHoldout(t))
	counts.categories[semanticCategoryAmbiguity] = minimumHoldoutCategoryCaseCount - 1
	if err := validateHoldoutCategoryCounts(counts.categories); err == nil {
		t.Fatalf("%s: category 最小件数の不足を受理しました", corpusV2HoldoutCoverageVerificationID)
	}

	counts = collectHoldoutRequirementCounts(corpusV2ValidHoldout(t))
	delete(counts.safetyVariants, holdoutSafetyVariantKey{
		coverageID: "boundary-no-unbounded-fanout",
		variant:    SafetyVariantOrdinary,
	})
	if err := validateHoldoutCoverageCounts(
		corpusSchemaVersionV2,
		counts.coverages,
	); err != nil {
		t.Fatalf("%s: safety 対以外の coverage 条件が崩れました: %v", corpusV2HoldoutCoverageVerificationID, err)
	}
	if err := validateHoldoutSafetyVariantPairs(
		corpusSchemaVersionV2,
		counts.safetyVariants,
	); err == nil {
		t.Fatalf("%s: safety の ordinary 欠落を受理しました", corpusV2HoldoutCoverageVerificationID)
	}
}

func corpusV2ValidHoldout(t *testing.T) []SemanticCase {
	t.Helper()
	base := holdoutRequirementsTestValidHoldout(t)
	holdout := make([]SemanticCase, 0, len(base)+11)
	for _, source := range base {
		var safetyVariant *SafetyVariant
		if value, exists := source.SafetyVariant(); exists {
			safetyVariant = &value
		}
		semanticCase, err := NewSemanticCase(SemanticCaseValues{
			ArtifactKind:   source.ArtifactKind(),
			SchemaVersion:  corpusSchemaVersionV2,
			CaseID:         source.CaseID(),
			LeakageGroupID: source.LeakageGroupID(),
			CoverageIDs:    source.CoverageIDs(),
			SafetyVariant:  safetyVariant,
			EnabledPacks:   source.EnabledPacks(),
			Request:        source.Request(),
			Expected:       source.Expected(),
		})
		if err != nil {
			t.Fatalf("%s: v1 carry-forward error = %v", corpusV2HoldoutCoverageVerificationID, err)
		}
		holdout = append(holdout, semanticCase)
	}
	for index, addition := range corpusV2AdditionalCoverageCases() {
		holdout = append(holdout, corpusV2CoverageCase(
			t,
			240+index,
			addition.coverageID,
			addition.variant,
		))
	}
	return holdout
}

func TestCorpusV2LeakageDigestsはManifestとの不一致で原IDを隠す(
	t *testing.T,
) {
	const secretLeakageID = "secret-holdout-leakage-group"
	holdout := []SemanticCase{
		corpusV2LeakageDigestCase(t, 4, secretLeakageID),
	}
	valid := corpusV2ContractManifest(
		t,
		computeHoldoutLeakageGroupDigests(holdout),
	)
	if err := validateManifestHoldoutLeakageGroupDigests(valid, holdout); err != nil {
		t.Fatalf("%s: 正しい digest 集合を拒否しました: %v", corpusV2LeakageDigestsVerificationID, err)
	}
	mismatch := corpusV2ContractManifest(t, []string{strings.Repeat("0", 64)})
	err := validateManifestHoldoutLeakageGroupDigests(mismatch, holdout)
	if err == nil {
		t.Fatalf("%s: digest 不一致を受理しました", corpusV2LeakageDigestsVerificationID)
	}
	if strings.Contains(err.Error(), secretLeakageID) {
		t.Fatalf("%s: error が raw leakageGroupId を含みます: %v", corpusV2LeakageDigestsVerificationID, err)
	}
}

func TestCorpusVersionは参照済みV9を変更せずV10を別版にする(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v9/manifest.json",
		repositoryCorpusV9ManifestSHA256,
	)

	prepared := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v10")
	if prepared.Manifest().SchemaVersion() != corpusSchemaVersionV2 ||
		prepared.Manifest().CorpusVersion() != "corpus-v10" {
		t.Fatalf(
			"%s: corpus-v10 の版境界が一致しません",
			corpusImmutableVersionVerificationID,
		)
	}
}

func corpusV2LeakageDigestCase(
	t *testing.T,
	index int,
	leakageGroupID string,
) SemanticCase {
	t.Helper()
	values := validSemanticCaseValues(t)
	values.SchemaVersion = corpusSchemaVersionV2
	values.CaseID = fmt.Sprintf("holdout-leakage-digest-%02d", index)
	values.LeakageGroupID = leakageGroupID
	semanticCase, err := NewSemanticCase(values)
	if err != nil {
		t.Fatalf(
			"%s: leakage fixture を作成できません: %v",
			corpusV2LeakageDigestsVerificationID,
			err,
		)
	}
	return semanticCase
}

type corpusV2AdditionalCoverageCase struct {
	coverageID string
	variant    SafetyVariant
}

func corpusV2AdditionalCoverageCases() []corpusV2AdditionalCoverageCase {
	return []corpusV2AdditionalCoverageCase{
		{coverageID: "boundary-no-unbounded-fanout", variant: SafetyVariantOrdinary},
		{coverageID: "boundary-no-unbounded-fanout", variant: SafetyVariantAdversarial},
		{coverageID: "boundary-unmarked-enumeration", variant: SafetyVariantOrdinary},
		{coverageID: "boundary-unmarked-enumeration", variant: SafetyVariantAdversarial},
		{coverageID: "boundary-unsupported-candidate-scope", variant: SafetyVariantOrdinary},
		{coverageID: "boundary-unsupported-candidate-scope", variant: SafetyVariantAdversarial},
		{coverageID: "boundary-unsupported-cue-context", variant: SafetyVariantOrdinary},
		{coverageID: "boundary-unsupported-cue-context", variant: SafetyVariantAdversarial},
		{coverageID: "structure-shared-terminal-cue"},
		{coverageID: "unsupported-relationship-analysis"},
		{coverageID: "unsupported-version-comparison"},
	}
}

func corpusV2CoverageCase(
	t *testing.T,
	index int,
	coverageID string,
	variant SafetyVariant,
) SemanticCase {
	t.Helper()
	values := validSemanticCaseValues(t)
	values.SchemaVersion = corpusSchemaVersionV2
	values.CaseID = fmt.Sprintf("holdout-v2-coverage-%03d", index)
	values.LeakageGroupID = fmt.Sprintf("holdout-v2-coverage-%03d", index)
	values.CoverageIDs = []string{coverageID}
	if variant != "" {
		values.SafetyVariant = &variant
	}
	semanticCase, err := NewSemanticCase(values)
	if err != nil {
		t.Fatalf("%s: coverage fixture error = %v", corpusV2HoldoutCoverageVerificationID, err)
	}
	return semanticCase
}

func corpusV2ContractManifest(t *testing.T, leakageDigests []string) Manifest {
	t.Helper()
	sets := make([]ManifestSet, 0, 3)
	for _, kind := range manifestSetKinds() {
		set, err := NewManifestSet(ManifestSetValues{Kind: kind})
		if err != nil {
			t.Fatalf("SOT-ENG-026: 空の manifest set を作成できません: %v", err)
		}
		sets = append(sets, set)
	}
	manifest, err := NewManifest(ManifestValues{
		ArtifactKind:                    ArtifactKindCorpusManifest,
		SchemaVersion:                   corpusSchemaVersionV2,
		CorpusVersion:                   "corpus-v10",
		Seed:                            1,
		HoldoutDigest:                   strings.Repeat("f", 64),
		HoldoutLeakageGroupDigests:      leakageDigests,
		RequiredCategoryIDs:             manifestRequiredCategoryIDs(),
		RequiredExecutionScenarioIDs:    manifestRequiredExecutionScenarioIDs(),
		RequiredDevelopmentAssertionIDs: manifestRequiredDevelopmentAssertionIDs(),
		Development:                     sets[0],
		Holdout:                         sets[1],
		Execution:                       sets[2],
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: v2 manifest を作成できません: %v", err)
	}
	return manifest
}
