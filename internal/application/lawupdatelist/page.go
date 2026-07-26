package lawupdatelist

import (
	"encoding/json"
	"fmt"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

// PageValues は、Page の作成に必要な値を保持する。
type PageValues struct {
	Items []model.SourcedResource[model.LawUpdate]
	Page  model.SourcePage
	Date  model.Date
}

// Page は、law.update.list@1 の型付き項目と件数を不変に保持する。
type Page struct {
	items []model.SourcedResource[model.LawUpdate]
	page  model.SourcePage
	date  model.Date
}

// NewPage は、入力を複製し、項目とページの従属制約を確認した Page を返す。
func NewPage(values PageValues) (Page, error) {
	page := Page{
		items: cloneItems(values.Items),
		page:  values.Page,
		date:  values.Date,
	}
	if err := page.Validate(); err != nil {
		return Page{}, err
	}
	return page, nil
}

// Items は、対象日の法令更新項目の複製を返す。
func (p Page) Items() []model.SourcedResource[model.LawUpdate] {
	return cloneItems(p.items)
}

// Page は、正確な総件数を含むページ情報を返す。
func (p Page) Page() model.SourcePage {
	return p.page
}

// Date は、更新一覧の対象日を返す。
func (p Page) Date() model.Date {
	return p.date
}

// Validate は、項目、件数および law.update.list@1 の識別子対応を確認する。
func (p Page) Validate() error {
	if err := p.date.Validate(); err != nil {
		return fmt.Errorf("date が有効ではありません: %w", err)
	}
	if err := p.page.Validate(); err != nil {
		return fmt.Errorf("page が有効ではありません: %w", err)
	}
	if p.page.ReturnedCount() != len(p.items) {
		return fmt.Errorf("page.returnedCount と items の件数が一致しません")
	}
	if _, exists := p.page.NextToken(); exists {
		return fmt.Errorf("law.update.list@1 では page.nextToken を使用できません")
	}
	totalCount, totalExists := p.page.TotalCount()
	totalRelation, relationExists := p.page.TotalRelation()
	if !totalExists || !relationExists || totalRelation != model.TotalRelationExact {
		return fmt.Errorf("page は exact の totalCount を持たなければなりません")
	}
	if totalCount != p.page.ReturnedCount() {
		return fmt.Errorf("page.totalCount と page.returnedCount が一致しません")
	}

	for index, item := range p.items {
		if err := validateItem(item, p.date); err != nil {
			return fmt.Errorf("items[%d] が有効ではありません: %w", index, err)
		}
	}
	return nil
}

// MarshalJSON は、SOT-IF-034 の項目名で更新一覧を表す。
func (p Page) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Items []model.SourcedResource[model.LawUpdate] `json:"items"`
		Page  model.SourcePage                         `json:"page"`
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

func validateItem(
	item model.SourcedResource[model.LawUpdate],
	date model.Date,
) error {
	if err := item.Validate(); err != nil {
		return err
	}

	key := item.Ref().Key()
	data := item.Data()
	switch {
	case key.ResourceType() != "law-update-list":
		return fmt.Errorf("ref.key.resourceType は law-update-list でなければなりません")
	case data.UpdatedOn() != date:
		return fmt.Errorf("data.updatedOn と要求日が一致しません")
	case key.ResourceID() != date.String():
		return fmt.Errorf("ref.key.resourceId と要求日が一致しません")
	case key.SourceID() != data.Source().ID():
		return fmt.Errorf("ref.key.sourceId と data.source.id が一致しません")
	}
	if _, exists := key.VersionID(); exists {
		return fmt.Errorf("ref.key.versionId は指定できません")
	}
	return nil
}

func cloneItems(
	values []model.SourcedResource[model.LawUpdate],
) []model.SourcedResource[model.LawUpdate] {
	cloned := make([]model.SourcedResource[model.LawUpdate], len(values))
	copy(cloned, values)
	return cloned
}
