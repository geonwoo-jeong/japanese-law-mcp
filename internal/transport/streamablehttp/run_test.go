package streamablehttp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestValidateLoopbackListenAddress(t *testing.T) {
	t.Parallel()

	for _, address := range []string{
		"127.0.0.1:8080",
		"127.255.255.254:65535",
		"[::1]:8080",
	} {
		if err := validateLoopbackListenAddress(address); err != nil {
			t.Fatalf("%q のエラー = %v", address, err)
		}
	}

	for _, address := range []string{
		"localhost:8080",
		"0.0.0.0:8080",
		"10.0.0.1:8080",
		"[::]:8080",
		"[::ffff:127.0.0.1]:8080",
		"[::1%lo0]:8080",
		"127.0.0.1:0",
	} {
		if err := validateLoopbackListenAddress(address); err == nil {
			t.Fatalf("%q を受理しました", address)
		}
	}
}

func TestRunRejectsInvalidAddressBeforeListening(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), nil, Options{ListenAddress: "0.0.0.0:8080"})
	if err == nil {
		t.Fatal("非 loopback の待受先を受理しました")
	}
}

func TestRunRejectsCanceledContextBeforeListening(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Run(ctx, nil, Options{ListenAddress: "127.0.0.1:49151"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() のエラー = %v, want context.Canceled", err)
	}
}

func TestHTTPServerBoundsRequestBodyReadTime(t *testing.T) {
	t.Parallel()

	server := newHTTPServer(
		context.Background(),
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	)
	if server.ReadTimeout <= 0 {
		t.Fatal("request body の読取り時間が制限されていません")
	}
}

func TestServeStopsAndCancelsRequests(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listener を開始できません: %v", err)
	}

	started := make(chan struct{})
	canceled := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		close(started)
		<-req.Context().Done()
		close(canceled)
		w.WriteHeader(http.StatusNoContent)
	})

	ctx, cancel := context.WithCancel(context.Background())
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- serve(ctx, listener, handler)
	}()

	requestResult := make(chan error, 1)
	go func() {
		request, requestErr := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			"http://"+listener.Addr().String()+"/mcp",
			nil,
		)
		if requestErr != nil {
			requestResult <- requestErr
			return
		}
		response, requestErr := http.DefaultClient.Do(request)
		if requestErr == nil {
			_ = response.Body.Close()
		}
		requestResult <- requestErr
	}()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("request が handler へ到達しませんでした")
	}
	cancel()

	select {
	case <-canceled:
	case <-time.After(3 * time.Second):
		t.Fatal("request context が中断されませんでした")
	}
	select {
	case serveErr := <-serveResult:
		if serveErr != nil {
			t.Fatalf("serve() のエラー = %v", serveErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("HTTP server が終了しませんでした")
	}
	select {
	case <-requestResult:
	case <-time.After(3 * time.Second):
		t.Fatal("HTTP request が終了しませんでした")
	}
}
