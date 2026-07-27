package releasecheck

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const maxChangelogBytes = 8 * 1024 * 1024

// BuildPublishNotes は、SOT-DEL-014 に従い変更履歴とリリース契約を結合する。
func BuildPublishNotes(
	changelogPath, releaseNotesPath, tag, repository string,
) ([]byte, error) {
	if strings.TrimSpace(changelogPath) == "" ||
		strings.TrimSpace(releaseNotesPath) == "" ||
		strings.TrimSpace(tag) == "" ||
		strings.TrimSpace(repository) == "" {
		return nil, fmt.Errorf(
			"変更履歴、リリース情報、tag および repository は必須です",
		)
	}
	contract, err := readValidatedContract(releaseNotesPath, tag, repository)
	if err != nil {
		return nil, err
	}
	changelog, err := readLimitedRegularFile(changelogPath, maxChangelogBytes)
	if err != nil {
		return nil, fmt.Errorf("変更履歴を読めません: %w", err)
	}
	if !utf8.Valid(changelog) {
		return nil, fmt.Errorf("変更履歴は有効な UTF-8 でなければなりません")
	}
	section, err := extractChangelogVersion(string(changelog), tag)
	if err != nil {
		return nil, err
	}
	return []byte(section + "\n\n---\n\n" + contract + "\n"), nil
}

func readValidatedContract(path, tag, repository string) (string, error) {
	if err := validateReleaseNotes(path, tag, repository); err != nil {
		return "", err
	}
	content, err := readLimitedRegularFile(path, maxReleaseNotesBytes)
	if err != nil {
		return "", fmt.Errorf("リリース情報を読めません: %w", err)
	}
	if !utf8.Valid(content) {
		return "", fmt.Errorf("リリース情報は有効な UTF-8 でなければなりません")
	}
	normalized := normalizeMarkdown(string(content))
	sections := parseH2Sections(normalized)
	return renderReleaseSections(sections), nil
}

func extractChangelogVersion(content, tag string) (string, error) {
	version := strings.TrimPrefix(tag, "v")
	normalized := normalizeMarkdown(content)
	lines := strings.Split(normalized, "\n")
	starts := matchingVersionSectionStarts(lines, version)
	switch len(starts) {
	case 0:
		return "", fmt.Errorf(
			"変更履歴に対象版 %s の H2 セクションが見つかりません",
			version,
		)
	case 1:
		end := nextH2SectionStart(lines, starts[0]+1)
		return strings.TrimSpace(strings.Join(lines[starts[0]:end], "\n")), nil
	default:
		return "", fmt.Errorf(
			"変更履歴に対象版 %s の H2 セクションが重複しています",
			version,
		)
	}
}

func matchingVersionSectionStarts(lines []string, version string) []int {
	starts := make([]int, 0, 1)
	fence := ""
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if nextFence, matched := markdownFence(trimmed, fence); matched {
			fence = nextFence
			continue
		}
		if fence == "" && isVersionHeading(line, version) {
			starts = append(starts, index)
		}
	}
	return starts
}

func nextH2SectionStart(lines []string, from int) int {
	fence := ""
	for index := from; index < len(lines); index++ {
		line := lines[index]
		trimmed := strings.TrimSpace(line)
		if nextFence, matched := markdownFence(trimmed, fence); matched {
			fence = nextFence
			continue
		}
		if fence == "" && strings.HasPrefix(line, "## ") {
			return index
		}
	}
	return len(lines)
}

func isVersionHeading(line, version string) bool {
	plain := "## " + version
	if line == plain {
		return true
	}
	if strings.HasPrefix(line, plain) &&
		validReleaseDateSuffix(strings.TrimPrefix(line, plain)) {
		return true
	}
	prefix := "## [" + version + "]("
	if !strings.HasPrefix(line, prefix) {
		return false
	}
	remainder := strings.TrimPrefix(line, prefix)
	linkEnd := strings.Index(remainder, ")")
	if linkEnd <= 0 {
		return false
	}
	suffix := remainder[linkEnd+1:]
	return suffix == "" || validReleaseDateSuffix(suffix)
}

func validReleaseDateSuffix(suffix string) bool {
	if len(suffix) != len(" (2006-01-02)") ||
		!strings.HasPrefix(suffix, " (") ||
		!strings.HasSuffix(suffix, ")") {
		return false
	}
	date := suffix[2 : len(suffix)-1]
	_, err := time.Parse(time.DateOnly, date)
	return err == nil
}

func renderReleaseSections(sections []releaseNotesSection) string {
	rendered := make([]string, 0, len(sections))
	for _, section := range sections {
		rendered = append(
			rendered,
			"## "+section.name+"\n\n"+strings.TrimSpace(section.body),
		)
	}
	return strings.Join(rendered, "\n\n")
}

func normalizeMarkdown(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}
