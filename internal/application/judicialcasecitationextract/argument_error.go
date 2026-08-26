package judicialcasecitationextract

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// ArgumentError は、入力値を露出せず invalid_argument を分類する。
type ArgumentError struct {
	field       string
	reason      string
	initialized bool
}

func newArgumentError(field string, reason string) ArgumentError {
	return ArgumentError{field: field, reason: reason, initialized: true}
}

func (e ArgumentError) Error() string {
	if err := e.Validate(); err != nil {
		return "判例引用抽出の入力値が契約を満たしていません"
	}
	return e.field + " " + e.reason
}

func (e ArgumentError) Code() model.ErrorCode { return model.ErrorCodeInvalidArgument }
func (e ArgumentError) Field() string         { return e.field }
func (e ArgumentError) Reason() string        { return e.reason }

func (e ArgumentError) Validate() error {
	if !e.initialized {
		return fmt.Errorf("ArgumentError は初期化されていません")
	}
	switch e.field {
	case "ref", "decision", "document":
	default:
		return fmt.Errorf("field が定義されていません")
	}
	if !utf8.ValidString(e.reason) || e.reason == "" ||
		strings.TrimSpace(e.reason) != e.reason || len(e.reason) > 256 ||
		strings.IndexFunc(e.reason, unicode.IsControl) >= 0 {
		return fmt.Errorf("reason が有効ではありません")
	}
	return nil
}
