package lawdocumentread

import "errors"

// ErrNotFound は、指定した法令、リビジョンまたは基準日以前の本文が存在しないことを表す。
var ErrNotFound = errors.New("指定した条件に該当する法令本文が見つかりません")
