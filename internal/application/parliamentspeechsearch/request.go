// Package parliamentspeechsearch は、国会発言検索 capability の型付き境界を提供する。
package parliamentspeechsearch

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/continuation"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	// CapabilityID は、国会発言検索 capability の識別子である。
	CapabilityID = "parliament.speech.search"
	// MajorVersion は、国会発言検索 capability のメジャーバージョンである。
	MajorVersion = 1
	// DefaultLimit は、limit を省略した場合の返却上限である。
	DefaultLimit = 20
	// MaxLimit は、一回の検索で指定できる返却上限である。
	MaxLimit = 30
	// MaxTextBytes は、検索文字列に許可する UTF-8 byte 数である。
	MaxTextBytes = 512
	// MaxTokenBytes は、継続トークンに許可する UTF-8 byte 数である。
	MaxTokenBytes = 4096
)

// House は、院名の正規化済み列挙値である。
type House string

const (
	HouseOfRepresentatives House = "house_of_representatives"
	HouseOfCouncillors     House = "house_of_councillors"
	BothHouses             House = "both_houses"
	ConferenceOfBothHouses House = "conference_of_both_houses"
)

// RequestValues は、Request の作成に必要な境界値を保持する。
type RequestValues struct {
	Query             string
	Speaker           string
	MeetingName       string
	House             House
	FromDate          *model.Date
	UntilDate         *model.Date
	Limit             *int
	ContinuationToken string
}

// Request は、parliament.speech.search@1 の正規化済み入力を不変に保持する。
type Request struct {
	query             string
	speaker           string
	meetingName       string
	house             House
	fromDate          *model.Date
	untilDate         *model.Date
	limit             int
	continuationToken string
}

// NewRequest は、検索条件を正規化し、検証済みの Request を返す。
func NewRequest(values RequestValues) (Request, error) {
	limit := DefaultLimit
	if values.Limit != nil {
		limit = *values.Limit
	}
	query := strings.TrimFunc(values.Query, unicode.IsSpace)
	speaker := strings.TrimFunc(values.Speaker, unicode.IsSpace)
	meetingName := strings.TrimFunc(values.MeetingName, unicode.IsSpace)
	for _, field := range []struct {
		name       string
		raw        string
		normalized string
	}{
		{name: "query", raw: values.Query, normalized: query},
		{name: "speaker", raw: values.Speaker, normalized: speaker},
		{name: "meetingName", raw: values.MeetingName, normalized: meetingName},
	} {
		if field.raw != "" && field.normalized == "" {
			return Request{}, fmt.Errorf("%s は空白だけにできません", field.name)
		}
	}
	request := Request{
		query:             query,
		speaker:           speaker,
		meetingName:       meetingName,
		house:             House(strings.TrimFunc(string(values.House), unicode.IsSpace)),
		fromDate:          cloneDate(values.FromDate),
		untilDate:         cloneDate(values.UntilDate),
		limit:             limit,
		continuationToken: values.ContinuationToken,
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func (r Request) Query() (string, bool)       { return optionalText(r.query) }
func (r Request) Speaker() (string, bool)     { return optionalText(r.speaker) }
func (r Request) MeetingName() (string, bool) { return optionalText(r.meetingName) }
func (r Request) Limit() int                  { return r.limit }

// House は、院名条件と有無を返す。
func (r Request) House() (House, bool) {
	return r.house, r.house != ""
}

// FromDate は、開催日の下限と有無を返す。
func (r Request) FromDate() (model.Date, bool) {
	if r.fromDate == nil {
		return model.Date{}, false
	}
	return *r.fromDate, true
}

// UntilDate は、開催日の上限と有無を返す。
func (r Request) UntilDate() (model.Date, bool) {
	if r.untilDate == nil {
		return model.Date{}, false
	}
	return *r.untilDate, true
}

// ContinuationToken は、継続トークンと有無を返す。
func (r Request) ContinuationToken() (string, bool) {
	return r.continuationToken, r.continuationToken != ""
}

// Validate は、parliament.speech.search@1 の共通入力制約を確認する。
func (r Request) Validate() error {
	conditions := 0
	for _, value := range []string{r.query, r.speaker, r.meetingName} {
		if value != "" {
			conditions++
		}
		if err := validateSearchText(value); err != nil {
			return err
		}
	}
	if r.house != "" {
		conditions++
		if !r.house.valid() {
			return fmt.Errorf("house は定義済みの列挙値でなければなりません")
		}
	}
	if r.fromDate != nil {
		conditions++
		if err := r.fromDate.Validate(); err != nil {
			return fmt.Errorf("fromDate が有効ではありません: %w", err)
		}
	}
	if r.untilDate != nil {
		conditions++
		if err := r.untilDate.Validate(); err != nil {
			return fmt.Errorf("untilDate が有効ではありません: %w", err)
		}
	}
	if conditions == 0 {
		return fmt.Errorf("query、speaker、meetingName、house、fromDate または untilDate の一つ以上が必要です")
	}
	if r.fromDate != nil && r.untilDate != nil &&
		r.fromDate.String() > r.untilDate.String() {
		return fmt.Errorf("fromDate は untilDate 以下でなければなりません")
	}
	if r.limit < 1 || r.limit > MaxLimit {
		return fmt.Errorf("limit は 1 以上 30 以下でなければなりません")
	}
	if !utf8.ValidString(r.continuationToken) {
		return fmt.Errorf("continuationToken は有効な UTF-8 でなければなりません")
	}
	if len(r.continuationToken) > MaxTokenBytes {
		return fmt.Errorf("continuationToken は UTF-8 で 4096 byte 以下でなければなりません")
	}
	if r.continuationToken != "" {
		return fmt.Errorf(
			"continuationToken は現在の provider では使用できません。最初の 30 件だけを取得してください",
		)
	}
	return nil
}

// ConditionObject は、継続条件 fingerprint 用の正規化済み JSON object を返す。
func (r Request) ConditionObject() (continuation.JSONObject, error) {
	if err := r.Validate(); err != nil {
		return continuation.JSONObject{}, err
	}
	raw, err := json.Marshal(struct {
		Query       string `json:"query,omitempty"`
		Speaker     string `json:"speaker,omitempty"`
		MeetingName string `json:"meetingName,omitempty"`
		House       House  `json:"house,omitempty"`
		FromDate    string `json:"fromDate,omitempty"`
		UntilDate   string `json:"untilDate,omitempty"`
		Limit       int    `json:"limit"`
	}{
		Query:       r.query,
		Speaker:     r.speaker,
		MeetingName: r.meetingName,
		House:       r.house,
		FromDate:    optionalDateString(r.fromDate),
		UntilDate:   optionalDateString(r.untilDate),
		Limit:       r.limit,
	})
	if err != nil {
		return continuation.JSONObject{}, fmt.Errorf("検索条件を JSON に変換できません: %w", err)
	}
	object, err := continuation.NewJSONObject(raw)
	if err != nil {
		return continuation.JSONObject{}, fmt.Errorf(
			"検索条件を canonical JSON object に変換できません: %w",
			err,
		)
	}
	return object, nil
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Request は JSON から直接復元できません。境界専用の入力型から NewRequest を使用してください",
	)
}

func (h House) valid() bool {
	switch h {
	case HouseOfRepresentatives,
		HouseOfCouncillors,
		BothHouses,
		ConferenceOfBothHouses:
		return true
	default:
		return false
	}
}

func validateSearchText(value string) error {
	if value == "" {
		return nil
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("検索文字列は有効な UTF-8 でなければなりません")
	}
	if len(value) > MaxTextBytes {
		return fmt.Errorf("検索文字列は UTF-8 で 512 byte 以下でなければなりません")
	}
	for index := 0; index < len(value); index++ {
		if value[index] <= 0x1f || value[index] == 0x7f {
			return fmt.Errorf("検索文字列に ASCII 制御文字を含めることはできません")
		}
	}
	return nil
}

func cloneDate(value *model.Date) *model.Date {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func optionalDateString(value *model.Date) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func optionalText(value string) (string, bool) {
	return value, value != ""
}
