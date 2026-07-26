package lawdocumentread

import (
	"context"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Port は、一つの provider が実装する law.document.read@1 の型付き境界である。
type Port interface {
	Read(context.Context, Request) (model.SourcedResource[model.LawDocumentRepresentation], error)
}
