package kokkai

import "encoding/json"

const speechSearchBusyMessage = "現在、混み合っております。もうしばらくしてから再度アクセスしてください。"

type speechSearchResponse struct {
	totalCount         int
	returnedCount      int
	nextRecordPosition *int
	records            []speechRecord
}

type speechRecord struct {
	speechID        string
	speechOrder     int
	speakerName     string
	speakerReading  *string
	speakerGroup    *string
	speakerPosition *string
	speakerRole     *string
	speechText      string
	startPage       *int
	speechURL       string
	meetingRecordID string
	imageKind       string
	session         int
	houseName       string
	meetingName     string
	issue           string
	meetingDate     string
	closing         *bool
	meetingURL      string
	pdfURL          *string
}

type rawSpeechSearchResponse struct {
	NumberOfRecords    json.RawMessage `json:"numberOfRecords"`
	NumberOfReturn     json.RawMessage `json:"numberOfReturn"`
	StartRecord        json.RawMessage `json:"startRecord"`
	NextRecordPosition json.RawMessage `json:"nextRecordPosition"`
	SpeechRecord       json.RawMessage `json:"speechRecord"`
	Message            json.RawMessage `json:"message"`
	Details            json.RawMessage `json:"details"`
}

type rawSpeechRecord struct {
	SpeechID      json.RawMessage `json:"speechID"`
	SpeechOrder   json.RawMessage `json:"speechOrder"`
	Speaker       json.RawMessage `json:"speaker"`
	SpeakerYomi   json.RawMessage `json:"speakerYomi"`
	SpeakerGroup  json.RawMessage `json:"speakerGroup"`
	SpeakerPos    json.RawMessage `json:"speakerPosition"`
	SpeakerRole   json.RawMessage `json:"speakerRole"`
	Speech        json.RawMessage `json:"speech"`
	StartPage     json.RawMessage `json:"startPage"`
	SpeechURL     json.RawMessage `json:"speechURL"`
	IssueID       json.RawMessage `json:"issueID"`
	ImageKind     json.RawMessage `json:"imageKind"`
	Session       json.RawMessage `json:"session"`
	NameOfHouse   json.RawMessage `json:"nameOfHouse"`
	NameOfMeeting json.RawMessage `json:"nameOfMeeting"`
	Issue         json.RawMessage `json:"issue"`
	Date          json.RawMessage `json:"date"`
	Closing       json.RawMessage `json:"closing"`
	MeetingURL    json.RawMessage `json:"meetingURL"`
	PDFURL        json.RawMessage `json:"pdfURL"`
}
