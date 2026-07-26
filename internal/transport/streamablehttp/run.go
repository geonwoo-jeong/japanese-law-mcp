package streamablehttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	requestReadTimeout = 5 * time.Second
	shutdownTimeout    = 5 * time.Second
)

// Options は、ローカル Streamable HTTP の起動オプションを表す。
type Options struct {
	ListenAddress  string
	AllowedOrigins []string
}

// Run は、MCP サーバーを検証済みの loopback 待受先へ接続する。
func Run(ctx context.Context, server *sdk.Server, options Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateLoopbackListenAddress(options.ListenAddress); err != nil {
		return err
	}

	listener, err := net.Listen("tcp", options.ListenAddress)
	if err != nil {
		return fmt.Errorf("loopback HTTP listener を開始できません: %w", err)
	}
	return serve(ctx, listener, NewHandler(server, options))
}

func validateLoopbackListenAddress(address string) error {
	if strings.TrimSpace(address) != address {
		return errors.New("listenAddress は空白を含まない loopback IP literal と port でなければなりません")
	}
	host, portText, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return errors.New("listenAddress は loopback IP literal と port でなければなりません")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("listenAddress の port は 1 以上 65535 以下でなければなりません")
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return errors.New("listenAddress は 127.0.0.0/8 または ::1 の IP literal でなければなりません")
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		if strings.Contains(host, ":") || ipv4[0] != 127 {
			return errors.New("listenAddress は 127.0.0.0/8 または ::1 の IP literal でなければなりません")
		}
		return nil
	}
	if !ip.Equal(net.IPv6loopback) {
		return errors.New("listenAddress は 127.0.0.0/8 または ::1 の IP literal でなければなりません")
	}
	return nil
}

func serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	httpServer := newHTTPServer(ctx, handler)

	serverResult := make(chan error, 1)
	go func() {
		err := httpServer.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverResult <- err
	}()

	select {
	case err := <-serverResult:
		if err != nil {
			return fmt.Errorf("loopback HTTP server を実行できません: %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			shutdownTimeout,
		)
		defer cancel()
		if err := httpServer.Shutdown(shutdownContext); err != nil {
			_ = httpServer.Close()
			return fmt.Errorf("loopback HTTP server を停止できません: %w", err)
		}
		if err := <-serverResult; err != nil {
			return fmt.Errorf("loopback HTTP server を停止できません: %w", err)
		}
		return nil
	}
}

func newHTTPServer(ctx context.Context, handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: requestReadTimeout,
		ReadTimeout:       requestReadTimeout,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
}
