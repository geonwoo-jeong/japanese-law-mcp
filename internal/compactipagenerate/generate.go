// Package compactipagenerate は、SOT-ENG-051 の固定 IPA 辞書を生成・照合する。
package compactipagenerate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
)

const (
	modulePath    = "github.com/ikawaha/kagome-dict/ipa"
	dataDirectory = "internal/nlp/kagome/data"
	manifestName  = "ipa-manifest.json"
)

// Options は、生成対象の repository と照合専用の選択を保持する。
type Options struct {
	Repository string
	Check      bool
}

type fileDigest struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type moduleIdentity struct {
	Path     string `json:"path"`
	Version  string `json:"version"`
	Sum      string `json:"sum"`
	GoModSum string `json:"goModSum"`
}

type generatorIdentity struct {
	Path      string       `json:"path"`
	Sources   []fileDigest `json:"sources"`
	Toolchain string       `json:"toolchain"`
}

type manifest struct {
	FormatVersion int               `json:"formatVersion"`
	Module        moduleIdentity    `json:"module"`
	Generator     generatorIdentity `json:"generator"`
	Inputs        []fileDigest      `json:"inputs"`
	Outputs       []fileDigest      `json:"outputs"`
	Retained      []fileDigest      `json:"retainedEntries"`
	Excluded      []string          `json:"excludedEntries"`
	Words         int               `json:"words"`
	Classes       int               `json:"classes"`
	Names         int               `json:"names"`
}

type artifactSet struct {
	files    map[string][]byte
	manifest manifest
}

// Run は、選択された module を検証し、生成物全体を再現または照合する。
func Run(ctx context.Context, options Options) error {
	if ctx == nil || options.Repository == "" {
		return fmt.Errorf("生成には context と repository が必要です")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	source, err := loadSource(ctx, options.Repository)
	if err != nil {
		return err
	}
	generated, err := buildArtifacts(source)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	sources, err := generatorSources(options.Repository)
	if err != nil {
		return err
	}
	generated.manifest.Generator = generatorIdentity{
		Path: "cmd/compact-ipa-generate", Sources: sources, Toolchain: runtime.Version(),
	}
	if options.Check {
		return checkArtifacts(options.Repository, generated)
	}
	encoded, err := encodeManifest(generated.manifest)
	if err != nil {
		return err
	}
	generated.files[manifestName] = encoded
	return replaceArtifacts(ctx, options.Repository, generated.files)
}

func encodeManifest(value manifest) ([]byte, error) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("IPA manifest を生成できません: %w", err)
	}
	return append(encoded, '\n'), nil
}

func decodeManifest(data []byte) (manifest, error) {
	limits := legalqueryartifact.JSONLimits{Depth: 8, Values: 2048, RejectNull: true}
	if err := legalqueryartifact.InspectJSONObject(data, limits); err != nil {
		return manifest{}, err
	}
	var value manifest
	if err := legalqueryartifact.DecodeClosed(data, &value); err != nil {
		return manifest{}, err
	}
	if value.FormatVersion != 1 {
		return manifest{}, fmt.Errorf("IPA manifest の形式が未対応です")
	}
	if !validToolchain(value.Generator.Toolchain) {
		return manifest{}, fmt.Errorf("IPA manifest の生成時 toolchain が不正です")
	}
	return value, nil
}

func validToolchain(value string) bool {
	if !strings.HasPrefix(value, "go1.") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(value, "go"), ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return false
		}
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				return false
			}
		}
	}
	return true
}

func checkArtifacts(repository string, expected artifactSet) error {
	raw, err := readRepositoryFile(repository, dataDirectory+"/"+manifestName)
	if err != nil {
		return fmt.Errorf("IPA manifest を読み込めません: %w", err)
	}
	actual, err := decodeManifest(raw)
	if err != nil {
		return err
	}
	// SOT-ENG-051: 保存済み生成時の版を保持し、照合を実行する Go の版で付け替えない。
	expected.manifest.Generator.Toolchain = actual.Generator.Toolchain
	if !reflect.DeepEqual(actual, expected.manifest) {
		return fmt.Errorf("IPA manifest が固定入力または生成器と一致しません")
	}
	canonical, err := encodeManifest(expected.manifest)
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, canonical) {
		return fmt.Errorf("IPA manifest の byte 列が再生成結果と一致しません")
	}
	for name, want := range expected.files {
		got, readErr := readRepositoryFile(repository, dataDirectory+"/"+name)
		if readErr != nil {
			return fmt.Errorf("IPA 生成物 %s を読み込めません: %w", name, readErr)
		}
		if !bytes.Equal(got, want) {
			return fmt.Errorf("IPA 生成物 %s の byte 列が再生成結果と一致しません", name)
		}
	}
	return checkDirectoryEntries(repository, expected.files)
}

func generatorSources(repository string) ([]fileDigest, error) {
	var result []fileDigest
	for _, directory := range []string{"cmd/compact-ipa-generate", "internal/compactipagenerate"} {
		entries, err := os.ReadDir(filepath.Join(repository, directory))
		if err != nil {
			return nil, fmt.Errorf("IPA 生成器を読み込めません: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			path := directory + "/" + entry.Name()
			data, err := readRepositoryFile(repository, path)
			if err != nil {
				return nil, fmt.Errorf("IPA 生成器 %s を読み込めません: %w", path, err)
			}
			result = append(result, digest(path, data))
		}
	}
	slices.SortFunc(result, func(a, b fileDigest) int { return strings.Compare(a.Path, b.Path) })
	return result, nil
}
