package lawv2

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestLawDocumentAdapterReadsExactRevision(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	var captured *http.Request
	adapter := newTestLawDocumentAdapter(
		t,
		now,
		make(chan struct{}, 1),
		doerFunc(func(request *http.Request) (*http.Response, error) {
			captured = request.Clone(request.Context())
			return response(
				http.StatusOK,
				string(readLawDocumentFixture(t)),
				map[string]string{"Content-Type": "application/xml"},
			), nil
		}),
	)
	request := newLawDocumentReadRequest(
		t,
		"322CO0000000016",
		"322CO0000000016_20240401_506CO0000000161",
		nil,
	)

	result, err := adapter.Read(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-011/024: Read() のエラー = %v", err)
	}
	if captured == nil ||
		!strings.HasSuffix(
			captured.URL.Path,
			"/322CO0000000016_20240401_506CO0000000161",
		) ||
		captured.URL.Query().Get("asof") != "" {
		t.Fatalf("SOT-IF-011: request = %#v", captured)
	}
	assertMappedLawDocument(t, result, now, false)
}

func TestLawDocumentAdapterReadsRevisionAtAsOf(t *testing.T) {
	t.Parallel()

	asOf := mustDate("2024-04-01")
	var captured *http.Request
	adapter := newTestLawDocumentAdapter(
		t,
		time.Now(),
		make(chan struct{}, 1),
		doerFunc(func(request *http.Request) (*http.Response, error) {
			captured = request.Clone(request.Context())
			return response(
				http.StatusOK,
				string(readLawDocumentFixture(t)),
				map[string]string{"Content-Type": "application/xml"},
			), nil
		}),
	)
	request := newLawDocumentReadRequest(
		t,
		"322CO0000000016",
		"",
		&asOf,
	)

	result, err := adapter.Read(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-011/024: Read() のエラー = %v", err)
	}
	if captured == nil ||
		!strings.HasSuffix(captured.URL.Path, "/322CO0000000016") ||
		captured.URL.Query().Get("asof") != "2024-04-01" {
		t.Fatalf("SOT-IF-011: request = %#v", captured)
	}
	assertMappedLawDocument(t, result, time.Time{}, true)
}

func TestLawDocumentAdapterRejectsProviderRangeBeforeHTTP(t *testing.T) {
	t.Parallel()

	attempts := 0
	adapter := newTestLawDocumentAdapter(
		t,
		time.Now(),
		make(chan struct{}, 1),
		doerFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.New("呼び出してはならない HTTP")
		}),
	)
	asOf := mustDate("2017-03-31")
	request := newLawDocumentReadRequest(
		t,
		"322CO0000000016",
		"",
		&asOf,
	)

	_, err := adapter.Read(context.Background(), request)
	assertLawDocumentSourceError(
		t,
		err,
		model.SourceErrorCodeUnsupportedQuery,
	)
	if attempts != 0 {
		t.Fatalf("SOT-IF-004/024: HTTP attempts = %d", attempts)
	}
}

func TestLawDocumentAdapterRejectsResponseIdentityMismatch(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		old string
		new string
	}{
		"lawId": {
			old: "<law_id>322CO0000000016</law_id>",
			new: "<law_id>別の法令</law_id>",
		},
		"revisionId": {
			old: "<law_revision_id>322CO0000000016_20240401_506CO0000000161</law_revision_id>",
			new: "<law_revision_id>322CO0000000016_20240501_506CO0000000162</law_revision_id>",
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body := strings.Replace(
				string(readLawDocumentFixture(t)),
				test.old,
				test.new,
				1,
			)
			adapter := newTestLawDocumentAdapter(
				t,
				time.Now(),
				make(chan struct{}, 1),
				doerFunc(func(*http.Request) (*http.Response, error) {
					return response(
						http.StatusOK,
						body,
						map[string]string{
							"Content-Type": "application/xml",
						},
					), nil
				}),
			)
			request := newLawDocumentReadRequest(
				t,
				"322CO0000000016",
				"322CO0000000016_20240401_506CO0000000161",
				nil,
			)
			_, err := adapter.Read(context.Background(), request)
			assertLawDocumentSourceError(
				t,
				err,
				model.SourceErrorCodeInvalidSourceResponse,
			)
		})
	}
}

func TestLawDocumentAdapterUsesSharedBusyGate(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	attempts := 0
	adapter := newTestLawDocumentAdapter(
		t,
		time.Now(),
		gate,
		doerFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.New("呼び出してはならない HTTP")
		}),
	)
	request := newLawDocumentReadRequest(
		t,
		"322CO0000000016",
		"",
		nil,
	)

	_, err := adapter.Read(context.Background(), request)
	assertLawDocumentSourceError(t, err, model.SourceErrorCodeSourceBusy)
	if attempts != 0 {
		t.Fatalf("SOT-IF-004/017: HTTP attempts = %d", attempts)
	}
}

func TestProductionLawDocumentAdapterCanBeConstructed(t *testing.T) {
	t.Parallel()

	adapter, err := NewLawDocumentAdapter()
	if err != nil || adapter == nil {
		t.Fatalf("NewLawDocumentAdapter() = %#v, %v", adapter, err)
	}
	if _, err := newLawDocumentAdapter(
		lawDocumentAdapterDependencies{},
	); err == nil {
		t.Fatal("依存関係がない adapter を受理した")
	}
}

func newTestLawDocumentAdapter(
	t *testing.T,
	now time.Time,
	gate chan struct{},
	doer httpDoer,
) *LawDocumentAdapter {
	t.Helper()

	client := mustTestClient(t, clientDependencies{
		doer: doer,
		now: func() time.Time {
			if now.IsZero() {
				return time.Now()
			}
			return now
		},
		sleep: func(context.Context, time.Duration) error {
			return nil
		},
	})
	adapter, err := newLawDocumentAdapter(lawDocumentAdapterDependencies{
		client: client,
		gate:   gate,
	})
	if err != nil {
		t.Fatalf("newLawDocumentAdapter() のエラー = %v", err)
	}
	return adapter
}

func newLawDocumentReadRequest(
	t *testing.T,
	resourceID string,
	versionID string,
	asOf *model.Date,
) lawdocumentread.Request {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     providerID,
		ResourceType: "law",
		ResourceID:   resourceID,
		VersionID:    versionID,
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できない: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できない: %v", err)
	}
	request, err := lawdocumentread.NewRequest(
		lawdocumentread.RequestValues{
			Resource: ref,
			AsOf:     asOf,
		},
	)
	if err != nil {
		t.Fatalf("lawdocumentread.NewRequest() のエラー = %v", err)
	}
	return request
}

func assertMappedLawDocument(
	t *testing.T,
	result model.SourcedResource[model.LawDocumentRepresentation],
	retrievedAt time.Time,
	wantAsOf bool,
) {
	t.Helper()

	ref := result.Ref()
	versionID, hasVersion := ref.Key().VersionID()
	if ref.ProviderID() != providerID ||
		ref.Key().SourceID() != providerID ||
		ref.Key().ResourceType() != "law" ||
		ref.Key().ResourceID() != "322CO0000000016" ||
		!hasVersion ||
		versionID != "322CO0000000016_20240401_506CO0000000161" {
		t.Fatalf("SOT-IF-011/024: ref = %#v", ref)
	}
	data := result.Data()
	if data.Format() != model.LawDocumentFormatXML ||
		data.Law().LawID() != ref.Key().ResourceID() ||
		data.Law().RevisionID() != versionID ||
		!strings.HasPrefix(data.Content(), "<Law ") ||
		data.Citation().URL() !=
			"https://laws.e-gov.go.jp/law/322CO0000000016/20240401_506CO0000000161" {
		t.Fatalf("SOT-IF-011/024: data = %#v", data)
	}
	_, hasAsOf := data.AsOf()
	if hasAsOf != wantAsOf {
		t.Fatalf("SOT-MODEL-017: asOf の有無 = %t", hasAsOf)
	}
	provenance := result.Provenance()
	if len(provenance) != 1 ||
		provenance[0].MediaType() != "application/xml" ||
		provenance[0].Transformation() !=
			model.ProvenanceTransformationExtracted {
		t.Fatalf("SOT-IF-011/015: provenance = %#v", provenance)
	}
	methodID, exists := provenance[0].MethodID()
	if !exists || methodID != "SOT-IF-011" {
		t.Fatalf("SOT-IF-011: methodId = %q, %t", methodID, exists)
	}
	if !retrievedAt.IsZero() &&
		!provenance[0].RetrievedAt().Equal(retrievedAt) {
		t.Fatalf("SOT-IF-015: retrievedAt = %v", provenance[0].RetrievedAt())
	}
}
