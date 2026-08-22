package lawv2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestParseLawSearchResponseAndMapItems(t *testing.T) {
	t.Parallel()

	response, nextOffset, err := parseLawSearchResponse(
		context.Background(),
		readLawSearchFixture(t, "law-search-normal.json"),
		1,
		0,
	)
	if err != nil {
		t.Fatalf("SOT-IF-054: parseLawSearchResponse() のエラー = %v", err)
	}
	if response.totalCount != 2 || response.count != 1 || len(response.laws) != 1 {
		t.Fatalf("SOT-IF-054: response = %#v", response)
	}
	if nextOffset == nil || *nextOffset != 1 {
		t.Fatalf("SOT-IF-054: nextOffset = %v", nextOffset)
	}

	retrievedAt := time.Date(2026, 7, 26, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	items, err := mapLawSearchItems(response, retrievedAt)
	if err != nil {
		t.Fatalf("SOT-IF-054/SOT-IF-015: mapLawSearchItems() のエラー = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("SOT-IF-022: items の長さ = %d", len(items))
	}
	assertMappedLawSearchItem(t, items[0], retrievedAt)
}

func TestParseLawSearchResponseAcceptsEmptyResult(t *testing.T) {
	t.Parallel()

	response, nextOffset, err := parseLawSearchResponse(
		context.Background(),
		readLawSearchFixture(t, "law-search-empty.json"),
		20,
		0,
	)
	if err != nil {
		t.Fatalf("SOT-IF-054/SOT-IF-022: 空結果のエラー = %v", err)
	}
	if response.totalCount != 0 || response.count != 0 || len(response.laws) != 0 {
		t.Fatalf("SOT-IF-022: 空結果 = %#v", response)
	}
	if nextOffset != nil {
		t.Fatalf("SOT-IF-054: 空結果の nextOffset = %d", *nextOffset)
	}
}

func TestParseLawSearchResponseDerivesMissingNextOffset(t *testing.T) {
	t.Parallel()

	_, nextOffset, err := parseLawSearchResponse(
		context.Background(),
		readLawSearchFixture(t, "law-search-page.json"),
		1,
		0,
	)
	if err != nil {
		t.Fatalf("SOT-IF-054: next_offset 導出のエラー = %v", err)
	}
	if nextOffset == nil || *nextOffset != 1 {
		t.Fatalf("SOT-IF-054: 導出した nextOffset = %v", nextOffset)
	}
}

func TestParseLawSearchResponseRejectsPageInvariantViolations(t *testing.T) {
	t.Parallel()

	validLaw := `{
		"law_info":{"law_id":"322CO0000000016"},
		"revision_info":{
			"law_revision_id":"322CO0000000016_20240401_506CO0000000161",
			"law_title":"地方自治法施行令"
		}
	}`
	tests := []struct {
		name   string
		body   string
		limit  int
		offset int
	}{
		{
			name:  "明示 null なのに残件がある",
			body:  fmt.Sprintf(`{"total_count":2,"count":1,"next_offset":null,"laws":[%s]}`, validLaw),
			limit: 1,
		},
		{
			name:  "count と配列長が異なる",
			body:  `{"total_count":1,"count":1,"next_offset":null,"laws":[]}`,
			limit: 1,
		},
		{
			name:  "count が limit を超える",
			body:  fmt.Sprintf(`{"total_count":1,"count":1,"next_offset":null,"laws":[%s]}`, validLaw),
			limit: 0,
		},
		{
			name:  "next_offset が offset+count と異なる",
			body:  fmt.Sprintf(`{"total_count":3,"count":1,"next_offset":2,"laws":[%s]}`, validLaw),
			limit: 1,
		},
		{
			name:  "残件があるのに前進しない",
			body:  `{"total_count":1,"count":0,"laws":[]}`,
			limit: 1,
		},
		{
			name:   "next_offset が int32 上限を超える",
			body:   fmt.Sprintf(`{"total_count":2147483648,"count":1,"next_offset":2147483648,"laws":[%s]}`, validLaw),
			limit:  1,
			offset: 2147483647,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := parseLawSearchResponse(
				context.Background(),
				[]byte(test.body),
				test.limit,
				test.offset,
			)
			assertLawSearchSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
		})
	}
}

func TestParseLawSearchResponseClassifiesStructuralError(t *testing.T) {
	t.Parallel()

	_, _, err := parseLawSearchResponse(
		context.Background(),
		readLawSearchFixture(t, "law-search-structural-error.json"),
		1,
		0,
	)
	assertLawSearchSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)

	_, _, err = parseLawSearchResponse(
		context.Background(),
		[]byte(`{"count":0,"laws":[]}`),
		1,
		0,
	)
	assertLawSearchSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)

	tests := []struct {
		name string
		body string
		code model.SourceErrorCode
	}{
		{
			name: "malformed JSON",
			body: `{"total_count":0`,
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "root type",
			body: `[]`,
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "laws missing",
			body: `{"total_count":0,"count":0}`,
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "laws null",
			body: `{"total_count":0,"count":0,"laws":null}`,
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "duplicate key",
			body: `{"total_count":0,"total_count":0,"count":0,"laws":[]}`,
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "required item value missing",
			body: `{"total_count":1,"count":1,"laws":[{"law_info":{},"revision_info":{}}]}`,
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "optional date type changed",
			body: `{"total_count":1,"count":1,"laws":[{"law_info":{"law_id":"322CO0000000016","promulgation_date":123},"revision_info":{"law_revision_id":"322CO0000000016_20240401_506CO0000000161","law_title":"地方自治法施行令"}}]}`,
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "optional null is omitted",
			body: `{"total_count":1,"count":1,"laws":[{"law_info":{"law_id":"322CO0000000016","law_num":null,"promulgation_date":null},"revision_info":{"law_revision_id":"322CO0000000016_20240401_506CO0000000161","law_title":"地方自治法施行令","amendment_enforcement_date":null}}]}`,
			code: "",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response, _, parseErr := parseLawSearchResponse(
				context.Background(),
				[]byte(test.body),
				1,
				0,
			)
			if test.code == "" {
				if parseErr != nil {
					t.Fatalf("SOT-IF-054: optional null を拒否しました: %v", parseErr)
				}
				if len(response.laws) != 1 ||
					response.laws[0].lawNumber != "" ||
					response.laws[0].promulgationDate != "" ||
					response.laws[0].revisionEffectiveDate != "" {
					t.Fatalf("SOT-IF-054: optional null の省略結果 = %#v", response.laws)
				}
				return
			}
			assertLawSearchSourceError(t, parseErr, test.code)
		})
	}
}

func TestParseLawSearchResponseAppliesJSONBudgets(t *testing.T) {
	t.Parallel()

	t.Run("parser input byte 上限", func(t *testing.T) {
		t.Parallel()

		_, _, err := parseLawSearchResponse(
			context.Background(),
			bytes.Repeat([]byte{' '}, lawSearchParserInputBytes+1),
			1,
			0,
		)
		assertLawSearchSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
	})

	t.Run("JSON value 数", func(t *testing.T) {
		t.Parallel()

		var builder strings.Builder
		builder.WriteString(`{"padding":[`)
		for index := 0; index < lawSearchJSONValues; index++ {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString("null")
		}
		builder.WriteString(`],"total_count":0,"count":0,"laws":[]}`)
		_, _, err := parseLawSearchResponse(
			context.Background(),
			[]byte(builder.String()),
			1,
			0,
		)
		assertLawSearchSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
	})

	t.Run("JSON depth", func(t *testing.T) {
		t.Parallel()

		body := `{"padding":` +
			strings.Repeat("[", lawSearchJSONDepth) +
			"null" +
			strings.Repeat("]", lawSearchJSONDepth) +
			`,"total_count":0,"count":0,"laws":[]}`
		_, _, err := parseLawSearchResponse(
			context.Background(),
			[]byte(body),
			1,
			0,
		)
		assertLawSearchSourceError(t, err, model.SourceErrorCodeUnsafeSourceContent)
	})
}

func TestParseLawSearchResponsePropagatesCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := parseLawSearchResponse(
		ctx,
		readLawSearchFixture(t, "law-search-normal.json"),
		1,
		0,
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-IF-015/SOT-ENG-010: error = %v", err)
	}
}

func TestMapLawSearchItemsRejectsRevisionWithDifferentLawPrefix(t *testing.T) {
	t.Parallel()

	response := lawSearchResponse{
		totalCount: 1,
		count:      1,
		laws: []lawSearchLaw{
			{
				lawID:      "322CO0000000016",
				revisionID: "別の法令_20240401_506CO0000000161",
				title:      "地方自治法施行令",
			},
		},
	}
	_, err := mapLawSearchItems(response, time.Now())
	assertLawSearchSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
}

func TestMapLawSearchItemsRejectsInvalidMappedValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		law         lawSearchLaw
		retrievedAt time.Time
	}{
		{
			name: "公布日",
			law: lawSearchLaw{
				lawID:            "322CO0000000016",
				revisionID:       "322CO0000000016_20240401_506CO0000000161",
				title:            "地方自治法施行令",
				promulgationDate: "1947-02-30",
			},
			retrievedAt: time.Now(),
		},
		{
			name: "リビジョン施行日",
			law: lawSearchLaw{
				lawID:                 "322CO0000000016",
				revisionID:            "322CO0000000016_20240401_506CO0000000161",
				title:                 "地方自治法施行令",
				revisionEffectiveDate: "2024-02-30",
			},
			retrievedAt: time.Now(),
		},
		{
			name: "取得時刻",
			law: lawSearchLaw{
				lawID:      "322CO0000000016",
				revisionID: "322CO0000000016_20240401_506CO0000000161",
				title:      "地方自治法施行令",
			},
		},
		{
			name: "URL に安全でない path",
			law: lawSearchLaw{
				lawID:      "322CO/0000000016",
				revisionID: "322CO/0000000016_20240401_506CO0000000161",
				title:      "地方自治法施行令",
			},
			retrievedAt: time.Now(),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := mapLawSearchItems(lawSearchResponse{
				totalCount: 1,
				count:      1,
				laws:       []lawSearchLaw{test.law},
			}, test.retrievedAt)
			assertLawSearchSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
		})
	}
}

func assertMappedLawSearchItem(
	t *testing.T,
	item model.SourcedResource[model.LawSummary],
	retrievedAt time.Time,
) {
	t.Helper()

	ref := item.Ref()
	if ref.ProviderID() != providerID ||
		ref.Key().SourceID() != providerID ||
		ref.Key().ResourceType() != lawSearchResourceType ||
		ref.Key().ResourceID() != "322CO0000000016" {
		t.Fatalf("SOT-IF-054/SOT-IF-022: ref = %#v", ref)
	}
	versionID, exists := ref.Key().VersionID()
	if !exists || versionID != "322CO0000000016_20240401_506CO0000000161" {
		t.Fatalf("SOT-IF-054: versionId = %q, %t", versionID, exists)
	}
	provenance := item.Provenance()
	if len(provenance) != 1 ||
		provenance[0].URL() != "https://laws.e-gov.go.jp/law/322CO0000000016/20240401_506CO0000000161" ||
		provenance[0].MediaType() != lawSearchMediaType ||
		provenance[0].Transformation() != model.ProvenanceTransformationNormalized ||
		!provenance[0].RetrievedAt().Equal(retrievedAt) {
		t.Fatalf("SOT-IF-054/SOT-IF-015: provenance = %#v", provenance)
	}
	methodID, exists := provenance[0].MethodID()
	if !exists || methodID != lawSearchMappingMethod {
		t.Fatalf("SOT-IF-054: methodId = %q, %t", methodID, exists)
	}

	encoded, err := json.Marshal(item.Data())
	if err != nil {
		t.Fatalf("SOT-MODEL-001: LawSummary JSON のエラー = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("SOT-MODEL-001: LawSummary JSON 復元のエラー = %v", err)
	}
	if got["lawId"] != "322CO0000000016" ||
		got["revisionId"] != "322CO0000000016_20240401_506CO0000000161" ||
		got["title"] != "地方自治法施行令" ||
		got["lawNumber"] != "昭和二十二年政令第十六号" ||
		got["promulgationDate"] != "1947-05-03" ||
		got["revisionEffectiveDate"] != "2024-04-01" {
		t.Fatalf("SOT-IF-054: LawSummary = %s", encoded)
	}
}

func assertLawSearchSourceError(
	t *testing.T,
	err error,
	want model.SourceErrorCode,
) {
	t.Helper()
	if err == nil {
		t.Fatalf("SOT-IF-054: error = nil, want %q", want)
	}
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("SOT-IF-054: error type = %T, want model.SourceError", err)
	}
	if sourceError.Code() != want {
		t.Fatalf("SOT-IF-054: error code = %q, want %q", sourceError.Code(), want)
	}
}

func readLawSearchFixture(t *testing.T, name string) []byte {
	t.Helper()
	var (
		body []byte
		err  error
	)
	switch name {
	case "law-search-normal.json":
		body, err = os.ReadFile("fixtures/law-search-normal.json")
	case "law-search-empty.json":
		body, err = os.ReadFile("fixtures/law-search-empty.json")
	case "law-search-page.json":
		body, err = os.ReadFile("fixtures/law-search-page.json")
	case "law-search-structural-error.json":
		body, err = os.ReadFile("fixtures/law-search-structural-error.json")
	default:
		t.Fatalf("未定義の fixture 名です: %q", name)
	}
	if err != nil {
		t.Fatalf("fixture %q を読み込めません: %v", name, err)
	}
	return body
}
