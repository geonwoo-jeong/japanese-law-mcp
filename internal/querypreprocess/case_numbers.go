package querypreprocess

import (
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func extractJudicialCaseNumbers(
	query string,
) ([]legalquery.JudicialCaseNumberMention, []byteSpan, error) {
	mentions := make([]legalquery.JudicialCaseNumberMention, 0)
	spans := make([]byteSpan, 0)
	for startByte := range query {
		if !hasJudicialEraPrefix(query[startByte:]) {
			continue
		}
		parsed, err := legalquery.ParseJudicialCaseNumberPrefix(
			query[startByte:],
		)
		if err != nil {
			continue
		}
		endByte := startByte + parsed.EndByte()
		if !hasCompleteJudicialCaseNumberBoundary(
			query,
			startByte,
			endByte,
		) {
			continue
		}
		span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
			StartByte: startByte,
			EndByte:   endByte,
		})
		if err != nil {
			return nil, nil, err
		}
		mention, err := legalquery.NewJudicialCaseNumberMention(
			legalquery.JudicialCaseNumberMentionValues{
				Span:         span,
				Surface:      query[startByte:endByte],
				Era:          parsed.Era(),
				Year:         parsed.Year(),
				CaseCode:     parsed.CaseCode(),
				SerialNumber: parsed.SerialNumber(),
			},
		)
		if err != nil {
			return nil, nil, err
		}
		mentions = append(mentions, mention)
		spans = append(spans, byteSpan{
			startByte: startByte,
			endByte:   endByte,
		})
	}
	return mentions, spans, nil
}

func hasJudicialEraPrefix(value string) bool {
	for _, era := range []string{"昭和", "平成", "令和"} {
		if len(value) >= len(era) && value[:len(era)] == era {
			return true
		}
	}
	return false
}

func hasCompleteJudicialCaseNumberBoundary(
	query string,
	startByte int,
	endByte int,
) bool {
	if startByte > 0 {
		previous, _ := utf8.DecodeLastRuneInString(query[:startByte])
		if isASCIIOrFullwidthDecimal(previous) {
			return false
		}
	}
	if endByte >= len(query) {
		return true
	}
	next, _ := utf8.DecodeRuneInString(query[endByte:])
	if isASCIIOrFullwidthDecimal(next) ||
		unicode.In(next, unicode.Latin) {
		return false
	}
	switch next {
	case '年', '第', '号',
		'.', '．',
		'+', '＋',
		'-', '−', '－',
		'(', ')', '（', '）':
		return false
	default:
		return true
	}
}

func isASCIIOrFullwidthDecimal(value rune) bool {
	return (value >= '0' && value <= '9') ||
		(value >= '０' && value <= '９')
}
