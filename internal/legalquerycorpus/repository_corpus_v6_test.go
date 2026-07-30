package legalquerycorpus

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const (
	repositoryCorpusV6Seed           = 20260727
	repositoryCorpusV6Development    = 32
	repositoryCorpusV6Holdout        = 240
	repositoryCorpusV6Execution      = 8
	repositoryCorpusV6HoldoutDigest  = "cb845657bd4e9af4fb45892516d14c30a1dcf5d77c7444b87d918a4601ab126d"
	repositoryCorpusV6ManifestSHA256 = "d3fb83ddd974100539f613e70ba7d547ec409626d10a80a49ddb35b671a6db47"
)

func TestRepositoryCorpusV6は予算統合caseの法概念根拠期待値を訂正する(
	t *testing.T,
) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
	previous := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v5")
	corrected := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v6")
	assertRepositoryCorpusV6Manifest(t, corrected.Manifest())
	assertRepositoryCorpusV6Correction(t, previous, corrected)
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v6/manifest.json",
		repositoryCorpusV6ManifestSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v6",
		corrected,
	)
	assertRepositoryCorpusImmutable(t, corrected)
}

func assertRepositoryCorpusV6Manifest(t *testing.T, manifest Manifest) {
	t.Helper()
	if manifest.CorpusVersion() != "corpus-v6" ||
		manifest.Seed() != repositoryCorpusV6Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV6HoldoutDigest ||
		manifest.Development().CaseCount() != repositoryCorpusV6Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV6Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV6Execution {
		t.Fatalf("SOT-ENG-024: corpus-v6 の固定値が一致しません")
	}
}

func assertRepositoryCorpusV6Correction(
	t *testing.T,
	previous Corpus,
	corrected Corpus,
) {
	t.Helper()
	for _, current := range repositoryCorpusV6BudgetCorrectionCases() {
		previousPlan := repositoryCorpusBudgetExpectedPlan(
			t,
			previous,
			current.caseID,
		)
		correctedPlan := repositoryCorpusBudgetExpectedPlan(
			t,
			corrected,
			current.caseID,
		)
		if got := previousPlan.Meanings()[0].EvidenceCodes(); !slices.Equal(got, current.previousEvidence) {
			t.Fatalf("SOT-ENG-024: %s の corpus-v5 訂正前 evidence = %#v", current.caseID, got)
		}
		if got := previousPlan.Meanings()[0].ConceptIDs(); len(got) != 0 {
			t.Fatalf("SOT-ENG-024: %s の corpus-v5 訂正前 conceptIds = %#v", current.caseID, got)
		}
		correctedMeaning := correctedPlan.Meanings()[0]
		if got := correctedMeaning.EvidenceCodes(); !slices.Equal(got, current.correctedEvidence) {
			t.Fatalf("SOT-ENG-024: %s の corpus-v6 訂正後 evidence = %#v", current.caseID, got)
		}
		if got := correctedMeaning.ConceptIDs(); !slices.Equal(got, current.correctedConcepts) {
			t.Fatalf("SOT-ENG-024: %s の corpus-v6 訂正後 conceptIds = %#v", current.caseID, got)
		}
	}
}

type repositoryCorpusV6BudgetCorrectionCase struct {
	caseID            string
	previousEvidence  []legalquery.EvidenceCode
	correctedEvidence []legalquery.EvidenceCode
	correctedConcepts []string
}

func repositoryCorpusV6BudgetCorrectionCases() []repositoryCorpusV6BudgetCorrectionCase {
	return []repositoryCorpusV6BudgetCorrectionCase{
		{
			caseID: "holdout-budget-01-explicit-date",
			previousEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceOfficialAlias,
			},
			correctedEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceOfficialAlias,
				legalquery.EvidenceLegalConcept,
			},
			correctedConcepts: []string{"adult-guardianship"},
		},
		{
			caseID: "holdout-budget-02",
			previousEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			correctedEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceLegalConcept,
			},
			correctedConcepts: []string{"adult-guardianship"},
		},
		{
			caseID: "holdout-budget-11",
			previousEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			correctedEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceLegalConcept,
			},
			correctedConcepts: []string{"adult-guardianship"},
		},
		{
			caseID: "holdout-budget-12",
			previousEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			correctedEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceLegalConcept,
			},
			correctedConcepts: []string{"adult-guardianship"},
		},
		{
			caseID: "holdout-budget-13",
			previousEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			correctedEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceLegalConcept,
			},
			correctedConcepts: []string{"adult-guardianship"},
		},
		{
			caseID: "holdout-budget-14",
			previousEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			correctedEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceLegalConcept,
			},
			correctedConcepts: []string{"adult-guardianship"},
		},
		{
			caseID: "holdout-budget-15",
			previousEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceOfficialAlias,
			},
			correctedEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceLegalConcept,
			},
			correctedConcepts: []string{"adult-guardianship"},
		},
		{
			caseID: "holdout-budget-16",
			previousEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			correctedEvidence: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceLegalConcept,
			},
			correctedConcepts: []string{"adult-guardianship"},
		},
	}
}

func repositoryCorpusBudgetExpectedPlan(
	t *testing.T,
	corpus Corpus,
	caseID string,
) ExpectedPlan {
	t.Helper()
	for _, semanticCase := range corpus.Holdout() {
		if semanticCase.CaseID() != caseID {
			continue
		}
		plan, ok := semanticCase.Expected().(ExpectedPlan)
		if !ok {
			t.Fatalf("SOT-ENG-026: %s が plan ではありません", caseID)
		}
		return plan
	}
	t.Fatalf("SOT-ENG-026: %s がありません", caseID)
	return ExpectedPlan{}
}
