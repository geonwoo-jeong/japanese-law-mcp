package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	keyTransport      = "transport"
	keyRequestTimeout = "requestTimeout"
	keyListenAddress  = "listenAddress"
	keyAllowedOrigins = "allowedOrigins"
	keyDiagnostics    = "diagnostics"
)

// LoadOptions は、一回の設定読込みに使用する外部入力を保持する。
type LoadOptions struct {
	Flags         *pflag.FlagSet
	ConfigFile    string
	UserConfigDir func() (string, error)
}

// Load は、すべての設定入力を優先順位どおりに集約して検証する。
func Load(options LoadOptions) (Config, error) {
	settings := viper.NewWithOptions(viper.KeyDelimiter("::"))
	settings.AllowEmptyEnv(true)

	if err := readConfigFile(settings, options); err != nil {
		return Config{}, err
	}
	fileValues, err := loadStructuredFileValues(settings.ConfigFileUsed())
	if err != nil {
		return Config{}, NewValidationError(err)
	}
	if err := validateFileSettings(settings); err != nil {
		return Config{}, err
	}

	setDefaults(settings)
	if err := bindKnownEnvironment(settings); err != nil {
		return Config{}, err
	}
	if err := bindFlags(settings, options.Flags); err != nil {
		return Config{}, err
	}

	values, err := decodeValues(settings)
	if err != nil {
		return Config{}, err
	}
	values.Providers = fileValues.providers
	values.ProviderRoutes = fileValues.providerRoutes
	values.ExtensionPacks = fileValues.extensionPacks
	return New(values)
}

func readConfigFile(settings *viper.Viper, options LoadOptions) error {
	if options.ConfigFile != "" {
		if !supportedConfigExtension(filepath.Ext(options.ConfigFile)) {
			return fmt.Errorf("指定した設定ファイルの形式には対応していません: %q", options.ConfigFile)
		}
		settings.SetConfigFile(options.ConfigFile)
		if err := settings.ReadInConfig(); err != nil {
			return fmt.Errorf("指定した設定ファイルを読み込めません: %q", options.ConfigFile)
		}
		return nil
	}

	userConfigDir := options.UserConfigDir
	if userConfigDir == nil {
		userConfigDir = os.UserConfigDir
	}
	dir, err := userConfigDir()
	if err != nil {
		return fmt.Errorf("利用者設定ディレクトリを確認できません")
	}
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("利用者設定ディレクトリが空です")
	}

	path := filepath.Join(dir, "japanese-law-mcp", "config.yaml")
	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("利用者設定ファイルを確認できません: %q", path)
	}

	settings.SetConfigFile(path)
	if err := settings.ReadInConfig(); err != nil {
		return fmt.Errorf("利用者設定ファイルを読み込めません: %q", path)
	}
	return nil
}

func supportedConfigExtension(extension string) bool {
	switch strings.ToLower(extension) {
	case ".yaml", ".yml", ".json", ".toml":
		return true
	default:
		return false
	}
}

func validateFileSettings(settings *viper.Viper) error {
	keys := settings.AllKeys()
	sort.Strings(keys)
	for _, key := range keys {
		if !knownConfigKey(key) {
			return fmt.Errorf("未知の設定項目 %q が指定されています", key)
		}
	}

	fileSettings := settings.AllSettings()
	for key, value := range fileSettings {
		switch canonicalConfigKey(key) {
		case keyTransport, keyRequestTimeout, keyListenAddress:
			if _, ok := value.(string); !ok {
				return fmt.Errorf("設定項目 %s の型が正しくありません", canonicalConfigKey(key))
			}
		case keyAllowedOrigins:
			if !isStringSlice(value) {
				return fmt.Errorf("設定項目 allowedOrigins の型が正しくありません")
			}
		case keyDiagnostics:
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("設定項目 diagnostics の型が正しくありません")
			}
		case keyProviders, keyProviderRoutes, keyExtensionPacks:
			// 構造化 namespace は、空 object と atomic object を保持する専用 decoder で検証する。
		}
	}
	return nil
}

func knownConfigKey(key string) bool {
	return canonicalConfigKey(topLevelConfigKey(key)) != ""
}

func canonicalConfigKey(key string) string {
	switch strings.ToLower(key) {
	case strings.ToLower(keyTransport):
		return keyTransport
	case strings.ToLower(keyRequestTimeout):
		return keyRequestTimeout
	case strings.ToLower(keyListenAddress):
		return keyListenAddress
	case strings.ToLower(keyAllowedOrigins):
		return keyAllowedOrigins
	case strings.ToLower(keyDiagnostics):
		return keyDiagnostics
	case strings.ToLower(keyProviders):
		return keyProviders
	case strings.ToLower(keyProviderRoutes):
		return keyProviderRoutes
	case strings.ToLower(keyExtensionPacks):
		return keyExtensionPacks
	default:
		return ""
	}
}

func topLevelConfigKey(key string) string {
	top, _, _ := strings.Cut(key, "::")
	return top
}

func isStringSlice(value any) bool {
	switch values := value.(type) {
	case []string:
		return true
	case []any:
		for _, item := range values {
			if _, ok := item.(string); !ok {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func setDefaults(settings *viper.Viper) {
	defaults := Default()
	settings.SetDefault(keyTransport, string(defaults.Transport()))
	settings.SetDefault(keyRequestTimeout, defaults.RequestTimeout().String())
	settings.SetDefault(keyListenAddress, defaults.ListenAddress())
	settings.SetDefault(keyAllowedOrigins, defaults.AllowedOrigins())
	settings.SetDefault(keyDiagnostics, defaults.Diagnostics())
}

func bindKnownEnvironment(settings *viper.Viper) error {
	bindings := [][2]string{
		{keyTransport, "JAPANESE_LAW_MCP_TRANSPORT"},
		{keyRequestTimeout, "JAPANESE_LAW_MCP_REQUEST_TIMEOUT"},
		{keyListenAddress, "JAPANESE_LAW_MCP_LISTEN_ADDRESS"},
		{keyAllowedOrigins, "JAPANESE_LAW_MCP_ALLOWED_ORIGINS"},
		{keyDiagnostics, "JAPANESE_LAW_MCP_DIAGNOSTICS"},
	}
	for _, binding := range bindings {
		if err := settings.BindEnv(binding[0], binding[1]); err != nil {
			return fmt.Errorf("環境変数を設定項目 %s に関連付けられません", binding[0])
		}
	}
	return nil
}

func bindFlags(settings *viper.Viper, flags *pflag.FlagSet) error {
	if flags == nil {
		return nil
	}
	bindings := [][2]string{
		{keyTransport, "transport"},
		{keyRequestTimeout, "request-timeout"},
		{keyListenAddress, "listen-address"},
		{keyAllowedOrigins, "allowed-origin"},
		{keyDiagnostics, "diagnostics"},
	}
	for _, binding := range bindings {
		flag := flags.Lookup(binding[1])
		if flag == nil {
			continue
		}
		if err := settings.BindPFlag(binding[0], flag); err != nil {
			return fmt.Errorf("フラグ --%s を設定項目 %s に関連付けられません", binding[1], binding[0])
		}
	}
	return nil
}

func decodeValues(settings *viper.Viper) (Values, error) {
	timeout, err := time.ParseDuration(settings.GetString(keyRequestTimeout))
	if err != nil {
		return Values{}, fmt.Errorf("設定項目 requestTimeout を duration として解釈できません")
	}

	origins, err := decodeOrigins(settings.Get(keyAllowedOrigins))
	if err != nil {
		return Values{}, err
	}

	diagnostics, err := strconv.ParseBool(settings.GetString(keyDiagnostics))
	if err != nil {
		return Values{}, fmt.Errorf("設定項目 diagnostics を boolean として解釈できません")
	}

	return Values{
		Transport:      settings.GetString(keyTransport),
		RequestTimeout: timeout,
		ListenAddress:  settings.GetString(keyListenAddress),
		AllowedOrigins: origins,
		Diagnostics:    diagnostics,
	}, nil
}

func decodeOrigins(value any) ([]string, error) {
	switch origins := value.(type) {
	case nil:
		return make([]string, 0), nil
	case string:
		if origins == "" {
			return make([]string, 0), nil
		}
		parts := strings.Split(origins, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			result = append(result, strings.TrimSpace(part))
		}
		return result, nil
	case []string:
		result := make([]string, len(origins))
		copy(result, origins)
		return result, nil
	case []any:
		result := make([]string, 0, len(origins))
		for _, origin := range origins {
			text, ok := origin.(string)
			if !ok {
				return nil, fmt.Errorf("設定項目 allowedOrigins を HTTPS Origin の配列として解釈できません")
			}
			result = append(result, text)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("設定項目 allowedOrigins を HTTPS Origin の配列として解釈できません")
	}
}
