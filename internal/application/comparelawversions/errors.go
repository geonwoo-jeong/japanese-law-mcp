package comparelawversions

import "errors"

// ErrInvalidSourceResponse は、共通 capability の結果が公開契約を満たさないことを表す。
var ErrInvalidSourceResponse = errors.New("法令版間比較の情報源応答が公開契約を満たしません")
