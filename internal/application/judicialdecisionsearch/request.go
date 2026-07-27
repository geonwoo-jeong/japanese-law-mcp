// Package judicialdecisionsearch は、裁判例検索 capability の型付き境界を提供する。
package judicialdecisionsearch

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/continuation"
)

const (
	// CapabilityID は、裁判例検索 capability の識別子である。
	CapabilityID = "judicial-decision.search"
	// MajorVersion は、裁判例検索 capability のメジャーバージョンである。
	MajorVersion = 1
	// DefaultLimit は、limit を省略した場合の返却上限である。
	DefaultLimit = 20
	// MaxLimit は、一回の検索で指定できる返却上限である。
	MaxLimit = 30
	// MaxQueryBytes は、検索語に許可する UTF-8 byte 数である。
	MaxQueryBytes = 512
	// MaxTokenBytes は、継続トークンに許可する UTF-8 byte 数である。
	MaxTokenBytes = 4096
)

// RequestValues は、Request の作成に必要な境界値を保持する。
type RequestValues struct {
	Query             string
	Limit             *int
	ContinuationToken string
}

// Request は、judicial-decision.search@1 の正規化済み入力を不変に保持する。
type Request struct {
	query             string
	limit             int
	continuationToken string
}

// NewRequest は、検索条件を正規化し、検証済みの Request を返す。
func NewRequest(values RequestValues) (Request, error) {
	limit := DefaultLimit
	if values.Limit != nil {
		limit = *values.Limit
	}
	request := Request{
		query:             strings.TrimFunc(values.Query, unicode.IsSpace),
		limit:             limit,
		continuationToken: values.ContinuationToken,
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

// Query は、外側の Unicode whitespace を除いた検索語を返す。
func (r Request) Query() string {
	return r.query
}

// Limit は、既定値適用後の返却上限を返す。
func (r Request) Limit() int {
	return r.limit
}

// ContinuationToken は、継続トークンと有無を返す。
func (r Request) ContinuationToken() (string, bool) {
	return r.continuationToken, r.continuationToken != ""
}

// Validate は、judicial-decision.search@1 の共通入力制約を確認する。
func (r Request) Validate() error {
	if !utf8.ValidString(r.query) {
		return fmt.Errorf("query は有効な UTF-8 でなければなりません")
	}
	if r.query == "" {
		return fmt.Errorf("query は一文字以上でなければなりません")
	}
	if len(r.query) > MaxQueryBytes {
		return fmt.Errorf("query は UTF-8 で 512 byte 以下でなければなりません")
	}
	if containsASCIIControl(r.query) {
		return fmt.Errorf("query に ASCII 制御文字を含めることはできません")
	}
	if r.limit < 1 || r.limit > MaxLimit {
		return fmt.Errorf("limit は 1 以上 30 以下でなければなりません")
	}
	if !utf8.ValidString(r.continuationToken) {
		return fmt.Errorf("continuationToken は有効な UTF-8 でなければなりません")
	}
	if len(r.continuationToken) > MaxTokenBytes {
		return fmt.Errorf("continuationToken は UTF-8 で 4096 byte 以下でなければなりません")
	}
	return nil
}

// ConditionObject は、継続条件 fingerprint 用の正規化済み JSON object を返す。
func (r Request) ConditionObject() (continuation.JSONObject, error) {
	if err := r.Validate(); err != nil {
		return continuation.JSONObject{}, err
	}
	raw, err := json.Marshal(struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}{
		Query: r.query,
		Limit: r.limit,
	})
	if err != nil {
		return continuation.JSONObject{}, fmt.Errorf("検索条件を JSON に変換できません: %w", err)
	}
	object, err := continuation.NewJSONObject(raw)
	if err != nil {
		return continuation.JSONObject{}, fmt.Errorf(
			"検索条件を canonical JSON object に変換できません: %w",
			err,
		)
	}
	return object, nil
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Request は JSON から直接復元できません。境界専用の入力型から NewRequest を使用してください",
	)
}

func containsASCIIControl(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] <= 0x1f || value[index] == 0x7f {
			return true
		}
	}
	return false
}
