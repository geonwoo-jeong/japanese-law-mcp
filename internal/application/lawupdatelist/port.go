package lawupdatelist

import "context"

// Port は、一つの provider が実装する law.update.list@1 の型付き境界である。
type Port interface {
	List(context.Context, Request) (Page, error)
}
