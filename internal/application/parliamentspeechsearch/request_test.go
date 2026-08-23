package parliamentspeechsearch_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestRequestConstants(t *testing.T) {
	t.Parallel()

	if parliamentspeechsearch.CapabilityID != "parliament.speech.search" ||
		parliamentspeechsearch.MajorVersion != 1 ||
		parliamentspeechsearch.DefaultLimit != 20 ||
		parliamentspeechsearch.MaxLimit != 30 ||
		parliamentspeechsearch.MaxTokenBytes != 4096 {
		t.Fatal("SOT-IF-062: capability 定数が契約と一致しない")
	}
}

func TestNewRequestNormalizesOuterWhitespaceAndAppliesDefaultLimit(t *testing.T) {
	t.Parallel()

	fromDate := newDate(t, "2024-03-01")
	request, err := parliamentspeechsearch.NewRequest(
		parliamentspeechsearch.RequestValues{
			Query:    "\u3000\t 入管 法 \n",
			Speaker:  "\u00a0山田太郎\u3000",
			FromDate: &fromDate,
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-062: NewRequest() のエラー = %v", err)
	}
	if value, exists := request.Query(); !exists || value != "入管 法" {
		t.Fatalf("SOT-IF-062: Query() = %q, %t", value, exists)
	}
	if value, exists := request.Speaker(); !exists || value != "山田太郎" {
		t.Fatalf("SOT-IF-062: Speaker() = %q, %t", value, exists)
	}
	if request.Limit() != parliamentspeechsearch.DefaultLimit {
		t.Fatalf("SOT-IF-062: 既定 limit = %d", request.Limit())
	}
	if value, exists := request.FromDate(); !exists || value.String() != "2024-03-01" {
		t.Fatalf("SOT-IF-062: FromDate() = %v, %t", value, exists)
	}
}

func TestNewRequestAcceptsEachSearchCondition(t *testing.T) {
	t.Parallel()

	fromDate := newDate(t, "2024-03-01")
	untilDate := newDate(t, "2024-03-31")
	tests := []parliamentspeechsearch.RequestValues{
		{Query: "永住許可"},
		{Speaker: "法務大臣"},
		{MeetingName: "法務委員会"},
		{House: parliamentspeechsearch.HouseOfCouncillors},
		{FromDate: &fromDate},
		{UntilDate: &untilDate},
	}
	for _, values := range tests {
		values := values
		t.Run("condition", func(t *testing.T) {
			t.Parallel()

			if _, err := parliamentspeechsearch.NewRequest(values); err != nil {
				t.Fatalf("SOT-IF-062: 単独条件を拒否した: %v", err)
			}
		})
	}
}

func TestNewRequestAcceptsEachHouse(t *testing.T) {
	t.Parallel()

	for _, house := range []parliamentspeechsearch.House{
		parliamentspeechsearch.HouseOfRepresentatives,
		parliamentspeechsearch.HouseOfCouncillors,
		parliamentspeechsearch.BothHouses,
		parliamentspeechsearch.ConferenceOfBothHouses,
	} {
		house := house
		t.Run(string(house), func(t *testing.T) {
			t.Parallel()

			if _, err := parliamentspeechsearch.NewRequest(
				parliamentspeechsearch.RequestValues{House: house},
			); err != nil {
				t.Fatalf("SOT-IF-062: house %q を拒否した: %v", house, err)
			}
		})
	}
}

func TestNewRequestRejectsMissingOrInvalidConditions(t *testing.T) {
	t.Parallel()

	fromDate := newDate(t, "2024-04-01")
	untilDate := newDate(t, "2024-03-01")
	tests := map[string]parliamentspeechsearch.RequestValues{
		"条件なし": {},
		"未知の house": {
			House: parliamentspeechsearch.House("senate"),
		},
		"日付逆転": {
			FromDate:  &fromDate,
			UntilDate: &untilDate,
		},
		"query 不正 UTF-8": {
			Query: string([]byte{'a', 0xff}),
		},
		"speaker 制御文字": {
			Speaker: "山田\n太郎",
		},
		"meetingName 512 byte 超": {
			MeetingName: strings.Repeat("会", 171),
		},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := parliamentspeechsearch.NewRequest(values); err == nil {
				t.Fatal("SOT-IF-062: 不正な条件を受理した")
			}
		})
	}
}

func TestNewRequestRejectsInvalidLimitAndToken(t *testing.T) {
	t.Parallel()

	invalidLimit := parliamentspeechsearch.MaxLimit + 1
	if _, err := parliamentspeechsearch.NewRequest(
		parliamentspeechsearch.RequestValues{
			Query: "永住許可",
			Limit: &invalidLimit,
		},
	); err == nil {
		t.Fatal("SOT-IF-062: 上限超過の limit を受理した")
	}
	if _, err := parliamentspeechsearch.NewRequest(
		parliamentspeechsearch.RequestValues{
			Query:             "永住許可",
			ContinuationToken: "opaque-token",
		},
	); err == nil {
		t.Fatal("SOT-IF-062: continuationToken を受理した")
	}
	if _, err := parliamentspeechsearch.NewRequest(
		parliamentspeechsearch.RequestValues{
			Query:             "永住許可",
			ContinuationToken: strings.Repeat("x", parliamentspeechsearch.MaxTokenBytes+1),
		},
	); err == nil {
		t.Fatal("SOT-IF-062: 上限超過の continuationToken を受理した")
	}
}

func TestNewRequestRejectsWhitespaceOnlyOptionalText(t *testing.T) {
	t.Parallel()

	for name, values := range map[string]parliamentspeechsearch.RequestValues{
		"query": {
			Query: " \u3000\t ",
			House: parliamentspeechsearch.HouseOfCouncillors,
		},
		"speaker": {
			Speaker: " \u3000\t ",
			House:   parliamentspeechsearch.HouseOfCouncillors,
		},
		"meetingName": {
			MeetingName: " \u3000\t ",
			House:       parliamentspeechsearch.HouseOfCouncillors,
		},
	} {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := parliamentspeechsearch.NewRequest(values); err == nil {
				t.Fatal("SOT-IF-062: 空白だけの検索文字列を受理した")
			}
		})
	}
}

func TestRequestConditionObjectUsesNormalizedValues(t *testing.T) {
	t.Parallel()

	limit := 7
	fromDate := newDate(t, "2024-03-01")
	request, err := parliamentspeechsearch.NewRequest(
		parliamentspeechsearch.RequestValues{
			Query:       "\u3000永住 許可\u3000",
			Speaker:     "法務大臣",
			House:       parliamentspeechsearch.HouseOfRepresentatives,
			FromDate:    &fromDate,
			Limit:       &limit,
			MeetingName: "法務委員会",
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-062: NewRequest() のエラー = %v", err)
	}
	object, err := request.ConditionObject()
	if err != nil {
		t.Fatalf("SOT-IF-062: ConditionObject() のエラー = %v", err)
	}
	want := []byte(`{"fromDate":"2024-03-01","house":"house_of_representatives","limit":7,"meetingName":"法務委員会","query":"永住 許可","speaker":"法務大臣"}`)
	if !bytes.Equal(object.Bytes(), want) {
		t.Fatalf("SOT-IF-062: condition = %s、期待値 = %s", object.Bytes(), want)
	}
}

func TestRequestRejectsDirectJSONDecodeAndInvalidZeroValue(t *testing.T) {
	t.Parallel()

	var request parliamentspeechsearch.Request
	if err := json.Unmarshal([]byte(`{"query":"永住許可"}`), &request); err == nil {
		t.Fatal("SOT-IF-062: Request を JSON から直接復元できた")
	}
	if err := request.Validate(); err == nil {
		t.Fatal("SOT-IF-062: Request のゼロ値を受理した")
	}
	if _, err := request.ConditionObject(); err == nil {
		t.Fatal("SOT-IF-062: ゼロ値から condition object を作成できた")
	}
}

func newDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("Date を作成できない: %v", err)
	}
	return date
}
