package searchlaws

import "context"

// QueryResolver は、検証済み原文から一意な正式検索語を解決する。
type QueryResolver interface {
	Resolve(context.Context, string) (string, bool, error)
}
