package legalquerycorpus

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const (
	repositoryCorpusV7Seed           = 20260727
	repositoryCorpusV7Development    = 32
	repositoryCorpusV7Holdout        = 240
	repositoryCorpusV7Execution      = 8
	repositoryCorpusV7HoldoutDigest  = "8321e4249e8fea6b64a7113cc0a63a947d485a83c01480bd74aec7f0de313bee"
	repositoryCorpusV7ManifestSHA256 = "6be50f47cc6672acb5893ff468f8e5f0fe4bfba0a18d76de2b90aebc0086e168"
)

func TestRepositoryCorpusV7は複数Stepの正式名称根拠期待値を訂正する(
	t *testing.T,
) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
	previous := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v6")
	corrected := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v7")

	manifest := corrected.Manifest()
	if manifest.CorpusVersion() != "corpus-v7" ||
		manifest.Seed() != repositoryCorpusV7Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV7HoldoutDigest ||
		manifest.Development().CaseCount() != repositoryCorpusV7Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV7Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV7Execution {
		t.Fatalf("SOT-ENG-024: corpus-v7 の固定値が一致しません")
	}

	previousMeaning := repositoryCorpusBudgetExpectedPlan(
		t,
		previous,
		"holdout-budget-15",
	).Meanings()[0]
	correctedMeaning := repositoryCorpusBudgetExpectedPlan(
		t,
		corrected,
		"holdout-budget-15",
	).Meanings()[0]
	wantPrevious := []legalquery.EvidenceCode{
		legalquery.EvidenceOfficialIdentifier,
		legalquery.EvidenceStructuredReference,
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceLegalConcept,
	}
	wantCorrected := []legalquery.EvidenceCode{
		legalquery.EvidenceOfficialIdentifier,
		legalquery.EvidenceStructuredReference,
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceOfficialAlias,
		legalquery.EvidenceLegalConcept,
	}
	if got := previousMeaning.EvidenceCodes(); !slices.Equal(got, wantPrevious) {
		t.Fatalf("SOT-ENG-024: corpus-v6 の訂正前 evidence = %#v", got)
	}
	if got := correctedMeaning.EvidenceCodes(); !slices.Equal(got, wantCorrected) {
		t.Fatalf("SOT-ENG-024: corpus-v7 の訂正後 evidence = %#v", got)
	}
	if got := correctedMeaning.ConceptIDs(); !slices.Equal(got, []string{"adult-guardianship"}) {
		t.Fatalf("SOT-ENG-024: corpus-v7 の訂正後 conceptIds = %#v", got)
	}

	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v7/manifest.json",
		repositoryCorpusV7ManifestSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v7",
		corrected,
	)
	assertRepositoryCorpusImmutable(t, corrected)
}
