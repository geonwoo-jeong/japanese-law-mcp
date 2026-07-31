package legalqueryeval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	candidateBaselineFileMode  = 0o600
	candidateBaselineWriteSize = 64 << 10
)

// WriteCandidateBaseline は、検証済み report byte を予約済みの候補版へ一度だけ保存する。
//
// SOT-ENG-036 と SOT-ENG-038 に従い、既存版と default.json は変更しない。
func WriteCandidateBaseline(
	ctx context.Context,
	repositoryRoot string,
	baselineVersion string,
	canonicalReport []byte,
) error {
	if err := checkCandidateBaselineContext(ctx); err != nil {
		return err
	}
	if !baselineVersionPattern.MatchString(baselineVersion) {
		return fmt.Errorf("候補 baselineVersion が不正です")
	}
	if len(canonicalReport) < 1 || len(canonicalReport) > maximumStandardBaselineBytes {
		return fmt.Errorf("候補 baseline byte が上限外です")
	}
	reportBytes := bytes.Clone(canonicalReport)

	repository, repositoryInfo, repositoryPath, err := openCandidateBaselineRepository(repositoryRoot)
	if err != nil {
		return err
	}
	defer func() { _ = repository.Close() }()
	testdata, testdataInfo, err := openCandidateBaselineDirectory(repository, "testdata")
	if err != nil {
		return fmt.Errorf("候補 testdata directory を開けません: %w", err)
	}
	defer func() { _ = testdata.Close() }()
	legalquery, legalqueryInfo, err := openCandidateBaselineDirectory(testdata, "legalquery")
	if err != nil {
		return fmt.Errorf("候補 legalquery directory を開けません: %w", err)
	}
	defer func() { _ = legalquery.Close() }()

	schema, err := loadCandidateBaselineSchema(legalquery)
	if err != nil {
		return err
	}
	report, err := decodeRepositoryBaseline(reportBytes, schema)
	if err != nil {
		return fmt.Errorf("候補 baseline report が不正です: %w", err)
	}
	if report.BaselineVersion() != baselineVersion {
		return fmt.Errorf("候補 baselineVersion が report と一致しません")
	}
	if err := VerifyStandardAcceptance(report); err != nil {
		return fmt.Errorf("不合格 report は候補 baseline にできません: %w", err)
	}
	if err := checkCandidateBaselineContext(ctx); err != nil {
		return err
	}

	baselines, baselinesInfo, err := openCandidateBaselineDirectory(legalquery, "baselines")
	if err != nil {
		return fmt.Errorf("候補 baseline directory を開けません: %w", err)
	}
	defer func() { _ = baselines.Close() }()
	defaultInfo, err := validateCandidateBaselineRoot(baselines)
	if err != nil {
		return err
	}
	versions, versionsInfo, err := openCandidateBaselineDirectory(baselines, "versions")
	if err != nil {
		return fmt.Errorf("候補 baseline history を開けません: %w", err)
	}
	defer func() { _ = versions.Close() }()

	filename := baselineVersion + ".json"
	history, err := inspectCandidateBaselineHistory(versions, filename)
	if err != nil {
		return err
	}
	if history.target != nil {
		return fmt.Errorf("候補 baselineVersion は既に使用されています")
	}
	if err := validateCandidateBaselineHistoryBudget(
		history.count,
		history.totalBytes,
		int64(len(reportBytes)),
	); err != nil {
		return err
	}
	if err := checkCandidateBaselineContext(ctx); err != nil {
		return err
	}

	verify := func(created fs.FileInfo) error {
		if err := verifyCandidateBaselineRepository(
			repositoryPath,
			repository,
			repositoryInfo,
		); err != nil {
			return err
		}
		checks := []struct {
			parent   *os.Root
			name     string
			expected fs.FileInfo
		}{
			{repository, "testdata", testdataInfo},
			{testdata, "legalquery", legalqueryInfo},
			{legalquery, "baselines", baselinesInfo},
			{baselines, "versions", versionsInfo},
			{baselines, "default.json", defaultInfo},
		}
		for _, check := range checks {
			if err := verifyCandidateBaselineEntry(check.parent, check.name, check.expected); err != nil {
				return err
			}
		}
		currentDefault, rootErr := validateCandidateBaselineRoot(baselines)
		if rootErr != nil {
			return rootErr
		}
		if !os.SameFile(defaultInfo, currentDefault) {
			return fmt.Errorf("default.json が候補作成中に置換されました")
		}
		current, inspectErr := inspectCandidateBaselineHistory(versions, filename)
		if inspectErr != nil {
			return inspectErr
		}
		if current.target == nil || !os.SameFile(created, current.target) ||
			current.target.Size() != int64(len(reportBytes)) {
			return fmt.Errorf("作成した候補 baseline の identity が一致しません")
		}
		return validateBaselineHistoryBudget(current.count, current.totalBytes)
	}
	if err := createCandidateBaselineFile(ctx, versions, filename, reportBytes, verify); err != nil {
		return err
	}
	return nil
}

type candidateBaselineHistory struct {
	count      int
	totalBytes int64
	target     fs.FileInfo
}

func openCandidateBaselineRepository(
	path string,
) (*os.Root, fs.FileInfo, string, error) {
	if path == "" {
		return nil, nil, "", fmt.Errorf("候補 repository root が指定されていません")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, "", fmt.Errorf("候補 repository root を確認できません")
	}
	expected, err := os.Lstat(absolute)
	if err != nil || expected.Mode()&os.ModeSymlink != 0 || !expected.IsDir() {
		return nil, nil, "", fmt.Errorf("候補 repository root が有効ではありません")
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, nil, "", fmt.Errorf("候補 repository root を開けません")
	}
	if err := verifyCandidateBaselineRepository(absolute, root, expected); err != nil {
		_ = root.Close()
		return nil, nil, "", err
	}
	return root, expected, absolute, nil
}

func verifyCandidateBaselineRepository(
	absolute string,
	root *os.Root,
	expected fs.FileInfo,
) error {
	opened, err := root.Stat(".")
	if err != nil || !opened.IsDir() || !os.SameFile(expected, opened) {
		return fmt.Errorf("候補 repository root の descriptor が一致しません")
	}
	current, err := os.Lstat(absolute)
	if err != nil || current.Mode()&os.ModeSymlink != 0 ||
		!current.IsDir() || !os.SameFile(opened, current) {
		return fmt.Errorf("候補 repository root が処理中に変化しました")
	}
	return nil
}

func openCandidateBaselineDirectory(
	parent *os.Root,
	name string,
) (*os.Root, fs.FileInfo, error) {
	if err := validateCandidateBaselineSegment(name); err != nil {
		return nil, nil, err
	}
	expected, err := parent.Lstat(name)
	if err != nil || expected.Mode()&os.ModeSymlink != 0 || !expected.IsDir() {
		return nil, nil, fmt.Errorf("候補 directory %q が有効ではありません", name)
	}
	child, err := parent.OpenRoot(name)
	if err != nil {
		return nil, nil, fmt.Errorf("候補 directory %q を開けません", name)
	}
	if err := verifyCandidateBaselineEntry(parent, name, expected); err != nil {
		_ = child.Close()
		return nil, nil, err
	}
	opened, err := child.Stat(".")
	if err != nil || !opened.IsDir() || !os.SameFile(expected, opened) {
		_ = child.Close()
		return nil, nil, fmt.Errorf("候補 directory %q の descriptor が一致しません", name)
	}
	return child, expected, nil
}

func verifyCandidateBaselineEntry(
	parent *os.Root,
	name string,
	expected fs.FileInfo,
) error {
	current, err := parent.Lstat(name)
	if err != nil || current.Mode()&os.ModeSymlink != 0 ||
		!os.SameFile(expected, current) || current.Mode().Type() != expected.Mode().Type() {
		return fmt.Errorf("候補 path %q が処理中に変化しました", name)
	}
	return nil
}

func loadCandidateBaselineSchema(legalquery *os.Root) (baselineSchemaV1, error) {
	schemas, _, err := openCandidateBaselineDirectory(legalquery, "schemas")
	if err != nil {
		return baselineSchemaV1{}, fmt.Errorf("候補 baseline schema directory を開けません: %w", err)
	}
	defer func() { _ = schemas.Close() }()
	raw, err := readCandidateBaselineRegular(
		schemas,
		baselineSchemaFilename,
		maximumBaselineSchemaBytes,
	)
	if err != nil {
		return baselineSchemaV1{}, fmt.Errorf("候補 baseline schema を読めません: %w", err)
	}
	return newBaselineSchemaV1(raw)
}

func readCandidateBaselineRegular(
	root *os.Root,
	name string,
	maximumBytes int64,
) ([]byte, error) {
	if err := validateCandidateBaselineSegment(name); err != nil {
		return nil, err
	}
	expected, err := root.Lstat(name)
	if err != nil || expected.Mode()&os.ModeSymlink != 0 ||
		!expected.Mode().IsRegular() || expected.Size() < 1 || expected.Size() > maximumBytes {
		return nil, fmt.Errorf("候補 regular file が有効ではありません")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, fmt.Errorf("候補 regular file を開けません")
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || opened.Size() != expected.Size() ||
		!os.SameFile(expected, opened) {
		return nil, fmt.Errorf("候補 regular file の descriptor が一致しません")
	}
	data, err := io.ReadAll(io.LimitReader(file, maximumBytes+1))
	if err != nil || int64(len(data)) != opened.Size() {
		return nil, fmt.Errorf("候補 regular file を完全に読めません")
	}
	after, err := file.Stat()
	if err != nil || after.Size() != opened.Size() || !os.SameFile(opened, after) {
		return nil, fmt.Errorf("候補 regular file が読取り中に変化しました")
	}
	if err := verifyCandidateBaselineEntry(root, name, opened); err != nil {
		return nil, err
	}
	return data, nil
}

func validateCandidateBaselineRoot(root *os.Root) (fs.FileInfo, error) {
	directory, err := root.Open(".")
	if err != nil {
		return nil, fmt.Errorf("候補 baseline directory を列挙できません")
	}
	defer func() { _ = directory.Close() }()
	entries, err := directory.ReadDir(3)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("候補 baseline directory を列挙できません")
	}
	if len(entries) != 2 {
		return nil, fmt.Errorf("候補 baseline directory の entry が不正です")
	}
	slices.SortFunc(entries, func(left, right fs.DirEntry) int {
		return strings.Compare(left.Name(), right.Name())
	})
	if entries[0].Name() != "default.json" || entries[1].Name() != "versions" {
		return nil, fmt.Errorf("候補 baseline directory の entry が不正です")
	}
	defaultInfo, err := root.Lstat("default.json")
	if err != nil || defaultInfo.Mode()&os.ModeSymlink != 0 ||
		!defaultInfo.Mode().IsRegular() || defaultInfo.Size() < 1 ||
		defaultInfo.Size() > maximumStandardBaselineBytes {
		return nil, fmt.Errorf("default.json が有効な通常 file ではありません")
	}
	return defaultInfo, nil
}

func inspectCandidateBaselineHistory(
	root *os.Root,
	targetName string,
) (candidateBaselineHistory, error) {
	directory, err := root.Open(".")
	if err != nil {
		return candidateBaselineHistory{}, fmt.Errorf("候補 baseline history を列挙できません")
	}
	defer func() { _ = directory.Close() }()

	history := candidateBaselineHistory{}
	for {
		entries, readErr := directory.ReadDir(32)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return candidateBaselineHistory{}, fmt.Errorf("候補 baseline history を列挙できません")
		}
		for _, entry := range entries {
			if history.count == maximumBaselineHistoryFiles {
				return candidateBaselineHistory{}, fmt.Errorf("候補 baseline history の件数が上限を超えています")
			}
			name := entry.Name()
			info, infoErr := root.Lstat(name)
			if infoErr != nil || info.Mode()&os.ModeSymlink != 0 ||
				!info.Mode().IsRegular() || info.Size() < 1 ||
				info.Size() > maximumStandardBaselineBytes ||
				!strings.HasSuffix(name, ".json") ||
				!baselineVersionPattern.MatchString(strings.TrimSuffix(name, ".json")) {
				return candidateBaselineHistory{}, fmt.Errorf("候補 baseline history の entry が不正です")
			}
			history.count++
			if info.Size() > maximumBaselineHistoryBytes-history.totalBytes {
				return candidateBaselineHistory{}, fmt.Errorf("候補 baseline history の byte 合計が上限を超えています")
			}
			history.totalBytes += info.Size()
			if name == targetName {
				history.target = info
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
	}
	if history.count < 1 {
		return candidateBaselineHistory{}, fmt.Errorf("候補 baseline history は一件以上必要です")
	}
	return history, nil
}

func validateCandidateBaselineHistoryBudget(
	existingCount int,
	existingBytes int64,
	incomingBytes int64,
) error {
	if existingCount < 1 || existingCount >= maximumBaselineHistoryFiles {
		return fmt.Errorf("候補追加後の baseline history 件数が上限外です")
	}
	if existingBytes < 1 || incomingBytes < 1 ||
		existingBytes > maximumBaselineHistoryBytes-incomingBytes {
		return fmt.Errorf("候補追加後の baseline history byte 合計が上限外です")
	}
	return nil
}

func createCandidateBaselineFile(
	ctx context.Context,
	root *os.Root,
	name string,
	data []byte,
	verify func(fs.FileInfo) error,
) (resultErr error) {
	file, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, candidateBaselineFileMode)
	if err != nil {
		return fmt.Errorf("候補 baseline を exclusive create できません: %w", err)
	}
	created, err := file.Stat()
	if err != nil || !created.Mode().IsRegular() || created.Mode().Perm() != candidateBaselineFileMode {
		_ = file.Close()
		_ = removeCreatedCandidateBaseline(root, name, created)
		return fmt.Errorf("作成した候補 baseline の mode が不正です")
	}
	defer func() {
		closeErr := file.Close()
		if resultErr == nil && closeErr != nil {
			resultErr = fmt.Errorf("候補 baseline を閉じられません: %w", closeErr)
		}
		if resultErr != nil {
			if cleanupErr := removeCreatedCandidateBaseline(root, name, created); cleanupErr != nil {
				resultErr = fmt.Errorf("%w; 不完全な候補を除去できません: %w", resultErr, cleanupErr)
			}
		}
	}()

	for offset := 0; offset < len(data); {
		if err := checkCandidateBaselineContext(ctx); err != nil {
			return err
		}
		end := min(offset+candidateBaselineWriteSize, len(data))
		written, writeErr := file.Write(data[offset:end])
		if writeErr != nil {
			return fmt.Errorf("候補 baseline を書けません: %w", writeErr)
		}
		if written < 1 {
			return fmt.Errorf("候補 baseline の書込みが進みません")
		}
		offset += written
	}
	if err := checkCandidateBaselineContext(ctx); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("候補 baseline を永続化できません: %w", err)
	}
	if err := checkCandidateBaselineContext(ctx); err != nil {
		return err
	}
	if err := verifyCandidateBaselineFileBytes(ctx, file, data); err != nil {
		return err
	}
	final, err := file.Stat()
	if err != nil || !final.Mode().IsRegular() || !os.SameFile(created, final) ||
		final.Size() != int64(len(data)) || final.Mode().Perm() != candidateBaselineFileMode {
		return fmt.Errorf("候補 baseline が書込み中に変化しました")
	}
	if err := verify(final); err != nil {
		return err
	}
	return nil
}

func verifyCandidateBaselineFileBytes(
	ctx context.Context,
	file *os.File,
	expected []byte,
) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("候補 baseline の検証位置へ戻れません: %w", err)
	}
	buffer := make([]byte, candidateBaselineWriteSize)
	for offset := 0; offset < len(expected); {
		if err := checkCandidateBaselineContext(ctx); err != nil {
			return err
		}
		length := min(len(buffer), len(expected)-offset)
		if _, err := io.ReadFull(file, buffer[:length]); err != nil {
			return fmt.Errorf("候補 baseline の検証 byte を読めません: %w", err)
		}
		if !bytes.Equal(buffer[:length], expected[offset:offset+length]) {
			return fmt.Errorf("候補 baseline の保存 byte が caller の report と一致しません")
		}
		offset += length
	}
	var trailing [1]byte
	if _, err := file.Read(trailing[:]); !errors.Is(err, io.EOF) {
		if err != nil {
			return fmt.Errorf("候補 baseline の末尾を検証できません: %w", err)
		}
		return fmt.Errorf("候補 baseline の後に未知 byte があります")
	}
	return nil
}

func removeCreatedCandidateBaseline(
	root *os.Root,
	name string,
	created fs.FileInfo,
) error {
	if created == nil {
		return nil
	}
	current, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !os.SameFile(created, current) {
		return fmt.Errorf("作成した候補 baseline の identity が変化しました")
	}
	if err := root.Remove(name); err != nil {
		return fmt.Errorf("作成した候補 baseline を除去できません")
	}
	return nil
}

func validateCandidateBaselineSegment(name string) error {
	if !fs.ValidPath(name) || name == "." || strings.Contains(name, "/") {
		return fmt.Errorf("候補 path は一 segment に限ります")
	}
	return nil
}

func checkCandidateBaselineContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("候補 baseline context は nil にできません")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("候補 baseline 作成が取り消されました: %w", err)
	}
	return nil
}
