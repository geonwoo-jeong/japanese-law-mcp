package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestParliamentSpeechCopiesValuesAndMarshalsJSON(t *testing.T) {
	t.Parallel()

	reading := "やまだたろう"
	group := "自由民主党"
	position := "国務大臣"
	role := "答弁者"
	pdfURL := "https://kokkai.ndl.go.jp/txt/213214889X00520240315.pdf"
	startPage := 17
	closing := true

	speaker, err := model.NewParliamentSpeaker(model.ParliamentSpeakerValues{
		Name:     "山田太郎",
		Reading:  &reading,
		Group:    &group,
		Position: &position,
		Role:     &role,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-034: NewParliamentSpeaker() のエラー = %v", err)
	}
	meeting, err := model.NewParliamentMeetingReference(
		model.ParliamentMeetingReferenceValues{
			MeetingRecordID: "213214889X00520240315",
			ImageKind:       "会議録",
			Session:         213,
			HouseName:       "参議院",
			MeetingName:     "法務委員会",
			Issue:           "5号",
			MeetingDate:     newDate(t, "2024-03-15"),
			Closing:         &closing,
			MeetingURL:      "https://kokkai.ndl.go.jp/txt/213214889X00520240315",
			PDFURL:          &pdfURL,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-034: NewParliamentMeetingReference() のエラー = %v", err)
	}
	source := newDietInformationSource(t)

	got, err := model.NewParliamentSpeech(model.ParliamentSpeechValues{
		SpeechID:    "123456",
		SpeechOrder: 9,
		Speaker:     speaker,
		SpeechText:  "  本文一行目\n本文二行目  ",
		StartPage:   &startPage,
		SpeechURL:   "https://kokkai.ndl.go.jp/txt/213214889X00520240315/9",
		Meeting:     meeting,
		Source:      source,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-034: NewParliamentSpeech() のエラー = %v", err)
	}

	reading = "変更後"
	group = "変更後"
	position = "変更後"
	role = "変更後"
	pdfURL = "https://kokkai.ndl.go.jp/changed.pdf"
	startPage = 1
	closing = false

	if got.SpeechID() != "123456" ||
		got.SpeechOrder() != 9 ||
		got.SpeechText() != "本文一行目\n本文二行目" ||
		got.SpeechURL() != "https://kokkai.ndl.go.jp/txt/213214889X00520240315/9" ||
		got.Source() != source {
		t.Fatalf("SOT-MODEL-034: ParliamentSpeech = %#v", got)
	}
	if value, exists := got.StartPage(); !exists || value != 17 {
		t.Fatalf("SOT-MODEL-034: StartPage() = %d, %t", value, exists)
	}
	if value, exists := got.Speaker().Reading(); !exists || value != "やまだたろう" {
		t.Fatalf("SOT-MODEL-034: speaker.reading = %q, %t", value, exists)
	}
	if value, exists := got.Speaker().Group(); !exists || value != "自由民主党" {
		t.Fatalf("SOT-MODEL-034: speaker.group = %q, %t", value, exists)
	}
	if value, exists := got.Speaker().Position(); !exists || value != "国務大臣" {
		t.Fatalf("SOT-MODEL-034: speaker.position = %q, %t", value, exists)
	}
	if value, exists := got.Speaker().Role(); !exists || value != "答弁者" {
		t.Fatalf("SOT-MODEL-034: speaker.role = %q, %t", value, exists)
	}
	if got.Meeting().MeetingRecordID() != "213214889X00520240315" ||
		got.Meeting().ImageKind() != "会議録" ||
		got.Meeting().Session() != 213 ||
		got.Meeting().HouseName() != "参議院" ||
		got.Meeting().MeetingName() != "法務委員会" ||
		got.Meeting().Issue() != "5号" ||
		got.Meeting().MeetingDate().String() != "2024-03-15" ||
		got.Meeting().MeetingURL() != "https://kokkai.ndl.go.jp/txt/213214889X00520240315" {
		t.Fatalf("SOT-MODEL-034: meeting = %#v", got.Meeting())
	}
	if value, exists := got.Meeting().Closing(); !exists || !value {
		t.Fatalf("SOT-MODEL-034: meeting.closing = %t, %t", value, exists)
	}
	if value, exists := got.Meeting().PDFURL(); !exists ||
		value != "https://kokkai.ndl.go.jp/txt/213214889X00520240315.pdf" {
		t.Fatalf("SOT-MODEL-034: meeting.pdfUrl = %q, %t", value, exists)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-034: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/034: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/034: JSON を再解析できない: %v", err)
	}
	want := map[string]any{
		"speechId":    "123456",
		"speechOrder": float64(9),
		"speechText":  "本文一行目\n本文二行目",
		"startPage":   float64(17),
		"speechUrl":   "https://kokkai.ndl.go.jp/txt/213214889X00520240315/9",
		"speaker": map[string]any{
			"name":     "山田太郎",
			"reading":  "やまだたろう",
			"group":    "自由民主党",
			"position": "国務大臣",
			"role":     "答弁者",
		},
		"meeting": map[string]any{
			"meetingRecordId": "213214889X00520240315",
			"imageKind":       "会議録",
			"session":         float64(213),
			"houseName":       "参議院",
			"meetingName":     "法務委員会",
			"issue":           "5号",
			"meetingDate":     "2024-03-15",
			"closing":         true,
			"meetingUrl":      "https://kokkai.ndl.go.jp/txt/213214889X00520240315",
			"pdfUrl":          "https://kokkai.ndl.go.jp/txt/213214889X00520240315.pdf",
		},
		"source": map[string]any{
			"id":         "ndl-diet-records",
			"name":       "国会会議録検索システム",
			"publisher":  "国立国会図書館",
			"authority":  "official",
			"serviceUrl": "https://kokkai.ndl.go.jp/",
		},
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-009/034: JSON = %#v、期待値 = %#v", object, want)
	}
}

func TestParliamentSpeechOmitsAbsentOptionalValues(t *testing.T) {
	t.Parallel()

	speech, err := model.NewParliamentSpeech(validParliamentSpeechValues(t))
	if err != nil {
		t.Fatalf("SOT-MODEL-034: NewParliamentSpeech() のエラー = %v", err)
	}
	if _, exists := speech.StartPage(); exists {
		t.Fatal("SOT-MODEL-034: 省略した startPage が存在すると判定された")
	}
	for _, getter := range []func() (string, bool){
		speech.Speaker().Reading,
		speech.Speaker().Group,
		speech.Speaker().Position,
		speech.Speaker().Role,
		speech.Meeting().PDFURL,
	} {
		if value, exists := getter(); exists || value != "" {
			t.Fatalf("SOT-MODEL-034: optional string = %q, %t", value, exists)
		}
	}
	if _, exists := speech.Meeting().Closing(); exists {
		t.Fatal("SOT-MODEL-034: 省略した closing が存在すると判定された")
	}

	encoded, err := json.Marshal(speech)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/034: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/034: JSON を再解析できない: %v", err)
	}
	if _, exists := object["startPage"]; exists {
		t.Fatalf("SOT-MODEL-009/034: startPage が省略されていない: %s", encoded)
	}
	speaker := object["speaker"].(map[string]any)
	for _, key := range []string{"reading", "group", "position", "role"} {
		if _, exists := speaker[key]; exists {
			t.Fatalf("SOT-MODEL-009/034: speaker.%s が省略されていない", key)
		}
	}
	meeting := object["meeting"].(map[string]any)
	for _, key := range []string{"closing", "pdfUrl"} {
		if _, exists := meeting[key]; exists {
			t.Fatalf("SOT-MODEL-009/034: meeting.%s が省略されていない", key)
		}
	}
}

func TestParliamentSpeechRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	startPage := -1
	values := validParliamentSpeechValues(t)
	tests := map[string]model.ParliamentSpeechValues{
		"speechId の欠落": withParliamentSpeechChange(values, func(v *model.ParliamentSpeechValues) {
			v.SpeechID = ""
		}),
		"speechOrder が負数": withParliamentSpeechChange(values, func(v *model.ParliamentSpeechValues) {
			v.SpeechOrder = -1
		}),
		"speechText の欠落": withParliamentSpeechChange(values, func(v *model.ParliamentSpeechValues) {
			v.SpeechText = ""
		}),
		"startPage が負数": withParliamentSpeechChange(values, func(v *model.ParliamentSpeechValues) {
			v.StartPage = &startPage
		}),
		"speechUrl が外部 origin": withParliamentSpeechChange(values, func(v *model.ParliamentSpeechValues) {
			v.SpeechURL = "https://example.com/speech"
		}),
		"speaker.name の欠落": withParliamentSpeechChange(values, func(v *model.ParliamentSpeechValues) {
			v.Speaker = model.ParliamentSpeaker{}
		}),
		"meeting の欠落": withParliamentSpeechChange(values, func(v *model.ParliamentSpeechValues) {
			v.Meeting = model.ParliamentMeetingReference{}
		}),
		"source が補助情報": withParliamentSpeechChange(values, func(v *model.ParliamentSpeechValues) {
			source, err := model.NewInformationSource(model.InformationSourceValues{
				ID:         "supplementary",
				Name:       "補助",
				Publisher:  "補助",
				Authority:  model.AuthoritySupplementary,
				ServiceURL: "https://kokkai.ndl.go.jp/",
			})
			if err != nil {
				t.Fatalf("InformationSource を作成できない: %v", err)
			}
			v.Source = source
		}),
	}
	for name, candidate := range tests {
		name, candidate := name, candidate
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewParliamentSpeech(candidate); err == nil {
				t.Fatal("SOT-MODEL-034: 不正な ParliamentSpeech を受理した")
			}
		})
	}
}

func TestParliamentSpeechRejectsDirectJSONDecodeAndInvalidZeroValues(t *testing.T) {
	t.Parallel()

	var speech model.ParliamentSpeech
	if err := json.Unmarshal([]byte(`{"speechId":"123"}`), &speech); err == nil {
		t.Fatal("SOT-MODEL-009/034: ParliamentSpeech を JSON から直接復元できた")
	}
	if err := speech.Validate(); err == nil {
		t.Fatal("SOT-MODEL-034: ParliamentSpeech のゼロ値を受理した")
	}

	var speaker model.ParliamentSpeaker
	if err := json.Unmarshal([]byte(`{"name":"山田太郎"}`), &speaker); err == nil {
		t.Fatal("SOT-MODEL-009/034: ParliamentSpeaker を JSON から直接復元できた")
	}
	if err := speaker.Validate(); err == nil {
		t.Fatal("SOT-MODEL-034: ParliamentSpeaker のゼロ値を受理した")
	}

	var meeting model.ParliamentMeetingReference
	if err := json.Unmarshal([]byte(`{"meetingRecordId":"123"}`), &meeting); err == nil {
		t.Fatal("SOT-MODEL-009/034: ParliamentMeetingReference を JSON から直接復元できた")
	}
	if err := meeting.Validate(); err == nil {
		t.Fatal("SOT-MODEL-034: ParliamentMeetingReference のゼロ値を受理した")
	}
}

func validParliamentSpeechValues(t *testing.T) model.ParliamentSpeechValues {
	t.Helper()
	return model.ParliamentSpeechValues{
		SpeechID:    "123456",
		SpeechOrder: 3,
		Speaker:     mustParliamentSpeaker(t, validParliamentSpeaker(t)),
		SpeechText:  "本文",
		SpeechURL:   "https://kokkai.ndl.go.jp/txt/213214889X00520240315/3",
		Meeting:     mustParliamentMeetingReference(t, validParliamentMeetingReference(t)),
		Source:      newDietInformationSource(t),
	}
}

func validParliamentSpeaker(t *testing.T) model.ParliamentSpeakerValues {
	t.Helper()
	return model.ParliamentSpeakerValues{Name: "山田太郎"}
}

func validParliamentMeetingReference(t *testing.T) model.ParliamentMeetingReferenceValues {
	t.Helper()
	return model.ParliamentMeetingReferenceValues{
		MeetingRecordID: "213214889X00520240315",
		ImageKind:       "会議録",
		Session:         213,
		HouseName:       "参議院",
		MeetingName:     "法務委員会",
		Issue:           "5号",
		MeetingDate:     newDate(t, "2024-03-15"),
		MeetingURL:      "https://kokkai.ndl.go.jp/txt/213214889X00520240315",
	}
}

func mustParliamentSpeaker(
	t *testing.T,
	values model.ParliamentSpeakerValues,
) model.ParliamentSpeaker {
	t.Helper()
	speaker, err := model.NewParliamentSpeaker(values)
	if err != nil {
		t.Fatalf("ParliamentSpeaker を作成できない: %v", err)
	}
	return speaker
}

func mustParliamentMeetingReference(
	t *testing.T,
	values model.ParliamentMeetingReferenceValues,
) model.ParliamentMeetingReference {
	t.Helper()
	meeting, err := model.NewParliamentMeetingReference(values)
	if err != nil {
		t.Fatalf("ParliamentMeetingReference を作成できない: %v", err)
	}
	return meeting
}

func withParliamentSpeechChange(
	values model.ParliamentSpeechValues,
	change func(*model.ParliamentSpeechValues),
) model.ParliamentSpeechValues {
	change(&values)
	return values
}

func newDietInformationSource(t *testing.T) model.InformationSource {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "ndl-diet-records",
		Name:       "国会会議録検索システム",
		Publisher:  "国立国会図書館",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://kokkai.ndl.go.jp/",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できない: %v", err)
	}
	return source
}
