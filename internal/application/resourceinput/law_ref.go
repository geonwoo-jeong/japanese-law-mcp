// Package resourceinput は、能力間で共有する provider 非依存の資源入力検証を提供する。
package resourceinput

import (
	"fmt"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	maximumLawResourceIDBytes = 256
	maximumLawVersionIDBytes  = 512
)

// ValidateLawRef は、law.document.read@1 と law.article.read@1 が共有する参照制約を検証する。
// provider と source の採用状態および対応関係は registry で別に検証する。
func ValidateLawRef(field string, ref model.SourceResourceRef) error {
	if err := ref.Validate(); err != nil {
		return fmt.Errorf("%s が有効ではありません: %w", field, err)
	}
	key := ref.Key()
	if key.ResourceType() != "law" {
		return fmt.Errorf("%s.key.resourceType は law でなければなりません", field)
	}
	return ValidateLawIdentifiers(
		field+".key.resourceId",
		field+".key.versionId",
		key.ResourceID(),
		keyVersionID(key),
	)
}

// ValidateLawIdentifiers は、法令 ID と任意のリビジョン ID の共通文字制約を検証する。
func ValidateLawIdentifiers(
	resourceField string,
	versionField string,
	resourceID string,
	versionID string,
) error {
	if err := validateOpaqueIdentifier(
		resourceField,
		resourceID,
		maximumLawResourceIDBytes,
	); err != nil {
		return err
	}
	if versionID == "" {
		return nil
	}
	if err := validateOpaqueIdentifier(
		versionField,
		versionID,
		maximumLawVersionIDBytes,
	); err != nil {
		return err
	}
	return nil
}

func keyVersionID(key model.SourceResourceKey) string {
	versionID, _ := key.VersionID()
	return versionID
}

func validateOpaqueIdentifier(field string, value string, limit int) error {
	switch {
	case value == "":
		return fmt.Errorf("%s は一文字以上でなければなりません", field)
	case !utf8.ValidString(value):
		return fmt.Errorf("%s は有効な UTF-8 でなければなりません", field)
	case len(value) > limit:
		return fmt.Errorf("%s は UTF-8 で %d byte 以下でなければなりません", field, limit)
	case value[0] == ' ' || value[len(value)-1] == ' ':
		return fmt.Errorf("%s の先頭または末尾に U+0020 を含めることはできません", field)
	}
	for index := 0; index < len(value); index++ {
		if value[index] <= 0x1f || value[index] == 0x7f {
			return fmt.Errorf("%s に ASCII 制御文字を含めることはできません", field)
		}
	}
	return nil
}
