package releasecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPublishNotes(t *testing.T) {
	t.Parallel()

	repository := writeTestSOTRepository(t)
	releaseNotes := writeTestFile(t, "CURRENT.md", []byte(validPublishContract()))
	changelog := writeTestFile(t, "CHANGELOG.md", []byte(`# 変更履歴

## [1.2.4](https://example.test/compare/v1.2.3...v1.2.4) (2026-07-28)

### 機能

- 次の版

`+"```text"+`
## 1.2.3
`+"```"+`

## [1.2.3](https://example.test/compare/v1.2.2...v1.2.3) (2026-07-27)

### 修正

- 対象の変更

`+"```markdown"+`
## コード例
`+"```"+`

## 1.2.2

- 前の版
`))

	got, err := BuildPublishNotes(
		changelog,
		releaseNotes,
		"v1.2.3",
		repository,
	)
	if err != nil {
		t.Fatalf("BuildPublishNotes() のエラー = %v", err)
	}
	want := `## [1.2.3](https://example.test/compare/v1.2.2...v1.2.3) (2026-07-27)

### 修正

- 対象の変更

` + "```markdown" + `
## コード例
` + "```" + `

---

## 提供する SOT

- SOT-DEL-004

## 未実装の SOT 差分

なし

## 互換性のない変更

- SOT-DEL-007: 変更内容
`
	if string(got) != want {
		t.Fatalf("BuildPublishNotes() =\n%s\nwant\n%s", got, want)
	}
}

func TestBuildPublishNotesAcceptsPlainVersionHeadingAndCRLF(t *testing.T) {
	t.Parallel()

	repository := writeTestSOTRepository(t)
	releaseNotes := writeTestFile(
		t,
		"CURRENT.md",
		[]byte(strings.ReplaceAll(validPublishContract(), "\n", "\r\n")),
	)
	changelog := writeTestFile(
		t,
		"CHANGELOG.md",
		[]byte("# 変更履歴\r\n\r\n## 1.2.3\r\n\r\n- 修正\r\n"),
	)

	got, err := BuildPublishNotes(
		changelog,
		releaseNotes,
		"v1.2.3",
		repository,
	)
	if err != nil {
		t.Fatalf("BuildPublishNotes() のエラー = %v", err)
	}
	if strings.Contains(string(got), "\r") {
		t.Fatalf("出力に CR が残っています: %q", got)
	}
	if !strings.HasPrefix(string(got), "## 1.2.3\n\n- 修正\n\n---\n\n") {
		t.Fatalf("変更履歴セクション = %q", got)
	}
}

func TestBuildPublishNotesAcceptsPlainVersionHeadingWithDate(t *testing.T) {
	t.Parallel()

	got, err := BuildPublishNotes(
		writeTestFile(
			t,
			"CHANGELOG.md",
			[]byte("# 変更履歴\n\n## 1.2.3 (2026-07-27)\n\n- 初回リリース\n"),
		),
		writeTestFile(t, "CURRENT.md", []byte(validPublishContract())),
		"v1.2.3",
		writeTestSOTRepository(t),
	)
	if err != nil {
		t.Fatalf("BuildPublishNotes() のエラー = %v", err)
	}
	if !strings.HasPrefix(
		string(got),
		"## 1.2.3 (2026-07-27)\n\n- 初回リリース\n\n---\n\n",
	) {
		t.Fatalf("変更履歴セクション = %q", got)
	}
}

func TestBuildPublishNotesRejectsInvalidChangelog(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		changelog string
		wantErr   string
	}{
		"対象版なし": {
			changelog: "# 変更履歴\n\n## 1.2.2\n\n- 前の版\n",
			wantErr:   "見つかりません",
		},
		"対象版の重複": {
			changelog: "# 変更履歴\n\n## 1.2.3\n\n- 一件目\n\n## 1.2.3\n\n- 二件目\n",
			wantErr:   "重複",
		},
		"v 付き見出し": {
			changelog: "# 変更履歴\n\n## v1.2.3\n\n- 不正\n",
			wantErr:   "見つかりません",
		},
		"前方一致だけ": {
			changelog: "# 変更履歴\n\n## 1.2.30\n\n- 別の版\n",
			wantErr:   "見つかりません",
		},
		"リンクなし": {
			changelog: "# 変更履歴\n\n## [1.2.3]\n\n- 不正\n",
			wantErr:   "見つかりません",
		},
		"空のリンク": {
			changelog: "# 変更履歴\n\n## [1.2.3]()\n\n- 不正\n",
			wantErr:   "見つかりません",
		},
		"不正な日付": {
			changelog: "# 変更履歴\n\n## [1.2.3](https://example.test) (2026-99-99)\n\n- 不正\n",
			wantErr:   "見つかりません",
		},
		"plain 見出しの不正な日付": {
			changelog: "# 変更履歴\n\n## 1.2.3 (2026-99-99)\n\n- 不正\n",
			wantErr:   "見つかりません",
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := BuildPublishNotes(
				writeTestFile(t, "CHANGELOG.md", []byte(test.changelog)),
				writeTestFile(t, "CURRENT.md", []byte(validPublishContract())),
				"v1.2.3",
				writeTestSOTRepository(t),
			)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("BuildPublishNotes() のエラー = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestBuildPublishNotesValidatesReleaseContract(t *testing.T) {
	t.Parallel()

	_, err := BuildPublishNotes(
		writeTestFile(t, "CHANGELOG.md", []byte("## 1.2.3\n\n- 修正\n")),
		writeTestFile(t, "CURRENT.md", []byte(strings.Replace(
			validPublishContract(),
			"SOT-DEL-004",
			"SOT-DEL-999",
			1,
		))),
		"v1.2.3",
		writeTestSOTRepository(t),
	)
	if err == nil || !strings.Contains(err.Error(), "有効な SOT") {
		t.Fatalf("BuildPublishNotes() のエラー = %v", err)
	}
}

func TestBuildPublishNotesRetainsSemVerValidation(t *testing.T) {
	t.Parallel()

	_, err := BuildPublishNotes(
		writeTestFile(t, "CHANGELOG.md", []byte("## 1.2.3\n\n- 修正\n")),
		writeTestFile(t, "CURRENT.md", []byte(validPublishContract())),
		"1.2.3",
		writeTestSOTRepository(t),
	)
	if err == nil || !strings.Contains(err.Error(), "v で始まる SemVer") {
		t.Fatalf("BuildPublishNotes() のエラー = %v", err)
	}
}

func TestBuildPublishNotesRejectsOversizedChangelog(t *testing.T) {
	t.Parallel()

	_, err := BuildPublishNotes(
		writeTestFile(
			t,
			"CHANGELOG.md",
			[]byte(strings.Repeat("a", maxChangelogBytes+1)),
		),
		writeTestFile(t, "CURRENT.md", []byte(validPublishContract())),
		"v1.2.3",
		writeTestSOTRepository(t),
	)
	if err == nil || !strings.Contains(err.Error(), "大きすぎます") {
		t.Fatalf("BuildPublishNotes() のエラー = %v", err)
	}
}

func TestBuildPublishNotesRejectsNonRegularChangelog(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	_, err := BuildPublishNotes(
		directory,
		writeTestFile(t, "CURRENT.md", []byte(validPublishContract())),
		"v1.2.3",
		writeTestSOTRepository(t),
	)
	if err == nil || !strings.Contains(err.Error(), "通常ファイル") {
		t.Fatalf("BuildPublishNotes() のエラー = %v", err)
	}
}

func TestBuildPublishNotesRejectsSymlinkedChangelog(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "target.md")
	if err := os.WriteFile(target, []byte("## 1.2.3\n"), 0o600); err != nil {
		t.Fatalf("対象ファイルを作成できません: %v", err)
	}
	link := filepath.Join(root, "CHANGELOG.md")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink を作成できません: %v", err)
	}

	_, err := BuildPublishNotes(
		link,
		writeTestFile(t, "CURRENT.md", []byte(validPublishContract())),
		"v1.2.3",
		writeTestSOTRepository(t),
	)
	if err == nil || !strings.Contains(err.Error(), "通常ファイル") {
		t.Fatalf("BuildPublishNotes() のエラー = %v", err)
	}
}

func validPublishContract() string {
	return `# Japanese Law MCP v1.2.3 <!-- x-release-please-version -->

## 提供する SOT

- SOT-DEL-004

## 未実装の SOT 差分

なし

## 互換性のない変更

- SOT-DEL-007: 変更内容
`
}
