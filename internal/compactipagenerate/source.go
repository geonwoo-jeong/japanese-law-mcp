package compactipagenerate

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/sumdb/dirhash"
)

type sourceData struct {
	module     moduleIdentity
	dictionary []byte
	license    []byte
	notice     []byte
}

type moduleMetadata struct {
	Path     string
	Version  string
	Sum      string
	GoModSum string
	Zip      string
	Replace  *moduleMetadata
	Error    *struct{ Err string }
}

func loadSource(ctx context.Context, repository string) (sourceData, error) {
	version, err := requiredVersion(repository)
	if err != nil {
		return sourceData{}, err
	}
	identity, err := readGoSum(repository, version)
	if err != nil {
		return sourceData{}, err
	}
	selected, err := runGoMetadata(ctx, repository, "list", "-m", "-mod=readonly", "-json", modulePath)
	if err != nil {
		return sourceData{}, err
	}
	if selected.Path != modulePath || selected.Version != version || selected.Replace != nil {
		return sourceData{}, fmt.Errorf("IPA の選択版が go.mod の固定版と一致しません")
	}
	downloaded, err := runGoMetadata(ctx, repository, "mod", "download", "-json", modulePath+"@"+version)
	if err != nil {
		return sourceData{}, err
	}
	if downloaded.Path != modulePath || downloaded.Version != version || downloaded.Replace != nil {
		return sourceData{}, fmt.Errorf("IPA の取得物が固定版と一致しません")
	}
	if downloaded.Sum != identity.Sum || downloaded.GoModSum != identity.GoModSum {
		return sourceData{}, fmt.Errorf("IPA module の取得物が go.sum と一致しません")
	}
	archiveBytes, err := os.ReadFile(downloaded.Zip)
	if err != nil {
		return sourceData{}, fmt.Errorf("IPA module の取得物を読み込めません: %w", err)
	}
	return extractSource(archiveBytes, identity)
}

func requiredVersion(repository string) (string, error) {
	raw, err := readRepositoryFile(repository, "go.mod")
	if err != nil {
		return "", fmt.Errorf("go.mod を読み込めません: %w", err)
	}
	file, err := modfile.Parse("go.mod", raw, nil)
	if err != nil {
		return "", fmt.Errorf("go.mod を解釈できません: %w", err)
	}
	for _, item := range file.Replace {
		if item.Old.Path == modulePath {
			return "", fmt.Errorf("IPA module の置換は使用できません")
		}
	}
	for _, item := range file.Require {
		if item.Mod.Path == modulePath {
			return item.Mod.Version, nil
		}
	}
	return "", fmt.Errorf("go.mod に固定した IPA module がありません")
}

func runGoMetadata(ctx context.Context, repository string, args ...string) (moduleMetadata, error) {
	command := exec.CommandContext(ctx, "go", args...) //nolint:gosec // SOT-ENG-051: G204 は固定した Go の操作と検証対象 module の argv のみを受ける。
	command.Dir = repository
	output, err := command.Output()
	if err != nil {
		return moduleMetadata{}, fmt.Errorf("IPA module の取得情報を確認できません: %w", err)
	}
	var value moduleMetadata
	if err := json.Unmarshal(output, &value); err != nil {
		return moduleMetadata{}, fmt.Errorf("IPA module の取得情報を解釈できません: %w", err)
	}
	if value.Error != nil {
		return moduleMetadata{}, fmt.Errorf("IPA module を取得できません: %s", value.Error.Err)
	}
	return value, nil
}

func readGoSum(repository, version string) (moduleIdentity, error) {
	raw, err := readRepositoryFile(repository, "go.sum")
	if err != nil {
		return moduleIdentity{}, fmt.Errorf("go.sum を読み込めません: %w", err)
	}
	expected := map[string]string{version: "", version + "/go.mod": ""}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != modulePath {
			continue
		}
		previous, relevant := expected[fields[1]]
		if !relevant {
			continue
		}
		if previous != "" || !strings.HasPrefix(fields[2], "h1:") {
			return moduleIdentity{}, fmt.Errorf("IPA module の固定 checksum が不正または重複しています")
		}
		expected[fields[1]] = fields[2]
	}
	if expected[version] == "" || expected[version+"/go.mod"] == "" {
		return moduleIdentity{}, fmt.Errorf("IPA module の固定 checksum が go.sum にありません")
	}
	return moduleIdentity{Path: modulePath, Version: version, Sum: expected[version], GoModSum: expected[version+"/go.mod"]}, nil
}

// SOT-ENG-051: checksum の検証と抽出は、一度読み込んだ同じ module ZIP に対して行う。
func extractSource(data []byte, identity moduleIdentity) (sourceData, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return sourceData{}, fmt.Errorf("IPA module ZIP を読み込めません: %w", err)
	}
	entries, names, err := moduleEntries(archive, identity)
	if err != nil {
		return sourceData{}, err
	}
	sum, err := dirhash.Hash1(names, func(name string) (io.ReadCloser, error) { return entries[name].Open() })
	if err != nil {
		return sourceData{}, fmt.Errorf("IPA module ZIP を検証できません: %w", err)
	}
	if sum != identity.Sum {
		return sourceData{}, fmt.Errorf("IPA module ZIP の checksum が一致しません")
	}
	prefix := identity.Path + "@" + identity.Version + "/"
	gomod, err := readZipEntry(entries[prefix+"go.mod"])
	if err != nil {
		return sourceData{}, err
	}
	gomodSum, err := dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(gomod)), nil })
	if err != nil {
		return sourceData{}, fmt.Errorf("IPA の go.mod を検証できません: %w", err)
	}
	if gomodSum != identity.GoModSum {
		return sourceData{}, fmt.Errorf("IPA の go.mod checksum が一致しません")
	}
	source := sourceData{module: identity}
	for name, destination := range map[string]*[]byte{"ipa.dict": &source.dictionary, "LICENSE": &source.license, "NOTICE.txt": &source.notice} {
		value, err := readZipEntry(entries[prefix+name])
		if err != nil {
			return sourceData{}, err
		}
		*destination = value
	}
	return source, nil
}

func moduleEntries(archive *zip.Reader, identity moduleIdentity) (map[string]*zip.File, []string, error) {
	prefix := identity.Path + "@" + identity.Version + "/"
	entries := make(map[string]*zip.File, len(archive.File))
	names := make([]string, 0, len(archive.File))
	for _, entry := range archive.File {
		relative, ok := strings.CutPrefix(entry.Name, prefix)
		if !ok || !validRelativePath(relative) || !entry.Mode().IsRegular() {
			return nil, nil, fmt.Errorf("IPA module ZIP の path が不正です")
		}
		if _, duplicate := entries[entry.Name]; duplicate {
			return nil, nil, fmt.Errorf("IPA module ZIP の entry が重複しています")
		}
		entries[entry.Name] = entry
		names = append(names, entry.Name)
	}
	return entries, names, nil
}

func validRelativePath(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\\n\r") || strings.HasPrefix(value, "/") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func readZipEntry(entry *zip.File) ([]byte, error) {
	if entry == nil {
		return nil, fmt.Errorf("IPA の固定入力が module ZIP にありません")
	}
	reader, err := entry.Open()
	if err != nil {
		return nil, fmt.Errorf("IPA 入力 %s を開けません: %w", entry.Name, err)
	}
	data, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil {
		return nil, fmt.Errorf("IPA 入力 %s を読み込めません: %w", entry.Name, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("IPA 入力 %s を閉じられません: %w", entry.Name, closeErr)
	}
	return data, nil
}
