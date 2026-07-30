package legalquerycorpus

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const (
	repositoryCorpusV8Seed           = 20260727
	repositoryCorpusV8Development    = 32
	repositoryCorpusV8Holdout        = 240
	repositoryCorpusV8Execution      = 8
	repositoryCorpusV8HoldoutDigest  = "3791b6a237a89a020566fea10758f7e2afb5f11b894398dd218ba5f79ac2514d"
	repositoryCorpusV8ManifestSHA256 = "b3ea555b923dc10f470c6eb7b61c7862b34a8ad3f9051c1652ba166ee4c50e98"
)

func TestRepositoryCorpusV8は独立した形態素検索根拠の期待値を訂正する(
	t *testing.T,
) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
	previous := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v7")
	corrected := loadRepositoryCorpusVersion(t, repositoryRoot, "corpus-v8")

	assertRepositoryCorpusV8Manifest(t, corrected.Manifest())
	assertRepositoryCorpusV8Correction(t, previous, corrected)
	assertRepositoryCorpusV8ChangedEntries(t, previous.Manifest(), corrected.Manifest())
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v8/manifest.json",
		repositoryCorpusV8ManifestSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v8",
		corrected,
	)
	assertRepositoryCorpusImmutable(t, corrected)
}

func assertRepositoryCorpusV8Manifest(t *testing.T, manifest Manifest) {
	t.Helper()
	if manifest.CorpusVersion() != "corpus-v8" ||
		manifest.Seed() != repositoryCorpusV8Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV8HoldoutDigest ||
		manifest.Development().CaseCount() != repositoryCorpusV8Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV8Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV8Execution {
		t.Fatalf("SOT-ENG-024: corpus-v8 の固定値が一致しません")
	}
}

func assertRepositoryCorpusV8Correction(
	t *testing.T,
	previous Corpus,
	corrected Corpus,
) {
	t.Helper()
	for _, current := range repositoryCorpusV8CorrectionCases() {
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
		if got := previousMeaning.EvidenceCodes(); !slices.Equal(got, current.previous) {
			t.Fatalf(
				"SOT-ENG-024: %s の corpus-v7 訂正前 evidence = %#v",
				current.caseID,
				got,
			)
		}
		if got := correctedMeaning.EvidenceCodes(); !slices.Equal(got, current.corrected) {
			t.Fatalf(
				"SOT-ENG-024: %s の corpus-v8 訂正後 evidence = %#v",
				current.caseID,
				got,
			)
		}
	}
}

func assertRepositoryCorpusV8ChangedEntries(
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
	if want := repositoryCorpusV8CorrectionCaseIDs(); !slices.Equal(changed, want) {
		t.Fatalf("SOT-ENG-024: corpus-v8 の変更 fixture = %#v", changed)
	}
}

func sameManifestEntry(left ManifestEntry, right ManifestEntry) bool {
	return left.CaseID() == right.CaseID() && left.SHA256() == right.SHA256()
}

type repositoryCorpusV8CorrectionCase struct {
	caseID    string
	previous  []legalquery.EvidenceCode
	corrected []legalquery.EvidenceCode
}

func repositoryCorpusV8CorrectionCases() []repositoryCorpusV8CorrectionCase {
	return []repositoryCorpusV8CorrectionCase{
		{
			caseID: "holdout-budget-06-explicit-date",
			previous: []legalquery.EvidenceCode{
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceOfficialAlias,
			},
			corrected: []legalquery.EvidenceCode{
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceOfficialAlias,
				legalquery.EvidenceMorphologicalContext,
			},
		},
		{
			caseID: "holdout-intent-16",
			previous: []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			corrected: []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceMorphologicalContext,
			},
		},
		{
			caseID: "holdout-intent-17",
			previous: []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			corrected: []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceMorphologicalContext,
			},
		},
		{
			caseID: "holdout-intent-18",
			previous: []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			corrected: []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceMorphologicalContext,
			},
		},
		{
			caseID: "holdout-structure-03",
			previous: []legalquery.EvidenceCode{
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			corrected: []legalquery.EvidenceCode{
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceMorphologicalContext,
			},
		},
	}
}

func repositoryCorpusV8CorrectionCaseIDs() []string {
	result := make([]string, 0, len(repositoryCorpusV8CorrectionCases()))
	for _, current := range repositoryCorpusV8CorrectionCases() {
		result = append(result, current.caseID)
	}
	return result
}
