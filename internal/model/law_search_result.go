package model

import (
	"encoding/json"
	"fmt"
	"math"
)

// LawSearchResultValues は、LawSearchResult の作成に必要な値を保持する。
type LawSearchResultValues struct {
	TotalCount int
	Items      []LawSummary
	NextOffset *int
}

// LawSearchResult は、公開する法令名検索の一ページを不変に保持する。
type LawSearchResult struct {
	totalCount  int
	items       []LawSummary
	nextOffset  *int
	initialized bool
}

// NewLawSearchResult は、法令一覧と次の取得位置を複製して返す。
func NewLawSearchResult(values LawSearchResultValues) (LawSearchResult, error) {
	result := LawSearchResult{
		totalCount:  values.TotalCount,
		items:       cloneLawSummaries(values.Items),
		initialized: true,
	}
	if values.NextOffset != nil {
		nextOffset := *values.NextOffset
		result.nextOffset = &nextOffset
	}
	if err := result.Validate(); err != nil {
		return LawSearchResult{}, err
	}
	return result, nil
}

// TotalCount は、検索条件に該当する法令の総数を返す。
func (r LawSearchResult) TotalCount() int {
	return r.totalCount
}

// Items は、現在のページに含まれる法令の複製を返す。
func (r LawSearchResult) Items() []LawSummary {
	return cloneLawSummaries(r.items)
}

// NextOffset は、次の取得位置と有無を返す。
func (r LawSearchResult) NextOffset() (int, bool) {
	if r.nextOffset == nil {
		return 0, false
	}
	return *r.nextOffset, true
}

// Validate は、件数、法令および次の取得位置を確認する。
func (r LawSearchResult) Validate() error {
	if !r.initialized {
		return fmt.Errorf("LawSearchResult は NewLawSearchResult で作成しなければなりません")
	}
	if r.totalCount < 0 {
		return fmt.Errorf("totalCount は 0 以上でなければなりません")
	}
	if r.items == nil {
		return fmt.Errorf("items は空配列または値を持つ配列でなければなりません")
	}
	if len(r.items) > r.totalCount {
		return fmt.Errorf("items の件数は totalCount 以下でなければなりません")
	}
	for index, item := range r.items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("items[%d] が有効ではありません: %w", index, err)
		}
	}
	if r.nextOffset != nil {
		if *r.nextOffset < 0 || int64(*r.nextOffset) > math.MaxInt32 {
			return fmt.Errorf("nextOffset は 0 以上 2147483647 以下でなければなりません")
		}
		if *r.nextOffset > r.totalCount {
			return fmt.Errorf("nextOffset は totalCount 以下でなければなりません")
		}
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-006 の項目名で検索結果を表す。
func (r LawSearchResult) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		TotalCount int          `json:"totalCount"`
		Items      []LawSummary `json:"items"`
		NextOffset *int         `json:"nextOffset,omitempty"`
	}{
		TotalCount: r.totalCount,
		Items:      cloneLawSummaries(r.items),
		NextOffset: cloneOptionalInteger(r.nextOffset),
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*LawSearchResult) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawSearchResult は JSON から直接復元できません。NewLawSearchResult を使用してください",
	)
}

func cloneLawSummaries(values []LawSummary) []LawSummary {
	cloned := make([]LawSummary, len(values))
	copy(cloned, values)
	return cloned
}

func cloneOptionalInteger(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
