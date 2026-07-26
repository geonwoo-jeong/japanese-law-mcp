package lawsearch

import "context"

// Port は、一つの provider が実装する law.search@1 の型付き境界である。
type Port interface {
	Search(context.Context, Request) (Page, error)
}
