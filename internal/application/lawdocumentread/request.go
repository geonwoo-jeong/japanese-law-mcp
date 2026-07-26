package lawdocumentread

import (
	"fmt"
	"unicode/utf8"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const (
	// CapabilityID は、law.document.read capability の識別子である。
	CapabilityID = "law.document.read"
	// MajorVersion は、law.document.read capability のメジャーバージョンである。
	MajorVersion = 1

	maxResourceIDBytes = 256
	maxVersionIDBytes  = 512
)

// RequestValues は、Request の作成に必要な境界値を保持する。
type RequestValues struct {
	Resource model.SourceResourceRef
	AsOf     *model.Date
}

// Request は、law.document.read@1 の正規化済み入力を不変に保持する。
type Request struct {
	resource model.SourceResourceRef
	asOf     *model.Date
}

// NewRequest は、入力を複製して検証済みの Request を返す。
func NewRequest(values RequestValues) (Request, error) {
	request := Request{resource: values.Resource}
	if values.AsOf != nil {
		asOf := *values.AsOf
		request.asOf = &asOf
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

// Resource は、取得対象の法令参照を返す。
func (r Request) Resource() model.SourceResourceRef {
	return r.resource
}

// AsOf は、選択基準日と有無を返す。
func (r Request) AsOf() (model.Date, bool) {
	if r.asOf == nil {
		return model.Date{}, false
	}
	return *r.asOf, true
}

// Validate は、law.document.read@1 の入力制約を確認する。
func (r Request) Validate() error {
	if err := r.resource.Validate(); err != nil {
		return fmt.Errorf("resource が有効ではありません: %w", err)
	}
	key := r.resource.Key()
	if key.ResourceType() != "law" {
		return fmt.Errorf("resource.key.resourceType は law でなければなりません")
	}
	if err := validateOpaqueIdentifier("resource.key.resourceId", key.ResourceID(), maxResourceIDBytes); err != nil {
		return err
	}
	if versionID, exists := key.VersionID(); exists {
		if err := validateOpaqueIdentifier("resource.key.versionId", versionID, maxVersionIDBytes); err != nil {
			return err
		}
		if r.asOf != nil {
			return fmt.Errorf("resource.key.versionId と asOf は同時に指定できません")
		}
	}
	if r.asOf != nil {
		if err := r.asOf.Validate(); err != nil {
			return fmt.Errorf("asOf が有効ではありません: %w", err)
		}
	}
	return nil
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
