package releasecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReleaseNotes(t *testing.T) {
	t.Parallel()

	repository := writeTestSOTRepository(t)
	valid := `# Japanese Law MCP v1.2.3

## 提供する SOT

- SOT-DEL-004

## 未実装の SOT 差分

なし

## 互換性のない変更

- SOT-DEL-007: 変更内容
`
	tests := map[string]struct {
		tag     string
		notes   string
		wantErr string
	}{
		"有効": {
			tag:   "v1.2.3",
			notes: valid,
		},
		"code fence 内の H2 は数えない": {
			tag: "v1.2.3",
			notes: strings.Replace(
				valid,
				"- SOT-DEL-004",
				"- SOT-DEL-004\n\n```text\n## 偽のセクション\n```",
				1,
			),
		},
		"Release Please の版更新注釈": {
			tag: "v1.2.3",
			notes: strings.Replace(
				valid,
				"# Japanese Law MCP v1.2.3",
				"# Japanese Law MCP v1.2.3 <!-- x-release-please-version -->",
				1,
			),
		},
		"見出しの版が tag と異なる": {
			tag: "v1.2.3",
			notes: strings.Replace(
				valid,
				"# Japanese Law MCP v1.2.3",
				"# Japanese Law MCP v1.2.4",
				1,
			),
			wantErr: "見出し",
		},
		"見出しが不正": {
			tag:     "v1.2.3",
			notes:   strings.Replace(valid, "# Japanese Law MCP v1.2.3", "# release", 1),
			wantErr: "見出し",
		},
		"タグに v がない": {
			tag:     "1.2.3",
			notes:   valid,
			wantErr: "v で始まる SemVer",
		},
		"先行ゼロ": {
			tag:     "v01.2.3",
			notes:   valid,
			wantErr: "v で始まる SemVer",
		},
		"必須セクションがない": {
			tag: "v1.2.3",
			notes: strings.ReplaceAll(
				valid,
				"## 互換性のない変更",
				"## 移行情報",
			),
			wantErr: "H2 セクション",
		},
		"重複セクション": {
			tag: "v1.2.3",
			notes: valid + `
## 提供する SOT

- SOT-DEL-010
`,
			wantErr: "H2 セクション",
		},
		"提供 SOT がない": {
			tag: "v1.2.3",
			notes: strings.Replace(
				valid,
				"- SOT-DEL-004",
				"なし",
				1,
			),
			wantErr: "SOT ID",
		},
		"提供 SOT が存在しない": {
			tag: "v1.2.3",
			notes: strings.Replace(
				valid,
				"SOT-DEL-004",
				"SOT-XXX-000",
				1,
			),
			wantErr: "有効な SOT",
		},
		"未実装差分が空": {
			tag: "v1.2.3",
			notes: strings.Replace(
				valid,
				"## 未実装の SOT 差分\n\nなし",
				"## 未実装の SOT 差分\n",
				1,
			),
			wantErr: "本文",
		},
		"未実装差分の宣言が不正": {
			tag: "v1.2.3",
			notes: strings.Replace(
				valid,
				"## 未実装の SOT 差分\n\nなし",
				"## 未実装の SOT 差分\n\n検討中",
				1,
			),
			wantErr: "SOT ID または「なし」",
		},
		"互換性変更の宣言が不正": {
			tag: "v1.2.3",
			notes: strings.Replace(
				valid,
				"- SOT-DEL-007: 変更内容",
				"変更があります",
				1,
			),
			wantErr: "SOT ID または「なし」",
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := writeTestFile(t, "release-notes.md", []byte(test.notes))
			err := validateReleaseNotes(path, test.tag, repository)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("validateReleaseNotes() のエラー = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("validateReleaseNotes() のエラー = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestValidateReleaseNotesRejectsOversizedFile(t *testing.T) {
	t.Parallel()

	path := writeTestFile(
		t,
		"release-notes.md",
		[]byte(strings.Repeat("a", maxReleaseNotesBytes+1)),
	)
	err := validateReleaseNotes(path, "v1.2.3", writeTestSOTRepository(t))
	if err == nil || !strings.Contains(err.Error(), "大きすぎます") {
		t.Fatalf("validateReleaseNotes() のエラー = %v", err)
	}
}

func TestValidateReleaseNotesRejectsInactiveSOT(t *testing.T) {
	t.Parallel()

	repository := writeTestSOTRepository(t)
	writeTestSOT(t, repository, "SOT-DEL-099", "草案")
	path := writeTestFile(t, "release-notes.md", []byte(`# Japanese Law MCP v1.2.3

## 提供する SOT

- SOT-DEL-099

## 未実装の SOT 差分

なし

## 互換性のない変更

なし
`))
	err := validateReleaseNotes(path, "v1.2.3", repository)
	if err == nil || !strings.Contains(err.Error(), "有効な SOT") {
		t.Fatalf("validateReleaseNotes() のエラー = %v", err)
	}
}

func TestLoadActiveSOTIDsRejectsMalformedSOT(t *testing.T) {
	t.Parallel()

	repository := writeTestSOTRepository(t)
	path := filepath.Join(repository, "sot", "60-delivery", "malformed.md")
	if err := os.WriteFile(path, []byte("# 規定ではない文書\n"), 0o600); err != nil {
		t.Fatalf("不正な SOT を作成できません: %v", err)
	}
	if _, err := loadActiveSOTIDs(repository); err == nil ||
		!strings.Contains(err.Error(), "不正") {
		t.Fatalf("loadActiveSOTIDs() のエラー = %v", err)
	}
}

func writeTestSOTRepository(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	directory := filepath.Join(root, "sot", "60-delivery")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatalf("SOT directory を作成できません: %v", err)
	}
	for _, id := range []string{"SOT-DEL-004", "SOT-DEL-007"} {
		writeTestSOT(t, root, id, "有効")
	}
	return root
}

func writeTestSOT(t *testing.T, root, id, status string) {
	t.Helper()

	content := "# " + id + ": テスト規定\n\n- 状態: " + status +
		"\n\n## 規定\n\nテスト用。\n"
	path := filepath.Join(root, "sot", "60-delivery", strings.ToLower(id)+".md")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("SOT を作成できません: %v", err)
	}
}

func writeTestFile(t *testing.T, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("テストファイルを書き込めません: %v", err)
	}
	return path
}
