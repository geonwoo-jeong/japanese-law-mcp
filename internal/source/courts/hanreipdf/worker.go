package hanreipdf

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tsawler/tabula/core"
	"github.com/tsawler/tabula/reader"
)

const (
	privateWorkerEnv     = "JLMCP_PRIVATE_WORKER_MODE"
	privateWorkerModePDF = "judicial-citation-pdf-v1"
	privateWorkerTempEnv = "JLMCP_PRIVATE_WORKER_TMPDIR"
)

type workerFailure string

const (
	workerFailureInvalidResponse  workerFailure = "invalid_source_response"
	workerFailureResponseTooLarge workerFailure = "source_response_too_large"
	workerFailureProcessingLimit  workerFailure = "source_processing_limit"
	workerFailureUnsafeContent    workerFailure = "unsafe_source_content"
)

type workerEnvelope struct {
	Output  *workerOutput `json:"output,omitempty"`
	Failure workerFailure `json:"failure,omitempty"`
}

type workerError struct {
	failure workerFailure
}

func (e workerError) Error() string {
	return "PDF worker が安全に処理を完了できませんでした"
}

// RunPrivateWorkerIfRequested は、固定の非公開 mode 要求時だけ worker を実行する。
func RunPrivateWorkerIfRequested(
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	mode string,
) (bool, int) {
	if mode != privateWorkerModePDF {
		return false, 0
	}
	_ = stderr
	return true, runPrivateWorker(stdin, stdout, func(pdfBytes []byte) (workerOutput, error) {
		return parsePDFWithTabula(pdfBytes, os.Getenv(privateWorkerTempEnv))
	})
}

func runPrivateWorker(
	stdin io.Reader,
	stdout io.Writer,
	parse func([]byte) (workerOutput, error),
) (exitCode int) {
	return runPrivateWorkerWithLimit(
		stdin,
		stdout,
		parse,
		maximumPDFResponseBytes,
	)
}

func runPrivateWorkerWithLimit(
	stdin io.Reader,
	stdout io.Writer,
	parse func([]byte) (workerOutput, error),
	inputLimit int64,
) (exitCode int) {
	exitCode = 1
	defer func() {
		if recover() == nil {
			return
		}
		if encodeWorkerEnvelope(stdout, workerEnvelope{Failure: workerFailureProcessingLimit}) == nil {
			exitCode = 0
		}
	}()

	if inputLimit < 1 || inputLimit > maximumPDFResponseBytes {
		return writeWorkerFailure(stdout, workerFailureInvalidResponse)
	}
	pdfBytes, err := io.ReadAll(io.LimitReader(stdin, inputLimit+1))
	if err != nil {
		return writeWorkerFailure(stdout, workerFailureInvalidResponse)
	}
	if int64(len(pdfBytes)) > inputLimit {
		return writeWorkerFailure(stdout, workerFailureResponseTooLarge)
	}
	if len(pdfBytes) == 0 || !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		return writeWorkerFailure(stdout, workerFailureInvalidResponse)
	}
	if looksEncryptedPDF(pdfBytes) {
		return writeWorkerFailure(stdout, workerFailureUnsafeContent)
	}
	output, err := parse(pdfBytes)
	if err != nil {
		var classified workerError
		if errors.As(err, &classified) {
			return writeWorkerFailure(stdout, classified.failure)
		}
		return writeWorkerFailure(stdout, workerFailureInvalidResponse)
	}
	if err := encodeWorkerEnvelope(stdout, workerEnvelope{Output: &output}); err != nil {
		return 1
	}
	return 0
}

func writeWorkerFailure(stdout io.Writer, failure workerFailure) int {
	if encodeWorkerEnvelope(stdout, workerEnvelope{Failure: failure}) != nil {
		return 1
	}
	return 0
}

func encodeWorkerEnvelope(stdout io.Writer, envelope workerEnvelope) error {
	encoded, err := json.Marshal(envelope)
	if err != nil || len(encoded)+1 > maximumWorkerStdoutBytes {
		return fmt.Errorf("worker protocol を構成できません")
	}
	encoded = append(encoded, '\n')
	_, err = stdout.Write(encoded)
	return err
}

func parsePDFWithTabula(pdfBytes []byte, tempDirectory string) (workerOutput, error) {
	file, path, err := createWorkerTempPDF(pdfBytes, tempDirectory)
	if err != nil {
		return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
	}
	defer func() {
		_ = file.Close()
		_ = os.Remove(path)
	}()

	pdfReader, err := reader.NewReader(file)
	if err != nil {
		return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
	}
	decompressedBytes, err := inspectPDFBudgets(pdfReader)
	if err != nil {
		return workerOutput{}, err
	}
	pageCount, err := pdfReader.PageCount()
	if err != nil || pageCount < 1 {
		return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
	}
	if pageCount > maximumPageCount {
		return workerOutput{}, workerError{failure: workerFailureProcessingLimit}
	}

	occurrences := make([]workerMention, 0)
	totalTextBytes := 0
	textAvailable := false
	truncated := false
	for index := 0; index < pageCount; index++ {
		page, pageErr := pdfReader.GetPage(index)
		if pageErr != nil {
			return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
		}
		fragments, extractErr := pdfReader.ExtractTextFragments(page)
		if extractErr != nil {
			return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
		}
		var pageText strings.Builder
		for _, fragment := range fragments {
			if !utf8.ValidString(fragment.Text) || strings.ContainsRune(fragment.Text, utf8.RuneError) {
				return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
			}
			totalTextBytes += len(fragment.Text)
			if totalTextBytes > maximumExtractedTextBytes {
				return workerOutput{}, workerError{failure: workerFailureProcessingLimit}
			}
			pageText.WriteString(fragment.Text)
		}
		text := pageText.String()
		if hasUsableText(text) {
			textAvailable = true
		}
		if truncated {
			continue
		}
		pageOccurrences, pageTruncated := scanExtractedText(
			text,
			index+1,
			maximumOccurrences-len(occurrences),
		)
		occurrences = append(occurrences, pageOccurrences...)
		truncated = pageTruncated
	}
	if extraPage, extraErr := pdfReader.GetPage(pageCount); extraErr == nil && extraPage != nil {
		return workerOutput{}, workerError{failure: workerFailureProcessingLimit}
	}
	return workerOutput{
		PageCount:         pageCount,
		ObjectCount:       pdfReader.NumObjects(),
		DecompressedBytes: decompressedBytes,
		Occurrences:       occurrences,
		TextUnavailable:   !textAvailable,
		Truncated:         truncated,
	}, nil
}

func createWorkerTempPDF(pdfBytes []byte, tempDirectory string) (*os.File, string, error) {
	file, err := os.CreateTemp(tempDirectory, "japanese-law-mcp-hanreipdf-*.pdf")
	if err != nil {
		return nil, "", err
	}
	path := file.Name()
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(path)
	}
	if err := file.Chmod(0o600); err != nil {
		cleanup()
		return nil, "", err
	}
	if _, err := file.Write(pdfBytes); err != nil {
		cleanup()
		return nil, "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, "", err
	}
	return file, path, nil
}

type budgetInspection struct {
	reader            *reader.Reader
	decodedStreams    map[*core.Stream]struct{}
	decompressedBytes int
	visitedNodes      int
	limits            pdfInspectionLimits
}

type pdfInspectionLimits struct {
	objects           int
	depth             int
	decompressedBytes int
	visitedNodes      int
}

func inspectPDFBudgets(pdfReader *reader.Reader) (int, error) {
	return inspectPDFBudgetsWithLimits(pdfReader, pdfInspectionLimits{
		objects:           maximumObjectCount,
		depth:             maximumReferenceDepth,
		decompressedBytes: maximumDecompressedBytes,
		visitedNodes:      maximumObjectCount * (maximumReferenceDepth + 1),
	})
}

func inspectPDFBudgetsWithLimits(
	pdfReader *reader.Reader,
	limits pdfInspectionLimits,
) (int, error) {
	if pdfReader == nil || pdfReader.XRefTable() == nil {
		return 0, workerError{failure: workerFailureInvalidResponse}
	}
	if limits.objects < 1 || limits.depth < 1 ||
		limits.decompressedBytes < 1 || limits.visitedNodes < 1 {
		return 0, workerError{failure: workerFailureInvalidResponse}
	}
	objectCount := pdfReader.NumObjects()
	if objectCount < 1 || objectCount > limits.objects ||
		pdfReader.XRefTable().Size() > limits.objects {
		return 0, workerError{failure: workerFailureProcessingLimit}
	}
	inspection := budgetInspection{
		reader:         pdfReader,
		decodedStreams: make(map[*core.Stream]struct{}),
		limits:         limits,
	}
	if err := inspection.walk(pdfReader.Trailer(), 0, make(map[int]struct{})); err != nil {
		return 0, err
	}
	objectNumbers := make([]int, 0, pdfReader.XRefTable().Size())
	for number, entry := range pdfReader.XRefTable().Entries {
		if entry != nil && entry.InUse {
			objectNumbers = append(objectNumbers, number)
		}
	}
	sort.Ints(objectNumbers)
	for _, number := range objectNumbers {
		object, err := pdfReader.GetObject(number)
		if err != nil {
			return 0, workerError{failure: workerFailureInvalidResponse}
		}
		if err := inspection.walk(object, 0, map[int]struct{}{number: {}}); err != nil {
			return 0, err
		}
	}
	return inspection.decompressedBytes, nil
}

func (i *budgetInspection) walk(
	object core.Object,
	depth int,
	path map[int]struct{},
) error {
	if depth > i.limits.depth {
		return workerError{failure: workerFailureProcessingLimit}
	}
	i.visitedNodes++
	if i.visitedNodes > i.limits.visitedNodes {
		return workerError{failure: workerFailureProcessingLimit}
	}
	switch value := object.(type) {
	case core.IndirectRef:
		if _, cyclic := path[value.Number]; cyclic {
			return nil
		}
		resolved, err := i.reader.ResolveReference(value)
		if err != nil {
			return workerError{failure: workerFailureInvalidResponse}
		}
		nextPath := cloneReferencePath(path)
		nextPath[value.Number] = struct{}{}
		return i.walk(resolved, depth+1, nextPath)
	case core.Array:
		for _, child := range value {
			if err := i.walk(child, depth+1, path); err != nil {
				return err
			}
		}
	case core.Dict:
		if value.Get("EF") != nil {
			return workerError{failure: workerFailureUnsafeContent}
		}
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if err := i.walk(value[key], depth+1, path); err != nil {
				return err
			}
		}
	case *core.Stream:
		if streamType, ok := value.Dict.Get("Type").(core.Name); ok &&
			string(streamType) == "EmbeddedFile" {
			return workerError{failure: workerFailureUnsafeContent}
		}
		if err := i.walk(value.Dict, depth+1, path); err != nil {
			return err
		}
		if _, exists := i.decodedStreams[value]; exists {
			return nil
		}
		i.decodedStreams[value] = struct{}{}
		decoded, err := value.Decode()
		if err != nil {
			return workerError{failure: workerFailureInvalidResponse}
		}
		i.decompressedBytes += len(decoded)
		if i.decompressedBytes > i.limits.decompressedBytes {
			return workerError{failure: workerFailureProcessingLimit}
		}
	}
	return nil
}

func cloneReferencePath(path map[int]struct{}) map[int]struct{} {
	cloned := make(map[int]struct{}, len(path)+1)
	for number := range path {
		cloned[number] = struct{}{}
	}
	return cloned
}

func hasUsableText(value string) bool {
	for _, current := range value {
		if unicode.IsLetter(current) || unicode.IsNumber(current) || unicode.IsPunct(current) {
			return true
		}
	}
	return false
}

func looksEncryptedPDF(pdfBytes []byte) bool {
	return bytes.Contains(pdfBytes, []byte("/Encrypt"))
}

type workerRunner func(context.Context, []byte) (workerOutput, error)

func productionWorkerRunner(ctx context.Context, pdfBytes []byte) (workerOutput, error) {
	if err := ctx.Err(); err != nil {
		return workerOutput{}, err
	}
	executable, err := os.Executable()
	if err != nil {
		return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
	}
	return runPrivateWorkerProcessWithTempRoot(ctx, pdfBytes, executable, "", nil)
}

func runPrivateWorkerProcessWithTempRoot(
	ctx context.Context,
	pdfBytes []byte,
	executable string,
	tempRoot string,
	extraEnvironment []string,
) (workerOutput, error) {
	tempDirectory, err := os.MkdirTemp(tempRoot, "japanese-law-mcp-hanreipdf-worker-")
	if err != nil {
		return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
	}
	defer func() {
		_ = os.RemoveAll(tempDirectory)
	}()
	environment := []string{
		privateWorkerEnv + "=" + privateWorkerModePDF,
		privateWorkerTempEnv + "=" + tempDirectory,
	}
	environment = append(environment, extraEnvironment...)
	return runWorkerProcess(
		ctx,
		pdfBytes,
		executable,
		environment,
	)
}

func runWorkerProcess(
	ctx context.Context,
	pdfBytes []byte,
	executable string,
	environment []string,
) (workerOutput, error) {
	if err := ctx.Err(); err != nil {
		return workerOutput{}, err
	}
	if executable == "" || len(environment) == 0 {
		return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
	}
	// SOT-IF-071: os.Executable が返す同一 binary と固定引数だけを起動する。
	command := exec.CommandContext(ctx, executable)
	command.Env = append([]string(nil), environment...)
	command.Stdin = bytes.NewReader(pdfBytes)
	stdout := &limitedWorkerBuffer{limit: maximumWorkerStdoutBytes}
	command.Stdout = stdout
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
	}
	waited := make(chan error, 1)
	go func() {
		waited <- command.Wait()
	}()
	select {
	case waitErr := <-waited:
		if waitErr != nil || stdout.exceeded {
			return workerOutput{}, workerError{failure: workerFailureProcessingLimit}
		}
		return decodeWorkerEnvelope(stdout.Bytes())
	case <-ctx.Done():
		_ = command.Process.Kill()
		<-waited
		return workerOutput{}, ctx.Err()
	}
}

type limitedWorkerBuffer struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (b *limitedWorkerBuffer) Write(value []byte) (int, error) {
	if b.buffer.Len()+len(value) > b.limit {
		b.exceeded = true
		remaining := b.limit - b.buffer.Len()
		if remaining > 0 {
			_, _ = b.buffer.Write(value[:remaining])
		}
		return len(value), nil
	}
	return b.buffer.Write(value)
}

func (b *limitedWorkerBuffer) Bytes() []byte {
	return bytes.Clone(b.buffer.Bytes())
}

func decodeWorkerEnvelope(encoded []byte) (workerOutput, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var envelope workerEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
	}
	if envelope.Output != nil && envelope.Failure == "" {
		return *envelope.Output, nil
	}
	if envelope.Output == nil && validWorkerFailure(envelope.Failure) {
		return workerOutput{}, workerError{failure: envelope.Failure}
	}
	return workerOutput{}, workerError{failure: workerFailureInvalidResponse}
}

func validWorkerFailure(failure workerFailure) bool {
	return failure == workerFailureInvalidResponse ||
		failure == workerFailureResponseTooLarge ||
		failure == workerFailureProcessingLimit ||
		failure == workerFailureUnsafeContent
}
