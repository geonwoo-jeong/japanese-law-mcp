package lawrevisionlist

import "context"

// Port は、一つの provider が実装する law.revision.list@1 の境界である。
type Port interface {
	List(context.Context, Request) (Page, error)
}
