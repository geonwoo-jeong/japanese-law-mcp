package lawsearch

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/continuation"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const (
	// CapabilityID は、法令検索 capability の識別子である。
	CapabilityID = "law.search"
	// MajorVersion は、法令検索 capability のメジャーバージョンである。
	MajorVersion = 1
	// DefaultLimit は、limit を省略した場合のページ上限である。
	DefaultLimit = 20
	// MaxLimit は、一回の検索で指定できるページ上限である。
	MaxLimit = 100
	// MaxTokenBytes は、継続トークンに許可する UTF-8 byte 数である。
	MaxTokenBytes = 4096

	maxQueryBytes = 512
)

// RequestValues は、Request の作成に必要な境界値を保持する。
type RequestValues struct {
	Query             string
	AsOf              *model.Date
	Limit             *int
	ContinuationToken string
}

// Request は、law.search@1 の正規化済み入力を不変に保持する。
type Request struct {
	query             string
	asOf              *model.Date
	limit             int
	continuationToken string
}

// NewRequest は、入力を複製し、正規化と検証を終えた Request を返す。
func NewRequest(values RequestValues) (Request, error) {
	limit := DefaultLimit
	if values.Limit != nil {
		limit = *values.Limit
	}

	request := Request{
		query:             strings.Trim(values.Query, " "),
		limit:             limit,
		continuationToken: values.ContinuationToken,
	}
	if values.AsOf != nil {
		asOf := *values.AsOf
		request.asOf = &asOf
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

// Query は、正規化済みの検索語を返す。
func (r Request) Query() string {
	return r.query
}

// AsOf は、選択するリビジョンの基準日と有無を返す。
func (r Request) AsOf() (model.Date, bool) {
	if r.asOf == nil {
		return model.Date{}, false
	}
	return *r.asOf, true
}

// Limit は、既定値適用後のページ上限を返す。
func (r Request) Limit() int {
	return r.limit
}

// ContinuationToken は、継続トークンと有無を返す。
func (r Request) ContinuationToken() (string, bool) {
	return r.continuationToken, r.continuationToken != ""
}

// Validate は、law.search@1 の入力制約を確認する。
func (r Request) Validate() error {
	if !utf8.ValidString(r.query) {
		return fmt.Errorf("query は有効な UTF-8 でなければなりません")
	}
	if r.query == "" {
		return fmt.Errorf("query は一文字以上でなければなりません")
	}
	if len(r.query) > maxQueryBytes {
		return fmt.Errorf("query は UTF-8 で 512 byte 以下でなければなりません")
	}
	if containsASCIIControl(r.query) {
		return fmt.Errorf("query に ASCII 制御文字を含めることはできません")
	}
	if r.asOf != nil {
		if err := r.asOf.Validate(); err != nil {
			return fmt.Errorf("asOf が有効ではありません: %w", err)
		}
	}
	if r.limit < 1 || r.limit > MaxLimit {
		return fmt.Errorf("limit は 1 以上 100 以下でなければなりません")
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

	var asOf *string
	if r.asOf != nil {
		value := r.asOf.String()
		asOf = &value
	}
	raw, err := json.Marshal(struct {
		Query string  `json:"query"`
		AsOf  *string `json:"asOf"`
		Limit int     `json:"limit"`
	}{
		Query: r.query,
		AsOf:  asOf,
		Limit: r.limit,
	})
	if err != nil {
		return continuation.JSONObject{}, fmt.Errorf("検索条件を JSON に変換できません: %w", err)
	}
	object, err := continuation.NewJSONObject(raw)
	if err != nil {
		return continuation.JSONObject{}, fmt.Errorf("検索条件を canonical JSON object に変換できません: %w", err)
	}
	return object, nil
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
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
