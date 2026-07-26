package lawcontentsearch

import (
	"encoding/json"
	"fmt"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

// PageValues は、Page の作成に必要な値を保持する。
type PageValues struct {
	Items []model.SourcedResource[model.LawContentMatch]
	Page  model.SourcePage
}

// Page は、law.content.search@1 の型付き項目と継続取得情報を不変に保持する。
type Page struct {
	items []model.SourcedResource[model.LawContentMatch]
	page  model.SourcePage
}

// NewPage は、入力を複製し、項目とページの従属制約を確認した Page を返す。
func NewPage(values PageValues) (Page, error) {
	page := Page{
		items: cloneContentItems(values.Items),
		page:  values.Page,
	}
	if err := page.Validate(); err != nil {
		return Page{}, err
	}
	return page, nil
}

// Items は、現在のページに含まれる一致箇所の複製を返す。
func (p Page) Items() []model.SourcedResource[model.LawContentMatch] {
	return cloneContentItems(p.items)
}

// Page は、件数と継続取得情報を返す。
func (p Page) Page() model.SourcePage {
	return p.page
}

// Validate は、項目、ページ、資源識別子および一致位置を確認する。
func (p Page) Validate() error {
	if err := p.page.Validate(); err != nil {
		return fmt.Errorf("page が有効ではありません: %w", err)
	}
	if p.page.ReturnedCount() != len(p.items) {
		return fmt.Errorf("page.returnedCount と items の件数が一致しません")
	}

	seen := make(map[contentItemIdentity]struct{}, len(p.items))
	for index, item := range p.items {
		if err := validateContentItem(item); err != nil {
			return fmt.Errorf("items[%d] が有効ではありません: %w", index, err)
		}
		identity := contentItemIdentity{
			ref:      item.Ref(),
			location: item.Data().Location(),
		}
		if _, exists := seen[identity]; exists {
			return fmt.Errorf("items[%d] の ref と location の組が重複しています", index)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

// MarshalJSON は、SOT-IF-023 の項目名で検索ページを表す。
func (p Page) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Items []model.SourcedResource[model.LawContentMatch] `json:"items"`
		Page  model.SourcePage                               `json:"page"`
	}{
		Items: cloneContentItems(p.items),
		Page:  p.page,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*Page) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Page は JSON から直接復元できません。境界専用の入力型から NewPage を使用してください",
	)
}

type contentItemIdentity struct {
	ref      model.SourceResourceRef
	location string
}

func validateContentItem(item model.SourcedResource[model.LawContentMatch]) error {
	if err := item.Validate(); err != nil {
		return err
	}

	key := item.Ref().Key()
	law := item.Data().Law()
	switch {
	case key.ResourceType() != "law":
		return fmt.Errorf("ref.key.resourceType は law でなければなりません")
	case key.ResourceID() != law.LawID():
		return fmt.Errorf("ref.key.resourceId と data.law.lawId が一致しません")
	case key.SourceID() != law.Source().ID():
		return fmt.Errorf("ref.key.sourceId と data.law.source.id が一致しません")
	}
	versionID, exists := key.VersionID()
	if !exists || versionID != law.RevisionID() {
		return fmt.Errorf("ref.key.versionId と data.law.revisionId が一致しません")
	}

	provenance := item.Provenance()
	location, exists := provenance[len(provenance)-1].Location()
	if !exists {
		return fmt.Errorf("最後の provenance.location は必須です")
	}
	if location != item.Data().Location() {
		return fmt.Errorf("最後の provenance.location と data.location が一致しません")
	}
	return nil
}

func cloneContentItems(
	values []model.SourcedResource[model.LawContentMatch],
) []model.SourcedResource[model.LawContentMatch] {
	cloned := make([]model.SourcedResource[model.LawContentMatch], len(values))
	copy(cloned, values)
	return cloned
}
