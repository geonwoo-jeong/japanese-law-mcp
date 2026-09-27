package releasecheck

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	maxBinaryBytes = 512 * 1024 * 1024
	maxNoticeBytes = 64 * 1024
	ipaLicenseName = "IPA-LICENSE"
	ipaNoticeName  = "IPA-NOTICE.txt"
	sdkLicenseName = "MCP-SDK-LICENSE"
	sdkPatchName   = "MCP-SDK-PATCH.json"
)

// SOT-ENG-051/SOT-ENG-052/SOT-DEL-004/SOT-DEL-010: 許諾文、通知原文とパッチ識別情報は checkout の正本から読み、検証中だけ保持する。
type archiveNotices struct {
	license    string
	notice     string
	sdkLicense string
	sdkPatch   string
}

func loadArchiveNotices(repository string) (archiveNotices, error) {
	directory := filepath.Join(repository, "internal", "nlp", "kagome", "data")
	license, err := readArchiveNotice(filepath.Join(directory, ipaLicenseName))
	if err != nil {
		return archiveNotices{}, fmt.Errorf("IPA の許諾文を読めません: %w", err)
	}
	notice, err := readArchiveNotice(filepath.Join(directory, ipaNoticeName))
	if err != nil {
		return archiveNotices{}, fmt.Errorf("IPA の通知原文を読めません: %w", err)
	}
	sdkDirectory := filepath.Join(repository, "third_party", "modelcontextprotocol-go-sdk")
	sdkLicense, err := readArchiveNotice(filepath.Join(sdkDirectory, "LICENSE"))
	if err != nil {
		return archiveNotices{}, fmt.Errorf("SDK の許諾原文を読めません: %w", err)
	}
	patchDirectory := filepath.Join(repository, "third_party", "modelcontextprotocol-go-sdk-patch")
	distributionLicense, err := readArchiveNotice(filepath.Join(patchDirectory, sdkLicenseName))
	if err != nil {
		return archiveNotices{}, fmt.Errorf("SDK の配布許諾文を読めません: %w", err)
	}
	if !bytes.Equal(distributionLicense, sdkLicense) {
		return archiveNotices{}, fmt.Errorf("SDK の配布許諾文が許諾原文と一致しません")
	}
	sdkPatch, err := readArchiveNotice(filepath.Join(patchDirectory, sdkPatchName))
	if err != nil {
		return archiveNotices{}, fmt.Errorf("SDK のパッチ識別情報を読めません: %w", err)
	}
	notices := archiveNotices{
		license: string(license), notice: string(notice),
		sdkLicense: string(sdkLicense), sdkPatch: string(sdkPatch),
	}
	if err := notices.validate(); err != nil {
		return archiveNotices{}, err
	}
	return notices, nil
}

func readArchiveNotice(filename string) ([]byte, error) {
	info, err := os.Lstat(filename)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("配布付属文書の正本は通常ファイルでなければなりません")
	}
	if info.Size() <= 0 || info.Size() > maxNoticeBytes {
		return nil, fmt.Errorf("配布付属文書の正本のサイズが範囲外です")
	}
	file, err := os.Open(filename) //nolint:gosec // SOT-ENG-019: checkout 内の固定した通知名を開き、同一性と読込み上限を確認する。
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("配布付属文書の正本が検証中に置き換えられました")
	}
	content, err := io.ReadAll(io.LimitReader(file, maxNoticeBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) == 0 || len(content) > maxNoticeBytes {
		return nil, fmt.Errorf("配布付属文書の正本のサイズが範囲外です")
	}
	return content, nil
}

func (n archiveNotices) validate() error {
	if len(n.license) == 0 || len(n.license) > maxNoticeBytes ||
		len(n.notice) == 0 || len(n.notice) > maxNoticeBytes ||
		len(n.sdkLicense) == 0 || len(n.sdkLicense) > maxNoticeBytes ||
		len(n.sdkPatch) == 0 || len(n.sdkPatch) > maxNoticeBytes {
		return fmt.Errorf("配布付属文書の正本は空でない上限内の通常ファイルが必要です")
	}
	return nil
}

func validateArchive(archivePath, format, binaryName string, notices archiveNotices) error {
	if err := notices.validate(); err != nil {
		return err
	}
	info, err := os.Lstat(archivePath)
	if err != nil {
		return fmt.Errorf("アーカイブを確認できません: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("アーカイブは通常ファイルでなければなりません")
	}
	switch format {
	case "tar.gz":
		return validateTarGz(archivePath, binaryName, notices)
	case "zip":
		return validateZip(archivePath, binaryName, notices)
	default:
		return fmt.Errorf("対応していないアーカイブ形式です: %s", format)
	}
}

func validateTarGz(archivePath, binaryName string, notices archiveNotices) error {
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
	seen := make(map[string]struct{}, 5)
	for {
		header, nextErr := reader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return fmt.Errorf("tar を解析できません: %w", nextErr)
		}
		if err := validateArchiveEntry(
			header.Name,
			header.Typeflag == tar.TypeReg || header.Typeflag == 0,
			header.Size,
			binaryName,
		); err != nil {
			return err
		}
		if err := recordArchiveEntry(seen, header.Name); err != nil {
			return err
		}
		executable := header.FileInfo().Mode().Perm()&0o111 != 0
		if header.Name == binaryName && !executable {
			return fmt.Errorf("macOS 実行ファイルに実行権限がありません")
		}
		if header.Name != binaryName && executable {
			return fmt.Errorf("配布付属文書に実行権限を付けてはなりません")
		}
		if err := validateArchiveContent(reader, header.Name, binaryName, notices); err != nil {
			return err
		}
	}
	return validateArchiveEntryCount(seen)
}

func validateZip(archivePath, binaryName string, notices archiveNotices) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("zip を解析できません: %w", err)
	}
	defer func() { _ = reader.Close() }()

	seen := make(map[string]struct{}, 5)
	for _, entry := range reader.File {
		if err := recordArchiveEntry(seen, entry.Name); err != nil {
			return err
		}
		if err := validateZipEntry(entry, binaryName, notices); err != nil {
			return err
		}
	}
	return validateArchiveEntryCount(seen)
}

func validateZipEntry(entry *zip.File, binaryName string, notices archiveNotices) error {
	if entry.UncompressedSize64 > maxBinaryBytes {
		return fmt.Errorf("アーカイブ内のファイルが大きすぎます")
	}
	if err := validateArchiveEntry(
		entry.Name,
		entry.Mode().IsRegular(),
		int64(entry.UncompressedSize64),
		binaryName,
	); err != nil {
		return err
	}
	if entry.Name != binaryName && entry.Mode().Perm()&0o111 != 0 {
		return fmt.Errorf("配布付属文書に実行権限を付けてはなりません")
	}
	content, err := entry.Open()
	if err != nil {
		return fmt.Errorf("zip entry を開けません: %w", err)
	}
	contentErr := validateArchiveContent(content, entry.Name, binaryName, notices)
	closeErr := content.Close()
	if contentErr != nil {
		return contentErr
	}
	if closeErr != nil {
		return fmt.Errorf("zip entry を閉じられません: %w", closeErr)
	}
	return nil
}

func recordArchiveEntry(seen map[string]struct{}, name string) error {
	if _, duplicate := seen[name]; duplicate {
		return fmt.Errorf("アーカイブ内のファイルが重複しています: %s", name)
	}
	seen[name] = struct{}{}
	return nil
}

func validateArchiveEntryCount(seen map[string]struct{}) error {
	if len(seen) != 5 {
		return fmt.Errorf("アーカイブには実行ファイル一つ、IPA の通知ファイル二つと SDK の付属文書二つが必要です")
	}
	return nil
}

func validateArchiveContent(reader io.Reader, name, binaryName string, notices archiveNotices) error {
	if name == binaryName {
		written, err := io.Copy(io.Discard, io.LimitReader(reader, maxBinaryBytes+1))
		if err != nil {
			return fmt.Errorf("アーカイブ内の実行ファイルを検証できません: %w", err)
		}
		if written > maxBinaryBytes {
			return fmt.Errorf("アーカイブ内の実行ファイルが大きすぎます")
		}
		return nil
	}
	var expected string
	switch name {
	case ipaLicenseName:
		expected = notices.license
	case ipaNoticeName:
		expected = notices.notice
	case sdkLicenseName:
		expected = notices.sdkLicense
	case sdkPatchName:
		expected = notices.sdkPatch
	default:
		return fmt.Errorf("アーカイブ内に予期しないファイルがあります: %s", name)
	}
	content, err := io.ReadAll(io.LimitReader(reader, maxNoticeBytes+1))
	if err != nil {
		return fmt.Errorf("配布付属文書を検証できません: %w", err)
	}
	if len(content) > maxNoticeBytes {
		return fmt.Errorf("配布付属文書が大きすぎます")
	}
	if string(content) != expected {
		return fmt.Errorf("配布付属文書が原文と一致しません: %s", name)
	}
	return nil
}

func validateArchiveEntry(name string, regular bool, size int64, binaryName string) error {
	if !safeArchivePath(name) {
		return fmt.Errorf("アーカイブ内に不正なパスがあります: %s", name)
	}
	if !regular {
		return fmt.Errorf("アーカイブには通常ファイルだけを含めてください")
	}
	limit := int64(maxNoticeBytes)
	switch name {
	case binaryName:
		limit = maxBinaryBytes
	case ipaLicenseName, ipaNoticeName, sdkLicenseName, sdkPatchName:
	default:
		return fmt.Errorf("アーカイブ内に予期しないファイルがあります: %s", name)
	}
	if size < 0 || size > limit {
		return fmt.Errorf("アーカイブ内のファイルが大きすぎます: %s", name)
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
	notices archiveNotices,
) (string, error) {
	if err := validateArchive(archivePath, target.format, target.binaryName, notices); err != nil {
		return "", err
	}
	destination := filepath.Join(destinationDirectory, target.binaryName)
	switch target.format {
	case "tar.gz":
		return destination, extractTarGzBinary(archivePath, destination, target.binaryName)
	case "zip":
		return destination, extractZipBinary(archivePath, destination, target.binaryName)
	default:
		return "", fmt.Errorf("対応していないアーカイブ形式です: %s", target.format)
	}
}

func extractTarGzBinary(archivePath, destination, binaryName string) error {
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
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return fmt.Errorf("アーカイブ内に実行ファイルがありません: %s", binaryName)
		}
		if err != nil {
			return err
		}
		if header.Name == binaryName {
			return writeExtractedBinary(destination, reader)
		}
	}
}

func extractZipBinary(archivePath, destination, binaryName string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	for _, entry := range reader.File {
		if entry.Name != binaryName {
			continue
		}
		content, err := entry.Open()
		if err != nil {
			return err
		}
		defer func() { _ = content.Close() }()
		return writeExtractedBinary(destination, content)
	}
	return fmt.Errorf("アーカイブ内に実行ファイルがありません: %s", binaryName)
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
