package judicialcasenumber

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maximumSerialNumber = 99_999_999

// ParseResult は、事件番号 prefix の検証済み解析結果である。
type ParseResult struct {
	era          string
	year         int
	caseCode     string
	serialNumber int
	searchText   string
	endByte      int
}

func (p ParseResult) Era() string        { return p.era }
func (p ParseResult) Year() int          { return p.year }
func (p ParseResult) CaseCode() string   { return p.caseCode }
func (p ParseResult) SerialNumber() int  { return p.serialNumber }
func (p ParseResult) SearchText() string { return p.searchText }
func (p ParseResult) EndByte() int       { return p.endByte }

// ParsePrefix は、文字列先頭の完全な事件番号構造を解析する。
// 後続する通常の照会文は解析範囲に含めず、EndByte で境界を返す。
func ParsePrefix(value string) (ParseResult, error) {
	parser := parser{value: value}
	era, err := parser.consumeEra()
	if err != nil {
		return ParseResult{}, err
	}
	parser.consumeStructuralSpace()
	year, err := parser.consumeYear()
	if err != nil {
		return ParseResult{}, err
	}
	if err := validateYear(era, year); err != nil {
		return ParseResult{}, err
	}
	parser.consumeStructuralSpace()
	hasYearMarker := parser.consumeLiteral("年")
	if hasYearMarker {
		parser.consumeStructuralSpace()
	}
	caseCode, err := parser.consumeCaseCode()
	if err != nil {
		return ParseResult{}, err
	}
	parser.consumeStructuralSpace()
	hasSerialMarkers := parser.consumeLiteral("第")
	if hasSerialMarkers {
		parser.consumeStructuralSpace()
	}
	serialNumber, _, err := parser.consumeDecimalNumber()
	if err != nil {
		return ParseResult{}, fmt.Errorf("事件番号が有効ではありません: %w", err)
	}
	if serialNumber < 1 || serialNumber > maximumSerialNumber {
		return ParseResult{}, fmt.Errorf(
			"事件番号は 1 以上 %d 以下でなければなりません",
			maximumSerialNumber,
		)
	}
	serialText := strconv.Itoa(serialNumber)
	if hasSerialMarkers {
		parser.consumeStructuralSpace()
		if !parser.consumeLiteral("号") {
			return ParseResult{}, fmt.Errorf("第と号は対でなければなりません")
		}
	}

	var searchText strings.Builder
	searchText.WriteString(era)
	yearText := strconv.Itoa(year)
	if year == 1 && hasYearMarker {
		yearText = "元"
	}
	searchText.WriteString(yearText)
	if hasYearMarker {
		searchText.WriteString("年")
	}
	searchText.WriteString(caseCode)
	if hasSerialMarkers {
		searchText.WriteString("第")
	}
	searchText.WriteString(serialText)
	if hasSerialMarkers {
		searchText.WriteString("号")
	}
	return ParseResult{
		era:          era,
		year:         year,
		caseCode:     caseCode,
		serialNumber: serialNumber,
		searchText:   searchText.String(),
		endByte:      parser.offset,
	}, nil
}

type parser struct {
	value  string
	offset int
}

func (p *parser) consumeEra() (string, error) {
	for _, era := range []string{"昭和", "平成", "令和"} {
		if p.consumeLiteral(era) {
			return era, nil
		}
	}
	return "", fmt.Errorf("元号は昭和、平成または令和でなければなりません")
}

func (p *parser) consumeYear() (int, error) {
	if p.consumeLiteral("元") {
		return 1, nil
	}
	year, _, err := p.consumeDecimalNumber()
	if err != nil {
		return 0, fmt.Errorf("元号年が有効ではありません: %w", err)
	}
	return year, nil
}

func (p *parser) consumeCaseCode() (string, error) {
	prefix, prefixCount := p.consumeJapaneseCharacters()
	p.consumeStructuralSpace()

	open, size := p.nextRune()
	var closeRune rune
	switch open {
	case '(':
		closeRune = ')'
	case '（':
		closeRune = '）'
	default:
		return "", fmt.Errorf("事件符号には一組の括弧が必要です")
	}
	p.offset += size
	p.consumeStructuralSpace()
	core, coreCount := p.consumeJapaneseCharacters()
	if coreCount == 0 {
		return "", fmt.Errorf("事件符号の括弧内は一文字以上必要です")
	}
	p.consumeStructuralSpace()
	close, closeSize := p.nextRune()
	if close != closeRune {
		return "", fmt.Errorf("事件符号の括弧は対応しなければなりません")
	}
	p.offset += closeSize
	p.consumeStructuralSpace()
	suffix, suffixCount := p.consumeCaseCodeSuffix()

	characterCount := prefixCount + coreCount + suffixCount
	if characterCount < 1 || characterCount > 8 {
		return "", fmt.Errorf("事件符号は括弧を除いて一文字以上八文字以下でなければなりません")
	}
	return prefix + "(" + core + ")" + suffix, nil
}

func (p *parser) consumeCaseCodeSuffix() (string, int) {
	var value strings.Builder
	count := 0
	for p.offset < len(p.value) {
		current, size := p.nextRune()
		if current == '第' || !isCaseCodeCharacter(current) {
			break
		}
		value.WriteRune(current)
		count++
		p.offset += size
	}
	return value.String(), count
}

func (p *parser) consumeJapaneseCharacters() (string, int) {
	var value strings.Builder
	count := 0
	for p.offset < len(p.value) {
		current, size := p.nextRune()
		if !isCaseCodeCharacter(current) {
			break
		}
		value.WriteRune(current)
		count++
		p.offset += size
	}
	return value.String(), count
}

func (p *parser) consumeDecimalNumber() (int, string, error) {
	start := p.offset
	var normalized strings.Builder
	for p.offset < len(p.value) {
		current, size := p.nextRune()
		digit, ok := asciiDecimalDigit(current)
		if !ok {
			break
		}
		normalized.WriteByte(digit)
		p.offset += size
	}
	if p.offset == start {
		return 0, "", fmt.Errorf("十進数字が必要です")
	}
	number, err := strconv.Atoi(normalized.String())
	if err != nil {
		return 0, "", fmt.Errorf("十進数字が大きすぎます")
	}
	return number, normalized.String(), nil
}

func (p *parser) consumeStructuralSpace() {
	for p.offset < len(p.value) {
		current, size := p.nextRune()
		if !isStructuralSpace(current) {
			return
		}
		p.offset += size
	}
}

func isStructuralSpace(value rune) bool {
	if !unicode.IsSpace(value) || unicode.IsControl(value) {
		return false
	}
	switch value {
	case '\u2028', '\u2029':
		return false
	default:
		return true
	}
}

func (p *parser) consumeLiteral(value string) bool {
	if !strings.HasPrefix(p.value[p.offset:], value) {
		return false
	}
	p.offset += len(value)
	return true
}

func (p *parser) nextRune() (rune, int) {
	if p.offset >= len(p.value) {
		return utf8.RuneError, 0
	}
	return utf8.DecodeRuneInString(p.value[p.offset:])
}

func validateYear(era string, year int) error {
	maximum, exists := map[string]int{
		"昭和": 64,
		"平成": 31,
		"令和": 99,
	}[era]
	if !exists {
		return fmt.Errorf("元号は昭和、平成または令和でなければなりません")
	}
	if year < 1 || year > maximum {
		return fmt.Errorf("%sの年は 1 以上 %d 以下でなければなりません", era, maximum)
	}
	return nil
}

func isCaseCodeCharacter(value rune) bool {
	return unicode.In(value, unicode.Hiragana, unicode.Katakana, unicode.Han)
}

func asciiDecimalDigit(value rune) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return byte(value), true
	case value >= '０' && value <= '９':
		return byte('0' + value - '０'), true
	default:
		return 0, false
	}
}
