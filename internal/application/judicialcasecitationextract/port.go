package judicialcasecitationextract

import "context"

// Port は、一つの provider が実装する判例引用抽出の型付き境界である。
type Port interface {
	Extract(context.Context, Request) (Result, error)
}
