package qualitygate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const developmentPrinciplesPath = "docs/development-principles.md"

func verifyDevelopmentPrinciples(repository string) error {
	manifestPath := filepath.Join(
		repository,
		filepath.FromSlash("docs/development-principles.sha256"),
	)
	manifest, err := os.ReadFile(manifestPath) //nolint:gosec // SOT-ENG-020: 明示された検査対象内の固定パスだけを読む。
	if err != nil {
		return fmt.Errorf("開発原則のチェックサムを読めません: %w", err)
	}

	line := strings.TrimSuffix(string(manifest), "\n")
	if strings.ContainsAny(line, "\r\n") {
		return errors.New("開発原則のチェックサムは一行である必要があります")
	}
	fields := strings.Fields(line)
	if len(fields) != 2 || fields[1] != developmentPrinciplesPath {
		return fmt.Errorf(
			"開発原則のチェックサム対象は %s である必要があります",
			developmentPrinciplesPath,
		)
	}
	expected, err := hex.DecodeString(fields[0])
	if err != nil || len(expected) != sha256.Size {
		return errors.New("開発原則の SHA-256 形式が不正です")
	}

	document, err := os.ReadFile( //nolint:gosec // SOT-ENG-020: 固定済みの開発原則だけを読む。
		filepath.Join(repository, filepath.FromSlash(developmentPrinciplesPath)),
	)
	if err != nil {
		return fmt.Errorf("開発原則を読めません: %w", err)
	}
	actual := sha256.Sum256(document)
	if !strings.EqualFold(hex.EncodeToString(actual[:]), fields[0]) {
		return errors.New("開発原則のチェックサムが一致しません")
	}
	return nil
}
