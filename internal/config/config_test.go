package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	got := Default()

	if got.Transport() != TransportStdio {
		t.Fatalf("SOT-IF-005: transport = %q、期待値 = %q", got.Transport(), TransportStdio)
	}
	if got.RequestTimeout() != 30*time.Second {
		t.Fatalf("SOT-IF-005: requestTimeout = %s、期待値 = 30s", got.RequestTimeout())
	}
	if got.ListenAddress() != "127.0.0.1:8080" {
		t.Fatalf("SOT-IF-005: listenAddress = %q", got.ListenAddress())
	}
	if len(got.AllowedOrigins()) != 0 {
		t.Fatalf("SOT-IF-005: allowedOrigins = %v、期待値 = 空", got.AllowedOrigins())
	}
	if got.Diagnostics() {
		t.Fatal("SOT-IF-005: diagnostics = true、期待値 = false")
	}
}

func TestConfigOwnsAllowedOrigins(t *testing.T) {
	t.Parallel()

	origins := []string{"https://example.test"}
	got, err := New(Values{
		Transport:      string(TransportStreamableHTTP),
		RequestTimeout: 30 * time.Second,
		ListenAddress:  "127.0.0.1:8080",
		AllowedOrigins: origins,
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-015: New() のエラー = %v", err)
	}

	origins[0] = "https://changed.test"
	fromConfig := got.AllowedOrigins()
	fromConfig[0] = "https://also-changed.test"

	if !slices.Equal(got.AllowedOrigins(), []string{"https://example.test"}) {
		t.Fatalf("SOT-ARCH-015: 設定値が外部から変更された: %v", got.AllowedOrigins())
	}
}

func TestNewRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	valid := Values{
		Transport:      string(TransportStreamableHTTP),
		RequestTimeout: 30 * time.Second,
		ListenAddress:  "127.0.0.1:8080",
		AllowedOrigins: []string{"https://example.test"},
	}

	tests := map[string]Values{
		"トランスポート":         withTransport(valid, "invalid"),
		"タイムアウトの下限":       withTimeout(valid, 999*time.Millisecond),
		"タイムアウトの上限":       withTimeout(valid, 121*time.Second),
		"待受先":             withListenAddress(valid, "127.0.0.1"),
		"空白を含む待受先":        withListenAddress(valid, "invalid host:8080"),
		"HTTP Origin":     withOrigins(valid, "http://example.test"),
		"パスを含む Origin":    withOrigins(valid, "https://example.test/path"),
		"利用者情報を含む Origin": withOrigins(valid, "https://user@example.test"),
		"stdio の Origin":  withTransport(valid, string(TransportStdio)),
		"stdio の待受先変更":    withListenAddress(withTransport(valid, string(TransportStdio)), "127.0.0.1:9090"),
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := New(input); err == nil {
				t.Fatal("SOT-IF-005: New() のエラー = nil")
			}
		})
	}
}

func TestLoadUsesDefinedPrecedence(t *testing.T) {
	clearKnownEnvironment(t)

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	writeFile(t, configPath, `
transport: streamable-http
requestTimeout: 40s
listenAddress: 127.0.0.1:9090
allowedOrigins:
  - https://file.example
diagnostics: false
`)
	t.Setenv("JAPANESE_LAW_MCP_REQUEST_TIMEOUT", "20s")
	t.Setenv("JAPANESE_LAW_MCP_ALLOWED_ORIGINS", "https://env.example")

	flags := newFlagSet(t,
		"--request-timeout=10s",
		"--allowed-origin=https://flag.example",
		"--diagnostics",
	)
	got, err := Load(LoadOptions{
		Flags:         flags,
		ConfigFile:    configPath,
		UserConfigDir: fixedUserConfigDir(t.TempDir()),
	})
	if err != nil {
		t.Fatalf("SOT-IF-020: Load() のエラー = %v", err)
	}

	if got.Transport() != TransportStreamableHTTP {
		t.Fatalf("SOT-IF-020: transport = %q", got.Transport())
	}
	if got.RequestTimeout() != 10*time.Second {
		t.Fatalf("SOT-IF-020: requestTimeout = %s、期待値 = 10s", got.RequestTimeout())
	}
	if got.ListenAddress() != "127.0.0.1:9090" {
		t.Fatalf("SOT-IF-020: listenAddress = %q", got.ListenAddress())
	}
	if !slices.Equal(got.AllowedOrigins(), []string{"https://flag.example"}) {
		t.Fatalf("SOT-IF-020: allowedOrigins = %v", got.AllowedOrigins())
	}
	if !got.Diagnostics() {
		t.Fatal("SOT-IF-020: diagnostics = false、期待値 = true")
	}
}

func TestLoadReadsUserConfigWhenPresent(t *testing.T) {
	clearKnownEnvironment(t)

	dir := t.TempDir()
	configDir := filepath.Join(dir, "japanese-law-mcp")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(configDir, "config.yaml"), "requestTimeout: 12s\n")

	got, err := Load(LoadOptions{
		Flags:         newFlagSet(t),
		UserConfigDir: fixedUserConfigDir(dir),
	})
	if err != nil {
		t.Fatalf("SOT-IF-020: Load() のエラー = %v", err)
	}
	if got.RequestTimeout() != 12*time.Second {
		t.Fatalf("SOT-IF-020: requestTimeout = %s、期待値 = 12s", got.RequestTimeout())
	}
}

func TestLoadDoesNotSearchWorkingDirectory(t *testing.T) {
	clearKnownEnvironment(t)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config.yaml"), "requestTimeout: 12s\n")

	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("作業ディレクトリを復元できません: %v", err)
		}
	})

	got, err := Load(LoadOptions{
		Flags:         newFlagSet(t),
		UserConfigDir: fixedUserConfigDir(t.TempDir()),
	})
	if err != nil {
		t.Fatalf("SOT-IF-020: Load() のエラー = %v", err)
	}
	if got.RequestTimeout() != Default().RequestTimeout() {
		t.Fatalf("SOT-IF-020: 作業ディレクトリの設定を読み込んだ: %s", got.RequestTimeout())
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	clearKnownEnvironment(t)

	tests := map[string]struct {
		path    string
		content string
		want    string
	}{
		"ファイルなし": {
			path: "missing.yaml",
			want: "指定した設定ファイル",
		},
		"未知の項目": {
			path:    "unknown.yaml",
			content: "unknownKey: value\n",
			want:    "未知の設定項目",
		},
		"無効な値": {
			path:    "invalid.yaml",
			content: "requestTimeout: not-a-duration\n",
			want:    "requestTimeout",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, test.path)
			if test.content != "" {
				writeFile(t, path, test.content)
			}

			_, err := Load(LoadOptions{
				Flags:         newFlagSet(t),
				ConfigFile:    path,
				UserConfigDir: fixedUserConfigDir(t.TempDir()),
			})
			if err == nil {
				t.Fatal("SOT-IF-005 SOT-IF-020: Load() のエラー = nil")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("SOT-IF-005 SOT-IF-020: エラー = %q、期待する内容 = %q", err, test.want)
			}
		})
	}
}

func TestLoadDoesNotLeakStateBetweenRuns(t *testing.T) {
	clearKnownEnvironment(t)

	firstFlags := newFlagSet(t, "--request-timeout=9s")
	first, err := Load(LoadOptions{
		Flags:         firstFlags,
		UserConfigDir: fixedUserConfigDir(t.TempDir()),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Load(LoadOptions{
		Flags:         newFlagSet(t),
		UserConfigDir: fixedUserConfigDir(t.TempDir()),
	})
	if err != nil {
		t.Fatal(err)
	}

	if first.RequestTimeout() != 9*time.Second {
		t.Fatalf("SOT-ENG-014: 1 回目の requestTimeout = %s", first.RequestTimeout())
	}
	if second.RequestTimeout() != 30*time.Second {
		t.Fatalf("SOT-ENG-014: 2 回目の requestTimeout = %s", second.RequestTimeout())
	}
}

func TestLoadSupportedExplicitFormats(t *testing.T) {
	clearKnownEnvironment(t)

	tests := map[string]string{
		"config.yaml": "requestTimeout: 13s\n",
		"config.yml":  "requestTimeout: 13s\n",
		"config.json": "{\"requestTimeout\":\"13s\"}\n",
		"config.toml": "requestTimeout = \"13s\"\n",
	}

	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, name)
			writeFile(t, path, content)

			got, err := Load(LoadOptions{
				Flags:         newFlagSet(t),
				ConfigFile:    path,
				UserConfigDir: fixedUserConfigDir(t.TempDir()),
			})
			if err != nil {
				t.Fatalf("SOT-IF-020: Load() のエラー = %v", err)
			}
			if got.RequestTimeout() != 13*time.Second {
				t.Fatalf("SOT-IF-020: requestTimeout = %s", got.RequestTimeout())
			}
		})
	}
}

func TestLoadEnvironmentValues(t *testing.T) {
	clearKnownEnvironment(t)
	t.Setenv("JAPANESE_LAW_MCP_TRANSPORT", "streamable-http")
	t.Setenv("JAPANESE_LAW_MCP_ALLOWED_ORIGINS", " https://one.example , https://two.example ")
	t.Setenv("JAPANESE_LAW_MCP_DIAGNOSTICS", "true")

	got, err := Load(LoadOptions{
		Flags:         nil,
		UserConfigDir: fixedUserConfigDir(t.TempDir()),
	})
	if err != nil {
		t.Fatalf("SOT-IF-020: Load() のエラー = %v", err)
	}
	if !slices.Equal(got.AllowedOrigins(), []string{"https://one.example", "https://two.example"}) {
		t.Fatalf("SOT-IF-020: allowedOrigins = %v", got.AllowedOrigins())
	}
	if !got.Diagnostics() {
		t.Fatal("SOT-IF-020: diagnostics = false")
	}
}

func TestLoadEmptyOriginsEnvironmentOverridesFile(t *testing.T) {
	clearKnownEnvironment(t)

	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, `
transport: streamable-http
allowedOrigins:
  - https://file.example
`)
	t.Setenv("JAPANESE_LAW_MCP_ALLOWED_ORIGINS", "")

	got, err := Load(LoadOptions{
		Flags:         newFlagSet(t),
		ConfigFile:    path,
		UserConfigDir: fixedUserConfigDir(t.TempDir()),
	})
	if err != nil {
		t.Fatalf("SOT-IF-020: Load() のエラー = %v", err)
	}
	if len(got.AllowedOrigins()) != 0 {
		t.Fatalf("SOT-IF-020: allowedOrigins = %v、期待値 = 空", got.AllowedOrigins())
	}
}

func TestLoadRejectsFileTypeErrors(t *testing.T) {
	clearKnownEnvironment(t)

	tests := map[string]struct {
		name    string
		content string
		want    string
	}{
		"未対応の拡張子": {
			name:    "config.txt",
			content: "requestTimeout: 13s\n",
			want:    "対応していません",
		},
		"requestTimeout の型": {
			name:    "config.yaml",
			content: "requestTimeout: 13\n",
			want:    "requestTimeout",
		},
		"allowedOrigins の型": {
			name:    "config.yaml",
			content: "allowedOrigins: 13\n",
			want:    "allowedOrigins",
		},
		"diagnostics の型": {
			name:    "config.yaml",
			content: "diagnostics: enabled\n",
			want:    "diagnostics",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), test.name)
			writeFile(t, path, test.content)
			_, err := Load(LoadOptions{
				Flags:         newFlagSet(t),
				ConfigFile:    path,
				UserConfigDir: fixedUserConfigDir(t.TempDir()),
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("SOT-IF-020: エラー = %v、期待する内容 = %q", err, test.want)
			}
		})
	}
}

func TestLoadRejectsUserConfigErrors(t *testing.T) {
	clearKnownEnvironment(t)

	_, err := Load(LoadOptions{
		Flags: newFlagSet(t),
		UserConfigDir: func() (string, error) {
			return "", os.ErrPermission
		},
	})
	if err == nil || !strings.Contains(err.Error(), "利用者設定ディレクトリ") {
		t.Fatalf("SOT-IF-020: エラー = %v", err)
	}

	_, err = Load(LoadOptions{
		Flags:         newFlagSet(t),
		UserConfigDir: fixedUserConfigDir(""),
	})
	if err == nil || !strings.Contains(err.Error(), "利用者設定ディレクトリ") {
		t.Fatalf("SOT-IF-020: エラー = %v", err)
	}

	dir := t.TempDir()
	configDir := filepath.Join(dir, "japanese-law-mcp")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(configDir, "config.yaml"), "invalid: [\n")
	_, err = Load(LoadOptions{
		Flags:         newFlagSet(t),
		UserConfigDir: fixedUserConfigDir(dir),
	})
	if err == nil || !strings.Contains(err.Error(), "利用者設定ファイル") {
		t.Fatalf("SOT-IF-020: エラー = %v", err)
	}
}

func TestDecodeOriginsRejectsNonStringValues(t *testing.T) {
	t.Parallel()

	if _, err := decodeOrigins([]any{"https://example.test", 1}); err == nil {
		t.Fatal("SOT-IF-020: decodeOrigins() のエラー = nil")
	}
	if _, err := decodeOrigins(1); err == nil {
		t.Fatal("SOT-IF-020: decodeOrigins() のエラー = nil")
	}
}

func newFlagSet(t *testing.T, args ...string) *pflag.FlagSet {
	t.Helper()

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("transport", "", "")
	flags.Duration("request-timeout", 0, "")
	flags.String("listen-address", "", "")
	flags.StringArray("allowed-origin", nil, "")
	flags.Bool("diagnostics", false, "")
	if err := flags.Parse(args); err != nil {
		t.Fatalf("フラグを準備できません: %v", err)
	}
	return flags
}

func clearKnownEnvironment(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"JAPANESE_LAW_MCP_TRANSPORT",
		"JAPANESE_LAW_MCP_REQUEST_TIMEOUT",
		"JAPANESE_LAW_MCP_LISTEN_ADDRESS",
		"JAPANESE_LAW_MCP_ALLOWED_ORIGINS",
		"JAPANESE_LAW_MCP_DIAGNOSTICS",
	} {
		value, exists := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("環境変数 %s を解除できません: %v", key, err)
		}
		t.Cleanup(func() {
			var err error
			if exists {
				err = os.Setenv(key, value)
			} else {
				err = os.Unsetenv(key)
			}
			if err != nil {
				t.Errorf("環境変数 %s を復元できません: %v", key, err)
			}
		})
	}
}

func fixedUserConfigDir(dir string) func() (string, error) {
	return func() (string, error) {
		return dir, nil
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(strings.TrimPrefix(content, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
}

func withTransport(input Values, transport string) Values {
	input.Transport = transport
	return input
}

func withTimeout(input Values, timeout time.Duration) Values {
	input.RequestTimeout = timeout
	return input
}

func withListenAddress(input Values, address string) Values {
	input.ListenAddress = address
	return input
}

func withOrigins(input Values, origins ...string) Values {
	input.AllowedOrigins = origins
	return input
}
