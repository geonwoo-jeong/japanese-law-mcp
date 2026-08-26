package judicialcasecitationextract

import "errors"

// ErrNotFound は、詳細 HTML が示した全文 PDF を取得できないことを表す。
var ErrNotFound = errors.New("指定した裁判例の全文 PDF が見つかりません")
