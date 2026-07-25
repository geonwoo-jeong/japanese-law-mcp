package sotcheck

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var (
	headingPattern = regexp.MustCompile(`(?m)^# (SOT-([A-Z]+)-([0-9]{3})): .+$`)
	statusPattern  = regexp.MustCompile(`(?m)^- 状態: (草案|有効|廃止)$`)
	rulePattern    = regexp.MustCompile(`(?m)^## 規定$`)
	linkPattern    = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
)

func TestDevelopmentPrinciplesChecksum(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	checksumPath := filepath.Join(root, "docs", "development-principles.sha256")
	checksumText := readText(t, checksumPath)
	fields := strings.Fields(checksumText)
	if len(fields) != 2 {
		t.Fatalf("開発原則のチェックサム形式が正しくありません: %q", checksumText)
	}

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fields[1])))
	if err != nil {
		t.Fatal(err)
	}
	actual := sha256.Sum256(content)
	if hex.EncodeToString(actual[:]) != fields[0] {
		t.Fatal("開発原則のチェックサムが一致しません")
	}
}

func TestSOTStructure(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	sotRoot := filepath.Join(root, "sot")
	rules := collectRules(t, sotRoot)
	ids := make(map[string]string, len(rules))

	for _, path := range rules {
		path := path
		t.Run(relativePath(sotRoot, path), func(t *testing.T) {
			validateRule(t, path, ids)
		})
	}
}

func TestSOTIndexes(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	sotRoot := filepath.Join(root, "sot")
	entries, err := os.ReadDir(sotRoot)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(sotRoot, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			validateIndex(t, dir)
		})
	}
}

func TestMarkdownLinksExist(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	for _, base := range []string{"AGENTS.md", "docs", "sot", "wiki"} {
		path := filepath.Join(root, base)
		walkMarkdown(t, path, func(markdownPath string) {
			validateLinks(t, markdownPath)
		})
	}
}

func validateRule(t *testing.T, path string, ids map[string]string) {
	t.Helper()

	content := readText(t, path)
	headings := headingPattern.FindAllStringSubmatch(content, -1)
	if len(headings) != 1 {
		t.Fatalf("SOT 見出しは一つ必要です: %s", path)
	}
	if len(statusPattern.FindAllStringSubmatch(content, -1)) != 1 {
		t.Fatalf("許可された状態は一つ必要です: %s", path)
	}
	if len(rulePattern.FindAllString(content, -1)) != 1 {
		t.Fatalf("規定節は一つ必要です: %s", path)
	}

	id, number := headings[0][1], headings[0][3]
	if previous, exists := ids[id]; exists {
		t.Fatalf("SOT ID %s が重複しています: %s, %s", id, previous, path)
	}
	ids[id] = path

	fileNumber := strings.SplitN(filepath.Base(path), "-", 2)[0]
	if fileNumber != number[len(number)-2:] {
		t.Fatalf("SOT ID とファイル番号が一致しません: %s", path)
	}
}

func validateIndex(t *testing.T, dir string) {
	t.Helper()

	indexPath := filepath.Join(dir, "00-index.md")
	index := readText(t, indexPath)
	links := localMarkdownLinks(index)
	actual := make([]string, 0)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "00-index.md" || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		actual = append(actual, entry.Name())
	}
	slices.Sort(actual)
	slices.Sort(links)

	if !slices.Equal(actual, links) {
		t.Fatalf("索引と規則ファイルが一致しません\n規則: %v\n索引: %v", actual, links)
	}
}

func collectRules(t *testing.T, root string) []string {
	t.Helper()

	var paths []string
	walkMarkdown(t, root, func(path string) {
		if filepath.Base(path) != "00-index.md" {
			paths = append(paths, path)
		}
	})
	slices.Sort(paths)
	return paths
}

func walkMarkdown(t *testing.T, root string, visit func(string)) {
	t.Helper()

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".md" {
			visit(path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func validateLinks(t *testing.T, path string) {
	t.Helper()

	content := readText(t, path)
	for _, match := range linkPattern.FindAllStringSubmatch(content, -1) {
		target := strings.SplitN(match[1], "#", 2)[0]
		if target == "" || strings.Contains(target, "://") {
			continue
		}
		resolved := filepath.Join(filepath.Dir(path), filepath.FromSlash(target))
		if _, err := os.Stat(resolved); err != nil {
			t.Errorf("相対リンクの参照先がありません: %s -> %s", path, match[1])
		}
	}
}

func localMarkdownLinks(content string) []string {
	var links []string
	for _, match := range linkPattern.FindAllStringSubmatch(content, -1) {
		target := strings.SplitN(match[1], "#", 2)[0]
		if filepath.Dir(target) == "." && filepath.Ext(target) == ".md" {
			links = append(links, target)
		}
	}
	return links
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("リポジトリの位置を取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readText(t *testing.T, path string) string {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("ファイルを閉じられません: %v", err)
		}
	}()

	var builder strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		builder.WriteString(scanner.Text())
		builder.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return builder.String()
}

func relativePath(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return strconv.Quote(path)
	}
	return filepath.ToSlash(relative)
}
