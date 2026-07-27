package releasecheck

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	t.Parallel()

	notes := writeTestFile(t, "release-notes.md", []byte(`# Japanese Law MCP v1.2.3

## 提供する SOT

- SOT-DEL-004

## 未実装の SOT 差分

なし

## 互換性のない変更

なし
`))
	if err := Check(context.Background(), Request{
		ReleaseNotes: notes,
		Tag:          "v1.2.3",
		Repository:   writeTestSOTRepository(t),
	}); err != nil {
		t.Fatalf("Check() のエラー = %v", err)
	}
}

func TestCheckWithDistribution(t *testing.T) {
	t.Parallel()

	notes := writeTestFile(t, "release-notes.md", []byte(`# Japanese Law MCP v1.2.3

## 提供する SOT

- SOT-DEL-004

## 未実装の SOT 差分

なし

## 互換性のない変更

なし
`))
	dist := newValidDistribution(t, "v1.2.3", testCommit)
	if err := Check(context.Background(), Request{
		ReleaseNotes: notes,
		Tag:          "v1.2.3",
		Repository:   writeTestSOTRepository(t),
		Dist:         dist,
		Commit:       testCommit,
	}); err != nil {
		t.Fatalf("Check() のエラー = %v", err)
	}
}

func TestCheckRejectsInvalidOptionalCombinations(t *testing.T) {
	t.Parallel()

	tests := map[string]Request{
		"dist だけ": {
			ReleaseNotes: "notes.md", Tag: "v1.2.3", Repository: ".",
			Dist: "dist",
		},
		"commit だけ": {
			ReleaseNotes: "notes.md", Tag: "v1.2.3", Repository: ".",
			Commit: testCommit,
		},
		"target-os だけ": {
			ReleaseNotes: "notes.md", Tag: "v1.2.3", Repository: ".",
			TargetOS: "darwin",
		},
		"target-arch だけ": {
			ReleaseNotes: "notes.md", Tag: "v1.2.3", Repository: ".",
			TargetArch: "arm64",
		},
		"dist なしの target": {
			ReleaseNotes: "notes.md", Tag: "v1.2.3", Repository: ".",
			TargetOS: "darwin", TargetArch: "arm64",
		},
	}
	for name, request := range tests {
		request := request
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := Check(context.Background(), request)
			if err == nil || !strings.Contains(err.Error(), "組み合わせ") {
				t.Fatalf("Check() のエラー = %v", err)
			}
		})
	}
}

func TestFindReleaseTarget(t *testing.T) {
	t.Parallel()

	target, exists := findReleaseTarget("windows", "amd64", "1.2.3")
	if !exists || target.binaryName != "japanese-law-mcp.exe" {
		t.Fatalf("findReleaseTarget() = %#v, %t", target, exists)
	}
	if _, exists := findReleaseTarget("linux", "amd64", "1.2.3"); exists {
		t.Fatal("linux target が見つかりました")
	}
}

func TestSmokeTargetWithOfficialBinary(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempDir := t.TempDir()
	binary := filepath.Join(tempDir, "japanese-law-mcp")
	repository := filepath.Clean(filepath.Join("..", ".."))
	command := exec.CommandContext( //nolint:gosec // SOT-ENG-019: 固定した go build とテスト専用出力先だけを使用する。
		ctx,
		"go",
		"build",
		"-trimpath",
		"-ldflags",
		"-X github.com/geonwoo-jeong/japanese-law-mcp/internal/buildinfo.version=1.2.3",
		"-o",
		binary,
		"./cmd/japanese-law-mcp",
	)
	command.Dir = repository
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("公式実行ファイルを build できません: %v\n%s", err, output)
	}

	archive := filepath.Join(tempDir, "japanese-law-mcp_1.2.3_darwin_arm64.tar.gz")
	content, err := os.ReadFile(binary) //nolint:gosec // SOT-ENG-019: テスト専用 TempDir に build した固定成果物を読む。
	if err != nil {
		t.Fatalf("build した実行ファイルを読めません: %v", err)
	}
	writeTestTarGz(t, archive, []testArchiveEntry{{
		name: "japanese-law-mcp", content: string(content), typeflag: 0, mode: 0o755,
	}})

	err = smokeTarget(ctx, archive, releaseTarget{
		goos:        "darwin",
		goarch:      "arm64",
		format:      "tar.gz",
		binaryName:  "japanese-law-mcp",
		archiveName: filepath.Base(archive),
	}, "1.2.3")
	if err != nil {
		t.Fatalf("smokeTarget() のエラー = %v", err)
	}
}
