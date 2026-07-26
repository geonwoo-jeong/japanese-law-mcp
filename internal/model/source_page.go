package model

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

const maximumContinuationTokenBytes = 4096

// TotalRelation は、情報源が示した総件数の関係を表す。
type TotalRelation string

const (
	// TotalRelationExact は、totalCount が正確な総件数であることを表す。
	TotalRelationExact TotalRelation = "exact"
	// TotalRelationLowerBound は、totalCount が総件数の下限であることを表す。
	TotalRelationLowerBound TotalRelation = "lower_bound"
)

// SourcePageValues は、SourcePage の作成に必要な値を保持する。
type SourcePageValues struct {
	ReturnedCount int
	NextToken     string
	TotalCount    *int
	TotalRelation TotalRelation
}

// SourcePage は、今回返した件数と同じ条件の継続取得情報を表す不変なページである。
type SourcePage struct {
	returnedCount int
	nextToken     string
	totalCount    *int
	totalRelation TotalRelation
}

// NewSourcePage は、入力を複製して検証済みの SourcePage を返す。
func NewSourcePage(values SourcePageValues) (SourcePage, error) {
	page := SourcePage{
		returnedCount: values.ReturnedCount,
		nextToken:     values.NextToken,
		totalRelation: values.TotalRelation,
	}
	if values.TotalCount != nil {
		totalCount := *values.TotalCount
		page.totalCount = &totalCount
	}
	if err := page.Validate(); err != nil {
		return SourcePage{}, err
	}
	return page, nil
}

// ReturnedCount は、今回返した項目数を返す。
func (p SourcePage) ReturnedCount() int {
	return p.returnedCount
}

// NextToken は、次の取得位置を表す不透明な継続トークンと有無を返す。
func (p SourcePage) NextToken() (string, bool) {
	return p.nextToken, p.nextToken != ""
}

// TotalCount は、情報源が示した該当件数と有無を返す。
func (p SourcePage) TotalCount() (int, bool) {
	if p.totalCount == nil {
		return 0, false
	}
	return *p.totalCount, true
}

// TotalRelation は、総件数が示す関係と有無を返す。
func (p SourcePage) TotalRelation() (TotalRelation, bool) {
	return p.totalRelation, p.totalRelation != ""
}

// Validate は、SourcePage の件数、継続トークンおよび総件数の制約を確認する。
func (p SourcePage) Validate() error {
	if p.returnedCount < 0 {
		return fmt.Errorf("returnedCount は 0 以上でなければなりません")
	}
	if p.nextToken != "" {
		if !utf8.ValidString(p.nextToken) {
			return fmt.Errorf("nextToken は有効な UTF-8 でなければなりません")
		}
		if len(p.nextToken) > maximumContinuationTokenBytes {
			return fmt.Errorf("nextToken は UTF-8 で 4096 byte 以下でなければなりません")
		}
	}
	if p.totalCount == nil {
		if p.totalRelation != "" {
			return fmt.Errorf("totalCount がない場合は totalRelation を指定できません")
		}
		return nil
	}
	if *p.totalCount < 0 {
		return fmt.Errorf("totalCount は 0 以上でなければなりません")
	}
	if p.totalRelation != TotalRelationExact &&
		p.totalRelation != TotalRelationLowerBound {
		return fmt.Errorf("totalCount がある場合の totalRelation は exact または lower_bound でなければなりません")
	}
	if p.totalRelation == TotalRelationExact && p.returnedCount > *p.totalCount {
		return fmt.Errorf("totalRelation が exact の場合は returnedCount が totalCount 以下でなければなりません")
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-014 の項目名でページを表す。
func (p SourcePage) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	return json.Marshal(struct {
		ReturnedCount int           `json:"returnedCount"`
		NextToken     string        `json:"nextToken,omitempty"`
		TotalCount    *int          `json:"totalCount,omitempty"`
		TotalRelation TotalRelation `json:"totalRelation,omitempty"`
	}{
		ReturnedCount: p.returnedCount,
		NextToken:     p.nextToken,
		TotalCount:    p.totalCount,
		TotalRelation: p.totalRelation,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*SourcePage) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"SourcePage は JSON から直接復元できません。境界専用の入力型から NewSourcePage を使用してください",
	)
}
