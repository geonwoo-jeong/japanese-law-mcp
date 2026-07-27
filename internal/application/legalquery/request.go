// Package legalquery は、日本語の統合法情報照会を計画して実行するアプリケーション境界を提供する。
package legalquery

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	// DefaultLimitPerAttempt は、取得上限を省略した場合の既定値である。
	DefaultLimitPerAttempt = 10
	// MaxLimitPerAttempt は、一つの collection step に要求できる最大件数である。
	MaxLimitPerAttempt = 20
	// MaxQueryBytes は、照会文に許可する UTF-8 byte 数である。
	MaxQueryBytes = 2048
)

// RequestValues は、Request の作成に必要な境界値を保持する。
type RequestValues struct {
	Query           string
	Ref             *model.SourceResourceRef
	LimitPerAttempt *int
}

// Request は、transport に依存せず、構文と共通参照構造を検証した統合照会入力を不変に保持する。
type Request struct {
	query           string
	ref             *model.SourceResourceRef
	limitPerAttempt int
}

// NewRequest は、入力を複製し、外側の Unicode White_Space を除いて検証する。
func NewRequest(values RequestValues) (Request, error) {
	if !utf8.ValidString(values.Query) {
		return Request{}, invalidArgument(
			"query",
			"は有効な UTF-8 でなければなりません",
		)
	}

	limitPerAttempt := DefaultLimitPerAttempt
	if values.LimitPerAttempt != nil {
		limitPerAttempt = *values.LimitPerAttempt
	}
	request := Request{
		query:           strings.TrimFunc(values.Query, unicode.IsSpace),
		limitPerAttempt: limitPerAttempt,
	}
	if values.Ref != nil {
		ref := *values.Ref
		request.ref = &ref
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

// Query は、外側の Unicode White_Space を除いた照会文を返す。
func (r Request) Query() string {
	return r.query
}

// Ref は、入力で受け取った資源参照の複製と有無を返す。
func (r Request) Ref() (model.SourceResourceRef, bool) {
	if r.ref == nil {
		return model.SourceResourceRef{}, false
	}
	return *r.ref, true
}

// LimitPerAttempt は、既定値適用後の一試行当たり希望上限を返す。
func (r Request) LimitPerAttempt() int {
	return r.limitPerAttempt
}

// Validate は、統合照会 request 単体で確認できる入力制約を検証する。
// provider と source の採用状態および read capability との一致は、route 選択後に検証する。
func (r Request) Validate() error {
	switch {
	case !utf8.ValidString(r.query):
		return invalidArgument("query", "は有効な UTF-8 でなければなりません")
	case r.query == "":
		return invalidArgument("query", "は一文字以上でなければなりません")
	case len(r.query) > MaxQueryBytes:
		return invalidArgument("query", "は UTF-8 で 2048 byte 以下でなければなりません")
	case containsASCIIControl(r.query):
		return invalidArgument("query", "に ASCII 制御文字を含めることはできません")
	case r.limitPerAttempt < 1 || r.limitPerAttempt > MaxLimitPerAttempt:
		return invalidArgument("limitPerAttempt", "は 1 以上 20 以下でなければなりません")
	}
	if r.ref != nil {
		if err := r.ref.Validate(); err != nil {
			return invalidArgument("ref", "は有効な SourceResourceRef でなければなりません")
		}
		switch r.ref.Key().ResourceType() {
		case "law", "judicial-decision":
		default:
			return invalidArgument(
				"ref",
				"の resourceType は law または judicial-decision でなければなりません",
			)
		}
	}
	return nil
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Request は JSON から直接復元できません。境界専用の入力型から NewRequest を使用してください",
	)
}

func invalidArgument(field string, reason string) error {
	argumentError, err := NewArgumentError(field, reason)
	if err != nil {
		return fmt.Errorf("統合法情報照会の入力エラーを作成できません: %w", err)
	}
	return argumentError
}

func containsASCIIControl(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] <= 0x1f || value[index] == 0x7f {
			return true
		}
	}
	return false
}
