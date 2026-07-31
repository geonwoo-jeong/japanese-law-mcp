package legalqueryadoption

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestRepositoryCurrentは初回採用Tupleをcanonicalに解決する(t *testing.T) {
	const verificationID = "profile-set-initial-adoption-bootstrap"
	t.Chdir("../..")

	manifest, err := LoadCurrent(context.Background())
	if err != nil {
		t.Fatalf("%s: repository current を読めません: %v", verificationID, err)
	}
	if manifest.AdoptionID() !=
		"adoption-sha256-a405ea93cbd38a99f4c6cd084fb520dcec3d000ad7583f20bbe94b7ab9b84b94" {
		t.Fatalf("%s: adoptionId = %q", verificationID, manifest.AdoptionID())
	}
	if manifest.PreviousAdoptionID() != "" {
		t.Fatalf("%s: 初回 manifest に previous が存在します", verificationID)
	}
}

func TestLoadCurrentは初回Bootstrapのcurrentとhistoryを読む(t *testing.T) {
	t.Parallel()

	const verificationID = "evaluation-baseline-initial-bootstrap"
	root := t.TempDir()
	prepareAdoptionFixtureRoot(t, root)
	writeAdoptionFixtureFile(
		t,
		root,
		"testdata/legalquery/adoptions/current.json",
		[]byte("{\"artifactKind\":\"legal_query_adoption_pointer\",\"schemaVersion\":1,\"adoptionId\":\"adoption-sha256-5542b0e9508a125f5e86f07b1787b5c66b330a3fd04b6c4479877eddb09e7c09\"}\n"),
	)
	writeAdoptionFixtureFile(
		t,
		root,
		"testdata/legalquery/adoptions/history/adoption-sha256-5542b0e9508a125f5e86f07b1787b5c66b330a3fd04b6c4479877eddb09e7c09.json",
		[]byte("{\"artifactKind\":\"legal_query_adoption\",\"schemaVersion\":1,\"adoptionId\":\"adoption-sha256-5542b0e9508a125f5e86f07b1787b5c66b330a3fd04b6c4479877eddb09e7c09\",\"profileSetId\":\"default\",\"profileSetVersion\":\"profile-set-sha256-be9ce1499a7b6708a162c4ae2f4da9a340ed2883d3bd3480b2ec21989d11bf8f\",\"rankingVersion\":\"legal-query-ranking-2026-07-28-1\",\"compositionVersion\":\"composition-default-v1\",\"evaluatorVersion\":\"legal-query-evaluator-v1\",\"profiles\":[{\"profileId\":\"core\",\"profileVersion\":\"core-2026-07-30-33\",\"cueSetVersion\":\"core-cues-2026-07-30-15\"},{\"profileId\":\"judicial-cases\",\"profileVersion\":\"judicial-cases-2026-07-30-9\",\"cueSetVersion\":\"judicial-cases-cues-2026-07-30-4\"}],\"corpusVersion\":\"corpus-v9\",\"holdoutDigest\":\"c72a7a9504465f1d64a02ec9cb82d6a9faf6f382bee546c30439513f42d47557\",\"baselineVersion\":\"default-1\",\"baselineSha256\":\"0c820d882a6ab0e60535c73af3edc1533b7d13452a2ef26f0bde2fa4071bfb81\",\"catalogVersion\":\"unified-query-examples-v1\",\"catalogSha256\":\"6e27660d73290f5545d6dff34227c558a3609eadaca62a6d6e4a2a6088cafe0a\"}\n"),
	)

	manifest, err := loadCurrent(context.Background(), root)
	if err != nil {
		t.Fatalf("%s: current adoption を読み込めません: %v", verificationID, err)
	}
	if manifest.AdoptionID() != "adoption-sha256-5542b0e9508a125f5e86f07b1787b5c66b330a3fd04b6c4479877eddb09e7c09" {
		t.Fatalf("%s: adoptionId = %q", verificationID, manifest.AdoptionID())
	}
	if manifest.ProfileSetID() != "default" {
		t.Fatalf("%s: profileSetId = %q", verificationID, manifest.ProfileSetID())
	}
}

func TestLoadCurrentはcanonicalByte不一致とID不一致を拒否する(t *testing.T) {
	t.Parallel()

	const verificationID = "profile-set-adoption-canonical-bytes"
	root := t.TempDir()
	prepareAdoptionFixtureRoot(t, root)
	writeAdoptionFixtureFile(
		t,
		root,
		"testdata/legalquery/adoptions/current.json",
		[]byte("{ \"artifactKind\":\"legal_query_adoption_pointer\",\"schemaVersion\":1,\"adoptionId\":\"adoption-sha256-f12b5a173556693c95789fb3064e64f2f94b0a03d5e2ad77886333e6992fe953\"}\n"),
	)
	writeAdoptionFixtureFile(
		t,
		root,
		"testdata/legalquery/adoptions/history/adoption-sha256-f12b5a173556693c95789fb3064e64f2f94b0a03d5e2ad77886333e6992fe953.json",
		[]byte("{\"artifactKind\":\"legal_query_adoption\",\"schemaVersion\":1,\"adoptionId\":\"adoption-sha256-f12b5a173556693c95789fb3064e64f2f94b0a03d5e2ad77886333e6992fe953\",\"profileSetId\":\"default\",\"profileSetVersion\":\"profile-set-v1\",\"rankingVersion\":\"ranking-v1\",\"compositionVersion\":\"composition-v1\",\"evaluatorVersion\":\"legal-query-evaluator-v1\",\"profiles\":[{\"profileId\":\"core\",\"profileVersion\":\"core-v1\",\"cueSetVersion\":\"core-cues-v1\"}],\"corpusVersion\":\"corpus-v9\",\"holdoutDigest\":\"c72a7a9504465f1d64a02ec9cb82d6a9faf6f382bee546c30439513f42d47557\",\"baselineVersion\":\"default-1\",\"baselineSha256\":\"0c820d882a6ab0e60535c73af3edc1533b7d13452a2ef26f0bde2fa4071bfb81\",\"catalogVersion\":\"unified-query-examples-v1\",\"catalogSha256\":\"6e27660d73290f5545d6dff34227c558a3609eadaca62a6d6e4a2a6088cafe0a\"}\n"),
	)
	if _, err := loadCurrent(context.Background(), root); err == nil {
		t.Fatal(verificationID + ": non-canonical current pointer を受理しました")
	}

	root = t.TempDir()
	prepareAdoptionFixtureRoot(t, root)
	writeAdoptionFixtureFile(
		t,
		root,
		"testdata/legalquery/adoptions/current.json",
		[]byte("{\"artifactKind\":\"legal_query_adoption_pointer\",\"schemaVersion\":1,\"adoptionId\":\"adoption-sha256-f12b5a173556693c95789fb3064e64f2f94b0a03d5e2ad77886333e6992fe953\"}\n"),
	)
	writeAdoptionFixtureFile(
		t,
		root,
		"testdata/legalquery/adoptions/history/adoption-sha256-f12b5a173556693c95789fb3064e64f2f94b0a03d5e2ad77886333e6992fe953.json",
		[]byte("{\"artifactKind\":\"legal_query_adoption\",\"schemaVersion\":1,\"adoptionId\":\"adoption-sha256-2222222222222222222222222222222222222222222222222222222222222222\",\"profileSetId\":\"default\",\"profileSetVersion\":\"profile-set-v1\",\"rankingVersion\":\"ranking-v1\",\"compositionVersion\":\"composition-v1\",\"evaluatorVersion\":\"legal-query-evaluator-v1\",\"profiles\":[{\"profileId\":\"core\",\"profileVersion\":\"core-v1\",\"cueSetVersion\":\"core-cues-v1\"}],\"corpusVersion\":\"corpus-v9\",\"holdoutDigest\":\"c72a7a9504465f1d64a02ec9cb82d6a9faf6f382bee546c30439513f42d47557\",\"baselineVersion\":\"default-1\",\"baselineSha256\":\"0c820d882a6ab0e60535c73af3edc1533b7d13452a2ef26f0bde2fa4071bfb81\",\"catalogVersion\":\"unified-query-examples-v1\",\"catalogSha256\":\"6e27660d73290f5545d6dff34227c558a3609eadaca62a6d6e4a2a6088cafe0a\"}\n"),
	)
	if _, err := loadCurrent(context.Background(), root); err == nil {
		t.Fatal(verificationID + ": file 名と adoptionId の不一致を受理しました")
	}
}

func TestLoadCurrentはmissingPreviousとcycleを拒否する(t *testing.T) {
	t.Parallel()

	const verificationID = "profile-set-adoption-artifact-safety"
	root := t.TempDir()
	prepareAdoptionFixtureRoot(t, root)
	writeAdoptionFixtureFile(
		t,
		root,
		"testdata/legalquery/adoptions/current.json",
		[]byte("{\"artifactKind\":\"legal_query_adoption_pointer\",\"schemaVersion\":1,\"adoptionId\":\"adoption-sha256-ef07bb9f160fea80c0b1bba45d664b42f90fac7115de737535882bf55004d5ec\"}\n"),
	)
	writeAdoptionFixtureFile(
		t,
		root,
		"testdata/legalquery/adoptions/history/adoption-sha256-ef07bb9f160fea80c0b1bba45d664b42f90fac7115de737535882bf55004d5ec.json",
		[]byte("{\"artifactKind\":\"legal_query_adoption\",\"schemaVersion\":1,\"adoptionId\":\"adoption-sha256-ef07bb9f160fea80c0b1bba45d664b42f90fac7115de737535882bf55004d5ec\",\"previousAdoptionId\":\"adoption-sha256-4444444444444444444444444444444444444444444444444444444444444444\",\"profileSetId\":\"default\",\"profileSetVersion\":\"profile-set-v1\",\"rankingVersion\":\"ranking-v1\",\"compositionVersion\":\"composition-v1\",\"evaluatorVersion\":\"legal-query-evaluator-v1\",\"profiles\":[{\"profileId\":\"core\",\"profileVersion\":\"core-v1\",\"cueSetVersion\":\"core-cues-v1\"}],\"corpusVersion\":\"corpus-v9\",\"holdoutDigest\":\"c72a7a9504465f1d64a02ec9cb82d6a9faf6f382bee546c30439513f42d47557\",\"baselineVersion\":\"default-1\",\"baselineSha256\":\"0c820d882a6ab0e60535c73af3edc1533b7d13452a2ef26f0bde2fa4071bfb81\",\"catalogVersion\":\"unified-query-examples-v1\",\"catalogSha256\":\"6e27660d73290f5545d6dff34227c558a3609eadaca62a6d6e4a2a6088cafe0a\"}\n"),
	)
	if _, err := loadCurrent(context.Background(), root); err == nil {
		t.Fatal(verificationID + ": missing previousAdoptionId を受理しました")
	}

	root = t.TempDir()
	prepareAdoptionFixtureRoot(t, root)
	writeAdoptionFixtureFile(
		t,
		root,
		"testdata/legalquery/adoptions/current.json",
		[]byte("{\"artifactKind\":\"legal_query_adoption_pointer\",\"schemaVersion\":1,\"adoptionId\":\"adoption-sha256-5555555555555555555555555555555555555555555555555555555555555555\"}\n"),
	)
	writeAdoptionFixtureFile(
		t,
		root,
		"testdata/legalquery/adoptions/history/adoption-sha256-5555555555555555555555555555555555555555555555555555555555555555.json",
		[]byte("{\"artifactKind\":\"legal_query_adoption\",\"schemaVersion\":1,\"adoptionId\":\"adoption-sha256-5555555555555555555555555555555555555555555555555555555555555555\",\"previousAdoptionId\":\"adoption-sha256-6666666666666666666666666666666666666666666666666666666666666666\",\"profileSetId\":\"default\",\"profileSetVersion\":\"profile-set-v1\",\"rankingVersion\":\"ranking-v1\",\"compositionVersion\":\"composition-v1\",\"evaluatorVersion\":\"legal-query-evaluator-v1\",\"profiles\":[{\"profileId\":\"core\",\"profileVersion\":\"core-v1\",\"cueSetVersion\":\"core-cues-v1\"}],\"corpusVersion\":\"corpus-v9\",\"holdoutDigest\":\"c72a7a9504465f1d64a02ec9cb82d6a9faf6f382bee546c30439513f42d47557\",\"baselineVersion\":\"default-1\",\"baselineSha256\":\"0c820d882a6ab0e60535c73af3edc1533b7d13452a2ef26f0bde2fa4071bfb81\",\"catalogVersion\":\"unified-query-examples-v1\",\"catalogSha256\":\"6e27660d73290f5545d6dff34227c558a3609eadaca62a6d6e4a2a6088cafe0a\"}\n"),
	)
	writeAdoptionFixtureFile(
		t,
		root,
		"testdata/legalquery/adoptions/history/adoption-sha256-6666666666666666666666666666666666666666666666666666666666666666.json",
		[]byte("{\"artifactKind\":\"legal_query_adoption\",\"schemaVersion\":1,\"adoptionId\":\"adoption-sha256-6666666666666666666666666666666666666666666666666666666666666666\",\"previousAdoptionId\":\"adoption-sha256-5555555555555555555555555555555555555555555555555555555555555555\",\"profileSetId\":\"default\",\"profileSetVersion\":\"profile-set-v1\",\"rankingVersion\":\"ranking-v1\",\"compositionVersion\":\"composition-v1\",\"evaluatorVersion\":\"legal-query-evaluator-v1\",\"profiles\":[{\"profileId\":\"core\",\"profileVersion\":\"core-v1\",\"cueSetVersion\":\"core-cues-v1\"}],\"corpusVersion\":\"corpus-v9\",\"holdoutDigest\":\"c72a7a9504465f1d64a02ec9cb82d6a9faf6f382bee546c30439513f42d47557\",\"baselineVersion\":\"default-1\",\"baselineSha256\":\"0c820d882a6ab0e60535c73af3edc1533b7d13452a2ef26f0bde2fa4071bfb81\",\"catalogVersion\":\"unified-query-examples-v1\",\"catalogSha256\":\"6e27660d73290f5545d6dff34227c558a3609eadaca62a6d6e4a2a6088cafe0a\"}\n"),
	)
	if _, err := loadCurrent(context.Background(), root); err == nil {
		t.Fatal(verificationID + ": previousAdoptionId cycle を受理しました")
	}
}

func writeAdoptionFixtureFile(
	t *testing.T,
	root string,
	relative string,
	content []byte,
) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("adoption test dir を作れません: %v", err)
	}
	//nolint:gosec // SOT-ENG-019: t.TempDir 配下へ test が構成した固定 fixture だけを書く。
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("adoption test file を書けません: %v", err)
	}
}

func prepareAdoptionFixtureRoot(t *testing.T, root string) {
	t.Helper()

	schema, err := os.ReadFile(filepath.Join(
		"..",
		"..",
		"testdata",
		"legalquery",
		"adoptions",
		"schema-v1.json",
	))
	if err != nil {
		t.Fatalf("adoption schema fixture を読めません: %v", err)
	}
	writeAdoptionFixtureFile(
		t,
		root,
		"testdata/legalquery/adoptions/schema-v1.json",
		schema,
	)
}

func prepareCatalogFixtureRoot(t *testing.T, root string) {
	t.Helper()

	for _, relative := range []string{
		"docs/unified-query-examples/00-index.md",
		"docs/unified-query-examples/10-execution.md",
		"docs/unified-query-examples/20-clarification-and-unsupported.md",
	} {
		content, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatalf("catalog fixture を読めません: %v", err)
		}
		writeAdoptionFixtureFile(t, root, relative, content)
	}
}

func TestLoadCurrentはsymlinkを拒否する(t *testing.T) {
	t.Parallel()

	const verificationID = "profile-set-adoption-artifact-safety"
	root := t.TempDir()
	prepareAdoptionFixtureRoot(t, root)
	target := filepath.Join(root, "outside-current.json")
	if err := os.WriteFile(target, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("%s: symlink target を書けません: %v", verificationID, err)
	}
	currentPath := filepath.Join(root, "testdata/legalquery/adoptions/current.json")
	if err := os.MkdirAll(filepath.Dir(currentPath), 0o750); err != nil {
		t.Fatalf("%s: current dir を作れません: %v", verificationID, err)
	}
	if err := os.Symlink(target, currentPath); err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("symlink を作成できない環境です")
		}
		t.Fatalf("%s: current symlink を作れません: %v", verificationID, err)
	}
	if _, err := loadCurrent(context.Background(), root); err == nil {
		t.Fatal(verificationID + ": current.json symlink を受理しました")
	}
}

func TestVerifyCatalogはhistoryを除外し未知ディレクトリを拒否する(t *testing.T) {
	t.Parallel()

	const verificationID = "profile-set-adoption-artifact-safety"
	root := t.TempDir()
	prepareCatalogFixtureRoot(t, root)

	repository, err := legalqueryartifact.OpenRepository(root)
	if err != nil {
		t.Fatalf("%s: repository を開けません: %v", verificationID, err)
	}
	defer func() { _ = repository.Close() }()

	manifest := Manifest{
		corpusVersion:   "corpus-v9",
		holdoutDigest:   "c72a7a9504465f1d64a02ec9cb82d6a9faf6f382bee546c30439513f42d47557",
		baselineVersion: "default-1",
		catalogVersion:  "unified-query-examples-v1",
		catalogSHA256:   "b1372b4cc00f2e0a1aec41b51c3887cb134ba99f569e38524274674cd026da59",
	}
	corpus := loadCatalogFixtureCorpus(t)
	if err := verifyCatalog(context.Background(), repository, manifest, corpus); err != nil {
		t.Fatalf("%s: 正常カタログを拒否しました: %v", verificationID, err)
	}

	if err := os.MkdirAll(
		filepath.Join(root, "docs/unified-query-examples/history"),
		0o750,
	); err != nil {
		t.Fatalf("%s: history directory を作れません: %v", verificationID, err)
	}
	if err := verifyCatalog(context.Background(), repository, manifest, corpus); err != nil {
		t.Fatalf("%s: history directory を除外できません: %v", verificationID, err)
	}

	if err := os.MkdirAll(
		filepath.Join(root, "docs/unified-query-examples/extra"),
		0o750,
	); err != nil {
		t.Fatalf("%s: 未知 directory を作れません: %v", verificationID, err)
	}
	if err := verifyCatalog(context.Background(), repository, manifest, corpus); err == nil {
		t.Fatal(verificationID + ": 検索例カタログの未知 directory を受理しました")
	}
}

func TestVerifyCatalogはIndex版不一致と存在しないArtifactを拒否する(t *testing.T) {
	t.Parallel()

	const verificationID = "profile-set-production-evaluation-identity"
	tests := []struct {
		name     string
		file     string
		current  string
		replaced string
		want     string
	}{
		{
			name:     "corpusVersion",
			file:     "docs/unified-query-examples/00-index.md",
			current:  "- `corpusVersion`: `corpus-v9`",
			replaced: "- `corpusVersion`: `corpus-v8`",
			want:     "corpusVersion",
		},
		{
			name:     "baselineVersion",
			file:     "docs/unified-query-examples/00-index.md",
			current:  "- `baselineVersion`: `default-1`",
			replaced: "- `baselineVersion`: `default-2`",
			want:     "baselineVersion",
		},
		{
			name:     "verification_artifact",
			file:     "docs/unified-query-examples/10-execution.md",
			current:  "`corpus-v9:semantic:development-name-alias`",
			replaced: "`corpus-v9:semantic:development-does-not-exist`",
			want:     "verification_artifact",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			prepareCatalogFixtureRoot(t, root)
			rewriteCatalogFixture(t, root, tt.file, tt.current, tt.replaced)
			repository, err := legalqueryartifact.OpenRepository(root)
			if err != nil {
				t.Fatalf("%s: repository を開けません: %v", verificationID, err)
			}
			defer func() { _ = repository.Close() }()
			err = verifyCatalog(
				context.Background(),
				repository,
				catalogFixtureManifest(),
				loadCatalogFixtureCorpus(t),
			)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("%s: %s 不一致結果 = %v", verificationID, tt.name, err)
			}
		})
	}
}

func TestVerifyCatalogはentry数とbyte上限を超えるfixtureを拒否する(t *testing.T) {
	t.Parallel()

	const verificationID = "profile-set-adoption-artifact-safety"
	root := t.TempDir()
	prepareCatalogFixtureRoot(t, root)

	repository, err := legalqueryartifact.OpenRepository(root)
	if err != nil {
		t.Fatalf("%s: repository を開けません: %v", verificationID, err)
	}
	defer func() { _ = repository.Close() }()

	manifest := Manifest{
		corpusVersion:   "corpus-v9",
		holdoutDigest:   "c72a7a9504465f1d64a02ec9cb82d6a9faf6f382bee546c30439513f42d47557",
		baselineVersion: "default-1",
		catalogVersion:  "unified-query-examples-v1",
		catalogSHA256:   "b1372b4cc00f2e0a1aec41b51c3887cb134ba99f569e38524274674cd026da59",
	}
	corpus := loadCatalogFixtureCorpus(t)

	for index := 0; index < maximumCatalogEntries; index++ {
		writeAdoptionFixtureFile(
			t,
			root,
			filepath.ToSlash(filepath.Join(
				"docs/unified-query-examples",
				fmt.Sprintf("extra-%02d.md", index),
			)),
			[]byte("# extra\n"),
		)
	}
	if err := verifyCatalog(context.Background(), repository, manifest, corpus); err == nil {
		t.Fatal(verificationID + ": catalog entry 上限超過を受理しました")
	}

	root = t.TempDir()
	prepareCatalogFixtureRoot(t, root)
	repository, err = legalqueryartifact.OpenRepository(root)
	if err != nil {
		t.Fatalf("%s: byte 上限用 repository を開けません: %v", verificationID, err)
	}
	defer func() { _ = repository.Close() }()

	writeAdoptionFixtureFile(
		t,
		root,
		"docs/unified-query-examples/huge.md",
		[]byte(strings.Repeat("a", maximumCatalogMarkdownBytes+1)),
	)
	if err := verifyCatalog(context.Background(), repository, manifest, corpus); err == nil {
		t.Fatal(verificationID + ": catalog byte 上限超過を受理しました")
	}
}

func loadCatalogFixtureCorpus(t *testing.T) legalquerycorpus.Corpus {
	t.Helper()

	corpus, err := legalquerycorpus.Load(
		context.Background(),
		filepath.Join("..", ".."),
		"testdata/legalquery/corpus-v9",
	)
	if err != nil {
		t.Fatalf("catalog 用 corpus fixture を読めません: %v", err)
	}
	return corpus
}

func catalogFixtureManifest() Manifest {
	return Manifest{
		corpusVersion:   "corpus-v9",
		holdoutDigest:   "c72a7a9504465f1d64a02ec9cb82d6a9faf6f382bee546c30439513f42d47557",
		baselineVersion: "default-1",
		catalogVersion:  "unified-query-examples-v1",
		catalogSHA256:   "b1372b4cc00f2e0a1aec41b51c3887cb134ba99f569e38524274674cd026da59",
	}
}

func rewriteCatalogFixture(
	t *testing.T,
	root string,
	relative string,
	current string,
	replaced string,
) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relative))
	//nolint:gosec // SOT-ENG-019: t.TempDir 配下の固定 catalog fixture 名だけを test 内で読む。
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("catalog fixture を読めません: %v", err)
	}
	updated := strings.Replace(string(raw), current, replaced, 1)
	if updated == string(raw) {
		t.Fatalf("catalog fixture の置換対象がありません: %s", relative)
	}
	//nolint:gosec // SOT-ENG-019: t.TempDir 配下の同じ固定 catalog fixture だけを test 内で更新する。
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatalf("catalog fixture を更新できません: %v", err)
	}
}
