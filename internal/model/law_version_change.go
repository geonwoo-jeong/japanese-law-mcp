package model

import (
	"encoding/json"
	"fmt"
)

// LawVersionChangeKind は、条の変更種別を表す。
type LawVersionChangeKind string

const (
	LawVersionChangeKindAdded    LawVersionChangeKind = "added"
	LawVersionChangeKindRemoved  LawVersionChangeKind = "removed"
	LawVersionChangeKindModified LawVersionChangeKind = "modified"
)

// LawVersionChangeReason は、同じ条を変更と判定した理由を表す。
type LawVersionChangeReason string

const (
	LawVersionChangeReasonLocation  LawVersionChangeReason = "location"
	LawVersionChangeReasonText      LawVersionChangeReason = "text"
	LawVersionChangeReasonStructure LawVersionChangeReason = "structure"
)

// LawVersionChangeValues は、条単位の変更を構築する値を保持する。
type LawVersionChangeValues struct {
	ChangeKind    LawVersionChangeKind
	ChangeReasons []LawVersionChangeReason
	Before        *LawVersionArticle
	After         *LawVersionArticle
}

// LawVersionChange は、追加、削除又は変更された一条を表す。
type LawVersionChange struct {
	changeKind    LawVersionChangeKind
	changeReasons []LawVersionChangeReason
	before        *LawVersionArticle
	after         *LawVersionArticle
}

// NewLawVersionChange は、入力を複製して検証済みの変更を返す。
func NewLawVersionChange(values LawVersionChangeValues) (LawVersionChange, error) {
	change := LawVersionChange{
		changeKind:    values.ChangeKind,
		changeReasons: cloneLawVersionChangeReasons(values.ChangeReasons),
		before:        cloneOptionalLawVersionArticle(values.Before),
		after:         cloneOptionalLawVersionArticle(values.After),
	}
	if err := change.Validate(); err != nil {
		return LawVersionChange{}, err
	}
	return change, nil
}

func (c LawVersionChange) ChangeKind() LawVersionChangeKind { return c.changeKind }
func (c LawVersionChange) ChangeReasons() []LawVersionChangeReason {
	return cloneLawVersionChangeReasons(c.changeReasons)
}
func (c LawVersionChange) Before() (LawVersionArticle, bool) {
	if c.before == nil {
		return LawVersionArticle{}, false
	}
	return *c.before, true
}
func (c LawVersionChange) After() (LawVersionArticle, bool) {
	if c.after == nil {
		return LawVersionArticle{}, false
	}
	return *c.after, true
}

// Validate は、変更種別、条の存在側、同一性及び理由の整合を検証する。
func (c LawVersionChange) Validate() error {
	switch c.changeKind {
	case LawVersionChangeKindAdded:
		if c.before != nil || c.after == nil || c.changeReasons != nil {
			return fmt.Errorf("added は after だけを持ち、changeReasons を持ちません")
		}
		return c.validateArticles()
	case LawVersionChangeKindRemoved:
		if c.before == nil || c.after != nil || c.changeReasons != nil {
			return fmt.Errorf("removed は before だけを持ち、changeReasons を持ちません")
		}
		return c.validateArticles()
	case LawVersionChangeKindModified:
		if c.before == nil || c.after == nil || len(c.changeReasons) == 0 {
			return fmt.Errorf("modified は before、after 及び一件以上の changeReasons を必須とします")
		}
	default:
		return fmt.Errorf("changeKind が定義されていません")
	}
	if err := c.validateArticles(); err != nil {
		return err
	}
	if !c.before.location.HasSameIdentity(c.after.location) {
		return fmt.Errorf("modified の before と after は同じ条同一性でなければなりません")
	}
	if err := validateLawVersionChangeReasons(c.changeReasons); err != nil {
		return err
	}
	hasLocation := containsLawVersionReason(c.changeReasons, LawVersionChangeReasonLocation)
	hasText := containsLawVersionReason(c.changeReasons, LawVersionChangeReasonText)
	hasStructure := containsLawVersionReason(c.changeReasons, LawVersionChangeReasonStructure)
	locationChanged := !c.before.location.HasSameAuxiliaryLocation(c.after.location)
	textChanged := c.before.text != c.after.text
	structureChanged := c.before.structureFingerprint != c.after.structureFingerprint
	if hasLocation != locationChanged {
		return fmt.Errorf("location の変更有無と changeReasons が一致しません")
	}
	if hasText != textChanged {
		return fmt.Errorf("text の変更有無と changeReasons が一致しません")
	}
	if hasStructure != structureChanged {
		return fmt.Errorf("structure の変更有無と changeReasons が一致しません")
	}
	return nil
}

func (c LawVersionChange) validateArticles() error {
	if c.before != nil {
		if err := c.before.Validate(); err != nil {
			return fmt.Errorf("before が有効ではありません: %w", err)
		}
	}
	if c.after != nil {
		if err := c.after.Validate(); err != nil {
			return fmt.Errorf("after が有効ではありません: %w", err)
		}
	}
	return nil
}

func validateLawVersionChangeReasons(values []LawVersionChangeReason) error {
	order := map[LawVersionChangeReason]int{
		LawVersionChangeReasonLocation:  0,
		LawVersionChangeReasonText:      1,
		LawVersionChangeReasonStructure: 2,
	}
	previous := -1
	for index, value := range values {
		position, exists := order[value]
		if !exists {
			return fmt.Errorf("changeReasons[%d] が定義されていません", index)
		}
		if position <= previous {
			return fmt.Errorf("changeReasons は location、text、structure の順に重複なく並べなければなりません")
		}
		previous = position
	}
	return nil
}

func containsLawVersionReason(values []LawVersionChangeReason, want LawVersionChangeReason) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// MarshalJSON は、存在する側だけを含む変更項目を返す。
func (c LawVersionChange) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ChangeKind    LawVersionChangeKind     `json:"changeKind"`
		ChangeReasons []LawVersionChangeReason `json:"changeReasons,omitempty"`
		Before        *LawVersionArticle       `json:"before,omitempty"`
		After         *LawVersionArticle       `json:"after,omitempty"`
	}{
		ChangeKind:    c.changeKind,
		ChangeReasons: cloneLawVersionChangeReasons(c.changeReasons),
		Before:        cloneOptionalLawVersionArticle(c.before),
		After:         cloneOptionalLawVersionArticle(c.after),
	})
}

func (*LawVersionChange) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("LawVersionChange は JSON から直接復元できません。NewLawVersionChange を使用してください")
}

func cloneOptionalLawVersionArticle(value *LawVersionArticle) *LawVersionArticle {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneLawVersionChangeReasons(values []LawVersionChangeReason) []LawVersionChangeReason {
	if values == nil {
		return nil
	}
	cloned := make([]LawVersionChangeReason, len(values))
	copy(cloned, values)
	return cloned
}
