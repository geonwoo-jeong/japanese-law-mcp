// Package searchlawcontent は、公開 search_law_content facade の型付き境界を提供する。
package searchlawcontent

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	// DefaultLimit は、limit を省略した場合の一致箇所上限である。
	DefaultLimit = 20
	// MaxLimit は、一回の検索で指定できる一致箇所上限である。
	MaxLimit = 100
	// MaxQueryBytes は、検索式に許可する UTF-8 byte 数である。
	MaxQueryBytes = 2048
)

// RequestValues は、公開 search_law_content 入力の境界値を保持する。
type RequestValues struct {
	Query  string
	AsOf   *model.Date
	Limit  *int
	Offset *int
}

// Request は、検証済みの e-Gov 本文検索式と公開 offset を不変に保持する。
type Request struct {
	query       string
	asOf        *model.Date
	limit       int
	offset      int
	initialized bool
}

// NewRequest は、入力を複製し、正規化と検索式検証を終えた Request を返す。
func NewRequest(values RequestValues) (Request, error) {
	limit := DefaultLimit
	if values.Limit != nil {
		limit = *values.Limit
	}
	offset := 0
	if values.Offset != nil {
		offset = *values.Offset
	}
	request := Request{
		query:       strings.Trim(values.Query, " "),
		limit:       limit,
		offset:      offset,
		initialized: true,
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

// Query は、正規化と検証を終えた e-Gov 検索式を返す。
func (r Request) Query() string {
	return r.query
}

// AsOf は、検索基準日と有無を返す。
func (r Request) AsOf() (model.Date, bool) {
	if r.asOf == nil {
		return model.Date{}, false
	}
	return *r.asOf, true
}

// Limit は、既定値適用後の一致箇所上限を返す。
func (r Request) Limit() int {
	return r.limit
}

// Offset は、既定値適用後の取得開始位置を返す。
func (r Request) Offset() int {
	return r.offset
}

// Validate は、公開 search_law_content の入力制約を確認する。
func (r Request) Validate() error {
	if !r.initialized {
		return fmt.Errorf("Request は NewRequest で作成しなければなりません")
	}
	switch {
	case !utf8.ValidString(r.query):
		return fmt.Errorf("query は有効な UTF-8 でなければなりません")
	case r.query == "":
		return fmt.Errorf("query は一文字以上でなければなりません")
	case len(r.query) > MaxQueryBytes:
		return fmt.Errorf("query は UTF-8 で 2048 byte 以下でなければなりません")
	case containsASCIIControl(r.query):
		return fmt.Errorf("query に ASCII 制御文字を含めることはできません")
	}
	if err := validateSearchExpression(r.query); err != nil {
		return err
	}
	if r.asOf != nil {
		if err := r.asOf.Validate(); err != nil {
			return fmt.Errorf("asOf が有効ではありません: %w", err)
		}
		if r.asOf.String() < "2017-04-01" {
			return fmt.Errorf("asOf は 2017-04-01 以降でなければなりません")
		}
	}
	if r.limit < 1 || r.limit > MaxLimit {
		return fmt.Errorf("limit は 1 以上 100 以下でなければなりません")
	}
	if r.offset < 0 || int64(r.offset) > math.MaxInt32 {
		return fmt.Errorf("offset は 0 以上 2147483647 以下でなければなりません")
	}
	return nil
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Request は JSON から直接復元できません。NewRequest を使用してください",
	)
}

func validateSearchExpression(query string) error {
	if strings.ContainsAny(query, "*?") {
		return validateWildcardExpression(query)
	}
	parser := logicalExpressionParser{query: query}
	if err := parser.parseExpression(); err != nil {
		return fmt.Errorf("query は許可された論理検索式でなければなりません: %w", err)
	}
	if parser.position != len(parser.query) {
		return fmt.Errorf("query は許可された論理検索式でなければなりません")
	}
	return nil
}

func validateWildcardExpression(query string) error {
	hasLiteral := false
	for _, value := range query {
		switch value {
		case '*', '?':
		case ' ', '|', '!', '(', ')':
			return fmt.Errorf(
				"ワイルドカード式に U+0020 または論理演算子を含めることはできません",
			)
		default:
			hasLiteral = true
		}
	}
	if !hasLiteral {
		return fmt.Errorf("ワイルドカード式には通常文字が一文字以上必要です")
	}
	return nil
}

type logicalExpressionParser struct {
	query    string
	position int
}

func (p *logicalExpressionParser) parseExpression() error {
	if err := p.parseConjunction(); err != nil {
		return err
	}
	for p.peek('|') {
		p.position++
		if err := p.parseConjunction(); err != nil {
			return err
		}
	}
	return nil
}

func (p *logicalExpressionParser) parseConjunction() error {
	if err := p.parseFactor(); err != nil {
		return err
	}
	for p.peek(' ') {
		for p.peek(' ') {
			p.position++
		}
		if err := p.parseFactor(); err != nil {
			return err
		}
	}
	return nil
}

func (p *logicalExpressionParser) parseFactor() error {
	if p.peek('!') {
		p.position++
	}
	if p.peek('(') {
		p.position++
		if err := p.parseExpression(); err != nil {
			return err
		}
		if !p.peek(')') {
			return fmt.Errorf("丸括弧が対応していません")
		}
		p.position++
		return nil
	}
	return p.parseTerm()
}

func (p *logicalExpressionParser) parseTerm() error {
	start := p.position
	for p.position < len(p.query) {
		value, size := utf8.DecodeRuneInString(p.query[p.position:])
		if strings.ContainsRune(" |!()*?", value) {
			break
		}
		p.position += size
	}
	if p.position == start {
		return fmt.Errorf("検索語が必要です")
	}
	return nil
}

func (p *logicalExpressionParser) peek(value byte) bool {
	return p.position < len(p.query) && p.query[p.position] == value
}

func containsASCIIControl(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] <= 0x1f || value[index] == 0x7f {
			return true
		}
	}
	return false
}
