package comparelawversions

import (
	"context"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Port は、公開 compare_law_versions を実行する型付き境界である。
type Port interface {
	Compare(context.Context, Request) (model.LawVersionComparison, error)
}
