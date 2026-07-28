package legalquerycorpus

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const (
	repositoryCorpusV3Seed           = 20260727
	repositoryCorpusV3Development    = 31
	repositoryCorpusV3Holdout        = 240
	repositoryCorpusV3Execution      = 7
	repositoryCorpusV3HoldoutDigest  = "25b06db5e29ada2922e970a8c569ee9d2e73ce22fae4ffe3ce5880043d3543b8"
	repositoryCorpusV3ManifestSHA256 = "890d0340bd34845b54b4aa87e121f3ead6aafbb38f89dcb171e5ee864e0759a0"
)

func TestRepositoryCorpusV3は開発用根拠を補完する(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
	corpus, err := Load(
		context.Background(),
		repositoryRoot,
		"testdata/legalquery/corpus-v3",
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024, SOT-ENG-026: corpus-v3 Load() error = %v", err)
	}
	manifest := corpus.Manifest()
	if manifest.CorpusVersion() != "corpus-v3" ||
		manifest.Seed() != repositoryCorpusV3Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV3HoldoutDigest {
		t.Fatalf(
			"SOT-ENG-024: corpus-v3 の固定値 = (%q, %d, %q)",
			manifest.CorpusVersion(),
			manifest.Seed(),
			manifest.HoldoutDigest(),
		)
	}
	if manifest.Development().CaseCount() != repositoryCorpusV3Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV3Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV3Execution ||
		len(corpus.Development()) != repositoryCorpusV3Development ||
		len(corpus.Holdout()) != repositoryCorpusV3Holdout ||
		len(corpus.Execution()) != repositoryCorpusV3Execution {
		t.Fatalf(
			"SOT-ENG-024: corpus-v3 の集合件数 = (%d, %d, %d)",
			len(corpus.Development()),
			len(corpus.Holdout()),
			len(corpus.Execution()),
		)
	}

	expectedEvidence := map[string][]legalquery.EvidenceCode{
		"development-structure-article-grounded": {
			legalquery.EvidenceStructuredReference,
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
			legalquery.EvidenceOfficialAlias,
		},
		"development-typo-deletion-grounded": {
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
			legalquery.EvidenceOfficialAlias,
			legalquery.EvidenceUniqueTypoCorrection,
		},
		"development-typo-substitution-grounded": {
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
			legalquery.EvidenceOfficialAlias,
			legalquery.EvidenceUniqueTypoCorrection,
		},
	}
	for _, fixture := range corpus.Development() {
		want, exists := expectedEvidence[fixture.CaseID()]
		if !exists {
			continue
		}
		plan, ok := fixture.Expected().(ExpectedPlan)
		if !ok || len(plan.Meanings()) != 1 {
			t.Fatalf(
				"SOT-ENG-026: %s の期待値は一つの plan meaning ではありません",
				fixture.CaseID(),
			)
		}
		if got := plan.Meanings()[0].EvidenceCodes(); !slices.Equal(got, want) {
			t.Fatalf(
				"SOT-MODEL-022: %s の evidenceCodes = %#v, want %#v",
				fixture.CaseID(),
				got,
				want,
			)
		}
		delete(expectedEvidence, fixture.CaseID())
	}
	if len(expectedEvidence) != 0 {
		t.Fatalf(
			"SOT-ENG-026: corpus-v3 に補正済み case がありません: %#v",
			expectedEvidence,
		)
	}
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v3/manifest.json",
		repositoryCorpusV3ManifestSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v3",
		corpus,
	)
	assertRepositoryCorpusImmutable(t, corpus)
	for _, fixture := range corpus.Development() {
		assertExpectedDatesAreGrounded(t, fixture)
		assertOfficialIdentifierEvidenceIsGrounded(t, fixture)
	}
	for _, fixture := range corpus.Holdout() {
		assertExpectedDatesAreGrounded(t, fixture)
		assertOfficialIdentifierEvidenceIsGrounded(t, fixture)
	}
}
