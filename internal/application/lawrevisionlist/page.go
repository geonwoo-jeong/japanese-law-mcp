package lawrevisionlist

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type PageValues struct {
	LawID string
	Items []model.SourcedResource[model.LawRevision]
	Page  model.SourcePage
}

// Page は、一つの法令に属する完全な改正履歴を不変に保持する。
type Page struct {
	lawID string
	items []model.SourcedResource[model.LawRevision]
	page  model.SourcePage
}

func NewPage(values PageValues) (Page, error) {
	page := Page{lawID: values.LawID, items: cloneLawRevisionResources(values.Items), page: values.Page}
	if err := page.Validate(); err != nil {
		return Page{}, err
	}
	return page, nil
}

func (p Page) LawID() string { return p.lawID }
func (p Page) Items() []model.SourcedResource[model.LawRevision] {
	return cloneLawRevisionResources(p.items)
}
func (p Page) Page() model.SourcePage { return p.page }

func (p Page) Validate() error {
	if p.lawID == "" {
		return fmt.Errorf("lawId は必須です")
	}
	if err := p.page.Validate(); err != nil {
		return fmt.Errorf("page が有効ではありません: %w", err)
	}
	if _, exists := p.page.NextToken(); exists {
		return fmt.Errorf("law.revision.list@1 では nextToken を使用できません")
	}
	totalCount, totalExists := p.page.TotalCount()
	totalRelation, relationExists := p.page.TotalRelation()
	if p.page.ReturnedCount() != len(p.items) ||
		!totalExists || !relationExists ||
		totalRelation != model.TotalRelationExact || totalCount != len(p.items) {
		return fmt.Errorf("page の正確な件数と items が一致しません")
	}
	seen := make(map[string]struct{}, len(p.items))
	for index, item := range p.items {
		if err := validateLawRevisionResource(item, p.lawID); err != nil {
			return fmt.Errorf("items[%d] が有効ではありません: %w", index, err)
		}
		revisionID := item.Data().RevisionID()
		if _, exists := seen[revisionID]; exists {
			return fmt.Errorf("items[%d].revisionId が重複しています", index)
		}
		seen[revisionID] = struct{}{}
	}
	return nil
}

func (p Page) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		LawID string                                     `json:"lawId"`
		Items []model.SourcedResource[model.LawRevision] `json:"items"`
		Page  model.SourcePage                           `json:"page"`
	}{LawID: p.lawID, Items: cloneLawRevisionResources(p.items), Page: p.page})
}

func (*Page) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("Page は JSON から直接復元できません。NewPage を使用してください")
}

func validateLawRevisionResource(
	item model.SourcedResource[model.LawRevision],
	lawID string,
) error {
	if err := item.Validate(); err != nil {
		return err
	}
	key := item.Ref().Key()
	data := item.Data()
	versionID, versionExists := key.VersionID()
	switch {
	case data.LawID() != lawID:
		return fmt.Errorf("data.lawId と page.lawId が一致しません")
	case key.ResourceType() != "law":
		return fmt.Errorf("ref.key.resourceType は law でなければなりません")
	case key.ResourceID() != data.LawID():
		return fmt.Errorf("ref.key.resourceId と data.lawId が一致しません")
	case !versionExists || versionID != data.RevisionID():
		return fmt.Errorf("ref.key.versionId と data.revisionId が一致しません")
	case key.SourceID() != data.Source().ID():
		return fmt.Errorf("ref.key.sourceId と data.source.id が一致しません")
	default:
		return nil
	}
}

func cloneLawRevisionResources(
	values []model.SourcedResource[model.LawRevision],
) []model.SourcedResource[model.LawRevision] {
	cloned := make([]model.SourcedResource[model.LawRevision], len(values))
	copy(cloned, values)
	return cloned
}
