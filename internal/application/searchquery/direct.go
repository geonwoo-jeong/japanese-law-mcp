package searchquery

import "context"

// ResolveDirectMatches は、同じ不変索引で完全一致・正規化・誤記だけを照合する。
// SOT-ARCH-030: 分離済み検索語では形態素解析を再実行しない。
func (r *Resolver) ResolveDirectMatches(
	ctx context.Context,
	query string,
) ([]Match, error) {
	if r == nil || isNilInterface(r.analyzer) {
		return r.ResolveMatches(ctx, query)
	}
	return (&Resolver{
		analyzer:   directOnlyAnalyzer{},
		exact:      r.exact,
		normalized: r.normalized,
		fuzzy:      r.fuzzy,
	}).ResolveMatches(ctx, query)
}

type directOnlyAnalyzer struct{}

func (directOnlyAnalyzer) RegisteredTerms(
	ctx context.Context,
	_ string,
) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []string{}, nil
}
