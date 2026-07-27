package lawdocumentread

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/resourceinput"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	// CapabilityID は、law.document.read capability の識別子である。
	CapabilityID = "law.document.read"
	// MajorVersion は、law.document.read capability のメジャーバージョンである。
	MajorVersion = 1
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
	if err := resourceinput.ValidateLawRef("resource", r.resource); err != nil {
		return err
	}
	key := r.resource.Key()
	if _, exists := key.VersionID(); exists {
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
