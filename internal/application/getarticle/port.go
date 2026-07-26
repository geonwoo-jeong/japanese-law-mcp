// Package getarticle は、公開 get_article facade の型付きユースケース境界を提供する。
package getarticle

import (
	"context"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Port は、公開 get_article を実行する型付き境界である。
type Port interface {
	Get(context.Context, Request) (model.LawArticleFragment, error)
}
