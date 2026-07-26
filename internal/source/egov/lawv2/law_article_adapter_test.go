package lawv2

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLawArticleAdapterReadsExactRevision(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	var captured *http.Request
	adapter := newTestLawArticleAdapter(
		t,
		now,
		make(chan struct{}, 1),
		doerFunc(func(request *http.Request) (*http.Response, error) {
			captured = request.Clone(request.Context())
			return response(
				http.StatusOK,
				string(readLawArticleFixture(t)),
				map[string]string{"Content-Type": "application/xml"},
			), nil
		}),
	)
	location := mustLawArticleLocation(t, model.LawArticleProvisionMain, "1", 2)
	request := newLawArticleReadRequest(
		t,
		providerID,
		providerID,
		"322CO0000000016",
		"322CO0000000016_20240401_506CO0000000161",
		nil,
		location,
	)

	result, err := adapter.Read(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-012/025: Read() のエラー = %v", err)
	}
	if captured == nil ||
		!strings.HasSuffix(
			captured.URL.Path,
			"/322CO0000000016_20240401_506CO0000000161",
		) ||
		captured.URL.Query().Get("asof") != "" ||
		captured.URL.Query().Get("response_format") != "xml" ||
		captured.URL.Query().Get("law_full_text_format") != "xml" {
		t.Fatalf("SOT-IF-011/012: request = %#v", captured)
	}
	assertMappedLawArticle(t, result, location, now)
}

func TestLawArticleAdapterReadsAtAsOf(t *testing.T) {
	t.Parallel()

	asOf := mustDate("2024-04-01")
	var captured *http.Request
	adapter := newTestLawArticleAdapter(
		t,
		time.Now(),
		make(chan struct{}, 1),
		doerFunc(func(request *http.Request) (*http.Response, error) {
			captured = request.Clone(request.Context())
			return response(
				http.StatusOK,
				string(readLawArticleFixture(t)),
				map[string]string{"Content-Type": "application/xml"},
			), nil
		}),
	)
	location := mustLawArticleLocation(
		t,
		model.LawArticleProvisionSupplementary,
		"1",
		0,
	)
	request := newLawArticleReadRequest(
		t,
		providerID,
		providerID,
		"322CO0000000016",
		"",
		&asOf,
		location,
	)

	result, err := adapter.Read(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-012/025: Read() のエラー = %v", err)
	}
	if captured == nil ||
		!strings.HasSuffix(captured.URL.Path, "/322CO0000000016") ||
		captured.URL.Query().Get("asof") != "2024-04-01" {
		t.Fatalf("SOT-IF-011/012: request = %#v", captured)
	}
	assertMappedLawArticle(t, result, location, time.Time{})
}

func TestLawArticleAdapterRejectsProviderRangeAndBusyBeforeHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		gate chan struct{}
		asOf *model.Date
		code model.SourceErrorCode
	}{
		{
			name: "対象期間外",
			gate: make(chan struct{}, 1),
			asOf: func() *model.Date {
				date := mustDate("2017-03-31")
				return &date
			}(),
			code: model.SourceErrorCodeUnsupportedQuery,
		},
		{
			name: "共通 gate が使用中",
			gate: func() chan struct{} {
				gate := make(chan struct{}, 1)
				gate <- struct{}{}
				return gate
			}(),
			code: model.SourceErrorCodeSourceBusy,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			attempts := 0
			adapter := newTestLawArticleAdapter(
				t,
				time.Now(),
				test.gate,
				doerFunc(func(*http.Request) (*http.Response, error) {
					attempts++
					return nil, errors.New("呼び出してはならない HTTP")
				}),
			)
			request := newLawArticleReadRequest(
				t,
				providerID,
				providerID,
				"322CO0000000016",
				"",
				test.asOf,
				mustLawArticleLocation(t, model.LawArticleProvisionMain, "1", 0),
			)
			_, err := adapter.Read(context.Background(), request)
			assertLawArticleSourceError(t, err, test.code)
			if attempts != 0 {
				t.Fatalf("SOT-IF-004/012: HTTP attempts = %d", attempts)
			}
		})
	}
}

func TestLawArticleAdapterRejectsProviderAndSourceMismatch(t *testing.T) {
	t.Parallel()

	for name, ids := range map[string][2]string{
		"providerId": {"other-provider", providerID},
		"sourceId":   {providerID, "other-source"},
	} {
		name, ids := name, ids
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			attempts := 0
			adapter := newTestLawArticleAdapter(
				t,
				time.Now(),
				make(chan struct{}, 1),
				doerFunc(func(*http.Request) (*http.Response, error) {
					attempts++
					return nil, errors.New("呼び出してはならない HTTP")
				}),
			)
			request := newLawArticleReadRequest(
				t,
				ids[0],
				ids[1],
				"322CO0000000016",
				"",
				nil,
				mustLawArticleLocation(t, model.LawArticleProvisionMain, "1", 0),
			)
			if _, err := adapter.Read(context.Background(), request); err == nil {
				t.Fatal("SOT-IF-012: provider/source 不一致を受理した")
			}
			if attempts != 0 {
				t.Fatalf("SOT-IF-012: HTTP attempts = %d", attempts)
			}
		})
	}
}

func TestLawArticleAdapterRejectsResponseIdentityMismatch(t *testing.T) {
	t.Parallel()

	tests := map[string][2]string{
		"lawId": {
			"<law_id>322CO0000000016</law_id>",
			"<law_id>別の法令</law_id>",
		},
		"revisionId": {
			"<law_revision_id>322CO0000000016_20240401_506CO0000000161</law_revision_id>",
			"<law_revision_id>322CO0000000016_20240501_506CO0000000162</law_revision_id>",
		},
	}
	for name, replacement := range tests {
		name, replacement := name, replacement
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body := strings.Replace(
				string(readLawArticleFixture(t)),
				replacement[0],
				replacement[1],
				1,
			)
			adapter := newTestLawArticleAdapter(
				t,
				time.Now(),
				make(chan struct{}, 1),
				doerFunc(func(*http.Request) (*http.Response, error) {
					return response(
						http.StatusOK,
						body,
						map[string]string{"Content-Type": "application/xml"},
					), nil
				}),
			)
			request := newLawArticleReadRequest(
				t,
				providerID,
				providerID,
				"322CO0000000016",
				"322CO0000000016_20240401_506CO0000000161",
				nil,
				mustLawArticleLocation(t, model.LawArticleProvisionMain, "1", 0),
			)
			_, err := adapter.Read(context.Background(), request)
			assertLawArticleSourceError(
				t,
				err,
				model.SourceErrorCodeInvalidSourceResponse,
			)
		})
	}
}

func TestProductionLawArticleAdapterCanBeConstructed(t *testing.T) {
	t.Parallel()

	adapter, err := NewLawArticleAdapter()
	if err != nil || adapter == nil {
		t.Fatalf("NewLawArticleAdapter() = %#v, %v", adapter, err)
	}
	if _, err := newLawArticleAdapter(lawArticleAdapterDependencies{}); err == nil {
		t.Fatal("依存関係がない adapter を受理した")
	}
}

func newTestLawArticleAdapter(
	t *testing.T,
	now time.Time,
	gate chan struct{},
	doer httpDoer,
) *LawArticleAdapter {
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
	adapter, err := newLawArticleAdapter(lawArticleAdapterDependencies{
		client: client,
		gate:   gate,
	})
	if err != nil {
		t.Fatalf("newLawArticleAdapter() のエラー = %v", err)
	}
	return adapter
}

func newLawArticleReadRequest(
	t *testing.T,
	requestProviderID string,
	sourceID string,
	resourceID string,
	versionID string,
	asOf *model.Date,
	location model.LawArticleLocation,
) lawarticleread.Request {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: "law",
		ResourceID:   resourceID,
		VersionID:    versionID,
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できない: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: requestProviderID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できない: %v", err)
	}
	request, err := lawarticleread.NewRequest(lawarticleread.RequestValues{
		Resource: ref,
		AsOf:     asOf,
		Location: location,
	})
	if err != nil {
		t.Fatalf("lawarticleread.NewRequest() のエラー = %v", err)
	}
	return request
}

func assertMappedLawArticle(
	t *testing.T,
	result model.SourcedResource[model.LawArticleFragment],
	location model.LawArticleLocation,
	retrievedAt time.Time,
) {
	t.Helper()

	ref := result.Ref()
	versionID, hasVersionID := ref.Key().VersionID()
	if ref.ProviderID() != providerID ||
		ref.Key().SourceID() != providerID ||
		ref.Key().ResourceType() != "law" ||
		ref.Key().ResourceID() != "322CO0000000016" ||
		!hasVersionID ||
		versionID != "322CO0000000016_20240401_506CO0000000161" {
		t.Fatalf("SOT-IF-012/025: ref = %#v", ref)
	}
	data := result.Data()
	citationLocation, hasCitationLocation := data.Citation().Location()
	wantLocation := lawArticleCitationLocation(location)
	if data.Law().LawID() != ref.Key().ResourceID() ||
		data.Law().RevisionID() != versionID ||
		data.Location() != location ||
		data.Format() != model.LawArticleFormatXML ||
		!strings.HasPrefix(data.Content(), "<") ||
		!hasCitationLocation ||
		citationLocation != wantLocation ||
		data.Citation().URL() !=
			"https://laws.e-gov.go.jp/law/322CO0000000016/20240401_506CO0000000161" {
		t.Fatalf("SOT-IF-012/025: data = %#v", data)
	}
	provenance := result.Provenance()
	if len(provenance) != 1 {
		t.Fatalf("SOT-IF-012/015: provenance = %#v", provenance)
	}
	provenanceLocation, hasProvenanceLocation := provenance[0].Location()
	methodID, hasMethodID := provenance[0].MethodID()
	if provenance[0].MediaType() != "application/xml" ||
		provenance[0].Transformation() != model.ProvenanceTransformationExtracted ||
		!hasProvenanceLocation ||
		provenanceLocation != wantLocation ||
		!hasMethodID ||
		methodID != "SOT-IF-012" {
		t.Fatalf("SOT-IF-012/015: provenance = %#v", provenance)
	}
	if !retrievedAt.IsZero() &&
		!provenance[0].RetrievedAt().Equal(retrievedAt) {
		t.Fatalf("SOT-IF-015: retrievedAt = %v", provenance[0].RetrievedAt())
	}
}
