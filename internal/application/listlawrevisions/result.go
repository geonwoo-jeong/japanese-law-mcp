package listlawrevisions

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type ResultValues struct {
	LawID      string
	TotalCount int
	Items      []model.LawRevision
}

// Result は、一つの法令の完全な改正履歴を公開用に保持する。
type Result struct {
	lawID      string
	totalCount int
	items      []model.LawRevision
}

func NewResult(values ResultValues) (Result, error) {
	result := Result{
		lawID: values.LawID, totalCount: values.TotalCount,
		items: cloneLawRevisions(values.Items),
	}
	if err := result.Validate(); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (r Result) LawID() string              { return r.lawID }
func (r Result) TotalCount() int            { return r.totalCount }
func (r Result) Items() []model.LawRevision { return cloneLawRevisions(r.items) }

func (r Result) Validate() error {
	if r.lawID == "" {
		return fmt.Errorf("lawId は必須です")
	}
	if r.totalCount < 0 || r.totalCount != len(r.items) {
		return fmt.Errorf("totalCount と items の件数が一致しません")
	}
	seen := make(map[string]struct{}, len(r.items))
	for index, item := range r.items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("items[%d] が有効ではありません: %w", index, err)
		}
		if item.LawID() != r.lawID {
			return fmt.Errorf("items[%d].lawId と lawId が一致しません", index)
		}
		if _, exists := seen[item.RevisionID()]; exists {
			return fmt.Errorf("items[%d].revisionId が重複しています", index)
		}
		seen[item.RevisionID()] = struct{}{}
	}
	return nil
}

func (*Result) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("Result は JSON から直接復元できません。NewResult を使用してください")
}

func cloneLawRevisions(values []model.LawRevision) []model.LawRevision {
	cloned := make([]model.LawRevision, len(values))
	copy(cloned, values)
	return cloned
}
