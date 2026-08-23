package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const dietSpeechesUsageNotice = "国会会議録の発言は発言者等が著作権を有する場合があります。利用条件を確認してください。発言は現行法令または法的結論を示すものではありません。"

type searchDietSpeechesInputSchema struct {
	Query             string `json:"query,omitempty"`
	Speaker           string `json:"speaker,omitempty"`
	MeetingName       string `json:"meetingName,omitempty"`
	House             string `json:"house,omitempty"`
	FromDate          string `json:"fromDate,omitempty"`
	UntilDate         string `json:"untilDate,omitempty"`
	Limit             int    `json:"limit,omitempty"`
	ContinuationToken string `json:"continuationToken,omitempty"`
}

type searchDietSpeechesOutput struct {
	UsageNotice string                         `json:"usageNotice"`
	Items       []searchDietSpeechesOutputItem `json:"items"`
	Page        searchJudicialCasesOutputPage  `json:"page"`
}

type searchDietSpeechesOutputItem struct {
	Ref        searchJudicialCasesOutputRef    `json:"ref"`
	Provenance []searchJudicialCasesOutputProv `json:"provenance"`
	Data       searchDietSpeechesOutputData    `json:"data"`
}

type searchDietSpeechesOutputData struct {
	SpeechID    string                          `json:"speechId"`
	SpeechOrder int                             `json:"speechOrder"`
	Speaker     searchDietSpeechesOutputSpeaker `json:"speaker"`
	SpeechText  string                          `json:"speechText"`
	StartPage   *int                            `json:"startPage,omitempty"`
	SpeechURL   string                          `json:"speechUrl"`
	Meeting     searchDietSpeechesOutputMeeting `json:"meeting"`
	Source      searchJudicialCasesOutputSource `json:"source"`
}

type searchDietSpeechesOutputSpeaker struct {
	Name     string `json:"name"`
	Reading  string `json:"reading,omitempty"`
	Group    string `json:"group,omitempty"`
	Position string `json:"position,omitempty"`
	Role     string `json:"role,omitempty"`
}

type searchDietSpeechesOutputMeeting struct {
	MeetingRecordID string `json:"meetingRecordId"`
	ImageKind       string `json:"imageKind"`
	Session         int    `json:"session"`
	HouseName       string `json:"houseName"`
	MeetingName     string `json:"meetingName"`
	Issue           string `json:"issue"`
	MeetingDate     string `json:"meetingDate"`
	Closing         *bool  `json:"closing,omitempty"`
	MeetingURL      string `json:"meetingUrl"`
	PDFURL          string `json:"pdfUrl,omitempty"`
}

func addSearchDietSpeechesTool(
	server *sdk.Server,
	searcher parliamentspeechsearch.Port,
) {
	inputSchema := mustSchemaFor[searchDietSpeechesInputSchema]()
	for _, key := range []string{"query", "speaker", "meetingName"} {
		inputSchema.Properties[key] = withUTF8ByteLimit(
			inputSchema.Properties[key],
			nil,
			parliamentspeechsearch.MaxTextBytes,
			"有効な UTF-8 で、正規化後に 512 byte 以下の検索文字列。",
		)
	}
	inputSchema.Properties["limit"].Minimum = jsonschemaPtr(1.0)
	inputSchema.Properties["limit"].Maximum = jsonschemaPtr(30.0)
	inputSchema.Properties["limit"].Default = json.RawMessage("20")
	inputSchema.Properties["continuationToken"] = withUTF8ByteLimit(
		inputSchema.Properties["continuationToken"],
		nil,
		parliamentspeechsearch.MaxTokenBytes,
		"有効な UTF-8 で 4096 byte 以下の継続トークン。",
	)
	server.AddTool(&sdk.Tool{
		Name:         "search_diet_speeches",
		Description:  "国会会議録検索システムの公式発言を検索し、発言本文と会議情報を共通形式で返します。",
		InputSchema:  inputSchema,
		OutputSchema: mustSchemaFor[searchDietSpeechesOutput](),
	}, func(ctx context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return callSearchDietSpeeches(ctx, searcher, request.Params.Arguments)
	})
}

func callSearchDietSpeeches(
	ctx context.Context,
	searcher parliamentspeechsearch.Port,
	arguments json.RawMessage,
) (*sdk.CallToolResult, error) {
	if isNilParliamentSpeechSearchPort(searcher) {
		return errorToolResult(newInternalErrorResult())
	}
	request, err := decodeSearchDietSpeechesInput(arguments)
	if err != nil {
		return errorToolResult(newInvalidArgumentResult(err))
	}
	page, err := searcher.Search(ctx, request)
	if err != nil {
		return errorToolResult(mapSearchDietSpeechesError(err))
	}
	if err := page.Validate(); err != nil {
		return errorToolResult(newInternalErrorResult())
	}
	return successSearchDietSpeechesToolResult(mapSearchDietSpeechesOutput(page))
}

func decodeSearchDietSpeechesInput(
	arguments json.RawMessage,
) (parliamentspeechsearch.Request, error) {
	input := bytes.TrimSpace(arguments)
	if len(input) == 0 ||
		!hasValidJSONSurrogatePairs(input) ||
		input[0] != '{' ||
		input[len(input)-1] != '}' {
		return parliamentspeechsearch.Request{},
			fmt.Errorf("入力は JSON object でなければなりません")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil || fields == nil {
		return parliamentspeechsearch.Request{}, fmt.Errorf("入力は JSON object でなければなりません")
	}
	for key := range fields {
		switch key {
		case "query", "speaker", "meetingName", "house", "fromDate", "untilDate", "limit", "continuationToken":
		default:
			return parliamentspeechsearch.Request{}, fmt.Errorf("定義していない入力項目は使用できません")
		}
	}

	var values parliamentspeechsearch.RequestValues
	if raw, ok := fields["query"]; ok {
		if isJSONNull(raw) {
			return parliamentspeechsearch.Request{}, fmt.Errorf("query に null は使用できません")
		}
		if err := decodeStrictJSONString(raw, &values.Query); err != nil {
			return parliamentspeechsearch.Request{}, fmt.Errorf("query は文字列でなければなりません")
		}
	}
	if raw, ok := fields["speaker"]; ok {
		if isJSONNull(raw) {
			return parliamentspeechsearch.Request{}, fmt.Errorf("speaker に null は使用できません")
		}
		if err := decodeStrictJSONString(raw, &values.Speaker); err != nil {
			return parliamentspeechsearch.Request{}, fmt.Errorf("speaker は文字列でなければなりません")
		}
	}
	if raw, ok := fields["meetingName"]; ok {
		if isJSONNull(raw) {
			return parliamentspeechsearch.Request{}, fmt.Errorf("meetingName に null は使用できません")
		}
		if err := decodeStrictJSONString(raw, &values.MeetingName); err != nil {
			return parliamentspeechsearch.Request{}, fmt.Errorf("meetingName は文字列でなければなりません")
		}
	}
	if raw, ok := fields["house"]; ok {
		if isJSONNull(raw) {
			return parliamentspeechsearch.Request{}, fmt.Errorf("house に null は使用できません")
		}
		var house string
		if err := decodeStrictJSONString(raw, &house); err != nil {
			return parliamentspeechsearch.Request{}, fmt.Errorf("house は文字列でなければなりません")
		}
		values.House = parliamentspeechsearch.House(house)
	}
	if raw, ok := fields["fromDate"]; ok {
		if isJSONNull(raw) {
			return parliamentspeechsearch.Request{}, fmt.Errorf("fromDate に null は使用できません")
		}
		text, err := decodeSearchDietSpeechesDateString(raw, "fromDate")
		if err != nil {
			return parliamentspeechsearch.Request{}, err
		}
		date, err := model.NewDate(text)
		if err != nil {
			return parliamentspeechsearch.Request{}, fmt.Errorf("fromDate は実在する YYYY-MM-DD でなければなりません")
		}
		values.FromDate = &date
	}
	if raw, ok := fields["untilDate"]; ok {
		if isJSONNull(raw) {
			return parliamentspeechsearch.Request{}, fmt.Errorf("untilDate に null は使用できません")
		}
		text, err := decodeSearchDietSpeechesDateString(raw, "untilDate")
		if err != nil {
			return parliamentspeechsearch.Request{}, err
		}
		date, err := model.NewDate(text)
		if err != nil {
			return parliamentspeechsearch.Request{}, fmt.Errorf("untilDate は実在する YYYY-MM-DD でなければなりません")
		}
		values.UntilDate = &date
	}
	if raw, ok := fields["limit"]; ok {
		if isJSONNull(raw) {
			return parliamentspeechsearch.Request{}, fmt.Errorf("limit に null は使用できません")
		}
		limit, err := decodeExactJudicialSearchLimit(raw)
		if err != nil {
			return parliamentspeechsearch.Request{}, fmt.Errorf("limit は整数でなければなりません")
		}
		values.Limit = &limit
	}
	if raw, ok := fields["continuationToken"]; ok {
		if isJSONNull(raw) {
			return parliamentspeechsearch.Request{}, fmt.Errorf("continuationToken に null は使用できません")
		}
		if err := decodeStrictJSONString(raw, &values.ContinuationToken); err != nil {
			return parliamentspeechsearch.Request{}, fmt.Errorf("continuationToken は文字列でなければなりません")
		}
	}
	return parliamentspeechsearch.NewRequest(values)
}

func decodeSearchDietSpeechesDateString(
	raw json.RawMessage,
	field string,
) (string, error) {
	var value string
	if err := decodeStrictJSONString(raw, &value); err != nil {
		return "", fmt.Errorf("%s は文字列でなければなりません", field)
	}
	return value, nil
}

func mapSearchDietSpeechesOutput(
	page parliamentspeechsearch.Page,
) searchDietSpeechesOutput {
	items := page.Items()
	outputItems := make([]searchDietSpeechesOutputItem, len(items))
	for index, item := range items {
		outputItems[index] = searchDietSpeechesOutputItem{
			Ref:        mapSearchJudicialCasesRef(item.Ref()),
			Provenance: mapSearchJudicialCasesProvenance(item.Provenance()),
			Data:       mapSearchDietSpeechData(item.Data()),
		}
	}
	return searchDietSpeechesOutput{
		UsageNotice: dietSpeechesUsageNotice,
		Items:       outputItems,
		Page:        mapSearchJudicialCasesPage(page.Page()),
	}
}

func mapSearchDietSpeechData(
	data model.ParliamentSpeech,
) searchDietSpeechesOutputData {
	output := searchDietSpeechesOutputData{
		SpeechID:    data.SpeechID(),
		SpeechOrder: data.SpeechOrder(),
		Speaker:     mapSearchDietSpeechSpeaker(data.Speaker()),
		SpeechText:  data.SpeechText(),
		SpeechURL:   data.SpeechURL(),
		Meeting:     mapSearchDietSpeechMeeting(data.Meeting()),
		Source:      mapSearchJudicialCasesSource(data.Source()),
	}
	if startPage, exists := data.StartPage(); exists {
		output.StartPage = &startPage
	}
	return output
}

func mapSearchDietSpeechSpeaker(
	speaker model.ParliamentSpeaker,
) searchDietSpeechesOutputSpeaker {
	output := searchDietSpeechesOutputSpeaker{Name: speaker.Name()}
	output.Reading, _ = speaker.Reading()
	output.Group, _ = speaker.Group()
	output.Position, _ = speaker.Position()
	output.Role, _ = speaker.Role()
	return output
}

func mapSearchDietSpeechMeeting(
	meeting model.ParliamentMeetingReference,
) searchDietSpeechesOutputMeeting {
	output := searchDietSpeechesOutputMeeting{
		MeetingRecordID: meeting.MeetingRecordID(),
		ImageKind:       meeting.ImageKind(),
		Session:         meeting.Session(),
		HouseName:       meeting.HouseName(),
		MeetingName:     meeting.MeetingName(),
		Issue:           meeting.Issue(),
		MeetingDate:     meeting.MeetingDate().String(),
		MeetingURL:      meeting.MeetingURL(),
	}
	if closing, exists := meeting.Closing(); exists {
		output.Closing = &closing
	}
	if pdfURL, exists := meeting.PDFURL(); exists {
		output.PDFURL = pdfURL
	}
	return output
}

func mapSearchDietSpeechesError(err error) model.ErrorResult {
	var sourceError model.SourceError
	if errors.As(err, &sourceError) {
		result, buildErr := model.NewErrorResultFromSourceError(sourceError)
		if buildErr == nil {
			return result
		}
	}
	return newInvalidArgumentResult(err)
}

func successSearchDietSpeechesToolResult(
	output searchDietSpeechesOutput,
) (*sdk.CallToolResult, error) {
	payload, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}
	return &sdk.CallToolResult{
		Content: []sdk.Content{
			&sdk.TextContent{Text: string(payload)},
		},
		StructuredContent: json.RawMessage(payload),
		IsError:           false,
	}, nil
}

func jsonschemaPtr(value float64) *float64 {
	return &value
}
