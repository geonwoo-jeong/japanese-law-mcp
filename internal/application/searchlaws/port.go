package searchlaws

import (
	"context"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

// Port は、公開 offset 契約で法令名検索を実行する型付き境界である。
type Port interface {
	Search(context.Context, Request) (model.LawSearchResult, error)
}
