package listlawrevisions

import "errors"

// ErrInvalidSourceResponse は、共通改正履歴が公開契約と一致しないことを表す。
var ErrInvalidSourceResponse = errors.New("情報源の法令改正履歴が公開契約と一致しません")
