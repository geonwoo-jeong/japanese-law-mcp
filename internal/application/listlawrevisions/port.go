package listlawrevisions

import "context"

// Port は、公開 list_law_revisions を実行する型付き境界である。
type Port interface {
	List(context.Context, Request) (Result, error)
}
