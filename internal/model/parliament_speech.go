package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

// ParliamentSpeechValues は、ParliamentSpeech の作成に必要な値を保持する。
type ParliamentSpeechValues struct {
	SpeechID    string
	SpeechOrder int
	Speaker     ParliamentSpeaker
	SpeechText  string
	StartPage   *int
	SpeechURL   string
	Meeting     ParliamentMeetingReference
	Source      InformationSource
}

// ParliamentSpeech は、国会会議録検索システムに登録された一つの発言を表す。
type ParliamentSpeech struct {
	speechID    string
	speechOrder int
	speaker     ParliamentSpeaker
	speechText  string
	startPage   *int
	speechURL   string
	meeting     ParliamentMeetingReference
	source      InformationSource
}

// ParliamentSpeakerValues は、発言者属性の作成に必要な値を保持する。
type ParliamentSpeakerValues struct {
	Name     string
	Reading  *string
	Group    *string
	Position *string
	Role     *string
}

// ParliamentSpeaker は、情報源が示す発言者属性を保持する。
type ParliamentSpeaker struct {
	name     string
	reading  *string
	group    *string
	position *string
	role     *string
}

// ParliamentMeetingReferenceValues は、会議録参照情報の作成に必要な値を保持する。
type ParliamentMeetingReferenceValues struct {
	MeetingRecordID string
	ImageKind       string
	Session         int
	HouseName       string
	MeetingName     string
	Issue           string
	MeetingDate     Date
	Closing         *bool
	MeetingURL      string
	PDFURL          *string
}

// ParliamentMeetingReference は、発言が属する会議録の参照情報を表す。
type ParliamentMeetingReference struct {
	meetingRecordID string
	imageKind       string
	session         int
	houseName       string
	meetingName     string
	issue           string
	meetingDate     Date
	closing         *bool
	meetingURL      string
	pdfURL          *string
}

// NewParliamentSpeech は、入力を複製して検証済みの発言を返す。
func NewParliamentSpeech(values ParliamentSpeechValues) (ParliamentSpeech, error) {
	speech := ParliamentSpeech{
		speechID:    values.SpeechID,
		speechOrder: values.SpeechOrder,
		speaker:     values.Speaker,
		speechText:  strings.TrimFunc(values.SpeechText, unicode.IsSpace),
		startPage:   cloneOptionalInt(values.StartPage),
		speechURL:   values.SpeechURL,
		meeting:     values.Meeting,
		source:      values.Source,
	}
	if err := speech.Validate(); err != nil {
		return ParliamentSpeech{}, err
	}
	return speech, nil
}

// NewParliamentSpeaker は、入力を複製して検証済みの発言者属性を返す。
func NewParliamentSpeaker(values ParliamentSpeakerValues) (ParliamentSpeaker, error) {
	speaker := ParliamentSpeaker{
		name:     values.Name,
		reading:  cloneOptionalString(values.Reading),
		group:    cloneOptionalString(values.Group),
		position: cloneOptionalString(values.Position),
		role:     cloneOptionalString(values.Role),
	}
	if err := speaker.Validate(); err != nil {
		return ParliamentSpeaker{}, err
	}
	return speaker, nil
}

// NewParliamentMeetingReference は、入力を複製して検証済みの会議録参照を返す。
func NewParliamentMeetingReference(
	values ParliamentMeetingReferenceValues,
) (ParliamentMeetingReference, error) {
	meeting := ParliamentMeetingReference{
		meetingRecordID: values.MeetingRecordID,
		imageKind:       values.ImageKind,
		session:         values.Session,
		houseName:       values.HouseName,
		meetingName:     values.MeetingName,
		issue:           values.Issue,
		meetingDate:     values.MeetingDate,
		closing:         cloneOptionalBool(values.Closing),
		meetingURL:      values.MeetingURL,
		pdfURL:          cloneOptionalString(values.PDFURL),
	}
	if err := meeting.Validate(); err != nil {
		return ParliamentMeetingReference{}, err
	}
	return meeting, nil
}

func (s ParliamentSpeech) SpeechID() string                    { return s.speechID }
func (s ParliamentSpeech) SpeechOrder() int                    { return s.speechOrder }
func (s ParliamentSpeech) Speaker() ParliamentSpeaker          { return s.speaker }
func (s ParliamentSpeech) SpeechText() string                  { return s.speechText }
func (s ParliamentSpeech) SpeechURL() string                   { return s.speechURL }
func (s ParliamentSpeech) Meeting() ParliamentMeetingReference { return s.meeting }
func (s ParliamentSpeech) Source() InformationSource           { return s.source }
func (s ParliamentSpeaker) Name() string                       { return s.name }
func (m ParliamentMeetingReference) MeetingRecordID() string   { return m.meetingRecordID }
func (m ParliamentMeetingReference) ImageKind() string         { return m.imageKind }
func (m ParliamentMeetingReference) Session() int              { return m.session }
func (m ParliamentMeetingReference) HouseName() string         { return m.houseName }
func (m ParliamentMeetingReference) MeetingName() string       { return m.meetingName }
func (m ParliamentMeetingReference) Issue() string             { return m.issue }
func (m ParliamentMeetingReference) MeetingDate() Date         { return m.meetingDate }
func (m ParliamentMeetingReference) MeetingURL() string        { return m.meetingURL }

// StartPage は、情報源が示した掲載開始頁と有無を返す。
func (s ParliamentSpeech) StartPage() (int, bool) {
	if s.startPage == nil {
		return 0, false
	}
	return *s.startPage, true
}

// Reading は、発言者よみと有無を返す。
func (s ParliamentSpeaker) Reading() (string, bool) { return optionalStringValue(s.reading) }

// Group は、発言者所属会派と有無を返す。
func (s ParliamentSpeaker) Group() (string, bool) { return optionalStringValue(s.group) }

// Position は、発言者肩書きと有無を返す。
func (s ParliamentSpeaker) Position() (string, bool) {
	return optionalStringValue(s.position)
}

// Role は、発言者役割と有無を返す。
func (s ParliamentSpeaker) Role() (string, bool) { return optionalStringValue(s.role) }

// Closing は、閉会中フラグと有無を返す。
func (m ParliamentMeetingReference) Closing() (bool, bool) {
	if m.closing == nil {
		return false, false
	}
	return *m.closing, true
}

// PDFURL は、会議録 PDF URL と有無を返す。
func (m ParliamentMeetingReference) PDFURL() (string, bool) {
	return optionalStringValue(m.pdfURL)
}

// Validate は、発言の必須項目と従属制約を確認する。
func (s ParliamentSpeech) Validate() error {
	switch {
	case s.speechID == "":
		return fmt.Errorf("国会発言の speechId は必須です")
	case s.speechOrder < 0:
		return fmt.Errorf("speechOrder は 0 以上でなければなりません")
	case s.speechText == "":
		return fmt.Errorf("speechText は必須です")
	case !isNDLOfficialHTTPSURL(s.speechURL):
		return fmt.Errorf("speechUrl は https://kokkai.ndl.go.jp/ の URL でなければなりません")
	}
	if s.startPage != nil && *s.startPage < 0 {
		return fmt.Errorf("startPage は 0 以上でなければなりません")
	}
	if err := s.speaker.Validate(); err != nil {
		return fmt.Errorf("speaker が有効ではありません: %w", err)
	}
	if err := s.meeting.Validate(); err != nil {
		return fmt.Errorf("meeting が有効ではありません: %w", err)
	}
	if err := s.source.Validate(); err != nil {
		return fmt.Errorf("source が有効ではありません: %w", err)
	}
	if s.source.Authority() != AuthorityOfficial ||
		!isNDLOfficialHTTPSURL(s.source.ServiceURL()) {
		return fmt.Errorf("source は国立国会図書館の公式情報源でなければなりません")
	}
	return nil
}

// Validate は、発言者属性の必須項目と省略可能項目を確認する。
func (s ParliamentSpeaker) Validate() error {
	if s.name == "" {
		return fmt.Errorf("speaker.name は必須です")
	}
	for field, value := range map[string]*string{
		"reading":  s.reading,
		"group":    s.group,
		"position": s.position,
		"role":     s.role,
	} {
		if value != nil && *value == "" {
			return fmt.Errorf("speaker.%s は空文字にできません", field)
		}
	}
	return nil
}

// Validate は、会議録参照の必須項目と URL を確認する。
func (m ParliamentMeetingReference) Validate() error {
	switch {
	case m.meetingRecordID == "":
		return fmt.Errorf("meeting.meetingRecordId は必須です")
	case m.imageKind == "":
		return fmt.Errorf("meeting.imageKind は必須です")
	case m.session < 0:
		return fmt.Errorf("meeting.session は 0 以上でなければなりません")
	case m.houseName == "":
		return fmt.Errorf("meeting.houseName は必須です")
	case m.meetingName == "":
		return fmt.Errorf("meeting.meetingName は必須です")
	case m.issue == "":
		return fmt.Errorf("meeting.issue は必須です")
	case !isNDLOfficialHTTPSURL(m.meetingURL):
		return fmt.Errorf("meeting.meetingUrl は https://kokkai.ndl.go.jp/ の URL でなければなりません")
	}
	if err := m.meetingDate.Validate(); err != nil {
		return fmt.Errorf("meeting.meetingDate が有効ではありません: %w", err)
	}
	if m.pdfURL != nil {
		if *m.pdfURL == "" {
			return fmt.Errorf("meeting.pdfUrl は空文字にできません")
		}
		if !isNDLOfficialHTTPSURL(*m.pdfURL) {
			return fmt.Errorf("meeting.pdfUrl は https://kokkai.ndl.go.jp/ の URL でなければなりません")
		}
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-034 の項目名で発言者属性を表す。
func (s ParliamentSpeaker) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Name     string  `json:"name"`
		Reading  *string `json:"reading,omitempty"`
		Group    *string `json:"group,omitempty"`
		Position *string `json:"position,omitempty"`
		Role     *string `json:"role,omitempty"`
	}{
		Name:     s.name,
		Reading:  cloneOptionalString(s.reading),
		Group:    cloneOptionalString(s.group),
		Position: cloneOptionalString(s.position),
		Role:     cloneOptionalString(s.role),
	})
}

// MarshalJSON は、SOT-MODEL-034 の項目名で会議録参照を表す。
func (m ParliamentMeetingReference) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		MeetingRecordID string  `json:"meetingRecordId"`
		ImageKind       string  `json:"imageKind"`
		Session         int     `json:"session"`
		HouseName       string  `json:"houseName"`
		MeetingName     string  `json:"meetingName"`
		Issue           string  `json:"issue"`
		MeetingDate     Date    `json:"meetingDate"`
		Closing         *bool   `json:"closing,omitempty"`
		MeetingURL      string  `json:"meetingUrl"`
		PDFURL          *string `json:"pdfUrl,omitempty"`
	}{
		MeetingRecordID: m.meetingRecordID,
		ImageKind:       m.imageKind,
		Session:         m.session,
		HouseName:       m.houseName,
		MeetingName:     m.meetingName,
		Issue:           m.issue,
		MeetingDate:     m.meetingDate,
		Closing:         cloneOptionalBool(m.closing),
		MeetingURL:      m.meetingURL,
		PDFURL:          cloneOptionalString(m.pdfURL),
	})
}

// MarshalJSON は、SOT-MODEL-034 の項目名で国会発言を表す。
func (s ParliamentSpeech) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		SpeechID    string                     `json:"speechId"`
		SpeechOrder int                        `json:"speechOrder"`
		Speaker     ParliamentSpeaker          `json:"speaker"`
		SpeechText  string                     `json:"speechText"`
		StartPage   *int                       `json:"startPage,omitempty"`
		SpeechURL   string                     `json:"speechUrl"`
		Meeting     ParliamentMeetingReference `json:"meeting"`
		Source      InformationSource          `json:"source"`
	}{
		SpeechID:    s.speechID,
		SpeechOrder: s.speechOrder,
		Speaker:     s.speaker,
		SpeechText:  s.speechText,
		StartPage:   cloneOptionalInt(s.startPage),
		SpeechURL:   s.speechURL,
		Meeting:     s.meeting,
		Source:      s.source,
	})
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*ParliamentSpeech) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ParliamentSpeech は JSON から直接復元できません。境界専用の入力型から NewParliamentSpeech を使用してください",
	)
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*ParliamentSpeaker) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ParliamentSpeaker は JSON から直接復元できません。境界専用の入力型から NewParliamentSpeaker を使用してください",
	)
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*ParliamentMeetingReference) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ParliamentMeetingReference は JSON から直接復元できません。境界専用の入力型から NewParliamentMeetingReference を使用してください",
	)
}

func isNDLOfficialHTTPSURL(value string) bool {
	if !isHTTPSURL(value) {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil &&
		strings.EqualFold(parsed.Host, "kokkai.ndl.go.jp") &&
		parsed.User == nil &&
		parsed.Fragment == ""
}
