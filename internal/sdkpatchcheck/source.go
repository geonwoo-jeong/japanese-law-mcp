package sdkpatchcheck

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/sumdb/dirhash"
)

type moduleMetadata struct {
	Path     string
	Version  string
	Sum      string
	GoModSum string
	Zip      string
	Replace  *moduleMetadata
	Error    *struct{ Err string }
	Origin   *struct{ Hash string }
}

func loadArchive(ctx context.Context, repository string, identity moduleIdentity) ([]byte, error) {
	selected, err := runGoMetadata(ctx, repository, true, "list", "-m", "-mod=readonly", "-json", ModulePath)
	if err != nil {
		return nil, err
	}
	if selected.Path != ModulePath || selected.Version != UpstreamVersion || selected.Replace == nil ||
		selected.Replace.Path != "./"+sourceDirectory || selected.Replace.Version != "" {
		return nil, fmt.Errorf("選択中の SDK が version 限定の固定 copy と一致しません")
	}
	downloaded, err := downloadMetadata(ctx)
	if err != nil {
		return nil, err
	}
	if downloaded.Path != ModulePath || downloaded.Version != UpstreamVersion || downloaded.Replace != nil ||
		downloaded.Sum != identity.ZipSum || downloaded.GoModSum != identity.GoModSum || downloaded.Zip == "" {
		return nil, fmt.Errorf("元 SDK の取得 identity が固定 manifest と一致しません")
	}
	if downloaded.Origin != nil && downloaded.Origin.Hash != identity.OriginCommit {
		return nil, fmt.Errorf("元 SDK の取得 commit が固定 manifest と一致しません")
	}
	return readArchiveFile(downloaded.Zip)
}

// SOT-ENG-020: 明示版の go mod download は go.sum を更新し得るため、
// 検査対象から分離した一時 module で実行する。元 SDK の cache へ patch しない。
func downloadMetadata(ctx context.Context) (metadata moduleMetadata, err error) {
	directory, err := os.MkdirTemp("", "jlm-sdk-upstream-")
	if err != nil {
		return moduleMetadata{}, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(directory)) }()
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.invalid/jlm-sdk-upstream-check\n\ngo 1.25.0\n"), 0600); err != nil {
		return moduleMetadata{}, err
	}
	return runGoMetadata(ctx, directory, false, "mod", "download", "-json", ModulePath+"@"+UpstreamVersion)
}

func runGoMetadata(ctx context.Context, repository string, readonly bool, args ...string) (moduleMetadata, error) {
	command := exec.CommandContext(ctx, "go", args...) //nolint:gosec // SOT-ENG-020: 固定 Go 操作と固定 module identity の argv だけを渡し、shell を使わない。
	command.Dir = repository
	command.Env = metadataEnvironment(os.Environ(), readonly)
	output, err := command.Output()
	if err != nil {
		return moduleMetadata{}, fmt.Errorf("SDK の module identity を照合できません: %w", err)
	}
	var value moduleMetadata
	if err := json.Unmarshal(output, &value); err != nil {
		return moduleMetadata{}, fmt.Errorf("SDK の module identity を解釈できません: %w", err)
	}
	if value.Error != nil {
		return moduleMetadata{}, fmt.Errorf("元 SDK を取得できません: %s", value.Error.Err)
	}
	return value, nil
}

func metadataEnvironment(inherited []string, readonly bool) []string {
	controlled := map[string]bool{"GOENV": true, "GOWORK": true, "GOTOOLCHAIN": true, "GOFLAGS": true, "GO111MODULE": true}
	result := make([]string, 0, len(inherited)+len(controlled))
	for _, entry := range inherited {
		name, _, _ := strings.Cut(entry, "=")
		if !controlled[name] {
			result = append(result, entry)
		}
	}
	flags := "GOFLAGS=-mod=readonly -p=1"
	if !readonly {
		// 対象 repository から分離した一時 module にだけ取得 checksum を書く。
		flags = "GOFLAGS=-mod=mod -p=1"
	}
	return append(result, "GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", flags, "GO111MODULE=on")
}

func readArchiveFile(filename string) (data []byte, err error) {
	info, err := os.Lstat(filename)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxArchiveBytes {
		return nil, fmt.Errorf("元 SDK ZIP は上限内の通常 file でなければなりません")
	}
	file, err := os.Open(filename) //nolint:gosec // SOT-ENG-020: 固定 go mod download の返した ZIP を上限・原 hash とともに照合する。
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("元 SDK ZIP が照合中に置き換えられました")
	}
	data, err = io.ReadAll(io.LimitReader(file, maxArchiveBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > maxArchiveBytes {
		return nil, fmt.Errorf("元 SDK ZIP がサイズ上限を超えています")
	}
	return data, nil
}

func extractArchive(data []byte, manifest patchManifest) (map[string][]byte, error) {
	if digest(data) != manifest.Module.ZipSHA256 {
		return nil, fmt.Errorf("元 SDK ZIP の raw SHA-256 が一致しません")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("元 SDK ZIP を開けません: %w", err)
	}
	prefix := ModulePath + "@" + UpstreamVersion + "/"
	entries := make(map[string]*zip.File, len(archive.File))
	names := make([]string, 0, len(archive.File))
	for _, entry := range archive.File {
		relative, ok := strings.CutPrefix(entry.Name, prefix)
		if !ok || !validRelativePath(relative) || !entry.Mode().IsRegular() || entry.UncompressedSize64 > maxSourceBytes {
			return nil, fmt.Errorf("元 SDK ZIP に不正な entry があります")
		}
		if _, duplicate := entries[entry.Name]; duplicate {
			return nil, fmt.Errorf("元 SDK ZIP の entry が重複しています")
		}
		entries[entry.Name] = entry
		names = append(names, entry.Name)
	}
	if len(entries) != len(manifest.UpstreamFiles) {
		return nil, fmt.Errorf("元 SDK ZIP と manifest の file 件数が一致しません")
	}
	sum, err := dirhash.Hash1(names, func(name string) (io.ReadCloser, error) { return entries[name].Open() })
	if err != nil {
		return nil, fmt.Errorf("元 SDK ZIP の h1 を計算できません: %w", err)
	}
	if sum != manifest.Module.ZipSum {
		return nil, fmt.Errorf("元 SDK ZIP の h1 が一致しません")
	}
	files := make(map[string][]byte, len(entries))
	for _, expected := range manifest.UpstreamFiles {
		entry := entries[prefix+expected.Path]
		if entry == nil {
			return nil, fmt.Errorf("元 SDK ZIP に固定 file がありません: %s", expected.Path)
		}
		content, err := readZipFile(entry)
		if err != nil {
			return nil, err
		}
		if int64(len(content)) != expected.Bytes || digest(content) != expected.SHA256 {
			return nil, fmt.Errorf("元 SDK file の hash が一致しません: %s", expected.Path)
		}
		files[expected.Path] = content
	}
	gomod, exists := files["go.mod"]
	if !exists || digest(gomod) != manifest.Module.GoModSHA256 {
		return nil, fmt.Errorf("元 SDK go.mod の raw SHA-256 が一致しません")
	}
	gomodSum, err := dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(gomod)), nil
	})
	if err != nil {
		return nil, err
	}
	if gomodSum != manifest.Module.GoModSum {
		return nil, fmt.Errorf("元 SDK go.mod の h1 が一致しません")
	}
	return files, nil
}

func readZipFile(entry *zip.File) (data []byte, err error) {
	reader, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, reader.Close()) }()
	data, err = io.ReadAll(io.LimitReader(reader, maxSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxSourceBytes {
		return nil, fmt.Errorf("元 SDK file がサイズ上限を超えています")
	}
	return data, nil
}
