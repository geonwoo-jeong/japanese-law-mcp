package main

import (
	"context"
	"errors"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/config"
)

func TestRunServerReportsImplementationGap(t *testing.T) {
	t.Parallel()

	err := runServer(context.Background(), config.Default())
	if !errors.Is(err, errServerNotImplemented) {
		t.Fatalf("実装状況: runServer() のエラー = %v", err)
	}
}
