package legalquery

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/judicialcasenumber"
)

// JudicialCaseNumberMentionValues は、事件番号出現を構築する値である。
type JudicialCaseNumberMentionValues struct {
	Span         QuerySpan
	Surface      string
	Era          string
	Year         int
	CaseCode     string
	SerialNumber int
}

// JudicialCaseNumberMention は、原文に完全に明記された裁判事件番号を表す。
type JudicialCaseNumberMention struct {
	span         QuerySpan
	surface      string
	era          string
	year         int
	caseCode     string
	serialNumber int
	searchText   string
}

// NewJudicialCaseNumberMention は、原表記と構造値が一致する事件番号を返す。
func NewJudicialCaseNumberMention(
	values JudicialCaseNumberMentionValues,
) (JudicialCaseNumberMention, error) {
	mention := JudicialCaseNumberMention{
		span:         values.Span,
		surface:      values.Surface,
		era:          values.Era,
		year:         values.Year,
		caseCode:     values.CaseCode,
		serialNumber: values.SerialNumber,
	}
	if parsed, err := ParseJudicialCaseNumberPrefix(values.Surface); err == nil &&
		parsed.EndByte() == len(values.Surface) {
		mention.searchText = parsed.SearchText()
	}
	if err := mention.Validate(); err != nil {
		return JudicialCaseNumberMention{}, err
	}
	return mention, nil
}

func (m JudicialCaseNumberMention) Span() QuerySpan    { return m.span }
func (m JudicialCaseNumberMention) Surface() string    { return m.surface }
func (m JudicialCaseNumberMention) Era() string        { return m.era }
func (m JudicialCaseNumberMention) Year() int          { return m.year }
func (m JudicialCaseNumberMention) CaseCode() string   { return m.caseCode }
func (m JudicialCaseNumberMention) SerialNumber() int  { return m.serialNumber }
func (m JudicialCaseNumberMention) SearchText() string { return m.searchText }

// Validate は、事件番号の原表記、構造値および検索語の一致を確認する。
func (m JudicialCaseNumberMention) Validate() error {
	if err := validateMentionBase(m.span, m.surface); err != nil {
		return err
	}
	parsed, err := ParseJudicialCaseNumberPrefix(m.surface)
	if err != nil {
		return fmt.Errorf("surface は完全な事件番号でなければなりません: %w", err)
	}
	if parsed.EndByte() != len(m.surface) {
		return fmt.Errorf("surface は事件番号以外の文字を含められません")
	}
	if m.era != parsed.Era() ||
		m.year != parsed.Year() ||
		m.caseCode != parsed.CaseCode() ||
		m.serialNumber != parsed.SerialNumber() ||
		m.searchText != parsed.SearchText() {
		return fmt.Errorf("事件番号の構造値は surface から導出した値と一致しなければなりません")
	}
	return nil
}

// JudicialCaseNumberParse は、事件番号 prefix の検証済み解析結果である。
type JudicialCaseNumberParse = judicialcasenumber.ParseResult

// ParseJudicialCaseNumberPrefix は、文字列先頭の完全な事件番号構造を解析する。
// 後続する通常の照会文は解析範囲に含めず、EndByte で境界を返す。
func ParseJudicialCaseNumberPrefix(
	value string,
) (JudicialCaseNumberParse, error) {
	return judicialcasenumber.ParsePrefix(value)
}
