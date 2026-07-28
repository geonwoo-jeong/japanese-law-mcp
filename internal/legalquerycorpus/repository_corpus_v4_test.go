package legalquerycorpus

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const (
	repositoryCorpusV4Seed           = 20260727
	repositoryCorpusV4Development    = 31
	repositoryCorpusV4Holdout        = 240
	repositoryCorpusV4Execution      = 7
	repositoryCorpusV4HoldoutDigest  = "25b06db5e29ada2922e970a8c569ee9d2e73ce22fae4ffe3ce5880043d3543b8"
	repositoryCorpusV4ManifestSHA256 = "183a4f33c8cef09251d9c1cf4f3e29e789884d97dc5c3705c982ee703ef6df36"
)

func TestRepositoryCorpusV4は製品辞書にない前提を除く(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
	corpus, err := Load(
		context.Background(),
		repositoryRoot,
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024, SOT-ENG-026: corpus-v4 Load() error = %v", err)
	}
	manifest := corpus.Manifest()
	if manifest.CorpusVersion() != "corpus-v4" ||
		manifest.Seed() != repositoryCorpusV4Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV4HoldoutDigest {
		t.Fatalf(
			"SOT-ENG-024: corpus-v4 の固定値 = (%q, %d, %q)",
			manifest.CorpusVersion(),
			manifest.Seed(),
			manifest.HoldoutDigest(),
		)
	}
	if manifest.Development().CaseCount() != repositoryCorpusV4Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV4Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV4Execution ||
		len(corpus.Development()) != repositoryCorpusV4Development ||
		len(corpus.Holdout()) != repositoryCorpusV4Holdout ||
		len(corpus.Execution()) != repositoryCorpusV4Execution {
		t.Fatalf(
			"SOT-ENG-024: corpus-v4 の集合件数 = (%d, %d, %d)",
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
			"SOT-ENG-026: corpus-v4 に補正済み case がありません: %#v",
			expectedEvidence,
		)
	}
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v4/manifest.json",
		repositoryCorpusV4ManifestSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v4",
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
