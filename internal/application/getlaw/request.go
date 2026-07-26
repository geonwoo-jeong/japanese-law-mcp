// Package getlaw は、公開 get_law facade の型付きユースケース境界を提供する。
package getlaw

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const maxLawIDBytes = 256

// RequestValues は、公開 get_law 入力の境界値を保持する。
type RequestValues struct {
	LawID string
	AsOf  *model.Date
}

// Request は、正規化済みの公開法令本文取得入力を不変に保持する。
type Request struct {
	lawID string
	asOf  *model.Date
}

// NewRequest は、lawId の端にある U+0020 を除き、公開入力制約を確認する。
func NewRequest(values RequestValues) (Request, error) {
	request := Request{lawID: strings.Trim(values.LawID, " ")}
	if values.AsOf != nil {
		asOf := *values.AsOf
		request.asOf = &asOf
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

// LawID は、正規化済みの公式法令識別子を返す。
func (r Request) LawID() string {
	return r.lawID
}

// AsOf は、検索基準日と有無を返す。
func (r Request) AsOf() (model.Date, bool) {
	if r.asOf == nil {
		return model.Date{}, false
	}
	return *r.asOf, true
}

// Validate は、公開 get_law の入力制約を確認する。
func (r Request) Validate() error {
	switch {
	case r.lawID == "":
		return fmt.Errorf("lawId は一文字以上でなければなりません")
	case !utf8.ValidString(r.lawID):
		return fmt.Errorf("lawId は有効な UTF-8 でなければなりません")
	case len(r.lawID) > maxLawIDBytes:
		return fmt.Errorf("lawId は UTF-8 で %d byte 以下でなければなりません", maxLawIDBytes)
	}
	for index := 0; index < len(r.lawID); index++ {
		if r.lawID[index] <= 0x1f || r.lawID[index] == 0x7f {
			return fmt.Errorf("lawId に ASCII 制御文字を含めることはできません")
		}
	}
	if r.asOf != nil {
		if err := r.asOf.Validate(); err != nil {
			return fmt.Errorf("asOf が有効ではありません: %w", err)
		}
		if r.asOf.String() < "2017-04-01" {
			return fmt.Errorf("asOf は 2017-04-01 以降でなければなりません")
		}
	}
	return nil
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Request は JSON から直接復元できません。NewRequest を使用してください",
	)
}
