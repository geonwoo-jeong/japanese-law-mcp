package config

import (
	"fmt"
	"time"
)

// Transport は、MCP サーバーが使用する通信方式を表す。
type Transport string

const (
	TransportStdio          Transport = "stdio"
	TransportStreamableHTTP Transport = "streamable-http"
)

// Values は、検証前の起動設定値を保持する。
type Values struct {
	Transport      string
	RequestTimeout time.Duration
	ListenAddress  string
	AllowedOrigins []string
	Diagnostics    bool
	Providers      map[string]ProviderConfig
	ProviderRoutes map[string]ProviderRoute
}

// Config は、検証済みで変更不能な起動設定を保持する。
type Config struct {
	transport      Transport
	requestTimeout time.Duration
	listenAddress  string
	allowedOrigins []string
	diagnostics    bool
	providers      map[string]ProviderConfig
	providerRoutes map[ProviderRouteKey]ProviderRoute
}

// Default は、SOT-IF-026 と SOT-IF-029 が定める既定の起動設定を返す。
func Default() Config {
	providers, err := resolveProviderConfigs(nil)
	if err != nil {
		panic("組込み provider 設定を構成できません")
	}
	providerRoutes, err := resolveProviderRoutes(nil)
	if err != nil {
		panic("組込み provider route を構成できません")
	}
	return Config{
		transport:      TransportStdio,
		requestTimeout: 30 * time.Second,
		listenAddress:  "127.0.0.1:8080",
		allowedOrigins: make([]string, 0),
		diagnostics:    false,
		providers:      providers,
		providerRoutes: providerRoutes,
	}
}

// New は、入力値を検証し、外部から変更できない Config を返す。
func New(values Values) (Config, error) {
	if err := validateValues(values); err != nil {
		return Config{}, fmt.Errorf(
			"設定を検証できません: %w",
			NewValidationError(err),
		)
	}
	providers, err := resolveProviderConfigs(values.Providers)
	if err != nil {
		return Config{}, fmt.Errorf(
			"設定を検証できません: %w",
			NewValidationError(err),
		)
	}
	providerRoutes, err := resolveProviderRoutes(values.ProviderRoutes)
	if err != nil {
		return Config{}, fmt.Errorf(
			"設定を検証できません: %w",
			NewValidationError(err),
		)
	}

	origins := make([]string, len(values.AllowedOrigins))
	copy(origins, values.AllowedOrigins)

	return Config{
		transport:      Transport(values.Transport),
		requestTimeout: values.RequestTimeout,
		listenAddress:  values.ListenAddress,
		allowedOrigins: origins,
		diagnostics:    values.Diagnostics,
		providers:      providers,
		providerRoutes: providerRoutes,
	}, nil
}

// Transport は、検証済みのトランスポートを返す。
func (c Config) Transport() Transport {
	return c.transport
}

// RequestTimeout は、検証済みの外部リクエストタイムアウトを返す。
func (c Config) RequestTimeout() time.Duration {
	return c.requestTimeout
}

// ListenAddress は、検証済みの HTTP 待受先を返す。
func (c Config) ListenAddress() string {
	return c.listenAddress
}

// AllowedOrigins は、検証済みの HTTPS Origin の複製を返す。
func (c Config) AllowedOrigins() []string {
	origins := make([]string, len(c.allowedOrigins))
	copy(origins, c.allowedOrigins)
	return origins
}

// Diagnostics は、一時診断が有効かを返す。
func (c Config) Diagnostics() bool {
	return c.diagnostics
}

// Providers は、検証済み provider 設定の再帰的な複製を返す。
func (c Config) Providers() map[string]ProviderConfig {
	return cloneProviderConfigs(c.providers)
}

// Provider は、providerId に一致する検証済み設定の複製を返す。
func (c Config) Provider(providerID string) (ProviderConfig, bool) {
	provider, exists := c.providers[providerID]
	if !exists {
		return ProviderConfig{}, false
	}
	return cloneProviderConfig(provider), true
}

// ProviderRoutes は、検証済み provider route の再帰的な複製を返す。
func (c Config) ProviderRoutes() map[ProviderRouteKey]ProviderRoute {
	return cloneProviderRoutes(c.providerRoutes)
}

// ProviderRoute は、能力と major version に一致する route の複製を返す。
func (c Config) ProviderRoute(key ProviderRouteKey) (ProviderRoute, bool) {
	route, exists := c.providerRoutes[key]
	if !exists {
		return ProviderRoute{}, false
	}
	return cloneProviderRoute(route), true
}
