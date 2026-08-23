package parliamentspeechsearch

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// PageValues は、Page の作成に必要な値を保持する。
type PageValues struct {
	Items []model.SourcedResource[model.ParliamentSpeech]
	Page  model.SourcePage
}

// Page は、国会発言検索の型付き項目とページ情報を不変に保持する。
type Page struct {
	items       []model.SourcedResource[model.ParliamentSpeech]
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

// Items は、順序と重複を保持した発言項目の複製を返す。
func (p Page) Items() []model.SourcedResource[model.ParliamentSpeech] {
	return cloneItems(p.items)
}

// Page は、件数と継続取得情報を返す。
func (p Page) Page() model.SourcePage {
	return p.page
}

// Validate は、項目、ページおよび発言資源参照の対応を確認する。
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
	if len(p.items) > MaxLimit {
		return fmt.Errorf("items は %d 件以下でなければなりません", MaxLimit)
	}
	totalCount, hasTotalCount := p.page.TotalCount()
	if !hasTotalCount {
		return fmt.Errorf("page.totalCount は必須です")
	}
	totalRelation, hasTotalRelation := p.page.TotalRelation()
	if !hasTotalRelation || totalRelation != model.TotalRelationExact {
		return fmt.Errorf("page.totalRelation は exact でなければなりません")
	}
	if totalCount < len(p.items) {
		return fmt.Errorf("page.totalCount は items の件数以上でなければなりません")
	}
	for index, item := range p.items {
		if err := validateItem(item); err != nil {
			return fmt.Errorf("items[%d] が有効ではありません: %w", index, err)
		}
	}
	return nil
}

// MarshalJSON は、SOT-IF-062 の項目名で検索ページを表す。
func (p Page) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Items []model.SourcedResource[model.ParliamentSpeech] `json:"items"`
		Page  model.SourcePage                                `json:"page"`
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

func validateItem(item model.SourcedResource[model.ParliamentSpeech]) error {
	if err := item.Validate(); err != nil {
		return err
	}
	key := item.Ref().Key()
	data := item.Data()
	if key.ResourceType() != "parliament-speech" {
		return fmt.Errorf("ref.key.resourceType は parliament-speech でなければなりません")
	}
	if _, exists := key.VersionID(); exists {
		return fmt.Errorf("ref.key.versionId は指定できません")
	}
	if key.SourceID() != data.Source().ID() {
		return fmt.Errorf("ref.key.sourceId と data.source.id が一致しません")
	}
	if key.ResourceID() != data.SpeechID() {
		return fmt.Errorf("ref.key.resourceId と data.speechId が一致しません")
	}
	return nil
}

func cloneItems(
	values []model.SourcedResource[model.ParliamentSpeech],
) []model.SourcedResource[model.ParliamentSpeech] {
	cloned := make([]model.SourcedResource[model.ParliamentSpeech], len(values))
	copy(cloned, values)
	return cloned
}
