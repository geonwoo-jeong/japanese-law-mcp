package judicialdecisionsearch

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// PageValues は、Page の作成に必要な値を保持する。
type PageValues struct {
	Items []model.SourcedResource[model.JudicialDecisionSummary]
	Page  model.SourcePage
}

// Page は、裁判例検索の型付き項目と継続取得情報を不変に保持する。
type Page struct {
	items       []model.SourcedResource[model.JudicialDecisionSummary]
	page        model.SourcePage
	initialized bool
}

// NewPage は、入力を複製し、項目とページの従属制約を確認した Page を返す。
func NewPage(values PageValues) (Page, error) {
	page := Page{
		items:       cloneItems(values.Items),
		page:        values.Page,
		initialized: true,
	}
	if err := page.Validate(); err != nil {
		return Page{}, err
	}
	return page, nil
}

// Items は、DOM 順と重複を保持した裁判例項目の複製を返す。
func (p Page) Items() []model.SourcedResource[model.JudicialDecisionSummary] {
	return cloneItems(p.items)
}

// Page は、件数と継続取得情報を返す。
func (p Page) Page() model.SourcePage {
	return p.page
}

// Validate は、項目、ページおよび裁判例資源参照の対応を確認する。
func (p Page) Validate() error {
	if !p.initialized {
		return fmt.Errorf("Page は NewPage で作成しなければなりません")
	}
	if err := p.page.Validate(); err != nil {
		return fmt.Errorf("page が有効ではありません: %w", err)
	}
	if p.page.ReturnedCount() != len(p.items) {
		return fmt.Errorf("page.returnedCount と items の件数が一致しません")
	}
	for index, item := range p.items {
		if err := validateItem(item); err != nil {
			return fmt.Errorf("items[%d] が有効ではありません: %w", index, err)
		}
	}
	return nil
}

// MarshalJSON は、SOT-IF-041 の項目名で検索ページを表す。
func (p Page) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Items []model.SourcedResource[model.JudicialDecisionSummary] `json:"items"`
		Page  model.SourcePage                                       `json:"page"`
	}{
		Items: cloneItems(p.items),
		Page:  p.page,
	})
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*Page) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Page は JSON から直接復元できません。境界専用の入力型から NewPage を使用してください",
	)
}

func validateItem(item model.SourcedResource[model.JudicialDecisionSummary]) error {
	if err := item.Validate(); err != nil {
		return err
	}

	key := item.Ref().Key()
	data := item.Data()
	if key.ResourceType() != "judicial-decision" {
		return fmt.Errorf("ref.key.resourceType は judicial-decision でなければなりません")
	}
	if _, exists := key.VersionID(); exists {
		return fmt.Errorf("ref.key.versionId は指定できません")
	}
	if key.SourceID() != data.Source().ID() {
		return fmt.Errorf("ref.key.sourceId と data.source.id が一致しません")
	}
	return nil
}

func cloneItems(
	values []model.SourcedResource[model.JudicialDecisionSummary],
) []model.SourcedResource[model.JudicialDecisionSummary] {
	cloned := make([]model.SourcedResource[model.JudicialDecisionSummary], len(values))
	copy(cloned, values)
	return cloned
}
