package listlawupdates

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// ResultValues は、公開更新一覧結果の作成に必要な値を保持する。
type ResultValues struct {
	Date       model.Date
	TotalCount int
	Items      []model.LawUpdate
}

// Result は、対象日、正確な総件数および上限付き更新情報を不変に保持する。
type Result struct {
	date       model.Date
	totalCount int
	items      []model.LawUpdate
}

// NewResult は、入力を複製し、公開結果の従属制約を確認する。
func NewResult(values ResultValues) (Result, error) {
	result := Result{
		date:       values.Date,
		totalCount: values.TotalCount,
		items:      cloneLawUpdates(values.Items),
	}
	if err := result.Validate(); err != nil {
		return Result{}, err
	}
	return result, nil
}

// Date は、更新一覧の対象日を返す。
func (r Result) Date() model.Date {
	return r.date
}

// TotalCount は、公式一覧にある更新情報の正確な総件数を返す。
func (r Result) TotalCount() int {
	return r.totalCount
}

// ReturnedCount は、公開結果へ含めた更新情報の件数を返す。
func (r Result) ReturnedCount() int {
	return len(r.items)
}

// OmittedCount は、公開結果から省略した更新情報の件数を返す。
func (r Result) OmittedCount() int {
	return r.totalCount - len(r.items)
}

// Truncated は、公開結果から一件以上を省略したかを返す。
func (r Result) Truncated() bool {
	return r.OmittedCount() > 0
}

// Items は、法令更新情報の複製を返す。
func (r Result) Items() []model.LawUpdate {
	return cloneLawUpdates(r.items)
}

// Validate は、対象日、件数および各更新情報の対応を確認する。
func (r Result) Validate() error {
	if err := r.date.Validate(); err != nil {
		return fmt.Errorf("date が有効ではありません: %w", err)
	}
	if len(r.items) > MaxLimit {
		return fmt.Errorf("items は %d 件以下でなければなりません", MaxLimit)
	}
	if r.totalCount < 0 || r.totalCount < len(r.items) {
		return fmt.Errorf("totalCount は items の件数以上でなければなりません")
	}
	for index, item := range r.items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("items[%d] が有効ではありません: %w", index, err)
		}
		if item.UpdatedOn() != r.date {
			return fmt.Errorf("items[%d].updatedOn と date が一致しません", index)
		}
	}
	return nil
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*Result) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Result は JSON から直接復元できません。NewResult を使用してください",
	)
}

func cloneLawUpdates(values []model.LawUpdate) []model.LawUpdate {
	cloned := make([]model.LawUpdate, len(values))
	copy(cloned, values)
	return cloned
}
