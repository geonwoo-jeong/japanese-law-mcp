package releasecheck

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const maxBinaryBytes = 512 * 1024 * 1024

func validateArchive(archivePath, format, binaryName string) error {
	info, err := os.Lstat(archivePath)
	if err != nil {
		return fmt.Errorf("アーカイブを確認できません: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("アーカイブは通常ファイルでなければなりません")
	}
	switch format {
	case "tar.gz":
		return validateTarGz(archivePath, binaryName)
	case "zip":
		return validateZip(archivePath, binaryName)
	default:
		return fmt.Errorf("対応していないアーカイブ形式です: %s", format)
	}
}

func validateTarGz(archivePath, binaryName string) error {
	file, err := os.Open(archivePath) //nolint:gosec // SOT-ENG-019: 明示されたローカル配布物だけを検証用に開く。
	if err != nil {
		return fmt.Errorf("tar.gz を開けません: %w", err)
	}
	defer func() { _ = file.Close() }()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("gzip を解析できません: %w", err)
	}
	defer func() { _ = gzipReader.Close() }()

	reader := tar.NewReader(gzipReader)
	count := 0
	for {
		header, nextErr := reader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return fmt.Errorf("tar を解析できません: %w", nextErr)
		}
		count++
		if count > 1 {
			return fmt.Errorf("アーカイブには通常ファイルを一つだけ含めてください")
		}
		if err := validateArchiveEntry(
			header.Name,
			header.Typeflag == tar.TypeReg || header.Typeflag == 0,
			header.Size,
			binaryName,
		); err != nil {
			return err
		}
		if header.FileInfo().Mode().Perm()&0o111 == 0 {
			return fmt.Errorf("macOS 実行ファイルに実行権限がありません")
		}
		written, err := io.Copy(
			io.Discard,
			io.LimitReader(reader, maxBinaryBytes+1),
		)
		if err != nil {
			return fmt.Errorf("tar entry を検証できません: %w", err)
		}
		if written > maxBinaryBytes {
			return fmt.Errorf("アーカイブ内の実行ファイルが大きすぎます")
		}
	}
	if count != 1 {
		return fmt.Errorf("アーカイブには通常ファイルを一つだけ含めてください")
	}
	return nil
}

func validateZip(archivePath, binaryName string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("zip を解析できません: %w", err)
	}
	defer func() { _ = reader.Close() }()

	if len(reader.File) != 1 {
		return fmt.Errorf("アーカイブには通常ファイルを一つだけ含めてください")
	}
	entry := reader.File[0]
	if entry.UncompressedSize64 > maxBinaryBytes {
		return fmt.Errorf("アーカイブ内の実行ファイルが大きすぎます")
	}
	if err := validateArchiveEntry(
		entry.Name,
		entry.Mode().IsRegular(),
		int64(entry.UncompressedSize64),
		binaryName,
	); err != nil {
		return err
	}
	content, err := entry.Open()
	if err != nil {
		return fmt.Errorf("zip entry を開けません: %w", err)
	}
	written, copyErr := io.Copy(io.Discard, io.LimitReader(content, maxBinaryBytes+1))
	closeErr := content.Close()
	if copyErr != nil {
		return fmt.Errorf("zip entry を検証できません: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("zip entry を閉じられません: %w", closeErr)
	}
	if written > maxBinaryBytes {
		return fmt.Errorf("アーカイブ内の実行ファイルが大きすぎます")
	}
	return nil
}

func validateArchiveEntry(
	name string,
	regular bool,
	size int64,
	binaryName string,
) error {
	if !safeArchivePath(name) {
		return fmt.Errorf("アーカイブ内に不正なパスがあります: %s", name)
	}
	if !regular {
		return fmt.Errorf("アーカイブには通常ファイルだけを含めてください")
	}
	if name != binaryName {
		return fmt.Errorf(
			"アーカイブ内の実行ファイル名 = %q, want %q",
			name,
			binaryName,
		)
	}
	if size < 0 || size > maxBinaryBytes {
		return fmt.Errorf("アーカイブ内の実行ファイルが大きすぎます")
	}
	return nil
}

func safeArchivePath(name string) bool {
	if name == "" || strings.Contains(name, `\`) || path.IsAbs(name) {
		return false
	}
	cleaned := path.Clean(name)
	return cleaned == name && cleaned != "." &&
		!strings.HasPrefix(cleaned, "../")
}

func extractArchiveBinary(
	archivePath string,
	target releaseTarget,
	destinationDirectory string,
) (string, error) {
	if err := validateArchive(archivePath, target.format, target.binaryName); err != nil {
		return "", err
	}
	destination := filepath.Join(destinationDirectory, target.binaryName)
	switch target.format {
	case "tar.gz":
		return destination, extractTarGzBinary(archivePath, destination)
	case "zip":
		return destination, extractZipBinary(archivePath, destination)
	default:
		return "", fmt.Errorf("対応していないアーカイブ形式です: %s", target.format)
	}
}

func extractTarGzBinary(archivePath, destination string) error {
	file, err := os.Open(archivePath) //nolint:gosec // SOT-ENG-019: 事前検証済みのローカル配布物だけを開く。
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer func() { _ = gzipReader.Close() }()
	reader := tar.NewReader(gzipReader)
	if _, err := reader.Next(); err != nil {
		return err
	}
	return writeExtractedBinary(destination, reader)
}

func extractZipBinary(archivePath, destination string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	content, err := reader.File[0].Open()
	if err != nil {
		return err
	}
	defer func() { _ = content.Close() }()
	return writeExtractedBinary(destination, content)
}

func writeExtractedBinary(destination string, source io.Reader) error {
	file, err := os.OpenFile( //nolint:gosec // SOT-ENG-019: 新規 TempDir と固定済み実行ファイル名から構成する。
		destination,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0o700,
	)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(source, maxBinaryBytes+1))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written > maxBinaryBytes {
		return fmt.Errorf("展開した実行ファイルが大きすぎます")
	}
	return nil
}
