package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestAddDiagnosticsWritesPublicFieldsOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		dependencies Dependencies
		arguments    map[string]any
		want         diagnosticEvent
		wantLeak     string
	}{
		{
			name:         "success",
			dependencies: Dependencies{SearchLaws: stubSearchLawsPort{}},
			arguments:    map[string]any{"query": "民法"},
			want: diagnosticEvent{
				Component: "mcp",
				Operation: "search_laws",
			},
			wantLeak: "民法",
		},
		{
			name: "invalid argument",
			dependencies: Dependencies{
				SearchLaws: stubSearchLawsPort{},
			},
			arguments: map[string]any{
				"query":   "秘密の検索語",
				"unknown": true,
			},
			want: diagnosticEvent{
				Component: "mcp",
				Operation: "search_laws",
				ErrorCode: "invalid_argument",
			},
			wantLeak: "秘密の検索語",
		},
		{
			name: "source error",
			dependencies: Dependencies{
				SearchLaws: errorSearchLawsPort{},
			},
			arguments: map[string]any{"query": "外部エラー時の秘密検索語"},
			want: diagnosticEvent{
				Component: "mcp",
				Operation: "search_laws",
				ErrorCode: "source_busy",
			},
			wantLeak: "外部エラー時の秘密検索語",
		},
		{
			name: "unified query invalid argument",
			dependencies: Dependencies{
				QueryLegalInformation: &recordingQueryLegalInformationPort{},
			},
			arguments: map[string]any{
				"query":   "診断へ出してはいけない統合照会",
				"unknown": true,
			},
			want: diagnosticEvent{
				Component: "mcp",
				Operation: "query_legal_information",
				ErrorCode: "invalid_argument",
			},
			wantLeak: "診断へ出してはいけない統合照会",
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			serverTransport, clientTransport := sdk.NewInMemoryTransports()
			var diagnostics bytes.Buffer
			server := NewServerWithDependencies(
				"test-version",
				testCase.dependencies,
			)
			if err := AddDiagnostics(server, &diagnostics); err != nil {
				t.Fatalf("AddDiagnostics() error = %v", err)
			}
			serverResult := make(chan error, 1)
			go func() {
				serverResult <- server.Run(ctx, serverTransport)
			}()

			client := sdk.NewClient(
				&sdk.Implementation{Name: "test-client", Version: "test-version"},
				nil,
			)
			session, err := client.Connect(ctx, clientTransport, nil)
			if err != nil {
				t.Fatalf("Connect() error = %v", err)
			}
			result, err := session.CallTool(ctx, &sdk.CallToolParams{
				Name:      testCase.want.Operation,
				Arguments: testCase.arguments,
			})
			if err != nil {
				t.Fatalf("CallTool() error = %v", err)
			}
			if testCase.want.ErrorCode == "" && result.IsError {
				t.Fatalf("CallTool() = error, want success: %#v", result)
			}
			if testCase.want.ErrorCode != "" && !result.IsError {
				t.Fatalf("CallTool() = success, want error: %#v", result)
			}
			if err := session.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			select {
			case runErr := <-serverResult:
				if runErr != nil {
					t.Fatalf("server run error = %v", runErr)
				}
			case <-ctx.Done():
				t.Fatalf("server shutdown wait error = %v", ctx.Err())
			}

			events := parseDiagnosticsTestEvents(t, diagnostics.String())
			if len(events) != 1 {
				t.Fatalf("diagnostics = %q, want 1 line", diagnostics.String())
			}
			if events[0] != testCase.want {
				t.Fatalf("diagnosticEvent = %#v, want %#v", events[0], testCase.want)
			}
			if strings.Contains(diagnostics.String(), testCase.wantLeak) {
				t.Fatalf("診断出力に入力値が含まれています: %q", diagnostics.String())
			}
			if strings.Contains(diagnostics.String(), "\"unknown\"") {
				t.Fatalf("診断出力に未許可項目名が含まれています: %q", diagnostics.String())
			}
		})
	}
}

func TestAddDiagnosticsHidesUnknownToolName(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	var diagnostics bytes.Buffer
	server := NewServer("test-version")
	if err := AddDiagnostics(server, &diagnostics); err != nil {
		t.Fatalf("AddDiagnostics() error = %v", err)
	}
	serverResult := make(chan error, 1)
	go func() {
		serverResult <- server.Run(ctx, serverTransport)
	}()

	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-client", Version: "test-version"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	_, err = session.CallTool(ctx, &sdk.CallToolParams{
		Name:      "unknown-secret-tool",
		Arguments: map[string]any{"query": "秘密値"},
	})
	if err == nil {
		t.Fatal("CallTool() error = nil, want protocol error")
	}
	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case runErr := <-serverResult:
		if runErr != nil {
			t.Fatalf("server run error = %v", runErr)
		}
	case <-ctx.Done():
		t.Fatalf("server shutdown wait error = %v", ctx.Err())
	}

	events := parseDiagnosticsTestEvents(t, diagnostics.String())
	if len(events) != 1 {
		t.Fatalf("diagnostics = %q, want 1 line", diagnostics.String())
	}
	want := diagnosticEvent{
		Component: "mcp",
		Operation: diagnosticToolCallMethod,
		ErrorCode: "internal_error",
	}
	if events[0] != want {
		t.Fatalf("diagnosticEvent = %#v, want %#v", events[0], want)
	}
	if strings.Contains(diagnostics.String(), "unknown-secret-tool") {
		t.Fatalf("診断出力に未知ツール名が含まれています: %q", diagnostics.String())
	}
	if strings.Contains(diagnostics.String(), "秘密値") {
		t.Fatalf("診断出力に入力値が含まれています: %q", diagnostics.String())
	}
}

func TestAddDiagnosticsIgnoresWriterFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		deps   Dependencies
		invoke func(context.Context, *sdk.ClientSession) error
	}{
		{
			name: "success",
			deps: Dependencies{SearchLaws: stubSearchLawsPort{}},
			invoke: func(ctx context.Context, session *sdk.ClientSession) error {
				result, err := session.CallTool(ctx, &sdk.CallToolParams{
					Name:      "search_laws",
					Arguments: map[string]any{"query": "民法"},
				})
				if err != nil {
					return err
				}
				if result == nil || result.IsError {
					return fmt.Errorf("success result = %#v", result)
				}
				return nil
			},
		},
		{
			name: "public error",
			deps: Dependencies{SearchLaws: stubSearchLawsPort{}},
			invoke: func(ctx context.Context, session *sdk.ClientSession) error {
				result, err := session.CallTool(ctx, &sdk.CallToolParams{
					Name: "search_laws",
					Arguments: map[string]any{
						"query":   "秘密の検索語",
						"unknown": true,
					},
				})
				if err != nil {
					return err
				}
				if result == nil || !result.IsError {
					return fmt.Errorf("public error result = %#v", result)
				}
				return nil
			},
		},
		{
			name: "protocol error and server continuity",
			deps: Dependencies{SearchLaws: stubSearchLawsPort{}},
			invoke: func(ctx context.Context, session *sdk.ClientSession) error {
				_, err := session.CallTool(ctx, &sdk.CallToolParams{
					Name:      "unknown-secret-tool",
					Arguments: map[string]any{"query": "秘密値"},
				})
				if err == nil {
					return errors.New("protocol error = nil")
				}
				result, callErr := session.CallTool(ctx, &sdk.CallToolParams{
					Name:      "search_laws",
					Arguments: map[string]any{"query": "民法"},
				})
				if callErr != nil {
					return callErr
				}
				if result == nil || result.IsError {
					return fmt.Errorf("continuity result = %#v", result)
				}
				return nil
			},
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			serverTransport, clientTransport := sdk.NewInMemoryTransports()
			server := NewServerWithDependencies("test-version", testCase.deps)
			if err := AddDiagnostics(server, failingDiagnosticsWriter{}); err != nil {
				t.Fatalf("AddDiagnostics() error = %v", err)
			}
			serverResult := make(chan error, 1)
			go func() {
				serverResult <- server.Run(ctx, serverTransport)
			}()

			client := sdk.NewClient(
				&sdk.Implementation{Name: "test-client", Version: "test-version"},
				nil,
			)
			session, err := client.Connect(ctx, clientTransport, nil)
			if err != nil {
				t.Fatalf("Connect() error = %v", err)
			}
			if err := testCase.invoke(ctx, session); err != nil {
				t.Fatalf("invoke error = %v", err)
			}
			if err := session.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			select {
			case runErr := <-serverResult:
				if runErr != nil {
					t.Fatalf("server run error = %v", runErr)
				}
			case <-ctx.Done():
				t.Fatalf("server shutdown wait error = %v", ctx.Err())
			}
		})
	}
}

func TestAddDiagnosticsSerializesConcurrentWrites(t *testing.T) {
	t.Parallel()

	const callCount = 32
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	diagnostics := &overlapDetectingWriter{}
	server := NewServerWithDependencies(
		"test-version",
		Dependencies{SearchLaws: stubSearchLawsPort{}},
	)
	if err := AddDiagnostics(server, diagnostics); err != nil {
		t.Fatalf("AddDiagnostics() error = %v", err)
	}
	serverResult := make(chan error, 1)
	go func() {
		serverResult <- server.Run(ctx, serverTransport)
	}()

	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-client", Version: "test-version"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	var waitGroup sync.WaitGroup
	callErrors := make(chan error, callCount)
	for index := 0; index < callCount; index++ {
		index := index
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			result, callErr := session.CallTool(ctx, &sdk.CallToolParams{
				Name: "search_laws",
				Arguments: map[string]any{
					"query": fmt.Sprintf("秘密の検索語-%d", index),
				},
			})
			if callErr != nil {
				callErrors <- callErr
				return
			}
			if result == nil || result.IsError {
				callErrors <- fmt.Errorf("予期しない tool result: %#v", result)
			}
		}()
	}
	waitGroup.Wait()
	close(callErrors)
	for callErr := range callErrors {
		t.Errorf("CallTool() error = %v", callErr)
	}

	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case runErr := <-serverResult:
		if runErr != nil {
			t.Fatalf("server run error = %v", runErr)
		}
	case <-ctx.Done():
		t.Fatalf("server shutdown wait error = %v", ctx.Err())
	}

	if diagnostics.overlapped.Load() {
		t.Fatal("SOT-ARCH-008: 診断行の Write が並行して実行されました")
	}
	events := parseDiagnosticsTestEvents(t, diagnostics.String())
	if len(events) != callCount {
		t.Fatalf("diagnostics event count = %d, want %d", len(events), callCount)
	}
	for _, event := range events {
		want := diagnosticEvent{Component: "mcp", Operation: "search_laws"}
		if event != want {
			t.Fatalf("diagnosticEvent = %#v, want %#v", event, want)
		}
	}
	if strings.Contains(diagnostics.String(), "秘密の検索語") {
		t.Fatalf("診断出力に入力値が含まれています: %q", diagnostics.String())
	}
}

func TestAddDiagnosticsRejectsNilInputs(t *testing.T) {
	t.Parallel()

	if err := AddDiagnostics(nil, &bytes.Buffer{}); err == nil {
		t.Fatal("server=nil を拒否しません")
	}
	if err := AddDiagnostics(NewServer("test-version"), nil); err == nil {
		t.Fatal("output=nil を拒否しません")
	}
}

func TestDiagnosticNormalizationFailsClosed(t *testing.T) {
	t.Parallel()

	knownRequest := &sdk.CallToolRequest{
		Params: &sdk.CallToolParamsRaw{Name: "search_laws"},
	}
	unknownRequest := &sdk.CallToolRequest{
		Params: &sdk.CallToolParamsRaw{Name: "unknown_secret_tool"},
	}
	result := &sdk.CallToolResult{}
	var typedNilResult *sdk.CallToolResult

	operationTests := []struct {
		name    string
		request sdk.Request
		result  sdk.Result
		err     error
		want    string
	}{
		{
			name:    "registered tool",
			request: knownRequest,
			result:  result,
			want:    "search_laws",
		},
		{
			name:    "unregistered tool",
			request: unknownRequest,
			result:  result,
			want:    diagnosticToolCallMethod,
		},
		{
			name:    "protocol error",
			request: knownRequest,
			err:     errors.New("protocol error"),
			want:    "search_laws",
		},
		{
			name:    "typed nil result",
			request: knownRequest,
			result:  typedNilResult,
			want:    "search_laws",
		},
	}
	for _, testCase := range operationTests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := diagnosticOperation(
				testCase.request,
				testCase.result,
				testCase.err,
			); got != testCase.want {
				t.Fatalf("diagnosticOperation() = %q, want %q", got, testCase.want)
			}
		})
	}

	errorTests := []struct {
		name   string
		result sdk.Result
		err    error
		want   string
	}{
		{
			name: "success",
			result: &sdk.CallToolResult{
				IsError: false,
			},
		},
		{
			name: "public error",
			result: &sdk.CallToolResult{
				IsError: true,
				Content: []sdk.Content{
					&sdk.TextContent{
						Text: `{"code":"not_found","message":"固定値","retryable":false}`,
					},
				},
			},
			want: "not_found",
		},
		{
			name: "protocol error",
			err:  errors.New("protocol error"),
			want: "internal_error",
		},
		{
			name:   "typed nil result",
			result: typedNilResult,
			want:   "internal_error",
		},
		{
			name: "unknown public code",
			result: &sdk.CallToolResult{
				IsError: true,
				Content: []sdk.Content{
					&sdk.TextContent{Text: `{"code":"secret_code"}`},
				},
			},
			want: "internal_error",
		},
		{
			name: "multiple contents",
			result: &sdk.CallToolResult{
				IsError: true,
				Content: []sdk.Content{
					&sdk.TextContent{Text: `{"code":"not_found"}`},
					&sdk.TextContent{Text: `{"code":"source_busy"}`},
				},
			},
			want: "internal_error",
		},
	}
	for _, testCase := range errorTests {
		testCase := testCase
		t.Run("error code/"+testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := diagnosticErrorCode(
				testCase.result,
				testCase.err,
			); string(got) != testCase.want {
				t.Fatalf("diagnosticErrorCode() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestAddDiagnosticsSkipsNonToolCallMethods(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	var diagnostics bytes.Buffer
	server := NewServerWithDependencies(
		"test-version",
		Dependencies{SearchLaws: stubSearchLawsPort{}},
	)
	if err := AddDiagnostics(server, &diagnostics); err != nil {
		t.Fatalf("AddDiagnostics() error = %v", err)
	}
	serverResult := make(chan error, 1)
	go func() {
		serverResult <- server.Run(ctx, serverTransport)
	}()

	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-client", Version: "test-version"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if _, err := session.ListTools(ctx, nil); err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if diagnostics.Len() != 0 {
		t.Fatalf("non tools/call diagnostics = %q, want empty", diagnostics.String())
	}
	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case runErr := <-serverResult:
		if runErr != nil {
			t.Fatalf("server run error = %v", runErr)
		}
	case <-ctx.Done():
		t.Fatalf("server shutdown wait error = %v", ctx.Err())
	}
}

type failingDiagnosticsWriter struct{}

func (failingDiagnosticsWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

type overlapDetectingWriter struct {
	active     atomic.Int32
	overlapped atomic.Bool
	mu         sync.Mutex
	buffer     bytes.Buffer
}

func (w *overlapDetectingWriter) Write(value []byte) (int, error) {
	if w.active.Add(1) > 1 {
		w.overlapped.Store(true)
	}
	defer w.active.Add(-1)
	time.Sleep(time.Millisecond)

	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(value)
}

func (w *overlapDetectingWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.String()
}

func parseDiagnosticsTestEvents(t *testing.T, diagnostics string) []diagnosticEvent {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(diagnostics), "\n")
	events := make([]diagnosticEvent, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event diagnosticEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("diagnostics JSON error = %v, line=%q", err, line)
		}
		var object map[string]any
		if err := json.Unmarshal([]byte(line), &object); err != nil {
			t.Fatalf("diagnostics object error = %v, line=%q", err, line)
		}
		for key := range object {
			if key != "component" && key != "operation" && key != "errorCode" {
				t.Fatalf("禁止された診断項目 %q が含まれています: %q", key, line)
			}
		}
		events = append(events, event)
	}
	return events
}
