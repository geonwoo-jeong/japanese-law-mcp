package lawversioncompare

import "errors"

// ErrNotFound は、指定した法令または版が存在しないことを表す。
var ErrNotFound = errors.New("指定した法令版が見つかりません")
