package lawcontentsearch

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/continuation"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	// CapabilityID は、法令本文検索 capability の識別子である。
	CapabilityID = "law.content.search"
	// MajorVersion は、法令本文検索 capability のメジャーバージョンである。
	MajorVersion = 1
	// DefaultLimit は、limit を省略した場合のページ上限である。
	DefaultLimit = 20
	// MaxLimit は、一回の検索で指定できるページ上限である。
	MaxLimit = 100
	// MaxTokenBytes は、継続トークンに許可する UTF-8 byte 数である。
	MaxTokenBytes = 4096

	maxTermsPerGroup  = 8
	maxTermsTotal     = 16
	maxTermBytes      = 128
	maxTermBytesTotal = 2048
)

// RequestValues は、Request の作成に必要な境界値を保持する。
type RequestValues struct {
	AllTerms          []string
	AnyTerms          []string
	ExcludeTerms      []string
	AsOf              *model.Date
	Limit             *int
	ContinuationToken string
}

// Request は、law.content.search@1 の正規化済み入力を不変に保持する。
type Request struct {
	allTerms          []string
	anyTerms          []string
	excludeTerms      []string
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
		allTerms:          normalizeTerms(values.AllTerms),
		anyTerms:          normalizeTerms(values.AnyTerms),
		excludeTerms:      normalizeTerms(values.ExcludeTerms),
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

// AllTerms は、すべてを含む正規化済みの語を複製して返す。
func (r Request) AllTerms() []string {
	return cloneTerms(r.allTerms)
}

// AnyTerms は、いずれかを含む正規化済みの語を複製して返す。
func (r Request) AnyTerms() []string {
	return cloneTerms(r.anyTerms)
}

// ExcludeTerms は、含んではならない正規化済みの語を複製して返す。
func (r Request) ExcludeTerms() []string {
	return cloneTerms(r.excludeTerms)
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

// Validate は、law.content.search@1 の入力制約を確認する。
func (r Request) Validate() error {
	if r.allTerms == nil || r.anyTerms == nil || r.excludeTerms == nil {
		return fmt.Errorf("検索語の配列は正規化済みの空配列または値を持つ配列でなければなりません")
	}
	if len(r.allTerms) == 0 && len(r.anyTerms) == 0 {
		return fmt.Errorf("allTerms または anyTerms に一件以上の検索語が必要です")
	}
	if err := validateTermCounts(r); err != nil {
		return err
	}
	if err := validateTerms(r.allTerms, r.anyTerms, r.excludeTerms); err != nil {
		return err
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
		AllTerms     []string `json:"allTerms"`
		AnyTerms     []string `json:"anyTerms"`
		ExcludeTerms []string `json:"excludeTerms"`
		AsOf         *string  `json:"asOf"`
		Limit        int      `json:"limit"`
	}{
		AllTerms:     cloneTerms(r.allTerms),
		AnyTerms:     cloneTerms(r.anyTerms),
		ExcludeTerms: cloneTerms(r.excludeTerms),
		AsOf:         asOf,
		Limit:        r.limit,
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

func normalizeTerms(values []string) []string {
	normalized := make([]string, len(values))
	for index, value := range values {
		normalized[index] = strings.Trim(value, " ")
	}
	return normalized
}

func cloneTerms(values []string) []string {
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func validateTermCounts(request Request) error {
	if len(request.allTerms) > maxTermsPerGroup ||
		len(request.anyTerms) > maxTermsPerGroup ||
		len(request.excludeTerms) > maxTermsPerGroup {
		return fmt.Errorf("allTerms、anyTerms および excludeTerms はそれぞれ 8 件以下でなければなりません")
	}
	total := len(request.allTerms) + len(request.anyTerms) + len(request.excludeTerms)
	if total > maxTermsTotal {
		return fmt.Errorf("検索語は合計 16 件以下でなければなりません")
	}
	return nil
}

func validateTerms(groups ...[]string) error {
	seen := make(map[string]struct{})
	totalBytes := 0
	for _, group := range groups {
		for _, term := range group {
			if err := validateTerm(term); err != nil {
				return err
			}
			if _, exists := seen[term]; exists {
				return fmt.Errorf("正規化後の同じ検索語を重複して指定できません")
			}
			seen[term] = struct{}{}
			totalBytes += len(term)
		}
	}
	if totalBytes > maxTermBytesTotal {
		return fmt.Errorf("検索語は合計で UTF-8 の 2048 byte 以下でなければなりません")
	}
	return nil
}

func validateTerm(term string) error {
	switch {
	case !utf8.ValidString(term):
		return fmt.Errorf("検索語は有効な UTF-8 でなければなりません")
	case term == "":
		return fmt.Errorf("検索語は一文字以上でなければなりません")
	case term != strings.Trim(term, " "):
		return fmt.Errorf("検索語は先頭と末尾の U+0020 を除いた正規化済みの値でなければなりません")
	case len(term) > maxTermBytes:
		return fmt.Errorf("検索語は UTF-8 で 128 byte 以下でなければなりません")
	}
	for index := 0; index < len(term); index++ {
		value := term[index]
		if value <= 0x20 || value == 0x7f || strings.ContainsRune("|!()*?", rune(value)) {
			return fmt.Errorf("検索語に ASCII の空白、制御文字または検索演算子記号を含めることはできません")
		}
	}
	return nil
}
