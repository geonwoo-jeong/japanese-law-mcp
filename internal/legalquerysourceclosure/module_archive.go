package legalquerysourceclosure

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const maximumModuleZipPathBytes = 512

type moduleFileHash struct {
	name   string
	digest string
}

func inspectModuleArtifact(ctx context.Context, identity listedModule, artifact ModuleArtifact) (ModuleDependency, error) {
	if len(artifact.Zip) == 0 || len(artifact.Zip) > MaximumModuleZipBytes {
		return ModuleDependency{}, fmt.Errorf("module zip の原 byte size が上限外です")
	}
	if len(artifact.GoMod) == 0 || len(artifact.GoMod) > MaximumModuleGoModBytes {
		return ModuleDependency{}, fmt.Errorf("module go.mod の原 byte size が上限外です")
	}
	if artifact.ZipHash != identity.zipSum || !moduleSumPattern.MatchString(artifact.ZipHash) {
		return ModuleDependency{}, fmt.Errorf("module cache の ziphash が module identity と一致しません")
	}
	if err := verifyModuleGoMod(identity.path, artifact.GoMod); err != nil {
		return ModuleDependency{}, err
	}
	zipSum, entryCount, expandedBytes, err := inspectModuleZip(ctx, identity.path, identity.version, artifact.Zip)
	if err != nil {
		return ModuleDependency{}, err
	}
	if zipSum != identity.zipSum {
		return ModuleDependency{}, fmt.Errorf("module zip の h1 checksum が一致しません")
	}
	goModSum := singleFileH1("go.mod", artifact.GoMod)
	if goModSum != identity.goModSum {
		return ModuleDependency{}, fmt.Errorf("module go.mod の h1 checksum が一致しません")
	}
	return ModuleDependency{
		ModulePath:               identity.path,
		Version:                  identity.version,
		ModuleZipSum:             identity.zipSum,
		ModuleZipRawSHA256:       rawSHA256(artifact.Zip),
		ModuleZipByteLength:      int64(len(artifact.Zip)),
		ModuleZipEntryCount:      entryCount,
		ModuleExpandedByteLength: expandedBytes,
		ModuleGoModSum:           identity.goModSum,
		ModuleGoModRawSHA256:     rawSHA256(artifact.GoMod),
	}, nil
}

func verifyModuleGoMod(modulePath string, raw []byte) error {
	parsed, err := modfile.Parse("go.mod", raw, nil)
	if err != nil || parsed.Module == nil || parsed.Module.Mod.Path != modulePath {
		return fmt.Errorf("module go.mod の module path が一致しません")
	}
	return nil
}

func inspectModuleZip(ctx context.Context, modulePath string, version string, raw []byte) (string, int, int64, error) {
	escapedPath, err := module.EscapePath(modulePath)
	if err != nil {
		return "", 0, 0, fmt.Errorf("module path を escape できません")
	}
	escapedVersion, err := module.EscapeVersion(version)
	if err != nil {
		return "", 0, 0, fmt.Errorf("module version を escape できません")
	}
	prefix := escapedPath + "@" + escapedVersion + "/"
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return "", 0, 0, fmt.Errorf("module zip を解析できません")
	}
	if len(reader.File) == 0 || len(reader.File) > MaximumModuleZipEntries {
		return "", 0, 0, fmt.Errorf("module zip entry 数が上限外です")
	}
	seen := make(map[string]struct{}, len(reader.File))
	seenFolded := make(map[string]string, len(reader.File))
	fileHashes := make([]moduleFileHash, 0, len(reader.File))
	var expandedBytes int64
	for _, entry := range reader.File {
		if err := ctx.Err(); err != nil {
			return "", 0, 0, fmt.Errorf("module zip 検証が中止されました: %w", err)
		}
		if err := validateModuleZipEntry(entry, prefix, seen, seenFolded); err != nil {
			return "", 0, 0, err
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		if entry.UncompressedSize64 > MaximumModuleEntryBytes ||
			entry.UncompressedSize64 > uint64(MaximumModuleExpandBytes-expandedBytes) {
			return "", 0, 0, fmt.Errorf("module zip の展開 size が上限を超えています")
		}
		digest, size, err := hashModuleZipEntry(ctx, entry)
		if err != nil {
			return "", 0, 0, err
		}
		if size != int64(entry.UncompressedSize64) {
			return "", 0, 0, fmt.Errorf("module zip entry の宣言 size と原 byte が一致しません")
		}
		expandedBytes += size
		fileHashes = append(fileHashes, moduleFileHash{name: entry.Name, digest: digest})
	}
	if len(fileHashes) == 0 {
		return "", 0, 0, fmt.Errorf("module zip に通常 file がありません")
	}
	return moduleFilesH1(fileHashes), len(reader.File), expandedBytes, nil
}

func validateModuleZipEntry(
	entry *zip.File,
	prefix string,
	seen map[string]struct{},
	seenFolded map[string]string,
) error {
	name := entry.Name
	if name == "" || len(name) > maximumModuleZipPathBytes || !utf8.ValidString(name) ||
		strings.Contains(name, `\`) || strings.ContainsAny(name, "\x00\r\n") || !strings.HasPrefix(name, prefix) {
		return fmt.Errorf("module zip entry path が不正です")
	}
	relative := strings.TrimPrefix(name, prefix)
	isDirectory := entry.FileInfo().IsDir()
	if isDirectory {
		relative = strings.TrimSuffix(relative, "/")
	}
	if relative == "" {
		if !isDirectory || name != prefix {
			return fmt.Errorf("module zip entry path が不正です")
		}
	} else if !fs.ValidPath(relative) || relative == "." || path.Clean(relative) != relative {
		return fmt.Errorf("module zip entry path が正規化されていません")
	}
	mode := entry.Mode()
	if !isDirectory && !mode.IsRegular() {
		return fmt.Errorf("module zip entry は通常 file または directory でなければなりません")
	}
	if isDirectory && mode.Type() != fs.ModeDir {
		return fmt.Errorf("module zip directory の file type が不正です")
	}
	if _, exists := seen[name]; exists {
		return fmt.Errorf("module zip entry path が重複しています")
	}
	seen[name] = struct{}{}
	folded := strings.ToLower(name)
	if previous, exists := seenFolded[folded]; exists && previous != name {
		return fmt.Errorf("module zip entry path に case collision があります")
	}
	seenFolded[folded] = name
	return nil
}

func hashModuleZipEntry(ctx context.Context, entry *zip.File) (string, int64, error) {
	opened, err := entry.Open()
	if err != nil {
		return "", 0, fmt.Errorf("module zip entry を開けません")
	}
	hasher := sha256.New()
	//nolint:gosec // SOT-ENG-038: entry header と一件・module 合計上限を先に検証し、LimitReader で上限加算一 byteだけを読む。
	written, copyErr := io.Copy(hasher, io.LimitReader(contextReader{ctx: ctx, reader: opened}, MaximumModuleEntryBytes+1))
	closeErr := opened.Close()
	if copyErr != nil || closeErr != nil || written > MaximumModuleEntryBytes {
		return "", 0, fmt.Errorf("module zip entry を上限内で完全に読めません")
	}
	return hex.EncodeToString(hasher.Sum(nil)), written, nil
}

func moduleFilesH1(files []moduleFileHash) string {
	files = slices.Clone(files)
	slices.SortFunc(files, func(left, right moduleFileHash) int {
		return strings.Compare(left.name, right.name)
	})
	hasher := sha256.New()
	for _, file := range files {
		_, _ = fmt.Fprintf(hasher, "%s  %s\n", file.digest, file.name)
	}
	return "h1:" + base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}

func singleFileH1(name string, raw []byte) string {
	digest := sha256.Sum256(raw)
	return moduleFilesH1([]moduleFileHash{{name: name, digest: hex.EncodeToString(digest[:])}})
}
