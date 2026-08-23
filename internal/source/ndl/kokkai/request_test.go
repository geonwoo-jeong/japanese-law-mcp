package kokkai

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestBuildSpeechSearchHTTPRequestMapsAllConditionsInFixedOrder(t *testing.T) {
	t.Parallel()

	limit := 30
	request, err := parliamentspeechsearch.NewRequest(
		parliamentspeechsearch.RequestValues{
			Query:       "\u3000永住 % 許可\u3000",
			Speaker:     " 山田 太郎 ",
			MeetingName: " 予算 委員会 ",
			House:       parliamentspeechsearch.ConferenceOfBothHouses,
			FromDate:    mustSpeechSearchDate(t, "2024-01-02"),
			UntilDate:   mustSpeechSearchDate(t, "2024-12-31"),
			Limit:       &limit,
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-062: request を作成できません: %v", err)
	}

	got, err := buildSpeechSearchHTTPRequest(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-064: HTTP request を作成できません: %v", err)
	}
	wantRawQuery := "any=%E6%B0%B8%E4%BD%8F%20%25%20%E8%A8%B1%E5%8F%AF" +
		"&speaker=%E5%B1%B1%E7%94%B0%20%E5%A4%AA%E9%83%8E" +
		"&nameOfMeeting=%E4%BA%88%E7%AE%97%20%E5%A7%94%E5%93%A1%E4%BC%9A" +
		"&nameOfHouse=%E4%B8%A1%E9%99%A2%E5%8D%94%E8%AD%B0%E4%BC%9A" +
		"&from=2024-01-02&until=2024-12-31" +
		"&startRecord=1&maximumRecords=30&recordPacking=json"
	if got.Method != http.MethodGet ||
		got.URL.Scheme != "https" ||
		got.URL.Host != "kokkai.ndl.go.jp" ||
		got.URL.Path != "/api/speech" ||
		got.URL.User != nil ||
		got.URL.Fragment != "" ||
		got.URL.Opaque != "" ||
		got.URL.RawQuery != wantRawQuery {
		t.Fatal("SOT-IF-063/064: 固定要求 URL または query parameter が一致しません")
	}
	if got.Header.Get("Accept") != "application/json" ||
		got.Header.Get("Accept-Encoding") != "gzip" ||
		got.Header.Get("Authorization") != "" ||
		got.Header.Get("Cookie") != "" {
		t.Fatal("SOT-IF-063: 固定 header または認証情報の非送信条件が一致しません")
	}
}

func TestBuildSpeechSearchHTTPRequestMapsEveryHouseWithoutMerging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		house parliamentspeechsearch.House
		want  string
	}{
		{parliamentspeechsearch.HouseOfRepresentatives, "衆議院"},
		{parliamentspeechsearch.HouseOfCouncillors, "参議院"},
		{parliamentspeechsearch.BothHouses, "両院"},
		{parliamentspeechsearch.ConferenceOfBothHouses, "両院協議会"},
	}
	for _, test := range tests {
		test := test
		t.Run(string(test.house), func(t *testing.T) {
			t.Parallel()
			request, err := parliamentspeechsearch.NewRequest(
				parliamentspeechsearch.RequestValues{House: test.house},
			)
			if err != nil {
				t.Fatalf("SOT-IF-062: request を作成できません: %v", err)
			}
			got, err := buildSpeechSearchHTTPRequest(context.Background(), request)
			if err != nil {
				t.Fatalf("SOT-IF-064: HTTP request を作成できません: %v", err)
			}
			if got.URL.Query().Get("nameOfHouse") != test.want {
				t.Fatal("SOT-IF-064: nameOfHouse の対応が一致しません")
			}
		})
	}
}

func TestBuildSpeechSearchHTTPRequestUsesDefaultsAndOmitsAbsentConditions(t *testing.T) {
	t.Parallel()

	request, err := parliamentspeechsearch.NewRequest(
		parliamentspeechsearch.RequestValues{Query: "民法"},
	)
	if err != nil {
		t.Fatalf("SOT-IF-062: request を作成できません: %v", err)
	}
	got, err := buildSpeechSearchHTTPRequest(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-064: HTTP request を作成できません: %v", err)
	}
	want := "any=%E6%B0%91%E6%B3%95&startRecord=1&maximumRecords=20&recordPacking=json"
	if got.URL.RawQuery != want {
		t.Fatal("SOT-IF-064: 省略条件または既定 limit の query が一致しません")
	}
}

func TestBuildSpeechSearchHTTPRequestEncodesInputExactlyOnce(t *testing.T) {
	t.Parallel()

	request, err := parliamentspeechsearch.NewRequest(
		parliamentspeechsearch.RequestValues{Query: "A%20B+C/?"},
	)
	if err != nil {
		t.Fatalf("SOT-IF-062: request を作成できません: %v", err)
	}
	got, err := buildSpeechSearchHTTPRequest(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-064: HTTP request を作成できません: %v", err)
	}
	want := "any=A%2520B%2BC%2F%3F&startRecord=1&maximumRecords=20&recordPacking=json"
	if got.URL.RawQuery != want || strings.Contains(got.URL.RawQuery, "+") {
		t.Fatal("SOT-IF-064: query component の一回 encoding が一致しません")
	}
}

func TestBuildSpeechSearchHTTPRequestAppliesFullURLByteBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		speakerLen int
		wantError  bool
	}{
		{name: "2000 byte", speakerLen: 370},
		{name: "2001 byte", speakerLen: 371, wantError: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			query := strings.Repeat("あ", 170)
			speaker := strings.Repeat("a", test.speakerLen)
			request, err := parliamentspeechsearch.NewRequest(
				parliamentspeechsearch.RequestValues{
					Query:   query,
					Speaker: speaker,
				},
			)
			if err != nil {
				t.Fatalf("SOT-IF-062: request を作成できません: %v", err)
			}
			got, buildErr := buildSpeechSearchHTTPRequest(context.Background(), request)
			if test.wantError {
				if buildErr == nil {
					t.Fatal("SOT-IF-063/064: 2001 byte の URL を受理しました")
				}
				if strings.Contains(buildErr.Error(), query) ||
					strings.Contains(buildErr.Error(), speaker) ||
					strings.Contains(buildErr.Error(), "?") {
					t.Fatalf("SOT-IF-063: error が検索条件または query を含みます: %v", buildErr)
				}
				return
			}
			if buildErr != nil {
				t.Fatalf("SOT-IF-063/064: 2000 byte の URL を拒否しました: %v", buildErr)
			}
			if len(got.URL.String()) != 2000 {
				t.Fatalf("SOT-IF-063: URL byte 数 = %d", len(got.URL.String()))
			}
		})
	}
}

func TestBuildSpeechSearchHTTPRequestRejectsNilContext(t *testing.T) {
	t.Parallel()

	request, err := parliamentspeechsearch.NewRequest(
		parliamentspeechsearch.RequestValues{Query: "民法"},
	)
	if err != nil {
		t.Fatalf("SOT-IF-062: request を作成できません: %v", err)
	}
	if _, err := buildSpeechSearchHTTPRequest(nil, request); err == nil {
		t.Fatal("SOT-IF-063: nil context を受理しました")
	}
}

func mustSpeechSearchDate(t *testing.T, value string) *model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("日付 %q を作成できません: %v", value, err)
	}
	return &date
}
