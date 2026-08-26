package hanreipdf

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestDescriptorBindingAndRedirectPolicy(t *testing.T) {
	t.Parallel()

	descriptor := Descriptor()
	if descriptor.ProviderID() != providerID || descriptor.Source().ID() != sourceID {
		t.Fatalf("descriptor = %#v", descriptor)
	}
	binding, err := NewProviderBinding()
	if err != nil {
		t.Fatal(err)
	}
	if binding.Descriptor.ProviderID() != providerID || binding.CitationExtract == nil {
		t.Fatalf("binding = %#v", binding)
	}

	client := newProductionHTTPClient()
	if client.Transport == nil || client.CheckRedirect == nil {
		t.Fatalf("client = %#v", client)
	}
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"https://www.courts.go.jp/assets/hanrei/00001.pdf",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "secret")
	request.Header.Set("Cookie", "a=b")
	request.Header.Set("Proxy-Authorization", "proxy")
	request.Header.Set("Referer", "https://example.com")
	if err := courtsPDFRedirectPolicy(request, nil); err != nil {
		t.Fatalf("allowed redirect = %v", err)
	}
	if request.Header.Get("Authorization") != "" ||
		request.Header.Get("Cookie") != "" ||
		request.Header.Get("Proxy-Authorization") != "" ||
		request.Header.Get("Referer") != "" {
		t.Fatal("redirect policy が機密 header を除去していません")
	}
	badRequest, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"https://example.com/evil.pdf",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(courtsPDFRedirectPolicy(badRequest, nil), errUnsafeRedirect) {
		t.Fatal("外部 redirect を拒否していません")
	}
}

func TestErrorHelpersAndWorkerNormalization(t *testing.T) {
	t.Parallel()

	if operationFetch.SourceOperationName() != "GET /assets/hanrei/{id}.pdf" ||
		operationFetch.SourceOperationProviderID() != providerID {
		t.Fatal("operation metadata が不正です")
	}
	if err := operation("unknown").ValidateSourceOperation(); err == nil {
		t.Fatal("未知の operation を受理しました")
	}
	if codeForStatus(http.StatusTooManyRequests) != model.SourceErrorCodeRateLimited ||
		codeForStatus(http.StatusInternalServerError) != model.SourceErrorCodeSourceUnavailable ||
		codeForStatus(http.StatusBadRequest) != model.SourceErrorCodeInvalidSourceResponse {
		t.Fatal("HTTP status の正規化が不正です")
	}
	if !errors.Is(errorForHTTPStatus(http.StatusNotFound), judicialcasecitationextract.ErrNotFound) {
		t.Fatal("404 を ErrNotFound に変換していません")
	}
	assertHanreiPDFSourceErrorCode(
		t,
		errorForHTTPStatus(http.StatusTooManyRequests),
		model.SourceErrorCodeRateLimited,
	)

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(normalizeContextError(canceled, operationFetch, canceled.Err()), context.Canceled) {
		t.Fatal("cancel を維持していません")
	}
	deadline, deadlineCancel := context.WithDeadline(context.Background(), pastTime())
	defer deadlineCancel()
	assertHanreiPDFSourceErrorCode(
		t,
		normalizeContextError(deadline, operationFetch, deadline.Err()),
		model.SourceErrorCodeSourceTimeout,
	)
	assertHanreiPDFSourceErrorCode(
		t,
		normalizeContextError(context.Background(), operationFetch, io.ErrUnexpectedEOF),
		model.SourceErrorCodeSourceUnavailable,
	)
	assertHanreiPDFSourceErrorCode(
		t,
		normalizeWorkerRunError(context.Background(), context.Background(), workerError{failure: workerFailureResponseTooLarge}),
		model.SourceErrorCodeSourceResponseTooLarge,
	)
	assertHanreiPDFSourceErrorCode(
		t,
		normalizeWorkerRunError(context.Background(), context.Background(), workerError{failure: workerFailureUnsafeContent}),
		model.SourceErrorCodeUnsafeSourceContent,
	)
	assertHanreiPDFSourceErrorCode(
		t,
		normalizeWorkerRunError(context.Background(), context.Background(), workerError{failure: workerFailureInvalidResponse}),
		model.SourceErrorCodeInvalidSourceResponse,
	)
}

func TestReadPDFBodyAndContextHelpers(t *testing.T) {
	t.Parallel()

	value, err := readPDFBody(context.Background(), bytes.NewReader([]byte("%PDF-1.7\n%%EOF")))
	if err != nil || !bytes.HasPrefix(value, []byte("%PDF-")) {
		t.Fatalf("value=%q err=%v", value, err)
	}
	_, err = readPDFBody(context.Background(), bytes.NewReader(nil))
	assertHanreiPDFSourceErrorCode(t, err, model.SourceErrorCodeInvalidSourceResponse)
	tooLarge := append([]byte("%PDF-"), make([]byte, maximumPDFResponseBytes)...)
	_, err = readPDFBody(context.Background(), bytes.NewReader(tooLarge))
	assertHanreiPDFSourceErrorCode(t, err, model.SourceErrorCodeSourceResponseTooLarge)

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	reader := &contextReader{context: canceled, reader: bytes.NewReader([]byte("%PDF-1.7"))}
	if _, err := reader.Read(make([]byte, 4)); !errors.Is(err, context.Canceled) {
		t.Fatalf("read error = %v", err)
	}
}

func TestWorkerEntryPointsAndSmallHelpers(t *testing.T) {
	t.Parallel()

	if handled, code := RunPrivateWorkerIfRequested(
		bytes.NewReader(nil),
		&bytes.Buffer{},
		io.Discard,
		"other",
	); handled || code != 0 {
		t.Fatalf("handled=%t code=%d", handled, code)
	}
	if (workerError{failure: workerFailureInvalidResponse}).Error() == "" {
		t.Fatal("worker error message が空です")
	}
	if _, err := runWorkerProcess(context.Background(), []byte("%PDF-1.7"), "", []string{"A=B"}); err == nil {
		t.Fatal("空 executable を受理しました")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := productionWorkerRunner(canceled, []byte("%PDF-1.7")); !errors.Is(err, context.Canceled) {
		t.Fatalf("production worker error = %v", err)
	}

	tempDir := t.TempDir()
	file, path, err := createWorkerTempPDF([]byte("%PDF-1.7\n%%EOF"), tempDir)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	data, readErr := os.ReadFile(path) //nolint:gosec // SOT-IF-071: t.TempDir 配下へ worker が作成した固定 fixture だけを読む。
	if readErr != nil || string(data) != "%PDF-1.7\n%%EOF" {
		t.Fatalf("data=%q err=%v", data, readErr)
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %#o", info.Mode().Perm())
	}

	if prefix := utf8Prefix("あいうえお", 5); prefix != "あ" {
		t.Fatalf("prefix = %q", prefix)
	}
	if suffix := utf8Suffix("あいうえお", 5); suffix != "お" {
		t.Fatalf("suffix = %q", suffix)
	}
	if firstRune("令和") != '令' {
		t.Fatal("firstRune が不正です")
	}
	if base := filepath.Base(path); base == "" {
		t.Fatal("temp path が不正です")
	}
}

func pastTime() time.Time {
	return time.Date(2026, 8, 25, 0, 0, 0, 0, time.FixedZone("JST", 9*60*60))
}
