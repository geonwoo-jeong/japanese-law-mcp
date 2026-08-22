package lawv2

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/continuation"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/requestpacing"
)

func TestProductionEGovBindingsShareHTTPConcurrencyGroup(t *testing.T) {
	t.Parallel()

	manager, err := continuation.NewManager()
	if err != nil {
		t.Fatalf("continuation manager を作成できません: %v", err)
	}
	bindings, err := NewProviderBindings(manager)
	if err != nil {
		t.Fatalf("provider bindings を作成できません: %v", err)
	}
	search, searchOK := bindings.LawSearch.(*LawSearchAdapter)
	content, contentOK := bindings.LawContentSearch.(*LawContentSearchAdapter)
	document, documentOK := bindings.LawDocumentRead.(*LawDocumentAdapter)
	article, articleOK := bindings.LawArticleRead.(*LawArticleAdapter)
	if !searchOK || !contentOK || !documentOK || !articleOK {
		t.Fatalf("e-Gov binding の型が一致しません: %#v", bindings)
	}
	gates := []chan struct{}{
		search.dependencies.gate,
		content.dependencies.gate,
		document.dependencies.gate,
		article.dependencies.gate,
	}
	for index := 1; index < len(gates); index++ {
		if gates[index] != gates[0] {
			t.Fatalf("binding %d が別の egov-http gate を使用しています", index)
		}
	}
}

func TestEGovOperationsShareRetryPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(context.Context, lawClient) error
	}{
		{
			name: "GET /laws",
			call: func(ctx context.Context, client lawClient) error {
				_, err := client.fetch(ctx, lawSearchRequest{
					query: "民法",
					asOf:  mustDate("2026-07-26"),
					limit: 20,
				})
				return err
			},
		},
		{
			name: "GET /keyword",
			call: func(ctx context.Context, client lawClient) error {
				_, err := client.fetchLawContent(ctx, lawContentSearchRequest{
					keyword: "民法",
					asOf:    mustDate("2026-07-26"),
					limit:   1,
				})
				return err
			},
		},
		{
			name: "GET /law_data document",
			call: func(ctx context.Context, client lawClient) error {
				_, err := client.fetchLawDocument(ctx, lawDocumentRequest{
					identifier: "322CO0000000016",
				})
				return err
			},
		},
		{
			name: "GET /law_data article",
			call: func(ctx context.Context, client lawClient) error {
				_, err := client.fetchLawArticle(ctx, lawDocumentRequest{
					identifier: "322CO0000000016",
				})
				return err
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			attempts := 0
			delays := make([]time.Duration, 0, maximumRetries)
			client := mustTestClient(t, clientDependencies{
				doer: doerFunc(func(*http.Request) (*http.Response, error) {
					attempts++
					return response(http.StatusServiceUnavailable, "", nil), nil
				}),
				now: time.Now,
				sleep: func(_ context.Context, delay time.Duration) error {
					delays = append(delays, delay)
					return nil
				},
			})
			err := test.call(context.Background(), client)
			assertSourceErrorCode(t, err, model.SourceErrorCodeSourceUnavailable)
			if attempts != maximumRetries+1 {
				t.Fatalf("HTTP attempts = %d", attempts)
			}
			if len(delays) != len(retryBackoffs) {
				t.Fatalf("retry delays = %v", delays)
			}
			for index := range retryBackoffs {
				if delays[index] != retryBackoffs[index] {
					t.Fatalf("retry delays = %v", delays)
				}
			}
		})
	}
}

func TestEGovHTTPConcurrencyGroupRejectsCrossOperation(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{}, 1)
	started := make(chan struct{})
	release := make(chan struct{})
	search := newTestAdapter(
		t,
		time.Now(),
		gate,
		doerFunc(func(*http.Request) (*http.Response, error) {
			close(started)
			<-release
			return response(
				http.StatusOK,
				`{"total_count":0,"count":0,"laws":[]}`,
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
	)
	searchRequest := mustSearchRequest(t, lawsearch.RequestValues{Query: "民法"})
	requestContext := requestpacing.WithScope(context.Background())
	searchDone := make(chan error, 1)
	go func() {
		_, err := search.Search(
			requestContext,
			searchRequest,
		)
		searchDone <- err
	}()
	<-started

	attempts := 0
	unexpectedDoer := doerFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return nil, context.Canceled
	})
	content := newTestLawContentAdapter(t, time.Now(), gate, unexpectedDoer)
	_, contentErr := content.Search(
		requestContext,
		mustLawContentSearchRequest(t, lawcontentsearch.RequestValues{
			AllTerms: []string{"民法"},
		}),
	)
	assertLawContentSourceError(t, contentErr, model.SourceErrorCodeSourceBusy)

	document := newTestLawDocumentAdapter(t, time.Now(), gate, unexpectedDoer)
	_, documentErr := document.Read(
		requestContext,
		newLawDocumentReadRequest(t, "322CO0000000016", "", nil),
	)
	assertLawDocumentSourceError(t, documentErr, model.SourceErrorCodeSourceBusy)

	article := newTestLawArticleAdapter(t, time.Now(), gate, unexpectedDoer)
	_, articleErr := article.Read(
		requestContext,
		newLawArticleReadRequest(
			t,
			providerID,
			providerID,
			"322CO0000000016",
			"",
			nil,
			mustLawArticleLocation(t, model.LawArticleProvisionMain, "1", 0),
		),
	)
	assertLawArticleSourceError(t, articleErr, model.SourceErrorCodeSourceBusy)
	if attempts != 0 {
		t.Fatalf("後続 operation の HTTP attempts = %d", attempts)
	}

	close(release)
	if err := <-searchDone; err != nil {
		t.Fatalf("先行 /laws operation のエラー = %v", err)
	}
}
