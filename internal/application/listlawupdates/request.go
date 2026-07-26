// Package listlawupdates は、公開 list_law_updates の型付きユースケース境界を提供する。
package listlawupdates

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// RequestValues は、公開 list_law_updates 入力の境界値を保持する。
type RequestValues struct {
	Date model.Date
}

// Request は、検証済みの対象日を不変に保持する。
type Request struct {
	date model.Date
}

// NewRequest は、公開 list_law_updates の対象日を確認する。
func NewRequest(values RequestValues) (Request, error) {
	request := Request{date: values.Date}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

// Date は、更新一覧の対象日を返す。
func (r Request) Date() model.Date {
	return r.date
}

// Validate は、対象日が実在する暦日であることを確認する。
func (r Request) Validate() error {
	if err := r.date.Validate(); err != nil {
		return fmt.Errorf("date が有効ではありません: %w", err)
	}
	return nil
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Request は JSON から直接復元できません。NewRequest を使用してください",
	)
}
