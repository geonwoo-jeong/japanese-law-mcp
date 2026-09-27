package compactipagenerate

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/ikawaha/kagome-dict/dict"
)

func buildArtifacts(source sourceData) (artifactSet, error) {
	archive, err := zip.NewReader(bytes.NewReader(source.dictionary), int64(len(source.dictionary)))
	if err != nil {
		return artifactSet{}, fmt.Errorf("IPA 辞書 archive を読み込めません: %w", err)
	}
	if err := validateSourceParts(archive); err != nil {
		return artifactSet{}, err
	}
	filtered, retained, table, err := filterDictionary(archive)
	if err != nil {
		return artifactSet{}, err
	}
	packed, classes, err := encodePOS(table)
	if err != nil {
		return artifactSet{}, err
	}
	if err := verifyPreserved(filtered, retained); err != nil {
		return artifactSet{}, err
	}
	files := map[string][]byte{
		"ipa-tokenization.dict": filtered, "ipa-pos.bin": packed,
		"IPA-LICENSE": source.license, "IPA-NOTICE.txt": source.notice,
	}
	value := manifest{
		FormatVersion: 1, Module: source.module,
		Inputs:   []fileDigest{digest("ipa.dict", source.dictionary), digest("LICENSE", source.license), digest("NOTICE.txt", source.notice)},
		Retained: retained, Excluded: []string{dict.ContentDictFileName, dict.POSDictFileName},
		Words: len(table.POSs), Classes: classes, Names: len(table.NameList),
	}
	for _, name := range []string{"ipa-tokenization.dict", "ipa-pos.bin", "IPA-LICENSE", "IPA-NOTICE.txt"} {
		value.Outputs = append(value.Outputs, digest(dataDirectory+"/"+name, files[name]))
	}
	return artifactSet{files: files, manifest: value}, nil
}

func validateSourceParts(archive *zip.Reader) error {
	required := map[string]bool{
		dict.MorphDictFileName: false, dict.ContentMetaFileName: false, dict.IndexDictFileName: false,
		dict.ConnectionDictFileName: false, dict.CharDefDictFileName: false, dict.UnkDictFileName: false,
		dict.DictInfoFileName: false, dict.ContentDictFileName: false, dict.POSDictFileName: false,
	}
	for _, entry := range archive.File {
		seen, exists := required[entry.Name]
		if !exists || seen || !entry.Mode().IsRegular() {
			return fmt.Errorf("IPA 辞書 entry %q が不正または重複しています", entry.Name)
		}
		required[entry.Name] = true
	}
	for name, seen := range required {
		if !seen {
			return fmt.Errorf("IPA 辞書 entry %s がありません", name)
		}
	}
	return nil
}

func filterDictionary(archive *zip.Reader) ([]byte, []fileDigest, dict.POSTable, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	var retained []fileDigest
	var table dict.POSTable
	for _, entry := range archive.File {
		if entry.Name == dict.ContentDictFileName {
			continue
		}
		data, err := readZipEntry(entry)
		if err != nil {
			return nil, nil, table, err
		}
		if entry.Name == dict.POSDictFileName {
			table, err = dict.ReadPOSTable(bytes.NewReader(data))
			if err != nil {
				return nil, nil, table, fmt.Errorf("IPA の品詞表を読み込めません: %w", err)
			}
			continue
		}
		// SOT-ENG-051: 圧縮済み byte をそのまま写し、元の entry 順序を保つ。
		if err := writer.Copy(entry); err != nil {
			return nil, nil, table, fmt.Errorf("IPA 辞書部分を複写できません: %w", err)
		}
		retained = append(retained, digest(entry.Name, data))
	}
	if err := writer.Close(); err != nil {
		return nil, nil, table, fmt.Errorf("IPA 辞書 archive を閉じられません: %w", err)
	}
	return buffer.Bytes(), retained, table, nil
}

func verifyPreserved(data []byte, expected []fileDigest) error {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("生成した IPA 辞書を読み込めません: %w", err)
	}
	if len(archive.File) != len(expected) {
		return fmt.Errorf("生成した IPA 辞書の entry 数が一致しません")
	}
	for index, entry := range archive.File {
		raw, err := readZipEntry(entry)
		if err != nil {
			return err
		}
		if digest(entry.Name, raw) != expected[index] {
			return fmt.Errorf("IPA 辞書 entry %s が元の byte 列と一致しません", entry.Name)
		}
	}
	return nil
}

func digest(path string, data []byte) fileDigest {
	sum := sha256.Sum256(data)
	return fileDigest{Path: path, SHA256: hex.EncodeToString(sum[:]), Bytes: int64(len(data))}
}
