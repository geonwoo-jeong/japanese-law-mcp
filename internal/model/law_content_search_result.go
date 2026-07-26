package model

import (
	"encoding/json"
	"fmt"
	"math"
)

// LawContentSearchResultValues は、LawContentSearchResult の作成に必要な値を保持する。
type LawContentSearchResultValues struct {
	TotalCount int
	Items      []LawContentMatch
	NextOffset *int
}

// LawContentSearchResult は、法令本文検索の一ページを不変に保持する。
type LawContentSearchResult struct {
	totalCount  int
	items       []LawContentMatch
	nextOffset  *int
	initialized bool
}

// NewLawContentSearchResult は、一致箇所と次の取得位置を複製して返す。
func NewLawContentSearchResult(
	values LawContentSearchResultValues,
) (LawContentSearchResult, error) {
	result := LawContentSearchResult{
		totalCount:  values.TotalCount,
		items:       cloneLawContentMatches(values.Items),
		initialized: true,
	}
	if values.NextOffset != nil {
		nextOffset := *values.NextOffset
		result.nextOffset = &nextOffset
	}
	if err := result.Validate(); err != nil {
		return LawContentSearchResult{}, err
	}
	return result, nil
}

// TotalCount は、検索条件に該当する一致箇所の総数を返す。
func (r LawContentSearchResult) TotalCount() int {
	return r.totalCount
}

// Items は、現在のページに含まれる一致箇所の複製を返す。
func (r LawContentSearchResult) Items() []LawContentMatch {
	return cloneLawContentMatches(r.items)
}

// NextOffset は、次の取得位置と有無を返す。
func (r LawContentSearchResult) NextOffset() (int, bool) {
	if r.nextOffset == nil {
		return 0, false
	}
	return *r.nextOffset, true
}

// Validate は、件数、一致箇所および次の取得位置を確認する。
func (r LawContentSearchResult) Validate() error {
	if !r.initialized {
		return fmt.Errorf(
			"LawContentSearchResult は NewLawContentSearchResult で作成しなければなりません",
		)
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
		if *r.nextOffset < 1 || int64(*r.nextOffset) > math.MaxInt32 {
			return fmt.Errorf("nextOffset は 1 以上 2147483647 以下でなければなりません")
		}
		if *r.nextOffset >= r.totalCount {
			return fmt.Errorf("nextOffset は totalCount 未満でなければなりません")
		}
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-008 の項目名で検索結果を表す。
func (r LawContentSearchResult) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		TotalCount int               `json:"totalCount"`
		Items      []LawContentMatch `json:"items"`
		NextOffset *int              `json:"nextOffset,omitempty"`
	}{
		TotalCount: r.totalCount,
		Items:      cloneLawContentMatches(r.items),
		NextOffset: cloneOptionalInteger(r.nextOffset),
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*LawContentSearchResult) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawContentSearchResult は JSON から直接復元できません。NewLawContentSearchResult を使用してください",
	)
}

func cloneLawContentMatches(values []LawContentMatch) []LawContentMatch {
	cloned := make([]LawContentMatch, len(values))
	copy(cloned, values)
	return cloned
}
