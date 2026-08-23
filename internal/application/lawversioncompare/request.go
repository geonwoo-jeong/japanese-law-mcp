package lawversioncompare

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/resourceinput"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	CapabilityID = "law.version.compare"
	MajorVersion = 1
)

// RequestValues は、共通比較要求を構築する値を保持する。
type RequestValues struct {
	Resource model.SourceResourceRef
	Before   Selector
	After    Selector
}

// Request は、law.version.compare@1 の正規化済み入力を不変に保持する。
type Request struct {
	resource model.SourceResourceRef
	before   Selector
	after    Selector
}

// NewRequest は、検証済みの共通比較要求を返す。
func NewRequest(values RequestValues) (Request, error) {
	request := Request{
		resource: values.Resource,
		before:   values.Before,
		after:    values.After,
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func (r Request) Resource() model.SourceResourceRef { return r.resource }
func (r Request) Before() Selector                  { return r.before }
func (r Request) After() Selector                   { return r.after }

// Validate は、法令参照と前後 selector を外部呼出し前に検証する。
func (r Request) Validate() error {
	if err := resourceinput.ValidateLawRef("resource", r.resource); err != nil {
		return err
	}
	if _, exists := r.resource.Key().VersionID(); exists {
		return fmt.Errorf("resource.key.versionId は指定できません")
	}
	if err := r.before.Validate(); err != nil {
		return fmt.Errorf("before が有効ではありません: %w", err)
	}
	if err := r.after.Validate(); err != nil {
		return fmt.Errorf("after が有効ではありません: %w", err)
	}
	return nil
}
