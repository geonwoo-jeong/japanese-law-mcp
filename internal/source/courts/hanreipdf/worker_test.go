package hanreipdf

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestTabulaParsesSyntheticJapaneseTextAndPreservesOccurrences(t *testing.T) {
	t.Parallel()

	tempDirectory := t.TempDir()
	pdf := syntheticJapanesePDF(
		"令和6(受)123。平成30年（受）第10号。平成99(受)1。平成30年（受）第10号。",
	)
	output, err := parsePDFWithTabula(pdf, tempDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if output.PageCount != 1 || output.ObjectCount != 8 || output.TextUnavailable ||
		len(output.Occurrences) != 4 {
		t.Fatalf("output=%#v", output)
	}
	if output.Occurrences[0].DecisionIdentity != "令和6(受)123" ||
		output.Occurrences[1].DecisionIdentity != "平成30年(受)第10号" ||
		output.Occurrences[2].Reason == "" ||
		output.Occurrences[3].DecisionIdentity != "平成30年(受)第10号" {
		t.Fatalf("occurrences=%#v", output.Occurrences)
	}
	entries, err := os.ReadDir(tempDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("一時ファイルが残りました: %#v", entries)
	}
}

func TestTabulaTreatsSyntheticImageOnlyPDFAsTextUnavailable(t *testing.T) {
	t.Parallel()

	output, err := parsePDFWithTabula(syntheticImageOnlyPDF(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if output.PageCount != 1 || len(output.Occurrences) != 0 || !output.TextUnavailable {
		t.Fatalf("output=%#v", output)
	}
}

func TestPrivateWorkerRejectsMagicEncryptionAndOversizeWithoutDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		limit   int64
		failure workerFailure
	}{
		{"magic", []byte("not-pdf"), 64, workerFailureInvalidResponse},
		{"encrypted", []byte("%PDF-1.7\ntrailer << /Encrypt 8 0 R >>"), 64, workerFailureUnsafeContent},
		{"oversize", []byte("%PDF-1234"), 8, workerFailureResponseTooLarge},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout bytes.Buffer
			called := false
			code := runPrivateWorkerWithLimit(
				bytes.NewReader(test.input),
				&stdout,
				func([]byte) (workerOutput, error) {
					called = true
					return workerOutput{}, nil
				},
				test.limit,
			)
			if code != 0 || called {
				t.Fatalf("code=%d called=%t", code, called)
			}
			_, err := decodeWorkerEnvelope(stdout.Bytes())
			var classified workerError
			if !errors.As(err, &classified) || classified.failure != test.failure {
				t.Fatalf("error=%T %v stdout=%q", err, err, stdout.String())
			}
			if strings.Contains(stdout.String(), "Encrypt") || strings.Contains(stdout.String(), "not-pdf") {
				t.Fatalf("worker output に原文が含まれています: %q", stdout.String())
			}
		})
	}
}

func TestPrivateWorkerRecoversParserPanicWithoutStackOrPath(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	code := runPrivateWorker(
		bytes.NewReader(syntheticImageOnlyPDF()),
		&stdout,
		func([]byte) (workerOutput, error) {
			panic("/private/secret.pdf 本文")
		},
	)
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	_, err := decodeWorkerEnvelope(stdout.Bytes())
	var classified workerError
	if !errors.As(err, &classified) || classified.failure != workerFailureProcessingLimit {
		t.Fatalf("error=%T %v", err, err)
	}
	if strings.Contains(stdout.String(), "secret") || strings.Contains(stdout.String(), "panic") {
		t.Fatalf("worker output=%q", stdout.String())
	}
}

func TestInspectPDFBudgetsUsesInjectableSmallLimits(t *testing.T) {
	t.Parallel()

	pdfReader := mustSyntheticPDFReader(t, syntheticJapanesePDF("令和5(受)1"))

	_, err := inspectPDFBudgetsWithLimits(pdfReader, pdfInspectionLimits{
		objects:           maximumObjectCount,
		depth:             maximumReferenceDepth,
		decompressedBytes: 1,
		visitedNodes:      maximumObjectCount,
	})
	var classified workerError
	if !errors.As(err, &classified) || classified.failure != workerFailureProcessingLimit {
		t.Fatalf("error=%T %v", err, err)
	}
}

func TestInspectPDFBudgetsRejectsDeepObjectGraphWithSmallDepth(t *testing.T) {
	t.Parallel()

	pdfReader := mustSyntheticPDFReader(t, syntheticJapanesePDF("令和5(受)1"))
	_, err := inspectPDFBudgetsWithLimits(pdfReader, pdfInspectionLimits{
		objects:           maximumObjectCount,
		depth:             2,
		decompressedBytes: maximumDecompressedBytes,
		visitedNodes:      maximumObjectCount,
	})
	var classified workerError
	if !errors.As(err, &classified) || classified.failure != workerFailureProcessingLimit {
		t.Fatalf("error=%T %v", err, err)
	}
}

func TestInspectPDFBudgetsRejectsObjectCountWithSmallLimit(t *testing.T) {
	t.Parallel()

	pdfReader := mustSyntheticPDFReader(t, syntheticJapanesePDF("令和5(受)1"))
	_, err := inspectPDFBudgetsWithLimits(pdfReader, pdfInspectionLimits{
		objects:           2,
		depth:             maximumReferenceDepth,
		decompressedBytes: maximumDecompressedBytes,
		visitedNodes:      maximumObjectCount,
	})
	var classified workerError
	if !errors.As(err, &classified) || classified.failure != workerFailureProcessingLimit {
		t.Fatalf("error=%T %v", err, err)
	}
}

func TestTabulaRejectsDeclaredPageCountOverBudget(t *testing.T) {
	t.Parallel()

	_, err := parsePDFWithTabula(syntheticOversizedPageTreePDF(), t.TempDir())
	var classified workerError
	if !errors.As(err, &classified) || classified.failure != workerFailureProcessingLimit {
		t.Fatalf("error=%T %v", err, err)
	}
}

func TestDecodeWorkerEnvelopeRejectsUnknownOrMultipleValues(t *testing.T) {
	t.Parallel()

	values := [][]byte{
		[]byte(`{"unknown":true}`),
		[]byte(`{"failure":"other"}`),
		[]byte("{} {}"),
		[]byte(`{"output":{"pageCount":1},"failure":"unsafe_source_content"}`),
	}
	for _, value := range values {
		if _, err := decodeWorkerEnvelope(value); err == nil {
			t.Fatalf("protocol violation を受理しました: %q", value)
		}
	}
}

func TestLimitedWorkerBufferBoundsStoredBytes(t *testing.T) {
	t.Parallel()

	buffer := &limitedWorkerBuffer{limit: 4}
	if _, err := buffer.Write([]byte("123456")); err != nil {
		t.Fatal(err)
	}
	if !buffer.exceeded || string(buffer.Bytes()) != "1234" {
		t.Fatalf("buffer=%q exceeded=%t", buffer.Bytes(), buffer.exceeded)
	}
}

func TestPrivateWorkerEntryIsHiddenAndHandlesValidPDF(t *testing.T) {
	t.Parallel()

	if handled, code := RunPrivateWorkerIfRequested(
		bytes.NewReader(nil),
		io.Discard,
		io.Discard,
		"",
	); handled || code != 0 {
		t.Fatalf("handled=%t code=%d", handled, code)
	}
	var stdout bytes.Buffer
	handled, code := RunPrivateWorkerIfRequested(
		bytes.NewReader(syntheticImageOnlyPDF()),
		&stdout,
		io.Discard,
		privateWorkerModePDF,
	)
	if !handled || code != 0 {
		t.Fatalf("handled=%t code=%d stdout=%q", handled, code, stdout.String())
	}
	output, err := decodeWorkerEnvelope(stdout.Bytes())
	if err != nil || !output.TextUnavailable {
		t.Fatalf("output=%#v error=%v", output, err)
	}
}

func TestPrivateWorkerMapsParserFailureAndSuccessProtocol(t *testing.T) {
	t.Parallel()

	for _, failure := range []workerFailure{
		workerFailureInvalidResponse,
		workerFailureProcessingLimit,
	} {
		var stdout bytes.Buffer
		code := runPrivateWorkerWithLimit(
			bytes.NewReader([]byte("%PDF-x")),
			&stdout,
			func([]byte) (workerOutput, error) { return workerOutput{}, workerError{failure: failure} },
			16,
		)
		if code != 0 {
			t.Fatalf("code=%d", code)
		}
		_, err := decodeWorkerEnvelope(stdout.Bytes())
		var classified workerError
		if !errors.As(err, &classified) || classified.failure != failure {
			t.Fatalf("error=%T %v", err, err)
		}
		if classified.Error() == "" {
			t.Fatal("固定 worker error message が空です")
		}
	}

	want := workerOutput{
		PageCount: 1, ObjectCount: 1, Occurrences: []workerMention{}, TextUnavailable: true,
	}
	var stdout bytes.Buffer
	code := runPrivateWorkerWithLimit(
		bytes.NewReader([]byte("%PDF-x")),
		&stdout,
		func([]byte) (workerOutput, error) { return want, nil },
		16,
	)
	got, err := decodeWorkerEnvelope(stdout.Bytes())
	if code != 0 || err != nil || got.PageCount != want.PageCount || !got.TextUnavailable {
		t.Fatalf("code=%d got=%#v error=%v", code, got, err)
	}
}

func TestWorkerProcessRejectsInvalidInvocationBeforeStart(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runWorkerProcess(ctx, nil, "", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if _, err := runWorkerProcess(context.Background(), nil, "", nil); err == nil {
		t.Fatal("空の executable と環境を受理しました")
	}
}
