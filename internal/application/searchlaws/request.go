// Package searchlaws は、公開 search_laws facade の型付きユースケース境界を提供する。
package searchlaws

import (
	"fmt"
	"math"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// RequestValues は、公開 search_laws 入力の境界値を保持する。
type RequestValues struct {
	Query  string
	AsOf   *model.Date
	Limit  *int
	Offset *int
}

// Request は、公開 offset 契約を持つ正規化済み法令検索入力である。
type Request struct {
	common lawsearch.Request
	offset int
}

// NewRequest は、共通法令検索条件を再利用し、公開 facade 固有制約を確認する。
func NewRequest(values RequestValues) (Request, error) {
	common, err := lawsearch.NewRequest(lawsearch.RequestValues{
		Query: values.Query,
		AsOf:  values.AsOf,
		Limit: values.Limit,
	})
	if err != nil {
		return Request{}, err
	}
	offset := 0
	if values.Offset != nil {
		offset = *values.Offset
	}
	request := Request{
		common: common,
		offset: offset,
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func (r Request) Query() string {
	return r.common.Query()
}

func (r Request) AsOf() (model.Date, bool) {
	return r.common.AsOf()
}

func (r Request) Limit() int {
	return r.common.Limit()
}

func (r Request) Offset() int {
	return r.offset
}

// WithQuery は、検索条件を保持して query だけを再検証した新しい値を返す。
func (r Request) WithQuery(query string) (Request, error) {
	if err := r.Validate(); err != nil {
		return Request{}, err
	}
	values := RequestValues{
		Query: query,
	}
	if asOf, exists := r.AsOf(); exists {
		values.AsOf = &asOf
	}
	limit := r.Limit()
	offset := r.Offset()
	values.Limit = &limit
	values.Offset = &offset
	return NewRequest(values)
}

// Validate は、正規表現指定、収録開始日および数値 offset を確認する。
func (r Request) Validate() error {
	if err := r.common.Validate(); err != nil {
		return err
	}
	query := r.common.Query()
	if strings.HasPrefix(query, "/") && strings.HasSuffix(query, "/") {
		return fmt.Errorf("query は e-Gov の正規表現指定と区別できる値でなければなりません")
	}
	if asOf, exists := r.common.AsOf(); exists &&
		asOf.String() < "2017-04-01" {
		return fmt.Errorf("asOf は 2017-04-01 以降でなければなりません")
	}
	if r.offset < 0 || int64(r.offset) > math.MaxInt32 {
		return fmt.Errorf("offset は 0 以上 2147483647 以下でなければなりません")
	}
	if _, exists := r.common.ContinuationToken(); exists {
		return fmt.Errorf("公開 search_laws では continuationToken を使用できません")
	}
	return nil
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Request は JSON から直接復元できません。NewRequest を使用してください",
	)
}
