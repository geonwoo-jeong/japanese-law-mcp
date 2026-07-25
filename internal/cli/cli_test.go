package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/config"
	"github.com/rogpeppe/go-internal/testscript"
	"golang.org/x/tools/txtar"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"japanese-law-mcp-test": runTestCommand,
	})
}

func TestCLIContract(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:                 "testdata",
		RequireExplicitExec: true,
		RequireUniqueNames:  true,
		Setup: func(env *testscript.Env) error {
			env.Vars = filterScriptEnvironment(env.Vars)
			env.Setenv("XDG_CONFIG_HOME", filepath.Join(env.WorkDir, "config-home"))
			coverDir := filepath.Join(env.WorkDir, "gocoverdir")
			if err := os.MkdirAll(coverDir, 0o750); err != nil {
				return err
			}
			env.Setenv("GOCOVERDIR", coverDir)
			env.Setenv("JAPANESE_LAW_MCP_TEST_USER_CONFIG_DIR", filepath.Join(env.WorkDir, "config-home"))
			return nil
		},
	})
}

func TestFilterScriptEnvironment(t *testing.T) {
	t.Parallel()

	input := []string{
		"WORK=/work",
		"PATH=/bin",
		"TMPDIR=/tmp",
		"HOME=/home",
		"exe=",
		"GOTRACEBACK=system",
		"JAPANESE_LAW_MCP_TRANSPORT=stdio",
		"AWS_PROFILE=dev",
	}

	got := filterScriptEnvironment(input)

	for _, unexpected := range []string{
		"JAPANESE_LAW_MCP_TRANSPORT=stdio",
		"AWS_PROFILE=dev",
	} {
		if slices.Contains(got, unexpected) {
			t.Fatalf("SOT-ENG-015: 不要な環境変数が残っています: %q", unexpected)
		}
	}
	for _, expected := range []string{
		"WORK=/work",
		"PATH=/bin",
		"TMPDIR=/tmp",
		"HOME=/home",
		"exe=",
		"GOTRACEBACK=system",
	} {
		if !slices.Contains(got, expected) {
			t.Fatalf("SOT-ENG-015: 必要な環境変数が失われました: %q", expected)
		}
	}
}

func TestTxtarScenariosReferenceSOT(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob(filepath.Join("testdata", "*.txtar"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("SOT-ENG-015: txtar シナリオがありません")
	}

	sotPattern := regexp.MustCompile(`SOT-[A-Z]+-[0-9]{3}`)
	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()

			archive, err := txtar.ParseFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !sotPattern.Match(archive.Comment) {
				t.Fatalf("SOT-ENG-015: SOT ID がありません: %s", path)
			}
		})
	}
}

func TestExecuteClassifiesResults(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args       []string
		runner     Runner
		wantCode   int
		wantStderr string
	}{
		"成功": {
			runner:   func(context.Context, config.Config) error { return nil },
			wantCode: ExitSuccess,
		},
		"設定エラー": {
			args:       []string{"--transport=invalid"},
			runner:     func(context.Context, config.Config) error { return nil },
			wantCode:   ExitUsage,
			wantStderr: "設定",
		},
		"実行エラー": {
			runner:     func(context.Context, config.Config) error { return errors.New("テスト用の内部エラー") },
			wantCode:   ExitRuntime,
			wantStderr: "サーバーを実行できません",
		},
		"位置引数": {
			args:       []string{"unexpected"},
			runner:     func(context.Context, config.Config) error { return nil },
			wantCode:   ExitUsage,
			wantStderr: "位置引数",
		},
		"未知のフラグ": {
			args:       []string{"--unknown"},
			runner:     func(context.Context, config.Config) error { return nil },
			wantCode:   ExitUsage,
			wantStderr: "--help",
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Execute(Options{
				Context:       context.Background(),
				Args:          test.args,
				Stdin:         strings.NewReader(""),
				Stdout:        &stdout,
				Stderr:        &stderr,
				Version:       "test-version",
				UserConfigDir: fixedConfigDir(t.TempDir()),
				Run:           test.runner,
			})

			if code != test.wantCode {
				t.Fatalf("SOT-IF-021: 終了コード = %d、期待値 = %d、stderr = %q", code, test.wantCode, stderr.String())
			}
			if test.wantStderr != "" && !strings.Contains(stderr.String(), test.wantStderr) {
				t.Fatalf("SOT-IF-021: stderr = %q、期待する内容 = %q", stderr.String(), test.wantStderr)
			}
			if strings.Contains(stderr.String(), "Usage:") || strings.Contains(stderr.String(), "unknown flag") {
				t.Fatalf("開発原則 7: 英語のエラーを出力した: %q", stderr.String())
			}
		})
	}
}

func TestExecutePassesContextAndConfigToRunner(t *testing.T) {
	t.Parallel()

	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "value")
	var receivedContext context.Context
	var receivedConfig config.Config

	code := Execute(Options{
		Context:       ctx,
		Args:          []string{"--request-timeout=8s"},
		Stdin:         strings.NewReader(""),
		Stdout:        io.Discard,
		Stderr:        io.Discard,
		Version:       "test-version",
		UserConfigDir: fixedConfigDir(t.TempDir()),
		Run: func(ctx context.Context, cfg config.Config) error {
			receivedContext = ctx
			receivedConfig = cfg
			return nil
		},
	})

	if code != ExitSuccess {
		t.Fatalf("SOT-IF-021: 終了コード = %d", code)
	}
	if receivedContext.Value(contextKey{}) != "value" {
		t.Fatal("SOT-ENG-014: context が Runner に渡されていません")
	}
	if receivedConfig.RequestTimeout().String() != "8s" {
		t.Fatalf("SOT-IF-020: requestTimeout = %s", receivedConfig.RequestTimeout())
	}
}

func TestExecuteTreatsCancellationAsSuccess(t *testing.T) {
	t.Parallel()

	code := Execute(Options{
		Context:       context.Background(),
		Stdin:         strings.NewReader(""),
		Stdout:        io.Discard,
		Stderr:        io.Discard,
		Version:       "test-version",
		UserConfigDir: fixedConfigDir(t.TempDir()),
		Run: func(context.Context, config.Config) error {
			return context.Canceled
		},
	})

	if code != ExitSuccess {
		t.Fatalf("SOT-IF-021: 終了コード = %d、期待値 = %d", code, ExitSuccess)
	}
}

func TestHelpIsJapaneseAndDoesNotLoadConfiguration(t *testing.T) {
	t.Setenv("JAPANESE_LAW_MCP_TRANSPORT", "invalid")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute(Options{
		Context:       context.Background(),
		Args:          []string{"--help"},
		Stdin:         strings.NewReader(""),
		Stdout:        &stdout,
		Stderr:        &stderr,
		Version:       "test-version",
		UserConfigDir: fixedConfigDir(t.TempDir()),
		Run: func(context.Context, config.Config) error {
			return errors.New("呼び出してはなりません")
		},
	})

	if code != ExitSuccess {
		t.Fatalf("SOT-IF-019: 終了コード = %d、stderr = %q", code, stderr.String())
	}
	for _, want := range []string{"使用方法:", "利用可能なコマンド:", "フラグ:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("開発原則 7: help = %q、期待する内容 = %q", stdout.String(), want)
		}
	}
	for _, unwanted := range []string{"Usage:", "Available Commands:", "Flags:"} {
		if strings.Contains(stdout.String(), unwanted) {
			t.Fatalf("開発原則 7: help に英語の見出しがあります: %q", unwanted)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("SOT-IF-019: stderr = %q", stderr.String())
	}
}

func TestHelpCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args       []string
		wantCode   int
		wantOutput string
		wantError  string
	}{
		"サブコマンド": {
			args:       []string{"help", "version"},
			wantCode:   ExitSuccess,
			wantOutput: "バージョン情報を表示します",
		},
		"未知のコマンド": {
			args:      []string{"help", "unknown"},
			wantCode:  ExitUsage,
			wantError: "見つかりません",
		},
		"引数過多": {
			args:      []string{"help", "version", "extra"},
			wantCode:  ExitUsage,
			wantError: "一つだけ",
		},
		"version の位置引数": {
			args:      []string{"version", "extra"},
			wantCode:  ExitUsage,
			wantError: "位置引数",
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Execute(Options{
				Context:       context.Background(),
				Args:          test.args,
				Stdin:         strings.NewReader(""),
				Stdout:        &stdout,
				Stderr:        &stderr,
				Version:       "test-version",
				UserConfigDir: fixedConfigDir(t.TempDir()),
				Run:           func(context.Context, config.Config) error { return nil },
			})

			if code != test.wantCode {
				t.Fatalf("SOT-IF-021: 終了コード = %d、期待値 = %d", code, test.wantCode)
			}
			if test.wantOutput != "" && !strings.Contains(stdout.String(), test.wantOutput) {
				t.Fatalf("SOT-IF-019: stdout = %q、期待する内容 = %q", stdout.String(), test.wantOutput)
			}
			if test.wantError != "" && !strings.Contains(stderr.String(), test.wantError) {
				t.Fatalf("SOT-IF-019: stderr = %q、期待する内容 = %q", stderr.String(), test.wantError)
			}
		})
	}
}

func TestExecuteHandlesMissingOptionsAndOutputFailure(t *testing.T) {
	t.Parallel()

	var missingContextStderr bytes.Buffer
	if code := Execute(Options{
		Args:    []string{"version"},
		Stderr:  &missingContextStderr,
		Version: "test-version",
	}); code != ExitRuntime {
		t.Fatalf("SOT-ENG-010: 終了コード = %d、stderr = %q", code, missingContextStderr.String())
	}
	if !strings.Contains(missingContextStderr.String(), "実行コンテキスト") {
		t.Fatalf("SOT-ENG-010: stderr = %q", missingContextStderr.String())
	}

	var stderr bytes.Buffer
	if code := Execute(Options{
		Context: context.Background(),
		Args:    []string{"version"},
		Stdout:  failingWriter{},
		Stderr:  &stderr,
		Version: "test-version",
	}); code != ExitRuntime {
		t.Fatalf("SOT-IF-021: 終了コード = %d、stderr = %q", code, stderr.String())
	}

	reader := &emptyReader{}
	if _, err := reader.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("emptyReader.Read() のエラー = %v", err)
	}
	if (&commandError{message: "テスト"}).Error() != "テスト" {
		t.Fatal("commandError.Error() がメッセージを返しません")
	}
}

func runTestCommand() {
	userConfigDir := func() (string, error) {
		if dir := os.Getenv("JAPANESE_LAW_MCP_TEST_USER_CONFIG_DIR"); dir != "" {
			return dir, nil
		}
		return os.UserConfigDir()
	}
	runner := func(_ context.Context, cfg config.Config) error {
		if os.Getenv("JAPANESE_LAW_MCP_TEST_RUNTIME_ERROR") == "1" {
			return errors.New("テスト用の内部エラー")
		}
		snapshot := struct {
			Transport      string   `json:"transport"`
			RequestTimeout string   `json:"requestTimeout"`
			ListenAddress  string   `json:"listenAddress"`
			AllowedOrigins []string `json:"allowedOrigins"`
			Diagnostics    bool     `json:"diagnostics"`
		}{
			Transport:      string(cfg.Transport()),
			RequestTimeout: cfg.RequestTimeout().String(),
			ListenAddress:  cfg.ListenAddress(),
			AllowedOrigins: cfg.AllowedOrigins(),
			Diagnostics:    cfg.Diagnostics(),
		}
		return json.NewEncoder(os.Stdout).Encode(snapshot)
	}

	code := Execute(Options{
		Context:       context.Background(),
		Args:          os.Args[1:],
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		Version:       "test-version",
		UserConfigDir: userConfigDir,
		Run:           runner,
	})
	os.Exit(code)
}

func fixedConfigDir(dir string) func() (string, error) {
	return func() (string, error) {
		return dir, nil
	}
}

func filterScriptEnvironment(vars []string) []string {
	allowed := map[string]struct{}{
		"WORK":        {},
		"PATH":        {},
		"GOTRACEBACK": {},
		"HOME":        {},
		"USERPROFILE": {},
		"TMPDIR":      {},
		"TMP":         {},
		"devnull":     {},
		"/":           {},
		":":           {},
		"$":           {},
		"exe":         {},
		"GORACE":      {},
	}

	filtered := make([]string, 0, len(vars))
	for _, entry := range vars {
		name, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, exists := allowed[name]; exists {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("書込みに失敗しました")
}

func ExampleExecute() {
	code := Execute(Options{
		Context:       context.Background(),
		Args:          []string{"version"},
		Stdin:         strings.NewReader(""),
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		Version:       "test-version",
		UserConfigDir: fixedConfigDir("."),
		Run: func(context.Context, config.Config) error {
			return nil
		},
	})
	fmt.Println("終了コード:", code)

	// Output:
	// japanese-law-mcp test-version
	// 終了コード: 0
}
