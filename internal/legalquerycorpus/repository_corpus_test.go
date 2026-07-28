package legalquerycorpus

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	repositoryCorpusV1Seed           = 20260727
	repositoryCorpusV1Development    = 31
	repositoryCorpusV1Holdout        = 240
	repositoryCorpusV1Execution      = 7
	repositoryCorpusV1HoldoutDigest  = "5b909cc6d80a5d94664b7598b8824ebfc30ed39151172084b235dc210f0ab2ac"
	repositoryCorpusV1ManifestSHA256 = "db0e8c07ed7b00b6e8d9fa90783d30fae130a24943ae0c06588fb7cee988663c"
)

func TestRepositoryCorpusV1は公開Loaderですべて検証できる(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))

	const corpusDirectory = "testdata/legalquery/corpus-v1"
	corpus, err := Load(
		context.Background(),
		repositoryRoot,
		corpusDirectory,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024, SOT-ENG-026: corpus-v1 Load() error = %v", err)
	}
	if err := corpus.Validate(); err != nil {
		t.Fatalf("SOT-ENG-026: corpus-v1 Validate() error = %v", err)
	}
	if corpus.Manifest().CorpusVersion() != "corpus-v1" {
		t.Fatalf(
			"SOT-ENG-026: corpusVersion = %q",
			corpus.Manifest().CorpusVersion(),
		)
	}
	manifest := corpus.Manifest()
	if manifest.Seed() != repositoryCorpusV1Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV1HoldoutDigest {
		t.Fatalf(
			"SOT-ENG-024: corpus-v1 の固定値 = (%d, %q)",
			manifest.Seed(),
			manifest.HoldoutDigest(),
		)
	}
	if manifest.Development().CaseCount() != repositoryCorpusV1Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV1Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV1Execution ||
		len(corpus.Development()) != repositoryCorpusV1Development ||
		len(corpus.Holdout()) != repositoryCorpusV1Holdout ||
		len(corpus.Execution()) != repositoryCorpusV1Execution {
		t.Fatalf(
			"SOT-ENG-024: corpus-v1 の集合件数 = (%d, %d, %d)",
			len(corpus.Development()),
			len(corpus.Holdout()),
			len(corpus.Execution()),
		)
	}
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		corpusDirectory,
		corpus,
	)
	assertRepositoryCorpusImmutable(t, corpus)
}

func assertRepositoryCorpusManifestDigest(t *testing.T, repositoryRoot string) {
	t.Helper()

	data, err := fs.ReadFile(
		os.DirFS(repositoryRoot),
		"testdata/legalquery/corpus-v1/manifest.json",
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024: manifest の固定 byte を読めません: %v", err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	if digest != repositoryCorpusV1ManifestSHA256 {
		t.Fatalf("SOT-ENG-024: manifest SHA-256 = %q", digest)
	}
}

func assertRepositoryCorpusReproducible(
	t *testing.T,
	repositoryRoot string,
	corpusDirectory string,
	first Corpus,
) {
	t.Helper()

	second, err := Load(context.Background(), repositoryRoot, corpusDirectory)
	if err != nil {
		t.Fatalf("SOT-ENG-024: 二回目の Load() error = %v", err)
	}
	if second.Manifest().HoldoutDigest() != first.Manifest().HoldoutDigest() ||
		!reflect.DeepEqual(
			repositorySemanticCaseIDs(second.Development()),
			repositorySemanticCaseIDs(first.Development()),
		) ||
		!reflect.DeepEqual(
			repositorySemanticCaseIDs(second.Holdout()),
			repositorySemanticCaseIDs(first.Holdout()),
		) ||
		!reflect.DeepEqual(
			repositoryExecutionCaseIDs(second.Execution()),
			repositoryExecutionCaseIDs(first.Execution()),
		) {
		t.Fatal("SOT-ENG-024: 同じ corpus-v1 の Load 結果が再現しません")
	}
}

func assertRepositoryCorpusImmutable(t *testing.T, corpus Corpus) {
	t.Helper()

	development := corpus.Development()
	holdout := corpus.Holdout()
	execution := corpus.Execution()
	development[0] = SemanticCase{}
	holdout[0] = SemanticCase{}
	execution[0] = ExecutionCase{}
	manifest := corpus.Manifest()
	entries := manifest.Development().Cases()
	entries[0] = ManifestEntry{}
	categories := manifest.RequiredCategoryIDs()
	categories[0] = "changed"

	if corpus.Development()[0].CaseID() == "" ||
		corpus.Holdout()[0].CaseID() == "" ||
		corpus.Execution()[0].CaseID() == "" ||
		corpus.Manifest().Development().Cases()[0].CaseID() == "" ||
		corpus.Manifest().RequiredCategoryIDs()[0] == "changed" {
		t.Fatal("SOT-ENG-026: corpus-v1 の getter から内部状態が変更されました")
	}
}

func repositorySemanticCaseIDs(values []SemanticCase) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.CaseID())
	}
	return result
}

func repositoryExecutionCaseIDs(values []ExecutionCase) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.CaseID())
	}
	return result
}
