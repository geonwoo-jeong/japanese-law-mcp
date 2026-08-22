package lawv2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchlaws"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestEGovLawsRuntimeResponseClassification(t *testing.T) {
	t.Parallel()
	t.Run("egov-laws-runtime-response-classification", func(t *testing.T) {
		t.Parallel()

		bodies := []string{
			`{"count":0,"laws":[]}`,
			`{"total_count":0,"count":0,"laws":null}`,
			`{"total_count":0,"count":0,"laws":[]} {}`,
			`{"error":{"message":"外部本文を公開しない"}}`,
		}
		for _, body := range bodies {
			_, _, err := parseLawSearchResponse(
				context.Background(),
				[]byte(body),
				1,
				0,
			)
			assertLawSearchSourceError(
				t,
				err,
				model.SourceErrorCodeInvalidSourceResponse,
			)
		}

		client := mustTestClient(t, clientDependencies{
			doer: doerFunc(func(*http.Request) (*http.Response, error) {
				return response(
					http.StatusOK,
					`{"total_count":0,"count":0,"laws":[]}`,
					map[string]string{"Content-Type": "text/html"},
				), nil
			}),
			now:   time.Now,
			sleep: sleepWithContext,
		})
		_, err := client.fetch(context.Background(), lawSearchRequest{
			query: "民法",
			asOf:  mustDate("2026-07-30"),
			limit: 1,
		})
		assertSourceErrorCode(t, err, model.SourceErrorCodeInvalidSourceResponse)
	})
}

func TestEGovLawsContractChangeSeparation(t *testing.T) {
	t.Parallel()
	t.Run("egov-laws-contract-change-separation", func(t *testing.T) {
		t.Parallel()

		recorded := fixture(t, "fixtures/law-search-contract.json")
		if err := verifyRecordedLawSearchContract(recorded); err != nil {
			t.Fatalf("保存済み公式契約を拒否しました: %v", err)
		}
		changed := bytes.Replace(
			recorded,
			[]byte(`"successMediaType": "application/json"`),
			[]byte(`"successMediaType": "text/html"`),
			1,
		)
		if bytes.Equal(changed, recorded) {
			t.Fatal("公式契約 fixture を変更できませんでした")
		}
		assertSourceErrorCode(
			t,
			verifyRecordedLawSearchContract(changed),
			model.SourceErrorCodeSourceContractChanged,
		)

		_, _, runtimeErr := parseLawSearchResponse(
			context.Background(),
			[]byte(`{"total_count":0,"count":0,"laws":null}`),
			1,
			0,
		)
		assertSourceErrorCode(
			t,
			runtimeErr,
			model.SourceErrorCodeInvalidSourceResponse,
		)
	})
}

func TestEGovLawsFacadeCapabilityParserIdentity(t *testing.T) {
	t.Parallel()
	t.Run("egov-laws-facade-capability-parser-identity", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
		body := string(fixture(t, "fixtures/law-search-page.json"))
		newDoer := func() httpDoer {
			return doerFunc(func(*http.Request) (*http.Response, error) {
				return response(
					http.StatusOK,
					body,
					map[string]string{"Content-Type": "application/json"},
				), nil
			})
		}

		facade := newTestSearchLawsFacade(
			t,
			make(chan struct{}, 1),
			newDoer(),
		)
		adapter := newTestAdapter(
			t,
			now,
			make(chan struct{}, 1),
			newDoer(),
		)
		limit := 1
		offset := 0
		facadeResult, err := facade.Search(
			context.Background(),
			mustPublicSearchRequest(t, searchlaws.RequestValues{
				Query:  "地方自治",
				Limit:  &limit,
				Offset: &offset,
			}),
		)
		if err != nil {
			t.Fatalf("公開 facade の Search() エラー = %v", err)
		}
		capabilityPage, err := adapter.Search(
			context.Background(),
			mustSearchRequest(t, lawsearch.RequestValues{
				Query: "地方自治",
				Limit: &limit,
			}),
		)
		if err != nil {
			t.Fatalf("内部 capability の Search() エラー = %v", err)
		}
		facadeItems := facadeResult.Items()
		capabilityItems := capabilityPage.Items()
		if len(facadeItems) != len(capabilityItems) {
			t.Fatalf(
				"item 数が一致しません: facade=%d capability=%d",
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
					"mapping が一致しません: facade=%s capability=%s",
					facadeJSON,
					capabilityJSON,
				)
			}
			key := capabilityItems[index].Ref().Key()
			versionID, exists := key.VersionID()
			if key.ResourceID() != facadeItems[index].LawID() ||
				!exists || versionID != facadeItems[index].RevisionID() {
				t.Fatalf("item identity が一致しません: ref=%#v", key)
			}
		}

		classifications := []struct {
			name        string
			body        string
			contentType string
		}{
			{
				name:        "runtime 構造違反",
				body:        `{"total_count":0,"count":0,"laws":null}`,
				contentType: "application/json",
			},
			{
				name:        "非 JSON media type",
				body:        `{"total_count":0,"count":0,"laws":[]}`,
				contentType: "text/html",
			},
		}
		for _, classification := range classifications {
			classification := classification
			t.Run(classification.name, func(t *testing.T) {
				t.Parallel()

				invalidDoer := func() httpDoer {
					return doerFunc(func(*http.Request) (*http.Response, error) {
						return response(
							http.StatusOK,
							classification.body,
							map[string]string{"Content-Type": classification.contentType},
						), nil
					})
				}
				invalidFacade := newTestSearchLawsFacade(
					t,
					make(chan struct{}, 1),
					invalidDoer(),
				)
				invalidAdapter := newTestAdapter(
					t,
					now,
					make(chan struct{}, 1),
					invalidDoer(),
				)
				_, facadeErr := invalidFacade.Search(
					context.Background(),
					mustPublicSearchRequest(t, searchlaws.RequestValues{Query: "民法"}),
				)
				_, capabilityErr := invalidAdapter.Search(
					context.Background(),
					mustSearchRequest(t, lawsearch.RequestValues{Query: "民法"}),
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

func sourceErrorCode(t *testing.T, err error) model.SourceErrorCode {
	t.Helper()
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("error = %T %v, want model.SourceError", err, err)
	}
	return sourceError.Code()
}
