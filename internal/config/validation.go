package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	minimumRequestTimeout = time.Second
	maximumRequestTimeout = 120 * time.Second
)

func validateValues(values Values) error {
	transport := Transport(values.Transport)
	if transport != TransportStdio && transport != TransportStreamableHTTP {
		return fmt.Errorf("transport は %q または %q でなければなりません", TransportStdio, TransportStreamableHTTP)
	}
	if values.RequestTimeout < minimumRequestTimeout || values.RequestTimeout > maximumRequestTimeout {
		return fmt.Errorf("requestTimeout は 1 秒以上 120 秒以下でなければなりません")
	}
	if err := validateListenAddress(values.ListenAddress); err != nil {
		return err
	}
	for _, origin := range values.AllowedOrigins {
		if err := validateOrigin(origin); err != nil {
			return err
		}
	}
	if transport == TransportStdio && values.ListenAddress != Default().ListenAddress() {
		return fmt.Errorf("stdio では listenAddress を変更できません")
	}
	if transport == TransportStdio && len(values.AllowedOrigins) != 0 {
		return fmt.Errorf("stdio では allowedOrigins を指定できません")
	}
	return nil
}

func validateListenAddress(address string) error {
	if strings.TrimSpace(address) != address {
		return fmt.Errorf("listenAddress は空白を含まない host:port でなければなりません")
	}

	host, portText, err := net.SplitHostPort(address)
	if err != nil || host == "" || strings.Contains(host, "/") || strings.ContainsAny(host, " \t\r\n") {
		return fmt.Errorf("listenAddress は host:port 形式でなければなりません")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("listenAddress の port は 1 以上 65535 以下でなければなりません")
	}
	return nil
}

func validateOrigin(origin string) error {
	if origin == "" || strings.TrimSpace(origin) != origin {
		return fmt.Errorf("allowedOrigins の各値は空白を含まない HTTPS Origin でなければなりません")
	}

	parsed, err := url.Parse(origin)
	if err != nil ||
		parsed.Scheme != "https" ||
		parsed.Host == "" ||
		parsed.Hostname() == "" ||
		parsed.User != nil ||
		parsed.Path != "" ||
		parsed.RawQuery != "" ||
		parsed.ForceQuery ||
		parsed.Fragment != "" ||
		parsed.Opaque != "" {
		return fmt.Errorf("allowedOrigins の値 %q は HTTPS Origin ではありません", origin)
	}
	return nil
}
