package getlaw

import (
	"context"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Port は、公開 get_law を実行する型付き境界である。
type Port interface {
	Get(context.Context, Request) (model.LawDocument, error)
}
