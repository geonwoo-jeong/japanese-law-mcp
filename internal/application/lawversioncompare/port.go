package lawversioncompare

import (
	"context"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Port は、一つの provider が実装する law.version.compare@1 の型付き境界である。
type Port interface {
	Compare(context.Context, Request) (model.SourcedResource[model.LawVersionComparison], error)
}
