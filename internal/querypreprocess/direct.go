package querypreprocess

import (
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchquery"
)

// ResolveDirectMatches は、法令名の既存索引を共有して分離済み検索語を照合する。
// SOT-ARCH-030: 前処理や形態素解析を繰り返さず、照会結果も共有しない。
func (p *Preprocessor) ResolveDirectMatches(
	ctx context.Context,
	query string,
) ([]searchquery.Match, error) {
	if p == nil || p.lawResolver == nil {
		return nil, fmt.Errorf("法令名の直接照合器は初期化されていません")
	}
	return p.lawResolver.ResolveDirectMatches(ctx, query)
}
