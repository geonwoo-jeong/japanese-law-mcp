package continuation

import "errors"

var (
	// ErrInvalidToken は、継続トークンを安全に再開できないことを表す。
	ErrInvalidToken = errors.New("継続トークンが有効ではありません")

	errInvalidJSONValue   = errors.New("JSON 値が有効ではありません")
	errInvalidJSONObject  = errors.New("JSON object が有効ではありません")
	errInvalidCredential  = errors.New("credential が有効ではありません")
	errInvalidConfigScope = errors.New("provider configuration scope が有効ではありません")
	errInvalidManager     = errors.New("継続トークン管理器が有効ではありません")
	errInvalidIssueInput  = errors.New("継続トークンの発行条件が有効ではありません")
	errInvalidVerifyInput = errors.New("継続トークンの検証条件が有効ではありません")
	errTokenTooLarge      = errors.New("継続トークンが 4096 byte を超えます")
)
