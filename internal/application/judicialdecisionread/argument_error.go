package judicialdecisionread

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const maximumArgumentReasonBytes = 256

// ArgumentError は、外部呼出し前に拒否した詳細取得入力を安全に分類する。
type ArgumentError struct {
	field       string
	reason      string
	initialized bool
}

// NewArgumentError は、公開時に invalid_argument へ対応できる入力エラーを返す。
func NewArgumentError(field string, reason string) (ArgumentError, error) {
	result := ArgumentError{
		field:       field,
		reason:      reason,
		initialized: true,
	}
	if err := result.Validate(); err != nil {
		return ArgumentError{}, err
	}
	return result, nil
}

// Error は、入力値そのものを含まない安全な説明を返す。
func (e ArgumentError) Error() string {
	if err := e.Validate(); err != nil {
		return "裁判例詳細取得の入力値が契約を満たしていません"
	}
	return e.field + " " + e.reason
}

// Code は、公開エラーへ対応する分類を返す。
func (e ArgumentError) Code() model.ErrorCode {
	return model.ErrorCodeInvalidArgument
}

// Field は、修正が必要な入力項目を返す。
func (e ArgumentError) Field() string {
	return e.field
}

// Reason は、入力値を含まない修正理由を返す。
func (e ArgumentError) Reason() string {
	return e.reason
}

// Validate は、分類と公開可能な説明の不変条件を確認する。
func (e ArgumentError) Validate() error {
	if !e.initialized {
		return fmt.Errorf("ArgumentError は NewArgumentError で作成しなければなりません")
	}
	if e.field != "ref" {
		return fmt.Errorf("ArgumentError の field が定義されていません")
	}
	if !utf8.ValidString(e.reason) ||
		e.reason == "" ||
		strings.TrimSpace(e.reason) != e.reason ||
		len(e.reason) > maximumArgumentReasonBytes ||
		containsASCIIControlInReason(e.reason) {
		return fmt.Errorf("ArgumentError の reason が有効ではありません")
	}
	return nil
}

func containsASCIIControlInReason(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return true
		}
	}
	return false
}
