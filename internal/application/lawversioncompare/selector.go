package lawversioncompare

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/resourceinput"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// SelectorValues は、法令版の選択条件を構築する値を保持する。
type SelectorValues struct {
	RevisionID string
	AsOf       *model.Date
}

// Selector は、revisionId 又は asOf のどちらか一方を不変に保持する。
type Selector struct {
	revisionID string
	asOf       *model.Date
}

// NewSelector は、検証済みの法令版選択条件を返す。
func NewSelector(values SelectorValues) (Selector, error) {
	selector := Selector{
		revisionID: values.RevisionID,
		asOf:       cloneOptionalDate(values.AsOf),
	}
	if err := selector.Validate(); err != nil {
		return Selector{}, err
	}
	return selector, nil
}

func (s Selector) RevisionID() (string, bool) {
	return s.revisionID, s.revisionID != ""
}

func (s Selector) AsOf() (model.Date, bool) {
	if s.asOf == nil {
		return model.Date{}, false
	}
	return *s.asOf, true
}

// Validate は、二つの選択方法の排他性と不透明 ID の境界を検証する。
func (s Selector) Validate() error {
	if s.revisionID == "" && s.asOf == nil {
		return fmt.Errorf("revisionId または asOf のどちらか一方を必須とします")
	}
	if s.revisionID != "" && s.asOf != nil {
		return fmt.Errorf("revisionId と asOf を同時に指定できません")
	}
	if s.revisionID != "" {
		if err := resourceinput.ValidateLawIdentifiers(
			"lawId",
			"revisionId",
			"law",
			s.revisionID,
		); err != nil {
			return err
		}
	}
	if s.asOf != nil {
		if err := s.asOf.Validate(); err != nil {
			return fmt.Errorf("asOf が有効ではありません: %w", err)
		}
	}
	return nil
}

func cloneOptionalDate(value *model.Date) *model.Date {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
