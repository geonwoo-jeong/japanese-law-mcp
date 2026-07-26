package listlawupdates

import "context"

// Port は、公開 list_law_updates を実行する型付き境界である。
type Port interface {
	List(context.Context, Request) (Result, error)
}
