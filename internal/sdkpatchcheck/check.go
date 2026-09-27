package sdkpatchcheck

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"golang.org/x/mod/modfile"
)

// Run は、元 SDK の固定取得物、version 限定 replace、複製全体を読み取り専用で照合する。
func Run(ctx context.Context, repository string) error {
	return runCheck(ctx, repository, loadArchive)
}

func runCheck(ctx context.Context, repository string, load func(context.Context, string, moduleIdentity) ([]byte, error)) (err error) {
	if ctx == nil || repository == "" {
		return fmt.Errorf("SDK 照合には context と repository が必要です")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := os.OpenRoot(repository)
	if err != nil {
		return fmt.Errorf("SDK 照合の repository を開けません: %w", err)
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	raw, err := readRegular(root, patchDirectory+"/"+manifestName, maxManifestBytes)
	if err != nil {
		return err
	}
	manifest, err := decodeManifest(raw)
	if err != nil {
		return err
	}
	mod, err := readRegular(root, "go.mod", maxSourceBytes)
	if err != nil {
		return err
	}
	if err := verifyModuleReplacement(mod); err != nil {
		return err
	}
	archive, err := load(ctx, repository, manifest.Module)
	if err != nil {
		return err
	}
	return verifyCopy(ctx, root, manifest, archive)
}

func verifyModuleReplacement(data []byte) error {
	file, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return fmt.Errorf("SDK 照合の go.mod を解釈できません: %w", err)
	}
	requireCount, replaceCount := 0, 0
	for _, item := range file.Require {
		if item.Mod.Path == ModulePath {
			requireCount++
			if item.Mod.Version != UpstreamVersion {
				return fmt.Errorf("SDK require が元の固定版と一致しません")
			}
		}
	}
	for _, item := range file.Replace {
		if item.Old.Path == ModulePath {
			replaceCount++
			if item.Old.Version != UpstreamVersion || item.New.Path != "./"+sourceDirectory || item.New.Version != "" {
				return fmt.Errorf("SDK replace が version 限定の固定 copy を指していません")
			}
		}
	}
	if requireCount != 1 || replaceCount != 1 {
		return fmt.Errorf("SDK の固定 require と replace が一件ずつ必要です")
	}
	return nil
}

func verifyCopy(ctx context.Context, root *os.Root, manifest patchManifest, archive []byte) error {
	upstream, err := extractArchive(archive, manifest)
	if err != nil {
		return err
	}
	expected := make(map[string]fileDigest, len(manifest.UpstreamFiles)+len(manifest.Additions))
	for _, file := range manifest.UpstreamFiles {
		expected[file.Path] = file
	}
	change := manifest.Changes[0]
	expected[change.Path] = fileDigest{Path: change.Path, Bytes: change.Bytes, SHA256: change.AfterSHA256}
	for _, file := range manifest.Additions {
		expected[file.Path] = file
	}
	if err := verifyTree(ctx, root, expected); err != nil {
		return err
	}
	patch, err := readRegular(root, patchDirectory+"/"+manifest.PatchFile, maxSourceBytes)
	if err != nil {
		return err
	}
	if digest(patch) != manifest.PatchSHA256 {
		return fmt.Errorf("SDK 適用 patch の hash が一致しません")
	}
	license, err := readRegular(root, patchDirectory+"/MCP-SDK-LICENSE", maxManifestBytes)
	if err != nil {
		return err
	}
	if len(license) == 0 || !bytes.Equal(license, upstream["LICENSE"]) {
		return fmt.Errorf("SDK 配布許諾文が元 SDK の LICENSE と一致しません")
	}
	return ctx.Err()
}

func verifyTree(ctx context.Context, root *os.Root, expected map[string]fileDigest) error {
	allowedDirectories := map[string]bool{sourceDirectory: true}
	for name := range expected {
		for directory := path.Dir(sourceDirectory + "/" + name); directory != "."; directory = path.Dir(directory) {
			allowedDirectories[directory] = true
		}
	}
	if err := checkComponents(root, sourceDirectory); err != nil {
		return err
	}
	seen := make(map[string]bool, len(expected))
	err := fs.WalkDir(root.FS(), sourceDirectory, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if !allowedDirectories[name] {
				return fmt.Errorf("SDK copy に余分な directory があります: %s", name)
			}
			return nil
		}
		if name == sourceDirectory {
			return fmt.Errorf("SDK copy の root は directory でなければなりません")
		}
		relative := name[len(sourceDirectory)+1:]
		file, exists := expected[relative]
		if !exists || !entry.Type().IsRegular() {
			return fmt.Errorf("SDK copy に余分な file または symlink があります: %s", name)
		}
		data, err := readRegular(root, name, maxSourceBytes)
		if err != nil {
			return err
		}
		if int64(len(data)) != file.Bytes || digest(data) != file.SHA256 {
			return fmt.Errorf("SDK copy が固定された内容と一致しません: %s", name)
		}
		seen[relative] = true
		return nil
	})
	if err != nil {
		return err
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("SDK copy の file が不足しています")
	}
	return nil
}
