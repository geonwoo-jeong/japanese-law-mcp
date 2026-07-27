// Package judicialdecisionread は、公表裁判例の詳細を参照元の provider から取得する型付き境界を提供する。
package judicialdecisionread

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	// CapabilityID は、裁判例詳細取得 capability の識別子である。
	CapabilityID = "judicial-decision.read"
	// MajorVersion は、裁判例詳細取得 capability のメジャーバージョンである。
	MajorVersion = 1
)

// RequestValues は、Request の作成に必要な境界値を保持する。
type RequestValues struct {
	Ref model.SourceResourceRef
}

// Request は、検索結果から受け取った裁判例参照を変更せずに保持する。
type Request struct {
	ref         model.SourceResourceRef
	initialized bool
}

// NewRequest は、裁判例詳細取得に使用できる参照を検証して返す。
func NewRequest(values RequestValues) (Request, error) {
	request := Request{
		ref:         values.Ref,
		initialized: true,
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

// Ref は、検索結果から受け取った資源参照を返す。
func (r Request) Ref() model.SourceResourceRef {
	return r.ref
}

// Validate は、judicial-decision.read@1 の参照制約を確認する。
func (r Request) Validate() error {
	if !r.initialized {
		return fmt.Errorf("Request は NewRequest で作成しなければなりません")
	}
	if err := r.ref.Validate(); err != nil {
		return fmt.Errorf("ref が有効ではありません: %w", err)
	}
	key := r.ref.Key()
	if key.ResourceType() != "judicial-decision" {
		return fmt.Errorf(
			"ref.key.resourceType は judicial-decision でなければなりません",
		)
	}
	if _, exists := key.VersionID(); exists {
		return fmt.Errorf("ref.key.versionId は指定できません")
	}
	return nil
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Request は JSON から直接復元できません。NewRequest を使用してください",
	)
}
