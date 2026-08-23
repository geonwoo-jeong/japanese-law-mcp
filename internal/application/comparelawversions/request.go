// Package comparelawversions は、公開 compare_law_versions の型付きユースケース境界を提供する。
package comparelawversions

import (
	"fmt"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawversioncompare"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/resourceinput"
)

// RequestValues は、公開比較要求を構築する値を保持する。
type RequestValues struct {
	LawID  string
	Before lawversioncompare.Selector
	After  lawversioncompare.Selector
}

// Request は、一つの法令 ID と比較前後の選択条件を不変に保持する。
type Request struct {
	lawID  string
	before lawversioncompare.Selector
	after  lawversioncompare.Selector
}

// NewRequest は、端の U+0020 を除いた lawId と検証済み selector を保持する。
func NewRequest(values RequestValues) (Request, error) {
	request := Request{
		lawID:  strings.Trim(values.LawID, " "),
		before: values.Before,
		after:  values.After,
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func (r Request) LawID() string                      { return r.lawID }
func (r Request) Before() lawversioncompare.Selector { return r.before }
func (r Request) After() lawversioncompare.Selector  { return r.after }

// Validate は、公開入力を provider 呼出し前に検証する。
func (r Request) Validate() error {
	if err := resourceinput.ValidateLawIdentifiers("lawId", "revisionId", r.lawID, ""); err != nil {
		return err
	}
	if err := r.before.Validate(); err != nil {
		return fmt.Errorf("before が有効ではありません: %w", err)
	}
	if err := r.after.Validate(); err != nil {
		return fmt.Errorf("after が有効ではありません: %w", err)
	}
	return nil
}

func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("Request は JSON から直接復元できません。NewRequest を使用してください")
}
