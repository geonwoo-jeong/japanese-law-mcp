package legalquery

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// LegalQueryPagePreviewValues は、公開 page preview の作成値を保持する。
type LegalQueryPagePreviewValues struct {
	ReturnedCount int
	HasMore       *bool
	TotalCount    *int
	TotalRelation model.TotalRelation
}

// LegalQueryPagePreview は、継続位置を含まない最初の page の件数を表す。
type LegalQueryPagePreview struct {
	returnedCount int
	hasMore       *bool
	totalCount    *int
	totalRelation model.TotalRelation
}

// NewLegalQueryPagePreview は、省略可能値を複製して検証済み preview を返す。
func NewLegalQueryPagePreview(
	values LegalQueryPagePreviewValues,
) (LegalQueryPagePreview, error) {
	preview := LegalQueryPagePreview{
		returnedCount: values.ReturnedCount,
		totalRelation: values.TotalRelation,
	}
	if values.HasMore != nil {
		hasMore := *values.HasMore
		preview.hasMore = &hasMore
	}
	if values.TotalCount != nil {
		totalCount := *values.TotalCount
		preview.totalCount = &totalCount
	}
	if err := preview.Validate(); err != nil {
		return LegalQueryPagePreview{}, err
	}
	return preview, nil
}

// ReturnedCount は、この attempt で公開する item 数を返す。
func (p LegalQueryPagePreview) ReturnedCount() int {
	return p.returnedCount
}

// HasMore は、続きの有無と判定できたかを返す。
func (p LegalQueryPagePreview) HasMore() (bool, bool) {
	if p.hasMore == nil {
		return false, false
	}
	return *p.hasMore, true
}

// TotalCount は、情報源が示した件数と有無を返す。
func (p LegalQueryPagePreview) TotalCount() (int, bool) {
	if p.totalCount == nil {
		return 0, false
	}
	return *p.totalCount, true
}

// TotalRelation は、情報源件数の関係と有無を返す。
func (p LegalQueryPagePreview) TotalRelation() (model.TotalRelation, bool) {
	if p.totalRelation == "" {
		return "", false
	}
	return p.totalRelation, true
}

// Validate は、件数、hasMore および totalRelation の従属条件を確認する。
func (p LegalQueryPagePreview) Validate() error {
	if p.returnedCount < 0 ||
		p.returnedCount > MaxItemsPerCollectionStep {
		return fmt.Errorf("returnedCount は 0 以上 20 以下でなければなりません")
	}
	if p.totalCount == nil {
		if p.totalRelation != "" {
			return fmt.Errorf("totalCount がない場合は totalRelation を指定できません")
		}
		return nil
	}
	if *p.totalCount < p.returnedCount {
		return fmt.Errorf("totalCount は returnedCount 以上でなければなりません")
	}
	if p.totalRelation != model.TotalRelationExact &&
		p.totalRelation != model.TotalRelationLowerBound {
		return fmt.Errorf("totalRelation は exact または lower_bound でなければなりません")
	}
	if p.totalRelation == model.TotalRelationExact {
		return p.validateExactTotal()
	}
	if p.returnedCount < *p.totalCount {
		if p.hasMore == nil || !*p.hasMore {
			return fmt.Errorf(
				"totalRelation が lower_bound で totalCount が returnedCount を上回る場合は hasMore=true が必要です",
			)
		}
	}
	return nil
}

func (p LegalQueryPagePreview) validateExactTotal() error {
	if p.hasMore == nil {
		return fmt.Errorf("totalRelation が exact の場合は hasMore が必要です")
	}
	expectedHasMore := p.returnedCount < *p.totalCount
	if *p.hasMore != expectedHasMore {
		return fmt.Errorf("hasMore が exact の totalCount と一致しません")
	}
	return nil
}

// MarshalJSON は、continuation を持たない page preview を表す。
func (p LegalQueryPagePreview) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ReturnedCount int                 `json:"returnedCount"`
		HasMore       *bool               `json:"hasMore,omitempty"`
		TotalCount    *int                `json:"totalCount,omitempty"`
		TotalRelation model.TotalRelation `json:"totalRelation,omitempty"`
	}{
		ReturnedCount: p.returnedCount,
		HasMore:       cloneOptionalBool(p.hasMore),
		TotalCount:    cloneOptionalInt(p.totalCount),
		TotalRelation: p.totalRelation,
	})
}

// UnmarshalJSON は、result assembler を介さない直接復元を拒否する。
func (*LegalQueryPagePreview) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalQueryPagePreview は JSON から直接復元できません。NewLegalQueryPagePreview を使用してください",
	)
}

func cloneOptionalBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneOptionalInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
