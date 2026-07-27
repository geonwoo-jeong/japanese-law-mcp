package releasecheck

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	maxReleaseNotesBytes = 1024 * 1024
	maxSOTFileBytes      = 2 * 1024 * 1024
)

var (
	semverTagPattern = regexp.MustCompile(
		`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)` +
			`(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)` +
			`(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?` +
			`(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`,
	)
	sotIDPattern      = regexp.MustCompile(`\bSOT-[A-Z]+-[0-9]{3}\b`)
	sotHeadingPattern = regexp.MustCompile(
		`(?m)^# (SOT-[A-Z]+-[0-9]{3}): .+$`,
	)
	sotStatusPattern = regexp.MustCompile(
		`(?m)^- 状態: (草案|有効|廃止)$`,
	)
)

var requiredReleaseSections = []string{
	"提供する SOT",
	"未実装の SOT 差分",
	"互換性のない変更",
}

type releaseNotesSection struct {
	name string
	body string
}

func validateReleaseNotes(path, tag, repository string) error {
	if !semverTagPattern.MatchString(tag) {
		return fmt.Errorf("tag は v で始まる SemVer でなければなりません")
	}
	content, err := readLimitedRegularFile(path, maxReleaseNotesBytes)
	if err != nil {
		return fmt.Errorf("リリース情報を読めません: %w", err)
	}
	if err := validateReleaseHeading(string(content), tag); err != nil {
		return err
	}
	activeSOTIDs, err := loadActiveSOTIDs(repository)
	if err != nil {
		return fmt.Errorf("有効な SOT を読み込めません: %w", err)
	}
	sections := parseH2Sections(string(content))
	if err := validateReleaseSections(sections, activeSOTIDs); err != nil {
		return err
	}
	return nil
}

func validateReleaseHeading(content, tag string) error {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	heading, _, _ := strings.Cut(normalized, "\n")
	expected := "# Japanese Law MCP " + tag
	annotated := expected + " <!-- x-release-please-version -->"
	if heading != expected && heading != annotated {
		return fmt.Errorf(
			"リリース情報の見出しは「%s」でなければなりません",
			expected,
		)
	}
	return nil
}

func parseH2Sections(content string) []releaseNotesSection {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	sections := make([]releaseNotesSection, 0, len(requiredReleaseSections))
	current := -1
	fence := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if nextFence, matched := markdownFence(trimmed, fence); matched {
			fence = nextFence
		}
		if fence == "" && strings.HasPrefix(line, "## ") {
			sections = append(sections, releaseNotesSection{
				name: strings.TrimSpace(strings.TrimPrefix(line, "## ")),
			})
			current = len(sections) - 1
			continue
		}
		if current >= 0 {
			if sections[current].body != "" {
				sections[current].body += "\n"
			}
			sections[current].body += line
		}
	}
	return sections
}

func markdownFence(line, current string) (string, bool) {
	for _, marker := range []string{"```", "~~~"} {
		if !strings.HasPrefix(line, marker) {
			continue
		}
		if current == "" {
			return marker, true
		}
		if current == marker {
			return "", true
		}
	}
	return current, false
}

func validateReleaseSections(
	sections []releaseNotesSection,
	activeSOTIDs map[string]struct{},
) error {
	if len(sections) != len(requiredReleaseSections) {
		return fmt.Errorf(
			"リリース情報には正確な H2 セクションを 3 件記載してください",
		)
	}
	for index, expected := range requiredReleaseSections {
		section := sections[index]
		if section.name != expected {
			return fmt.Errorf(
				"H2 セクション %d は「%s」でなければなりません",
				index+1,
				expected,
			)
		}
		body := strings.TrimSpace(section.body)
		if body == "" {
			return fmt.Errorf("H2 セクション「%s」の本文がありません", expected)
		}
		ids := sotIDPattern.FindAllString(body, -1)
		if index == 0 {
			if len(ids) == 0 {
				return fmt.Errorf("提供する SOT の本文には SOT ID が必要です")
			}
		} else if len(ids) == 0 && !containsNoneDeclaration(body) {
			return fmt.Errorf(
				"H2 セクション「%s」には SOT ID または「なし」が必要です",
				expected,
			)
		}
		if len(ids) > 0 && containsNoneDeclaration(body) {
			return fmt.Errorf(
				"H2 セクション「%s」で SOT ID と「なし」は併記できません",
				expected,
			)
		}
		for _, id := range ids {
			if _, exists := activeSOTIDs[id]; !exists {
				return fmt.Errorf("有効な SOT ではありません: %s", id)
			}
		}
	}
	return nil
}

func loadActiveSOTIDs(repository string) (map[string]struct{}, error) {
	sotRoot := filepath.Join(repository, "sot")
	info, err := os.Lstat(sotRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("sot は directory ではありません")
	}

	active := make(map[string]struct{})
	err = filepath.WalkDir(sotRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() ||
			entry.Name() == "00-index.md" ||
			filepath.Ext(entry.Name()) != ".md" {
			return nil
		}
		content, readErr := readLimitedRegularFile(path, maxSOTFileBytes)
		if readErr != nil {
			return fmt.Errorf("%s を読めません: %w", path, readErr)
		}
		headings := sotHeadingPattern.FindAllSubmatch(content, -1)
		statuses := sotStatusPattern.FindAllSubmatch(content, -1)
		if len(headings) != 1 || len(statuses) != 1 {
			return fmt.Errorf("%s の SOT 見出しまたは状態が不正です", path)
		}
		if string(statuses[0][1]) != "有効" {
			return nil
		}
		id := string(headings[0][1])
		if _, duplicate := active[id]; duplicate {
			return fmt.Errorf("有効な SOT ID が重複しています: %s", id)
		}
		active[id] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(active) == 0 {
		return nil, fmt.Errorf("有効な SOT がありません")
	}
	return active, nil
}

func containsNoneDeclaration(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		declaration := strings.TrimSpace(line)
		declaration = strings.TrimSpace(strings.TrimPrefix(declaration, "-"))
		if declaration == "なし" {
			return true
		}
	}
	return false
}

func readLimitedRegularFile(path string, maximum int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("通常ファイルではありません")
	}
	if info.Size() > maximum {
		return nil, fmt.Errorf("ファイルが大きすぎます")
	}
	content, err := os.ReadFile(path) //nolint:gosec // SOT-ENG-019: Lstat 済みの明示的な検証入力を読む。
	if err != nil {
		return nil, err
	}
	return content, nil
}
