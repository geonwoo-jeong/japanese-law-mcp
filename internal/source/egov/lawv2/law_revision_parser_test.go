package lawv2

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLawRevisionParserRejectsInvalidRuntimeResponses(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"law_info 欠落":    `{"revisions":[]}`,
		"revisions null": `{"law_info":{"law_id":"law-1"},"revisions":null}`,
		"必須 title 欠落": `{
			"law_info":{"law_id":"law-1"},
			"revisions":[{"law_revision_id":"revision-1"}]
		}`,
		"boolean 型不正": `{
			"law_info":{"law_id":"law-1"},
			"revisions":[{
				"law_revision_id":"revision-1","law_title":"試験法",
				"remain_in_force":"false"
			}]
		}`,
		"履歴 ID 重複": `{
			"law_info":{"law_id":"law-1"},
			"revisions":[
				{"law_revision_id":"revision-1","law_title":"試験法"},
				{"law_revision_id":"revision-1","law_title":"試験法"}
			]
		}`,
	}
	for name, body := range tests {
		name, body := name, body
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := parseLawRevisionResponse(context.Background(), []byte(body))
			assertSourceErrorCode(t, err, model.SourceErrorCodeInvalidSourceResponse)
		})
	}
}

func TestLawRevisionMappingRejectsUnknownEnumsAndInvalidDates(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"未知 enum": `{
			"law_info":{"law_id":"law-1"},
			"revisions":[{
				"law_revision_id":"revision-1","law_title":"試験法",
				"amendment_type":"9","mission":"New"
			}]
		}`,
		"日付不正": `{
			"law_info":{"law_id":"law-1","promulgation_date":"2026-02-30"},
			"revisions":[{"law_revision_id":"revision-1","law_title":"試験法"}]
		}`,
		"日時不正": `{
			"law_info":{"law_id":"law-1"},
			"revisions":[{
				"law_revision_id":"revision-1","law_title":"試験法",
				"updated":"2026-08-23"
			}]
		}`,
	}
	for name, body := range tests {
		name, body := name, body
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			response, err := parseLawRevisionResponse(context.Background(), []byte(body))
			if err != nil {
				t.Fatalf("parse error = %v", err)
			}
			_, err = mapLawRevisions(
				response,
				time.Now(),
				"https://laws.e-gov.go.jp/api/2/law_revisions/law-1?response_format=json",
			)
			assertSourceErrorCode(t, err, model.SourceErrorCodeInvalidSourceResponse)
		})
	}
}

func TestLawRevisionParserUsesSpecifiedResourceBudget(t *testing.T) {
	t.Parallel()

	if lawRevisionParserInputBytes != 16*1024*1024 ||
		lawRevisionJSONValues != 200000 ||
		lawRevisionJSONDepth != 32 ||
		lawRevisionParseTimeout != 3*time.Second {
		t.Fatal("law-revisions-json の資源予算が一致しません")
	}
	deep := strings.Repeat("[", lawRevisionJSONDepth+1) +
		"0" + strings.Repeat("]", lawRevisionJSONDepth+1)
	_, err := parseLawRevisionResponse(context.Background(), []byte(deep))
	assertSourceErrorCode(t, err, model.SourceErrorCodeUnsafeSourceContent)
}
