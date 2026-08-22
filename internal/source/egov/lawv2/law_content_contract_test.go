package lawv2

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchlawcontent"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestEGovKeywordRuntimeResponseClassification(t *testing.T) {
	t.Parallel()
	t.Run("egov-keyword-runtime-response-classification", func(t *testing.T) {
		t.Parallel()

		bodies := []string{
			`{"sentence_count":0,"items":[]}`,
			`{"total_count":0,"sentence_count":0,"items":null}`,
			`{"total_count":1,"sentence_count":1,"items":[{"law_info":{},"revision_info":{},"sentences":null}]}`,
			`{"total_count":0,"sentence_count":0,"items":[]} {}`,
			`{"error":{"message":"外部本文を公開しない"}}`,
		}
		for _, body := range bodies {
			_, _, err := parseLawContentSearchResponse(
				context.Background(),
				[]byte(body),
				1,
				0,
			)
			assertLawContentSourceError(
				t,
				err,
				model.SourceErrorCodeInvalidSourceResponse,
			)
		}

		emptyLawNumber := `{
			"total_count":1,
			"sentence_count":1,
			"next_offset":null,
			"items":[{
				"law_info":{"law_id":"322CO0000000016","law_num":""},
				"revision_info":{
					"law_revision_id":"322CO0000000016_20240401_506CO0000000161",
					"law_title":"地方自治法施行令"
				},
				"sentences":[{"position":"MainProvision/Article[1]","text":"本文"}]
			}]
		}`
		_, _, err := parseLawContentSearchResponse(
			context.Background(),
			[]byte(emptyLawNumber),
			1,
			0,
		)
		assertLawContentSourceError(
			t,
			err,
			model.SourceErrorCodeInvalidSourceResponse,
		)

		optionalNull := `{
			"total_count":1,
			"sentence_count":1,
			"next_offset":null,
			"items":[{
				"law_info":{
					"law_id":"322CO0000000016",
					"law_num":null,
					"promulgation_date":null
				},
				"revision_info":{
					"law_revision_id":"322CO0000000016_20240401_506CO0000000161",
					"law_title":"地方自治法施行令",
					"amendment_enforcement_date":null
				},
				"sentences":[{"position":"MainProvision/Article[1]","text":"本文"}]
			}]
		}`
		parsed, _, err := parseLawContentSearchResponse(
			context.Background(),
			[]byte(optionalNull),
			1,
			0,
		)
		if err != nil {
			t.Fatalf("省略可能な null を拒否しました: %v", err)
		}
		mapped, err := mapLawContentItems(parsed, time.Now())
		if err != nil || len(mapped) != 1 {
			t.Fatalf("省略可能な null の mapping = %#v, %v", mapped, err)
		}
		law := mapped[0].Data().Law()
		if _, exists := law.LawNumber(); exists {
			t.Fatal("null の lawNumber を保持しました")
		}
		if _, exists := law.PromulgationDate(); exists {
			t.Fatal("null の promulgationDate を保持しました")
		}
		if _, exists := law.RevisionEffectiveDate(); exists {
			t.Fatal("null の revisionEffectiveDate を保持しました")
		}

		transportCases := []struct {
			name    string
			body    string
			headers map[string]string
		}{
			{
				name: "非 JSON media type",
				body: `{"total_count":0,"sentence_count":0,"items":[]}`,
				headers: map[string]string{
					"Content-Type": "text/html",
				},
			},
			{
				name: "壊れた gzip",
				body: "gzip ではない本文",
				headers: map[string]string{
					"Content-Type":     "application/json",
					"Content-Encoding": "gzip",
				},
			},
		}
		for _, transportCase := range transportCases {
			transportCase := transportCase
			t.Run(transportCase.name, func(t *testing.T) {
				t.Parallel()

				client := mustTestClient(t, clientDependencies{
					doer: doerFunc(func(*http.Request) (*http.Response, error) {
						return response(
							http.StatusOK,
							transportCase.body,
							transportCase.headers,
						), nil
					}),
					now:   time.Now,
					sleep: sleepWithContext,
				})
				_, fetchErr := client.fetchLawContent(
					context.Background(),
					lawContentSearchRequest{
						keyword: "民法",
						asOf:    mustDate("2026-07-30"),
						limit:   1,
					},
				)
				assertLawContentSourceError(
					t,
					fetchErr,
					model.SourceErrorCodeInvalidSourceResponse,
				)
			})
		}
	})
}

func TestEGovKeywordContractChangeSeparation(t *testing.T) {
	t.Parallel()
	t.Run("egov-keyword-contract-change-separation", func(t *testing.T) {
		t.Parallel()

		recorded := fixture(t, "fixtures/law-content-contract.json")
		if err := verifyRecordedLawContentContract(recorded); err != nil {
			t.Fatalf("保存済み公式契約を拒否しました: %v", err)
		}
		changed := bytes.Replace(
			recorded,
			[]byte(`"minimumItems": 1`),
			[]byte(`"minimumItems": 0`),
			1,
		)
		if bytes.Equal(changed, recorded) {
			t.Fatal("公式契約 fixture を変更できませんでした")
		}
		assertLawContentSourceError(
			t,
			verifyRecordedLawContentContract(changed),
			model.SourceErrorCodeSourceContractChanged,
		)

		_, _, runtimeErr := parseLawContentSearchResponse(
			context.Background(),
			[]byte(`{"total_count":0,"sentence_count":0,"items":null}`),
			1,
			0,
		)
		assertLawContentSourceError(
			t,
			runtimeErr,
			model.SourceErrorCodeInvalidSourceResponse,
		)
	})
}

func TestEGovKeywordFacadeCapabilityParserIdentity(t *testing.T) {
	t.Parallel()
	t.Run("egov-keyword-facade-capability-parser-identity", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
		body := string(fixture(t, "fixtures/law-content-normal.json"))
		newDoer := func(contentType string, responseBody string) httpDoer {
			return doerFunc(func(*http.Request) (*http.Response, error) {
				return response(
					http.StatusOK,
					responseBody,
					map[string]string{"Content-Type": contentType},
				), nil
			})
		}

		facade := newTestSearchLawContentFacade(
			t,
			make(chan struct{}, 1),
			newDoer("application/json", body),
		)
		adapter := newTestLawContentAdapter(
			t,
			now,
			make(chan struct{}, 1),
			newDoer("application/json", body),
		)
		limit := 2
		offset := 0
		facadeResult, err := facade.Search(
			context.Background(),
			mustPublicContentSearchRequest(t, searchlawcontent.RequestValues{
				Query:  "自治",
				Limit:  &limit,
				Offset: &offset,
			}),
		)
		if err != nil {
			t.Fatalf("公開 facade の Search() エラー = %v", err)
		}
		capabilityPage, err := adapter.Search(
			context.Background(),
			mustLawContentSearchRequest(t, lawcontentsearch.RequestValues{
				AllTerms: []string{"自治"},
				Limit:    &limit,
			}),
		)
		if err != nil {
			t.Fatalf("内部 capability の Search() エラー = %v", err)
		}
		facadeItems := facadeResult.Items()
		capabilityItems := capabilityPage.Items()
		if len(facadeItems) != 2 || len(capabilityItems) != len(facadeItems) {
			t.Fatalf(
				"展開件数が一致しません: facade=%d capability=%d",
				len(facadeItems),
				len(capabilityItems),
			)
		}
		for index := range facadeItems {
			facadeJSON, marshalErr := json.Marshal(facadeItems[index])
			if marshalErr != nil {
				t.Fatalf("facade item の JSON 化エラー = %v", marshalErr)
			}
			capabilityJSON, marshalErr := json.Marshal(capabilityItems[index].Data())
			if marshalErr != nil {
				t.Fatalf("capability item の JSON 化エラー = %v", marshalErr)
			}
			if !bytes.Equal(facadeJSON, capabilityJSON) {
				t.Fatalf(
					"mapping または順序が一致しません: facade=%s capability=%s",
					facadeJSON,
					capabilityJSON,
				)
			}
			if capabilityItems[index].Data().Location() != facadeItems[index].Location() {
				t.Fatalf("sentence identity が一致しません: index=%d", index)
			}
		}

		classifications := []struct {
			name        string
			body        string
			contentType string
		}{
			{
				name:        "runtime 構造違反",
				body:        `{"total_count":0,"sentence_count":0,"items":null}`,
				contentType: "application/json",
			},
			{
				name:        "非 JSON media type",
				body:        `{"total_count":0,"sentence_count":0,"items":[]}`,
				contentType: "text/html",
			},
		}
		for _, classification := range classifications {
			classification := classification
			t.Run(classification.name, func(t *testing.T) {
				t.Parallel()

				invalidFacade := newTestSearchLawContentFacade(
					t,
					make(chan struct{}, 1),
					newDoer(classification.contentType, classification.body),
				)
				invalidAdapter := newTestLawContentAdapter(
					t,
					now,
					make(chan struct{}, 1),
					newDoer(classification.contentType, classification.body),
				)
				_, facadeErr := invalidFacade.Search(
					context.Background(),
					mustPublicContentSearchRequest(
						t,
						searchlawcontent.RequestValues{Query: "民法"},
					),
				)
				_, capabilityErr := invalidAdapter.Search(
					context.Background(),
					mustLawContentSearchRequest(
						t,
						lawcontentsearch.RequestValues{AllTerms: []string{"民法"}},
					),
				)
				facadeCode := sourceErrorCode(t, facadeErr)
				capabilityCode := sourceErrorCode(t, capabilityErr)
				if facadeCode != model.SourceErrorCodeInvalidSourceResponse ||
					capabilityCode != facadeCode {
					t.Fatalf(
						"parser 判定が一致しません: facade=%v capability=%v",
						facadeErr,
						capabilityErr,
					)
				}
			})
		}
	})
}
