package legalquerycorpus

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const (
	repositoryCorpusV9Seed           = 20260727
	repositoryCorpusV9Development    = 32
	repositoryCorpusV9Holdout        = 240
	repositoryCorpusV9Execution      = 8
	repositoryCorpusV9HoldoutDigest  = "c72a7a9504465f1d64a02ec9cb82d6a9faf6f382bee546c30439513f42d47557"
	repositoryCorpusV9ManifestSHA256 = "1d81c2356730b4f309955a2b411fa1813e2fd70f041fa7949c71bf6c28846c96"
)

func TestRepositoryCorpusV9は法令名と法概念の期待値を訂正する(
	t *testing.T,
) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
	previous := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v8")
	corrected := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v9")

	assertRepositoryCorpusV9Manifest(t, corrected.Manifest())
	assertRepositoryCorpusV9Correction(t, previous, corrected)
	assertRepositoryCorpusV9ChangedEntries(t, previous.Manifest(), corrected.Manifest())
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v9/manifest.json",
		repositoryCorpusV9ManifestSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v9",
		corrected,
	)
	assertRepositoryCorpusImmutable(t, corrected)
}

func assertRepositoryCorpusV9Manifest(t *testing.T, manifest Manifest) {
	t.Helper()
	if manifest.CorpusVersion() != "corpus-v9" ||
		manifest.Seed() != repositoryCorpusV9Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV9HoldoutDigest ||
		manifest.Development().CaseCount() != repositoryCorpusV9Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV9Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV9Execution {
		t.Fatalf("SOT-ENG-024: corpus-v9 の固定値が一致しません")
	}
}

func assertRepositoryCorpusV9Correction(
	t *testing.T,
	previous Corpus,
	corrected Corpus,
) {
	t.Helper()
	for _, current := range repositoryCorpusV9CorrectionCases() {
		previousMeaning := repositoryCorpusBudgetExpectedPlan(
			t,
			previous,
			current.caseID,
		).Meanings()[0]
		correctedMeaning := repositoryCorpusBudgetExpectedPlan(
			t,
			corrected,
			current.caseID,
		).Meanings()[0]
		assertRepositoryCorpusV9Meaning(
			t,
			current.caseID+" の corpus-v8 訂正前",
			previousMeaning,
			current.previousEvidence,
			current.previousConcepts,
		)
		assertRepositoryCorpusV9Meaning(
			t,
			current.caseID+" の corpus-v9 訂正後",
			correctedMeaning,
			current.correctedEvidence,
			current.correctedConcepts,
		)
	}
}

func assertRepositoryCorpusV9Meaning(
	t *testing.T,
	label string,
	meaning ExpectedMeaning,
	evidence []legalquery.EvidenceCode,
	concepts []string,
) {
	t.Helper()
	if got := meaning.EvidenceCodes(); !slices.Equal(got, evidence) {
		t.Fatalf("SOT-ENG-024: %s evidence = %#v", label, got)
	}
	if got := meaning.ConceptIDs(); !slices.Equal(got, concepts) {
		t.Fatalf("SOT-ENG-024: %s conceptIds = %#v", label, got)
	}
}

func assertRepositoryCorpusV9ChangedEntries(
	t *testing.T,
	previous Manifest,
	corrected Manifest,
) {
	t.Helper()
	if !slices.EqualFunc(
		previous.Development().Cases(),
		corrected.Development().Cases(),
		sameManifestEntry,
	) || !slices.EqualFunc(
		previous.Execution().Cases(),
		corrected.Execution().Cases(),
		sameManifestEntry,
	) {
		t.Fatal("SOT-ENG-024: development または execution fixture が変わりました")
	}
	changed := make([]string, 0)
	previousHoldout := previous.Holdout().Cases()
	correctedHoldout := corrected.Holdout().Cases()
	for index, entry := range previousHoldout {
		if !sameManifestEntry(entry, correctedHoldout[index]) {
			changed = append(changed, entry.CaseID())
		}
	}
	if want := repositoryCorpusV9CorrectionCaseIDs(); !slices.Equal(changed, want) {
		t.Fatalf("SOT-ENG-024: corpus-v9 の変更 fixture = %#v", changed)
	}
}

type repositoryCorpusV9CorrectionCase struct {
	caseID            string
	previousEvidence  []legalquery.EvidenceCode
	previousConcepts  []string
	correctedEvidence []legalquery.EvidenceCode
	correctedConcepts []string
}

func repositoryCorpusV9CorrectionCases() []repositoryCorpusV9CorrectionCase {
	explicitAlias := []legalquery.EvidenceCode{
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceOfficialAlias,
	}
	implicitAlias := []legalquery.EvidenceCode{
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceOfficialAlias,
	}
	explicitTypo := []legalquery.EvidenceCode{
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceOfficialAlias,
		legalquery.EvidenceUniqueTypoCorrection,
	}
	implicitTypo := []legalquery.EvidenceCode{
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceOfficialAlias,
		legalquery.EvidenceUniqueTypoCorrection,
	}
	conceptEvidence := []legalquery.EvidenceCode{
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceLegalConcept,
	}
	return []repositoryCorpusV9CorrectionCase{
		{
			caseID:            "holdout-name-10",
			previousEvidence:  explicitAlias,
			correctedEvidence: implicitAlias,
		},
		{
			caseID:            "holdout-name-11",
			previousEvidence:  explicitAlias,
			correctedEvidence: explicitTypo,
		},
		{
			caseID:            "holdout-name-15",
			previousEvidence:  conceptEvidence,
			previousConcepts:  []string{"permanent-residence-permission"},
			correctedEvidence: conceptEvidence,
			correctedConcepts: []string{
				"permanent-residence",
				"permanent-residence-permission",
			},
		},
		{
			caseID:            "holdout-typo-08",
			previousEvidence:  explicitTypo,
			correctedEvidence: implicitTypo,
		},
		{
			caseID:            "holdout-typo-10",
			previousEvidence:  explicitTypo,
			correctedEvidence: implicitTypo,
		},
	}
}

func repositoryCorpusV9CorrectionCaseIDs() []string {
	result := make([]string, 0, len(repositoryCorpusV9CorrectionCases()))
	for _, current := range repositoryCorpusV9CorrectionCases() {
		result = append(result, current.caseID)
	}
	return result
}
