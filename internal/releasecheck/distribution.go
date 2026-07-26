package releasecheck

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxManifestBytes = 16 * 1024 * 1024
	checksumPrefix   = "sha256:"
)

type releaseMetadata struct {
	ProjectName string `json:"project_name"`
	Tag         string `json:"tag"`
	Version     string `json:"version"`
	Commit      string `json:"commit"`
}

type goreleaserArtifact struct {
	Name   string                  `json:"name"`
	Path   string                  `json:"path"`
	GoOS   string                  `json:"goos"`
	GoArch string                  `json:"goarch"`
	Type   string                  `json:"type"`
	Extra  goreleaserArtifactExtra `json:"extra"`
}

type goreleaserArtifactExtra struct {
	Binaries []string `json:"Binaries"`
	Checksum string   `json:"Checksum"`
	Format   string   `json:"Format"`
}

func validateDistribution(dist, tag, commit string) error {
	version := strings.TrimPrefix(tag, "v")
	targets := releaseTargets(version)
	checksums, err := validateChecksums(dist, version, targets)
	if err != nil {
		return err
	}
	if err := validateMetadata(dist, tag, version, commit); err != nil {
		return err
	}
	if err := validateArtifacts(dist, targets, checksums); err != nil {
		return err
	}
	if err := rejectUnexpectedArchives(dist, targets, version); err != nil {
		return err
	}
	for _, target := range targets {
		if err := validateArchive(
			target.archivePath(dist),
			target.format,
			target.binaryName,
		); err != nil {
			return fmt.Errorf("%s の検証に失敗しました: %w", target.archiveName, err)
		}
	}
	return nil
}

func checksumFileName(version string) string {
	return projectName + "_" + version + "_checksums.txt"
}

func validateChecksums(
	dist, version string,
	targets []releaseTarget,
) (map[string]string, error) {
	path := filepath.Join(dist, checksumFileName(version))
	content, err := readLimitedRegularFile(path, maxManifestBytes)
	if err != nil {
		return nil, fmt.Errorf("SHA-256 checksum を読めません: %w", err)
	}
	expected := make(map[string]releaseTarget, len(targets))
	for _, target := range targets {
		expected[target.archiveName] = target
	}
	checksums := make(map[string]string, len(targets))
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("SHA-256 checksum の行形式が不正です")
		}
		name := strings.TrimPrefix(fields[1], "*")
		if _, exists := expected[name]; !exists {
			return nil, fmt.Errorf("SHA-256 checksum に予期しない成果物があります: %s", name)
		}
		if _, exists := checksums[name]; exists {
			return nil, fmt.Errorf("SHA-256 checksum が重複しています: %s", name)
		}
		hash := strings.ToLower(fields[0])
		if !validSHA256(hash) {
			return nil, fmt.Errorf("SHA-256 checksum の形式が不正です: %s", name)
		}
		checksums[name] = hash
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("SHA-256 checksum を解析できません: %w", err)
	}
	if len(checksums) != len(targets) {
		return nil, fmt.Errorf(
			"SHA-256 checksum は %d 件必要です: got %d",
			len(targets),
			len(checksums),
		)
	}
	for _, target := range targets {
		actual, hashErr := hashRegularFile(target.archivePath(dist))
		if hashErr != nil {
			return nil, fmt.Errorf("アーカイブ %s を検証できません: %w", target.archiveName, hashErr)
		}
		if actual != checksums[target.archiveName] {
			return nil, fmt.Errorf("SHA-256 checksum が一致しません: %s", target.archiveName)
		}
	}
	return checksums, nil
}

func validateMetadata(dist, tag, version, commit string) error {
	content, err := readLimitedRegularFile(
		filepath.Join(dist, "metadata.json"),
		maxManifestBytes,
	)
	if err != nil {
		return fmt.Errorf("metadata.json を読めません: %w", err)
	}
	var metadata releaseMetadata
	if err := json.Unmarshal(content, &metadata); err != nil {
		return fmt.Errorf("metadata.json を解析できません: %w", err)
	}
	if metadata.ProjectName != projectName {
		return fmt.Errorf("metadata.json の project_name が一致しません")
	}
	if metadata.Tag != tag {
		return fmt.Errorf("metadata.json の tag が一致しません")
	}
	if metadata.Version != version {
		return fmt.Errorf("metadata.json の version が一致しません")
	}
	if metadata.Commit != commit {
		return fmt.Errorf("metadata.json の commit が一致しません")
	}
	return nil
}

func validateArtifacts(
	dist string,
	targets []releaseTarget,
	checksums map[string]string,
) error {
	content, err := readLimitedRegularFile(
		filepath.Join(dist, "artifacts.json"),
		maxManifestBytes,
	)
	if err != nil {
		return fmt.Errorf("artifacts.json を読めません: %w", err)
	}
	var artifacts []goreleaserArtifact
	if err := json.Unmarshal(content, &artifacts); err != nil {
		return fmt.Errorf("artifacts.json を解析できません: %w", err)
	}
	expected := make(map[string]releaseTarget, len(targets))
	for _, target := range targets {
		expected[target.archiveName] = target
	}
	found := make(map[string]struct{}, len(targets))
	for _, artifact := range artifacts {
		if strings.HasPrefix(strings.ToLower(artifact.Type), "source") {
			return fmt.Errorf("artifacts.json に source artifact を含めてはなりません")
		}
		if artifact.Type != "Archive" {
			continue
		}
		target, exists := expected[artifact.Name]
		if !exists {
			return fmt.Errorf("artifacts.json に予期しない Archive があります: %s", artifact.Name)
		}
		if _, duplicate := found[artifact.Name]; duplicate {
			return fmt.Errorf("artifacts.json の Archive が重複しています: %s", artifact.Name)
		}
		if err := validateArchiveArtifact(artifact, target, checksums[artifact.Name]); err != nil {
			return err
		}
		found[artifact.Name] = struct{}{}
	}
	if len(found) != len(targets) {
		return fmt.Errorf("artifacts.json の Archive は %d 件必要です", len(targets))
	}
	return nil
}

func validateArchiveArtifact(
	artifact goreleaserArtifact,
	target releaseTarget,
	checksum string,
) error {
	if artifact.GoOS != target.goos || artifact.GoArch != target.goarch {
		return fmt.Errorf("artifacts.json の対象 OS またはアーキテクチャが一致しません")
	}
	if filepath.Base(filepath.Clean(artifact.Path)) != artifact.Name {
		return fmt.Errorf("artifacts.json の Archive path が name と一致しません")
	}
	if artifact.Extra.Format != target.format {
		return fmt.Errorf("artifacts.json の Archive 形式が一致しません")
	}
	if len(artifact.Extra.Binaries) != 1 ||
		artifact.Extra.Binaries[0] != target.binaryName {
		return fmt.Errorf("artifacts.json の実行ファイル一覧が一致しません")
	}
	if strings.ToLower(artifact.Extra.Checksum) != checksumPrefix+checksum {
		return fmt.Errorf("artifacts.json の Archive checksum が一致しません")
	}
	return nil
}

func rejectUnexpectedArchives(
	dist string,
	targets []releaseTarget,
	version string,
) error {
	entries, err := os.ReadDir(dist)
	if err != nil {
		return fmt.Errorf("dist directory を読めません: %w", err)
	}
	expected := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		expected[target.archiveName] = struct{}{}
	}
	expectedChecksum := checksumFileName(version)
	for _, entry := range entries {
		name := entry.Name()
		if name == "checksums.txt" ||
			(strings.HasSuffix(name, "_checksums.txt") && name != expectedChecksum) {
			return fmt.Errorf("予期しない checksum ファイルがあります: %s", name)
		}
		if entry.IsDir() || !isArchiveName(name) {
			continue
		}
		if _, exists := expected[name]; !exists {
			return fmt.Errorf("予期しないアーカイブがあります: %s", name)
		}
	}
	return nil
}

func isArchiveName(name string) bool {
	return strings.HasSuffix(name, ".tar.gz") ||
		strings.HasSuffix(name, ".tgz") ||
		strings.HasSuffix(name, ".zip")
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func hashRegularFile(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("通常ファイルではありません")
	}
	file, err := os.Open(path) //nolint:gosec // SOT-ENG-019: 固定済み成果物名と明示された dist から構成する。
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
