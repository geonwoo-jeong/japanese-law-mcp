package hanreipdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const parseTimeout = 4 * time.Second

var sharedPDFGate = make(chan struct{}, 1)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type adapterDependencies struct {
	doer      httpDoer
	now       func() time.Time
	gate      chan struct{}
	runWorker workerRunner
	timeout   time.Duration
}

// JudicialDecisionCaseCitationExtractAdapter は、全文 PDF の一回解析を行う。
type JudicialDecisionCaseCitationExtractAdapter struct {
	dependencies adapterDependencies
}

var _ judicialcasecitationextract.Port = (*JudicialDecisionCaseCitationExtractAdapter)(nil)

// NewJudicialDecisionCaseCitationExtractAdapter は、固定 origin と同時実行 1 の adapter を返す。
func NewJudicialDecisionCaseCitationExtractAdapter() (*JudicialDecisionCaseCitationExtractAdapter, error) {
	return newJudicialDecisionCaseCitationExtractAdapter(adapterDependencies{
		doer:      newProductionHTTPClient(),
		now:       time.Now,
		gate:      sharedPDFGate,
		runWorker: productionWorkerRunner,
		timeout:   parseTimeout,
	})
}

func newJudicialDecisionCaseCitationExtractAdapter(
	dependencies adapterDependencies,
) (*JudicialDecisionCaseCitationExtractAdapter, error) {
	if dependencies.doer == nil || dependencies.now == nil || dependencies.gate == nil ||
		dependencies.runWorker == nil || dependencies.timeout <= 0 {
		return nil, fmt.Errorf("裁判所 PDF adapter の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("裁判所 PDF の同時実行枠は一件でなければなりません")
	}
	return &JudicialDecisionCaseCitationExtractAdapter{dependencies: dependencies}, nil
}

// Extract は、同一 request の詳細に属する全文 PDF だけを一回解析する。
func (a *JudicialDecisionCaseCitationExtractAdapter) Extract(
	ctx context.Context,
	request judicialcasecitationextract.Request,
) (judicialcasecitationextract.Result, error) {
	if ctx == nil {
		return judicialcasecitationextract.Result{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return judicialcasecitationextract.Result{}, err
	}
	if !isAllowedDocumentURL(request.Document().URL()) {
		return judicialcasecitationextract.Result{}, newSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			operationFetch,
			"",
		)
	}
	if err := a.acquire(ctx); err != nil {
		return judicialcasecitationextract.Result{}, err
	}
	defer a.release()

	pdfBytes, retrievedAt, err := a.fetchPDF(ctx, request.Document().URL())
	if err != nil {
		return judicialcasecitationextract.Result{}, err
	}
	processing, cancel := context.WithTimeout(ctx, a.dependencies.timeout)
	defer cancel()
	output, err := a.dependencies.runWorker(processing, pdfBytes)
	if err != nil {
		return judicialcasecitationextract.Result{}, normalizeWorkerRunError(ctx, processing, err)
	}
	return buildExtractResult(request, retrievedAt, pdfBytes, output)
}

func (a *JudicialDecisionCaseCitationExtractAdapter) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return normalizeContextError(ctx, operationFetch, err)
	}
	select {
	case a.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newSourceError(model.SourceErrorCodeSourceBusy, operationFetch, "")
	}
}

func (a *JudicialDecisionCaseCitationExtractAdapter) release() {
	<-a.dependencies.gate
}

func (a *JudicialDecisionCaseCitationExtractAdapter) fetchPDF(
	ctx context.Context,
	rawURL string,
) ([]byte, time.Time, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, time.Time{}, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			operationFetch,
			"",
		)
	}
	request.Header.Set("Accept", model.JudicialDocumentMediaTypePDF)
	response, err := a.dependencies.doer.Do(request)
	if err != nil {
		if errors.Is(err, errUnsafeRedirect) {
			return nil, time.Time{}, newSourceError(
				model.SourceErrorCodeUnsafeSourceContent,
				operationFetch,
				"",
			)
		}
		return nil, time.Time{}, normalizeContextError(ctx, operationFetch, err)
	}
	if response == nil || response.Body == nil {
		return nil, time.Time{}, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			operationFetch,
			"",
		)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		discardPDFBody(response.Body)
		return nil, time.Time{}, errorForHTTPStatus(response.StatusCode)
	}
	finalURL := request.URL
	if response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL
	}
	if !isAllowedDocumentURL(finalURL.String()) {
		return nil, time.Time{}, newSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			operationFetch,
			"",
		)
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, model.JudicialDocumentMediaTypePDF) {
		return nil, time.Time{}, newSourceError(
			model.SourceErrorCodeSourceContractChanged,
			operationFetch,
			"",
		)
	}
	if response.ContentLength > maximumPDFResponseBytes {
		return nil, time.Time{}, newSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			operationFetch,
			"",
		)
	}
	body, err := readPDFBody(ctx, response.Body)
	if err != nil {
		return nil, time.Time{}, err
	}
	if !bytes.HasPrefix(body, []byte("%PDF-")) {
		return nil, time.Time{}, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			operationFetch,
			"",
		)
	}
	return body, a.dependencies.now().Round(0), nil
}

func readPDFBody(ctx context.Context, body io.Reader) ([]byte, error) {
	value, err := io.ReadAll(io.LimitReader(
		&contextReader{context: ctx, reader: body},
		maximumPDFResponseBytes+1,
	))
	if err != nil {
		if ctx.Err() != nil {
			return nil, normalizeContextError(ctx, operationFetch, ctx.Err())
		}
		return nil, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			operationFetch,
			"",
		)
	}
	if len(value) > maximumPDFResponseBytes {
		return nil, newSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			operationFetch,
			"",
		)
	}
	if len(value) == 0 {
		return nil, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			operationFetch,
			"",
		)
	}
	return value, nil
}

type contextReader struct {
	context context.Context
	reader  io.Reader
}

func (r *contextReader) Read(value []byte) (int, error) {
	if err := r.context.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(value)
}

func discardPDFBody(body io.Reader) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 1024))
}

func isAllowedDocumentURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	cleaned := path.Clean(parsed.Path)
	return strings.EqualFold(parsed.Scheme, "https") &&
		strings.EqualFold(parsed.Host, "www.courts.go.jp") &&
		parsed.Port() == "" &&
		parsed.User == nil &&
		parsed.Opaque == "" &&
		parsed.RawPath == "" &&
		parsed.RawQuery == "" &&
		parsed.Fragment == "" &&
		parsed.Path == cleaned &&
		strings.HasPrefix(parsed.Path, documentPathPrefix) &&
		len(parsed.Path) > len(documentPathPrefix)
}

func normalizeWorkerRunError(
	parent context.Context,
	processing context.Context,
	err error,
) error {
	if errors.Is(parent.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(parent.Err(), context.DeadlineExceeded) ||
		errors.Is(processing.Err(), context.DeadlineExceeded) ||
		errors.Is(err, context.DeadlineExceeded) {
		return newSourceError(model.SourceErrorCodeSourceTimeout, operationParse, "")
	}
	var classified workerError
	if !errors.As(err, &classified) {
		return newSourceError(model.SourceErrorCodeInvalidSourceResponse, operationParse, "")
	}
	switch classified.failure {
	case workerFailureResponseTooLarge:
		return newSourceError(model.SourceErrorCodeSourceResponseTooLarge, operationParse, "")
	case workerFailureProcessingLimit:
		return newSourceError(model.SourceErrorCodeSourceProcessingLimit, operationParse, "")
	case workerFailureUnsafeContent:
		return newSourceError(model.SourceErrorCodeUnsafeSourceContent, operationParse, "")
	default:
		return newSourceError(model.SourceErrorCodeInvalidSourceResponse, operationParse, "")
	}
}

// ProviderBinding は、公開 registry へ未接続の descriptor と型付き port を保持する。
type ProviderBinding struct {
	Descriptor      model.ProviderDescriptor
	CitationExtract judicialcasecitationextract.Port
}

// NewProviderBinding は、判例引用 pack の有効化時に登録できる latent binding を返す。
func NewProviderBinding() (ProviderBinding, error) {
	adapter, err := NewJudicialDecisionCaseCitationExtractAdapter()
	if err != nil {
		return ProviderBinding{}, err
	}
	return ProviderBinding{Descriptor: Descriptor(), CitationExtract: adapter}, nil
}
