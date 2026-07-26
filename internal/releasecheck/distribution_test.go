package releasecheck

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"

func TestValidateDistribution(t *testing.T) {
	t.Parallel()

	dist := newValidDistribution(t, "v1.2.3", testCommit)
	if err := validateDistribution(dist, "v1.2.3", testCommit); err != nil {
		t.Fatalf("validateDistribution() のエラー = %v", err)
	}
}

func TestValidateDistributionRejectsInvalidArtifacts(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		mutate  func(t *testing.T, fixture *distributionFixture)
		wantErr string
	}{
		"アーカイブ不足": {
			mutate: func(t *testing.T, fixture *distributionFixture) {
				t.Helper()
				if err := os.Remove(filepath.Join(fixture.dir, fixture.names[0])); err != nil {
					t.Fatalf("アーカイブを削除できません: %v", err)
				}
			},
			wantErr: "アーカイブ",
		},
		"チェックサム不一致": {
			mutate: func(t *testing.T, fixture *distributionFixture) {
				t.Helper()
				path := filepath.Join(fixture.dir, fixture.names[0])
				if err := os.WriteFile(path, []byte("改ざん"), 0o600); err != nil {
					t.Fatalf("アーカイブを改ざんできません: %v", err)
				}
			},
			wantErr: "SHA-256",
		},
		"チェックサムに余分な項目": {
			mutate: func(t *testing.T, fixture *distributionFixture) {
				t.Helper()
				path := filepath.Join(fixture.dir, checksumFileName("1.2.3"))
				file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0) //nolint:gosec // SOT-ENG-019: テスト専用 TempDir の固定 fixture を更新する。
				if err != nil {
					t.Fatalf("checksums.txt を開けません: %v", err)
				}
				if _, err := fmt.Fprintf(file, "%064d  extra.zip\n", 0); err != nil {
					t.Fatalf("checksums.txt を追記できません: %v", err)
				}
				if err := file.Close(); err != nil {
					t.Fatalf("checksums.txt を閉じられません: %v", err)
				}
			},
			wantErr: "予期しない成果物",
		},
		"metadata の commit 不一致": {
			mutate: func(t *testing.T, fixture *distributionFixture) {
				t.Helper()
				writeJSONFixture(t, filepath.Join(fixture.dir, "metadata.json"), releaseMetadata{
					ProjectName: projectName,
					Tag:         "v1.2.3",
					Version:     "1.2.3",
					Commit:      strings.Repeat("a", 40),
				})
			},
			wantErr: "commit",
		},
		"Source artifact": {
			mutate: func(t *testing.T, fixture *distributionFixture) {
				t.Helper()
				fixture.artifacts = append(fixture.artifacts, goreleaserArtifact{
					Name: "source.tar.gz",
					Type: "Source",
				})
				writeJSONFixture(
					t,
					filepath.Join(fixture.dir, "artifacts.json"),
					fixture.artifacts,
				)
			},
			wantErr: "source artifact",
		},
		"Archive checksum 不一致": {
			mutate: func(t *testing.T, fixture *distributionFixture) {
				t.Helper()
				fixture.artifacts[0].Extra.Checksum = "sha256:" + strings.Repeat("0", 64)
				writeJSONFixture(
					t,
					filepath.Join(fixture.dir, "artifacts.json"),
					fixture.artifacts,
				)
			},
			wantErr: "artifacts.json",
		},
		"Archive binaries 不一致": {
			mutate: func(t *testing.T, fixture *distributionFixture) {
				t.Helper()
				fixture.artifacts[0].Extra.Binaries = []string{"README.md"}
				writeJSONFixture(
					t,
					filepath.Join(fixture.dir, "artifacts.json"),
					fixture.artifacts,
				)
			},
			wantErr: "実行ファイル",
		},
		"予期しない配布アーカイブ": {
			mutate: func(t *testing.T, fixture *distributionFixture) {
				t.Helper()
				path := filepath.Join(fixture.dir, "japanese-law-mcp_1.2.3_source.tar.gz")
				writeTestTarGz(t, path, []testArchiveEntry{{
					name: "source", content: "source", typeflag: 0, mode: 0o644,
				}})
			},
			wantErr: "予期しないアーカイブ",
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fixture := newDistributionFixture(t, "v1.2.3", testCommit)
			test.mutate(t, fixture)
			err := validateDistribution(fixture.dir, "v1.2.3", testCommit)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("validateDistribution() のエラー = %v, want %q", err, test.wantErr)
			}
		})
	}
}

type distributionFixture struct {
	dir       string
	names     []string
	artifacts []goreleaserArtifact
}

func newValidDistribution(t *testing.T, tag, commit string) string {
	t.Helper()

	return newDistributionFixture(t, tag, commit).dir
}

func newDistributionFixture(t *testing.T, tag, commit string) *distributionFixture {
	t.Helper()

	dir := t.TempDir()
	version := strings.TrimPrefix(tag, "v")
	targets := releaseTargets(version)
	names := make([]string, 0, len(targets))
	artifacts := make([]goreleaserArtifact, 0, len(targets))
	checksums := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.archiveName)
		path := filepath.Join(dir, target.archiveName)
		entry := testArchiveEntry{
			name: target.binaryName, content: "binary", typeflag: 0, mode: 0o755,
		}
		if target.format == "zip" {
			writeTestZip(t, path, []testArchiveEntry{entry})
		} else {
			writeTestTarGz(t, path, []testArchiveEntry{entry})
		}
		sum := fileSHA256(t, path)
		checksums = append(checksums, sum+"  "+target.archiveName)
		artifacts = append(artifacts, goreleaserArtifact{
			Name:   target.archiveName,
			Path:   filepath.Join("dist", target.archiveName),
			GoOS:   target.goos,
			GoArch: target.goarch,
			Type:   "Archive",
			Extra: goreleaserArtifactExtra{
				Binaries: []string{target.binaryName},
				Checksum: "sha256:" + sum,
				Format:   target.format,
			},
		})
	}
	if err := os.WriteFile(
		filepath.Join(dir, checksumFileName(version)),
		[]byte(strings.Join(checksums, "\n")+"\n"),
		0o600,
	); err != nil {
		t.Fatalf("checksums.txt を作成できません: %v", err)
	}
	writeJSONFixture(t, filepath.Join(dir, "metadata.json"), releaseMetadata{
		ProjectName: projectName,
		Tag:         tag,
		Version:     version,
		Commit:      commit,
	})
	writeJSONFixture(t, filepath.Join(dir, "artifacts.json"), artifacts)
	return &distributionFixture{dir: dir, names: names, artifacts: artifacts}
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path) //nolint:gosec // SOT-ENG-019: テスト専用 TempDir の固定 fixture を読む。
	if err != nil {
		t.Fatalf("ファイルを読めません: %v", err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func writeJSONFixture(t *testing.T, path string, value any) {
	t.Helper()

	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("JSON を生成できません: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("JSON を書き込めません: %v", err)
	}
}
