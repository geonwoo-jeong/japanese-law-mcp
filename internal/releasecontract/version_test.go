package releasecontract_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/releasecheck"
)

// SOT-DEL-004/SOT-DEL-014: manifest の版と更新注釈を既存のリリース検証へ結び付ける。
func validateReleasePleaseVersion(
	ctx context.Context,
	manifest map[string]string,
	notesPath, repository string,
) error {
	version, exists := manifest["."]
	if !exists || len(manifest) != 1 {
		return fmt.Errorf("release manifest にはルートの版だけが必要です: %#v", manifest)
	}
	current, err := os.ReadFile(notesPath) // #nosec G304 -- SOT-ENG-019: テストが構築したリポジトリ内の固定パスか一時 fixture だけを読む。
	if err != nil {
		return fmt.Errorf("現在のリリース契約を読み込めません: %w", err)
	}
	heading, _, _ := strings.Cut(strings.ReplaceAll(string(current), "\r\n", "\n"), "\n")
	tag := "v" + version
	if heading != "# Japanese Law MCP "+tag+" <!-- x-release-please-version -->" {
		return fmt.Errorf("現在のリリース契約の見出しには manifest と同じ版と版更新注釈が必要です")
	}
	return releasecheck.Check(ctx, releasecheck.Request{
		ReleaseNotes: notesPath,
		Tag:          tag,
		Repository:   repository,
	})
}

// SOT-DEL-004/SOT-DEL-014: 合成した版の更新を許可し、不正値と不整合を拒否する。
func TestReleasePleaseVersionConsistency(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		manifest map[string]string
		heading  string
		wantErr  string
	}{
		"最初の正式版":       {manifest: map[string]string{".": "1.0.0"}},
		"後続の正式版":       {manifest: map[string]string{".": "2.3.4"}},
		"プレリリースとビルド情報": {manifest: map[string]string{".": "2.3.4-rc.1+build.5"}},
		"ルートがない":       {manifest: map[string]string{"other": "1.0.0"}, wantErr: "ルートの版だけ"},
		"余分なパッケージ": {
			manifest: map[string]string{".": "1.0.0", "other": "1.0.0"}, wantErr: "ルートの版だけ",
		},
		"版の不一致": {
			manifest: map[string]string{".": "1.0.0"},
			heading:  "# Japanese Law MCP v2.3.4 <!-- x-release-please-version -->",
			wantErr:  "同じ版と版更新注釈",
		},
		"注釈がない": {
			manifest: map[string]string{".": "1.0.0"},
			heading:  "# Japanese Law MCP v1.0.0",
			wantErr:  "同じ版と版更新注釈",
		},
		"空の版":           {manifest: map[string]string{".": ""}, wantErr: "SemVer"},
		"不正な版":          {manifest: map[string]string{".": "invalid"}, wantErr: "SemVer"},
		"省略した版":         {manifest: map[string]string{".": "1.2"}, wantErr: "SemVer"},
		"先行ゼロ":          {manifest: map[string]string{".": "01.2.3"}, wantErr: "SemVer"},
		"manifest の接頭辞": {manifest: map[string]string{".": "v1.2.3"}, wantErr: "SemVer"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repository := t.TempDir()
			if err := os.Mkdir(filepath.Join(repository, "sot"), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(repository, "sot", "01-test.md"),
				[]byte("# SOT-DEL-004: 検証用\n\n- 状態: 有効\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			heading := test.heading
			if heading == "" {
				heading = "# Japanese Law MCP v" + test.manifest["."] + " <!-- x-release-please-version -->"
			}
			notesPath := filepath.Join(repository, "CURRENT.md")
			notes := heading + "\n\n## 提供する SOT\n\n- SOT-DEL-004\n" +
				"\n## 未実装の SOT 差分\n\nなし\n\n## 互換性のない変更\n\nなし\n"
			if err := os.WriteFile(notesPath, []byte(notes), 0o600); err != nil {
				t.Fatal(err)
			}
			err := validateReleasePleaseVersion(t.Context(), test.manifest, notesPath, repository)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("整合する版を拒否しました: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("検証結果 = %v, 必要なエラー = %q", err, test.wantErr)
			}
		})
	}
}
