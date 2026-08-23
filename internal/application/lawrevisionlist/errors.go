package lawrevisionlist

import "errors"

// ErrNotFound は、指定した法令が情報源に存在しないことを表す。
var ErrNotFound = errors.New("指定した法令の改正履歴が見つかりません")
