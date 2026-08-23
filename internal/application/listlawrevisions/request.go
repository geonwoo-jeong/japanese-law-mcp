// Package listlawrevisions は、公開 list_law_revisions の型付きユースケース境界を提供する。
package listlawrevisions

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawrevisionlist"
)

type RequestValues struct {
	LawIDOrNumber string
}

// Request は、共通能力と同じ法令指定を不変に保持する。
type Request struct {
	common lawrevisionlist.Request
}

func NewRequest(values RequestValues) (Request, error) {
	common, err := lawrevisionlist.NewRequest(lawrevisionlist.RequestValues{
		LawIDOrNumber: values.LawIDOrNumber,
	})
	if err != nil {
		return Request{}, err
	}
	return Request{common: common}, nil
}

func (r Request) LawIDOrNumber() string { return r.common.LawIDOrNumber() }

func (r Request) Validate() error { return r.common.Validate() }

func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("Request は JSON から直接復元できません。NewRequest を使用してください")
}
