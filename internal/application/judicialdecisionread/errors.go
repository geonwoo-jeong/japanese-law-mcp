package judicialdecisionread

import "errors"

// ErrNotFound は、参照が示す公表裁判例の公式詳細ページが存在しないことを表す。
var ErrNotFound = errors.New("指定した裁判例の詳細が見つかりません")
