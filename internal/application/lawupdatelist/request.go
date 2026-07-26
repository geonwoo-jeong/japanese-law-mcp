package lawupdatelist

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	// CapabilityID は、法令更新一覧 capability の識別子である。
	CapabilityID = "law.update.list"
	// MajorVersion は、法令更新一覧 capability のメジャーバージョンである。
	MajorVersion = 1
)

// RequestValues は、Request の作成に必要な境界値を保持する。
type RequestValues struct {
	Date model.Date
}

// Request は、law.update.list@1 の対象日を不変に保持する。
type Request struct {
	date model.Date
}

// NewRequest は、検証済みの対象日を持つ Request を返す。
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

// Validate は、law.update.list@1 の共通入力制約を確認する。
func (r Request) Validate() error {
	if err := r.date.Validate(); err != nil {
		return fmt.Errorf("date が有効ではありません: %w", err)
	}
	return nil
}

// MarshalJSON は、SOT-IF-034 の項目名で対象日を表す。
func (r Request) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Date model.Date `json:"date"`
	}{
		Date: r.date,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Request は JSON から直接復元できません。境界専用の入力型から NewRequest を使用してください",
	)
}
