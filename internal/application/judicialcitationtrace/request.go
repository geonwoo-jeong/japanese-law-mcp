// Package judicialcitationtrace は、一件の公表裁判例から 1-hop の引用 graph を組み立てる。
package judicialcitationtrace

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	// DefaultIncomingLimit は、被引用候補上限の既定値である。
	DefaultIncomingLimit = 5
	// MaximumIncomingLimit は、被引用候補上限の最大値である。
	MaximumIncomingLimit = 10
)

// RequestValues は、引用追跡入力の境界値を保持する。
type RequestValues struct {
	Ref           model.SourceResourceRef
	Direction     model.JudicialCitationRequestedDirection
	IncomingLimit *int
}

// Request は、外部呼出し前に検証した 1-hop 引用追跡入力である。
type Request struct {
	ref           model.SourceResourceRef
	direction     model.JudicialCitationRequestedDirection
	incomingLimit int
	initialized   bool
}

// NewRequest は、省略値を適用して検証済みの Request を返す。
func NewRequest(values RequestValues) (Request, error) {
	direction := values.Direction
	if direction == "" {
		direction = model.JudicialCitationRequestedDirectionBoth
	}
	limit := DefaultIncomingLimit
	if values.IncomingLimit != nil {
		limit = *values.IncomingLimit
	}
	request := Request{
		ref:           values.Ref,
		direction:     direction,
		incomingLimit: limit,
		initialized:   true,
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func (r Request) Ref() model.SourceResourceRef { return r.ref }
func (r Request) Direction() model.JudicialCitationRequestedDirection {
	return r.direction
}
func (r Request) IncomingLimit() int { return r.incomingLimit }

// Validate は、参照、方向および被引用候補上限を確認する。
func (r Request) Validate() error {
	if !r.initialized {
		return newArgumentError("ref", "は NewRequest で検証しなければなりません")
	}
	if err := r.ref.Validate(); err != nil {
		return newArgumentError("ref", "は有効な SourceResourceRef でなければなりません")
	}
	key := r.ref.Key()
	if key.ResourceType() != "judicial-decision" {
		return newArgumentError("ref", "の resourceType は judicial-decision でなければなりません")
	}
	if _, exists := key.VersionID(); exists {
		return newArgumentError("ref", "に versionId は指定できません")
	}
	switch r.direction {
	case model.JudicialCitationRequestedDirectionOutgoing,
		model.JudicialCitationRequestedDirectionIncoming,
		model.JudicialCitationRequestedDirectionBoth:
	default:
		return newArgumentError("direction", "は outgoing、incoming 又は both でなければなりません")
	}
	if r.incomingLimit < 1 || r.incomingLimit > MaximumIncomingLimit {
		return newArgumentError("incomingLimit", "は 1 以上 10 以下でなければなりません")
	}
	return nil
}

func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("Request は JSON から直接復元できません。NewRequest を使用してください")
}

var _ json.Unmarshaler = (*Request)(nil)
