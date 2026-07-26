package lawsearch

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// PageValues は、Page の作成に必要な値を保持する。
type PageValues struct {
	Items []model.SourcedResource[model.LawSummary]
	Page  model.SourcePage
}

// Page は、law.search@1 の型付き項目と継続取得情報を不変に保持する。
type Page struct {
	items []model.SourcedResource[model.LawSummary]
	page  model.SourcePage
}

// NewPage は、入力を複製し、項目とページの従属制約を確認した Page を返す。
func NewPage(values PageValues) (Page, error) {
	page := Page{
		items: cloneItems(values.Items),
		page:  values.Page,
	}
	if err := page.Validate(); err != nil {
		return Page{}, err
	}
	return page, nil
}

// Items は、現在のページに含まれる法令項目の複製を返す。
func (p Page) Items() []model.SourcedResource[model.LawSummary] {
	return cloneItems(p.items)
}

// Page は、件数と継続取得情報を返す。
func (p Page) Page() model.SourcePage {
	return p.page
}

// Validate は、項目、ページおよび law.search@1 の識別子対応を確認する。
func (p Page) Validate() error {
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

// MarshalJSON は、SOT-IF-022 の項目名で検索ページを表す。
func (p Page) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Items []model.SourcedResource[model.LawSummary] `json:"items"`
		Page  model.SourcePage                          `json:"page"`
	}{
		Items: cloneItems(p.items),
		Page:  p.page,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*Page) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Page は JSON から直接復元できません。境界専用の入力型から NewPage を使用してください",
	)
}

func validateItem(item model.SourcedResource[model.LawSummary]) error {
	if err := item.Validate(); err != nil {
		return err
	}

	key := item.Ref().Key()
	data := item.Data()
	switch {
	case key.ResourceType() != "law":
		return fmt.Errorf("ref.key.resourceType は law でなければなりません")
	case key.ResourceID() != data.LawID():
		return fmt.Errorf("ref.key.resourceId と data.lawId が一致しません")
	case key.SourceID() != data.Source().ID():
		return fmt.Errorf("ref.key.sourceId と data.source.id が一致しません")
	}
	versionID, exists := key.VersionID()
	if !exists || versionID != data.RevisionID() {
		return fmt.Errorf("ref.key.versionId と data.revisionId が一致しません")
	}
	return nil
}

func cloneItems(
	values []model.SourcedResource[model.LawSummary],
) []model.SourcedResource[model.LawSummary] {
	cloned := make([]model.SourcedResource[model.LawSummary], len(values))
	copy(cloned, values)
	return cloned
}
