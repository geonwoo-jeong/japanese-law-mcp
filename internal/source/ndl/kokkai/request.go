package kokkai

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
)

const (
	speechSearchEndpoint   = "https://kokkai.ndl.go.jp/api/speech"
	maximumRequestURLBytes = 2000
)

type speechSearchQueryParameter struct {
	name  string
	value string
}

func buildSpeechSearchHTTPRequest(
	ctx context.Context,
	request parliamentspeechsearch.Request,
) (*http.Request, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("国会発言検索 request が有効ではありません: %w", err)
	}

	parameters, err := speechSearchQueryParameters(request)
	if err != nil {
		return nil, err
	}
	requestURL := speechSearchEndpoint + "?" + encodeSpeechSearchQuery(parameters)
	if len(requestURL) > maximumRequestURLBytes {
		return nil, fmt.Errorf("国会発言検索の要求 URL は 2000 byte 以下でなければなりません")
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("国会発言検索の固定要求を作成できません")
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Accept-Encoding", "gzip")
	return httpRequest, nil
}

func speechSearchQueryParameters(
	request parliamentspeechsearch.Request,
) ([]speechSearchQueryParameter, error) {
	parameters := make([]speechSearchQueryParameter, 0, 9)
	parameters = appendOptionalSpeechSearchParameter(parameters, "any", request.Query)
	parameters = appendOptionalSpeechSearchParameter(parameters, "speaker", request.Speaker)
	parameters = appendOptionalSpeechSearchParameter(
		parameters,
		"nameOfMeeting",
		request.MeetingName,
	)
	if house, exists := request.House(); exists {
		mapped, err := mapSpeechSearchHouse(house)
		if err != nil {
			return nil, err
		}
		parameters = append(parameters, speechSearchQueryParameter{"nameOfHouse", mapped})
	}
	if from, exists := request.FromDate(); exists {
		parameters = append(parameters, speechSearchQueryParameter{"from", from.String()})
	}
	if until, exists := request.UntilDate(); exists {
		parameters = append(parameters, speechSearchQueryParameter{"until", until.String()})
	}
	return append(parameters,
		speechSearchQueryParameter{"startRecord", "1"},
		speechSearchQueryParameter{"maximumRecords", strconv.Itoa(request.Limit())},
		speechSearchQueryParameter{"recordPacking", "json"},
	), nil
}

func appendOptionalSpeechSearchParameter(
	parameters []speechSearchQueryParameter,
	name string,
	value func() (string, bool),
) []speechSearchQueryParameter {
	text, exists := value()
	if !exists {
		return parameters
	}
	return append(parameters, speechSearchQueryParameter{name: name, value: text})
}

func mapSpeechSearchHouse(house parliamentspeechsearch.House) (string, error) {
	switch house {
	case parliamentspeechsearch.HouseOfRepresentatives:
		return "衆議院", nil
	case parliamentspeechsearch.HouseOfCouncillors:
		return "参議院", nil
	case parliamentspeechsearch.BothHouses:
		return "両院", nil
	case parliamentspeechsearch.ConferenceOfBothHouses:
		return "両院協議会", nil
	default:
		return "", fmt.Errorf("国会発言検索の院名が定義されていません")
	}
}

func encodeSpeechSearchQuery(parameters []speechSearchQueryParameter) string {
	encoded := make([]string, len(parameters))
	for index, parameter := range parameters {
		encoded[index] = encodeSpeechSearchQueryComponent(parameter.name) + "=" +
			encodeSpeechSearchQueryComponent(parameter.value)
	}
	return strings.Join(encoded, "&")
}

func encodeSpeechSearchQueryComponent(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}
