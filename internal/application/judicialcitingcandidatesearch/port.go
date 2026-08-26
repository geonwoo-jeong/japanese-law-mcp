package judicialcitingcandidatesearch

import "context"

// Port は、一つの provider が実装する被引用候補検索の型付き境界である。
type Port interface {
	Search(context.Context, Request) (Result, error)
}
