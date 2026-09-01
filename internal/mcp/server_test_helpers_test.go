package mcp

import (
	"context"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func mustNewCompactServer(t *testing.T, dependencies Dependencies) *sdk.Server {
	t.Helper()
	server, err := NewServerWithDependenciesAndExposure(
		"test-version",
		dependencies,
		ToolExposureCompact,
	)
	if err != nil {
		t.Fatalf("compact MCP サーバーを構成できません: %v", err)
	}
	return server
}

func mustNewSessionlessCompactServer(
	t *testing.T,
	dependencies Dependencies,
) *sdk.Server {
	t.Helper()
	server, err := NewSessionlessServerWithDependenciesAndExposure(
		"test-version",
		dependencies,
		ToolExposureCompact,
	)
	if err != nil {
		t.Fatalf("sessionless compact MCP サーバーを構成できません: %v", err)
	}
	return server
}

func mustNewFullServer(t *testing.T, dependencies Dependencies) *sdk.Server {
	t.Helper()
	server, err := NewServerWithDependenciesAndExposure(
		"test-version",
		dependencies,
		ToolExposureFull,
	)
	if err != nil {
		t.Fatalf("full MCP サーバーを構成できません: %v", err)
	}
	return server
}

func mustNewSessionlessFullServer(
	t *testing.T,
	dependencies Dependencies,
) *sdk.Server {
	t.Helper()
	server, err := NewSessionlessServerWithDependenciesAndExposure(
		"test-version",
		dependencies,
		ToolExposureFull,
	)
	if err != nil {
		t.Fatalf("sessionless full MCP サーバーを構成できません: %v", err)
	}
	return server
}

func withPairedMCPClientSessions(
	t *testing.T,
	left *sdk.Server,
	right *sdk.Server,
	operation func(context.Context, *sdk.ClientSession, *sdk.ClientSession),
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	leftServerTransport, leftClientTransport := sdk.NewInMemoryTransports()
	rightServerTransport, rightClientTransport := sdk.NewInMemoryTransports()
	leftServerResult := make(chan error, 1)
	rightServerResult := make(chan error, 1)
	go func() {
		leftServerResult <- left.Run(ctx, leftServerTransport)
	}()
	go func() {
		rightServerResult <- right.Run(ctx, rightServerTransport)
	}()

	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-client", Version: "test-version"},
		nil,
	)
	leftSession, err := client.Connect(ctx, leftClientTransport, nil)
	if err != nil {
		t.Fatalf("left MCP connect = %v", err)
	}
	rightSession, err := client.Connect(ctx, rightClientTransport, nil)
	if err != nil {
		t.Fatalf("right MCP connect = %v", err)
	}

	operation(ctx, leftSession, rightSession)

	if err := leftSession.Close(); err != nil {
		t.Fatalf("left MCP close = %v", err)
	}
	if err := rightSession.Close(); err != nil {
		t.Fatalf("right MCP close = %v", err)
	}

	select {
	case err := <-leftServerResult:
		if err != nil {
			t.Fatalf("left MCP server = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("left MCP server close timeout = %v", ctx.Err())
	}

	select {
	case err := <-rightServerResult:
		if err != nil {
			t.Fatalf("right MCP server = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("right MCP server close timeout = %v", ctx.Err())
	}
}
