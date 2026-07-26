package searchlawcontent

import (
	"context"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Port は、公開 offset 契約で法令本文検索を実行する型付き境界である。
type Port interface {
	Search(context.Context, Request) (model.LawContentSearchResult, error)
}
