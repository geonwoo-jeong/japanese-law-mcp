package hanreipdf

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestAdapterReturnsNotFoundOn404(t *testing.T) {
	t.Parallel()

	decision, document := judicialcasecitationextracttestRequest(t)
	request, err := judicialcasecitationextract.NewRequest(
		judicialcasecitationextract.RequestValues{Decision: decision, Document: document},
	)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := newJudicialDecisionCaseCitationExtractAdapter(adapterDependencies{
		doer: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}, nil
		}),
		now:       func() time.Time { return mustRetrievedAt(t) },
		gate:      make(chan struct{}, 1),
		runWorker: func(context.Context, []byte) (workerOutput, error) { return workerOutput{}, errors.New("unused") },
		timeout:   parseTimeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Extract(context.Background(), request); !errors.Is(err, judicialcasecitationextract.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestAdapterRejectsUnsafeDocumentURLBeforeExternalCall(t *testing.T) {
	t.Parallel()

	decision, document := judicialcasecitationextracttestRequestWithURL(
		t,
		"https://www.courts.go.jp/assets/hanrei/../../evil.pdf",
	)
	request, err := judicialcasecitationextract.NewRequest(
		judicialcasecitationextract.RequestValues{Decision: decision, Document: document},
	)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := newJudicialDecisionCaseCitationExtractAdapter(adapterDependencies{
		doer: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("unsafe URL は外部取得前に拒否されるべきです")
			return nil, nil
		}),
		now:       func() time.Time { return mustRetrievedAt(t) },
		gate:      make(chan struct{}, 1),
		runWorker: func(context.Context, []byte) (workerOutput, error) { return workerOutput{}, nil },
		timeout:   parseTimeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Extract(context.Background(), request)
	assertHanreiPDFSourceErrorCode(t, err, model.SourceErrorCodeUnsafeSourceContent)
}

func TestAdapterMapsWorkerDeadlineToSourceTimeout(t *testing.T) {
	t.Parallel()

	request := mustExtractRequest(t)
	adapter, err := newJudicialDecisionCaseCitationExtractAdapter(adapterDependencies{
		doer: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{model.JudicialDocumentMediaTypePDF},
				},
				Body:          io.NopCloser(bytes.NewReader([]byte("%PDF-1.7\n%%EOF"))),
				ContentLength: int64(len([]byte("%PDF-1.7\n%%EOF"))),
				Request:       request,
			}, nil
		}),
		now:  func() time.Time { return time.Date(2026, 8, 26, 12, 1, 0, 0, time.FixedZone("JST", 9*60*60)) },
		gate: make(chan struct{}, 1),
		runWorker: func(ctx context.Context, _ []byte) (workerOutput, error) {
			<-ctx.Done()
			return workerOutput{}, ctx.Err()
		},
		timeout: time.Nanosecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = adapter.Extract(context.Background(), request)
	assertHanreiPDFSourceErrorCode(t, err, model.SourceErrorCodeSourceTimeout)
}

func TestAdapterCancellationWaitsForWorkerAndReleasesGate(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{}, 1)
	workerStarted := make(chan struct{})
	workerFinished := make(chan struct{})
	adapter, err := newJudicialDecisionCaseCitationExtractAdapter(adapterDependencies{
		doer: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			body := []byte("%PDF-fixture")
			return &http.Response{
				StatusCode:    http.StatusOK,
				Header:        http.Header{"Content-Type": []string{model.JudicialDocumentMediaTypePDF}},
				Body:          io.NopCloser(bytes.NewReader(body)),
				ContentLength: int64(len(body)),
				Request:       request,
			}, nil
		}),
		now:  func() time.Time { return mustRetrievedAt(t) },
		gate: gate,
		runWorker: func(ctx context.Context, _ []byte) (workerOutput, error) {
			close(workerStarted)
			<-ctx.Done()
			close(workerFinished)
			return workerOutput{}, ctx.Err()
		},
		timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	request := mustExtractRequest(t)
	result := make(chan error, 1)
	go func() {
		_, extractErr := adapter.Extract(ctx, request)
		result <- extractErr
	}()
	<-workerStarted
	cancel()
	if extractErr := <-result; !errors.Is(extractErr, context.Canceled) {
		t.Fatalf("error=%v", extractErr)
	}
	select {
	case <-workerFinished:
	default:
		t.Fatal("worker 終了前に Extract が戻りました")
	}
	if len(gate) != 0 {
		t.Fatalf("gate usage=%d", len(gate))
	}
}

func TestAdapterReturnsBusyWhenGateIsOccupied(t *testing.T) {
	t.Parallel()

	request := mustExtractRequest(t)
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	adapter, err := newJudicialDecisionCaseCitationExtractAdapter(adapterDependencies{
		doer: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("busy の場合は fetch しません")
			return nil, nil
		}),
		now:       func() time.Time { return mustRetrievedAt(t) },
		gate:      gate,
		runWorker: func(context.Context, []byte) (workerOutput, error) { return workerOutput{}, nil },
		timeout:   parseTimeout,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = adapter.Extract(context.Background(), request)
	assertHanreiPDFSourceErrorCode(t, err, model.SourceErrorCodeSourceBusy)
}

func TestAdapterFetchesPDFOnceAndMapsWorkerResult(t *testing.T) {
	t.Parallel()

	request := mustExtractRequest(t)
	calls := 0
	workerCalls := 0
	adapter, err := newJudicialDecisionCaseCitationExtractAdapter(adapterDependencies{
		doer: roundTripFunc(func(httpRequest *http.Request) (*http.Response, error) {
			calls++
			if httpRequest.Method != http.MethodGet ||
				httpRequest.Header.Get("Accept") != model.JudicialDocumentMediaTypePDF ||
				httpRequest.Header.Get("Authorization") != "" ||
				httpRequest.Header.Get("Cookie") != "" {
				t.Fatalf("request=%#v header=%#v", httpRequest, httpRequest.Header)
			}
			body := []byte("%PDF-fixture")
			return &http.Response{
				StatusCode:    http.StatusOK,
				Header:        http.Header{"Content-Type": []string{"application/pdf; version=1.7"}},
				Body:          io.NopCloser(bytes.NewReader(body)),
				ContentLength: int64(len(body)),
				Request:       httpRequest,
			}, nil
		}),
		now:  func() time.Time { return mustRetrievedAt(t) },
		gate: make(chan struct{}, 1),
		runWorker: func(_ context.Context, input []byte) (workerOutput, error) {
			workerCalls++
			if string(input) != "%PDF-fixture" {
				t.Fatalf("input=%q", input)
			}
			return workerOutput{
				PageCount:         1,
				ObjectCount:       4,
				DecompressedBytes: 10,
				Occurrences: []workerMention{{
					Page:             1,
					ReferenceText:    "平成30(受)10",
					DecisionIdentity: "平成30(受)10",
					Excerpt:          "平成30(受)10",
				}},
			}, nil
		},
		timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := adapter.Extract(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || workerCalls != 1 || result.OccurrenceCount() != 1 {
		t.Fatalf("calls=%d workerCalls=%d result=%#v", calls, workerCalls, result)
	}
}

func TestAdapterRejectsMIMEBadMagicAndDeclaredOversizeBeforeWorker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		contentType   string
		body          []byte
		contentLength int64
		want          model.SourceErrorCode
	}{
		{"mime", "text/html", []byte("%PDF-x"), 6, model.SourceErrorCodeSourceContractChanged},
		{"magic", "application/pdf", []byte("secret"), 6, model.SourceErrorCodeInvalidSourceResponse},
		{"size", "application/pdf", []byte("%PDF-x"), maximumPDFResponseBytes + 1, model.SourceErrorCodeSourceResponseTooLarge},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			workerCalled := false
			adapter, err := newJudicialDecisionCaseCitationExtractAdapter(adapterDependencies{
				doer: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode:    http.StatusOK,
						Header:        http.Header{"Content-Type": []string{test.contentType}},
						Body:          io.NopCloser(bytes.NewReader(test.body)),
						ContentLength: test.contentLength,
						Request:       request,
					}, nil
				}),
				now:  time.Now,
				gate: make(chan struct{}, 1),
				runWorker: func(context.Context, []byte) (workerOutput, error) {
					workerCalled = true
					return workerOutput{}, nil
				},
				timeout: time.Second,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = adapter.Extract(context.Background(), mustExtractRequest(t))
			assertHanreiPDFSourceErrorCode(t, err, test.want)
			if workerCalled {
				t.Fatal("検証失敗後に worker を呼びました")
			}
		})
	}
}

func TestHTTPStatusAndContextErrorsAreNormalized(t *testing.T) {
	t.Parallel()

	for status, want := range map[int]model.SourceErrorCode{
		http.StatusForbidden:           model.SourceErrorCodeInvalidSourceResponse,
		http.StatusTooManyRequests:     model.SourceErrorCodeRateLimited,
		http.StatusInternalServerError: model.SourceErrorCodeSourceUnavailable,
		http.StatusBadRequest:          model.SourceErrorCodeInvalidSourceResponse,
	} {
		if got := codeForStatus(status); got != want {
			t.Fatalf("status=%d got=%s want=%s", status, got, want)
		}
	}
	active := context.Background()
	assertHanreiPDFSourceErrorCode(
		t,
		normalizeContextError(active, operationFetch, errors.New("network")),
		model.SourceErrorCodeSourceUnavailable,
	)
	deadline, cancelDeadline := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelDeadline()
	assertHanreiPDFSourceErrorCode(
		t,
		normalizeContextError(deadline, operationFetch, context.DeadlineExceeded),
		model.SourceErrorCodeSourceTimeout,
	)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := normalizeContextError(canceled, operationFetch, context.Canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}

func TestWorkerFailuresAreNormalizedWithoutDetail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		failure workerFailure
		want    model.SourceErrorCode
	}{
		{workerFailureInvalidResponse, model.SourceErrorCodeInvalidSourceResponse},
		{workerFailureResponseTooLarge, model.SourceErrorCodeSourceResponseTooLarge},
		{workerFailureProcessingLimit, model.SourceErrorCodeSourceProcessingLimit},
		{workerFailureUnsafeContent, model.SourceErrorCodeUnsafeSourceContent},
	}
	for _, test := range tests {
		err := normalizeWorkerRunError(
			context.Background(),
			context.Background(),
			workerError{failure: test.failure},
		)
		assertHanreiPDFSourceErrorCode(t, err, test.want)
	}
	err := normalizeWorkerRunError(context.Background(), context.Background(), errors.New("秘密本文"))
	assertHanreiPDFSourceErrorCode(t, err, model.SourceErrorCodeInvalidSourceResponse)
	if strings.Contains(err.Error(), "秘密本文") {
		t.Fatalf("error=%v", err)
	}
}

func TestDocumentURLAndRedirectPolicyRemainOnFixedOrigin(t *testing.T) {
	t.Parallel()

	allowed := "https://www.courts.go.jp/assets/hanrei/123.pdf"
	if !isAllowedDocumentURL(allowed) {
		t.Fatalf("allowed URL を拒否しました: %s", allowed)
	}
	for _, rejected := range []string{
		"http://www.courts.go.jp/assets/hanrei/123.pdf",
		"https://example.com/assets/hanrei/123.pdf",
		"https://www.courts.go.jp/assets/hanrei/../secret.pdf",
		"https://www.courts.go.jp/assets/hanrei/123.pdf?token=x",
		"https://www.courts.go.jp/assets/hanrei%2F123.pdf",
	} {
		if isAllowedDocumentURL(rejected) {
			t.Fatalf("URL を受理しました: %s", rejected)
		}
	}
	allowedURL, _ := url.Parse(allowed)
	redirect := &http.Request{
		Method: http.MethodGet,
		URL:    allowedURL,
		Header: http.Header{
			"Authorization": []string{"secret"},
			"Cookie":        []string{"secret"},
			"Referer":       []string{"secret"},
		},
	}
	if err := courtsPDFRedirectPolicy(
		redirect,
		[]*http.Request{{Method: http.MethodGet, URL: allowedURL}},
	); err != nil {
		t.Fatal(err)
	}
	if redirect.Header.Get("Authorization") != "" || redirect.Header.Get("Cookie") != "" ||
		redirect.Header.Get("Referer") != "" {
		t.Fatalf("headers=%#v", redirect.Header)
	}
	badURL, _ := url.Parse("https://example.com/assets/hanrei/123.pdf")
	if err := courtsPDFRedirectPolicy(
		&http.Request{Method: http.MethodGet, URL: badURL},
		nil,
	); !errors.Is(err, errUnsafeRedirect) {
		t.Fatalf("error=%v", err)
	}
}

func TestConstructorsCreateLatentBindingAndRejectMissingDependencies(t *testing.T) {
	t.Parallel()

	adapter, err := NewJudicialDecisionCaseCitationExtractAdapter()
	if err != nil || adapter == nil {
		t.Fatalf("adapter=%#v error=%v", adapter, err)
	}
	binding, err := NewProviderBinding()
	if err != nil || binding.CitationExtract == nil ||
		binding.Descriptor.ProviderID() != providerID {
		t.Fatalf("binding=%#v error=%v", binding, err)
	}
	if _, err := newJudicialDecisionCaseCitationExtractAdapter(adapterDependencies{}); err == nil {
		t.Fatal("不足した依存関係を受理しました")
	}
	if _, err := newJudicialDecisionCaseCitationExtractAdapter(adapterDependencies{
		doer:      roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, nil }),
		now:       time.Now,
		gate:      make(chan struct{}, 2),
		runWorker: func(context.Context, []byte) (workerOutput, error) { return workerOutput{}, nil },
		timeout:   time.Second,
	}); err == nil {
		t.Fatal("容量 2 の gate を受理しました")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func assertHanreiPDFSourceErrorCode(
	t *testing.T,
	err error,
	code model.SourceErrorCode,
) {
	t.Helper()
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("error = %T %v, want model.SourceError", err, err)
	}
	if sourceError.Code() != code {
		t.Fatalf("code = %q, want %q", sourceError.Code(), code)
	}
}
