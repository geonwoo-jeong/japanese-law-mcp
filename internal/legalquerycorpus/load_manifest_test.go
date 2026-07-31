package legalquerycorpus

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifestはFixtureDirectoryを走査せずV2Identityを返す(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	writeManifestOnlyFixture(
		t,
		repository,
		"testdata/legalquery/schemas/legal-query-corpus-v2.schema.json",
		readRepositoryCorpusFile(t, filepath.Join(
			"..", "..", "testdata/legalquery/schemas/legal-query-corpus-v2.schema.json",
		)),
	)
	writeManifestOnlyFixture(
		t,
		repository,
		"testdata/legalquery/corpus-v10/manifest.json",
		readRepositoryCorpusFile(t, filepath.Join(
			"..", "..", "testdata/legalquery/corpus-v10/manifest.json",
		)),
	)
	for _, set := range []string{"development", "holdout", "execution"} {
		if err := os.MkdirAll(filepath.Join(
			repository, "testdata/legalquery/corpus-v10", set,
		), 0o750); err != nil {
			t.Fatalf("manifest-only fixture directory を作れません: %v", err)
		}
	}
	if err := os.Mkdir(filepath.Join(
		repository,
		"testdata/legalquery/corpus-v10/holdout/走査禁止",
	), 0o750); err != nil {
		t.Fatalf("走査禁止 sentinel を作れません: %v", err)
	}

	artifact, err := LoadManifest(
		t.Context(), repository, "testdata/legalquery/corpus-v10",
	)
	if err != nil {
		t.Fatalf("candidate-evaluation-consumed-digest-preflight: manifest だけを読めません: %v", err)
	}
	if artifact.Manifest().SchemaVersion() != 2 ||
		artifact.Manifest().CorpusVersion() != "corpus-v10" ||
		len(artifact.Manifest().HoldoutLeakageGroupDigests()) == 0 ||
		len(artifact.RawBytes()) == 0 || len(artifact.SHA256()) != 64 {
		t.Fatalf("candidate-evaluation-consumed-digest-preflight: manifest identity が不正です")
	}
}

func writeManifestOnlyFixture(t *testing.T, root, relative string, raw []byte) {
	t.Helper()
	target := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatalf("manifest-only fixture directory を作れません: %v", err)
	}
	if err := os.WriteFile(target, raw, 0o600); err != nil {
		t.Fatalf("manifest-only fixture file を書けません: %v", err)
	}
}
