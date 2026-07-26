package lawarticleread

import (
	"context"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

// Port は、一つの provider が実装する law.article.read@1 の型付き境界である。
type Port interface {
	Read(context.Context, Request) (model.SourcedResource[model.LawArticleFragment], error)
}
