// Package listlawupdates は、公開 list_law_updates の型付きユースケース境界を提供する。
package listlawupdates

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	// DefaultLimit は、limit を省略した場合の公開返却上限である。
	DefaultLimit = 50
	// MaxLimit は、一回の要求で指定できる公開返却上限である。
	MaxLimit = 512
)

// RequestValues は、公開 list_law_updates 入力の境界値を保持する。
type RequestValues struct {
	Date  model.Date
	Limit *int
}

// Request は、検証済みの対象日と実効返却上限を不変に保持する。
type Request struct {
	date  model.Date
	limit int
}

// NewRequest は、公開 list_law_updates の対象日と返却上限を確認する。
func NewRequest(values RequestValues) (Request, error) {
	limit := DefaultLimit
	if values.Limit != nil {
		limit = *values.Limit
	}
	request := Request{date: values.Date, limit: limit}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

// Date は、更新一覧の対象日を返す。
func (r Request) Date() model.Date {
	return r.date
}

// Limit は、既定値適用後の公開返却上限を返す。
func (r Request) Limit() int {
	return r.limit
}

// Validate は、対象日と実効返却上限を確認する。
func (r Request) Validate() error {
	if err := r.date.Validate(); err != nil {
		return fmt.Errorf("date が有効ではありません: %w", err)
	}
	if r.limit < 1 || r.limit > MaxLimit {
		return fmt.Errorf("limit は 1 以上 %d 以下でなければなりません", MaxLimit)
	}
	return nil
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Request は JSON から直接復元できません。NewRequest を使用してください",
	)
}
