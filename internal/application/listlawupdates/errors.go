package listlawupdates

import "errors"

// ErrInvalidSourceResponse は、共通更新一覧の結果が公開契約と一致しないことを表す。
var ErrInvalidSourceResponse = errors.New("情報源の更新一覧が公開契約と一致しません")
