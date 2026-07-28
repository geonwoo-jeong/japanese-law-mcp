package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type queryLegalInformationSchemaSnapshot struct {
	input  json.RawMessage
	output json.RawMessage
}

func TestQueryLegalInformationSchemaIsIdenticalAcrossServerModes(t *testing.T) {
	t.Parallel()

	dependencies := Dependencies{
		QueryLegalInformation: &recordingQueryLegalInformationPort{},
	}
	stateful := queryLegalInformationSchemaFromServer(
		t,
		NewServerWithDependencies("test-version", dependencies),
	)
	sessionless := queryLegalInformationSchemaFromServer(
		t,
		NewSessionlessServerWithDependencies("test-version", dependencies),
	)
	if string(stateful.input) != string(sessionless.input) {
		t.Fatal("stdio と Streamable HTTP の input schema が一致しません")
	}
	if string(stateful.output) != string(sessionless.output) {
		t.Fatal("stdio と Streamable HTTP の output schema が一致しません")
	}
}

func queryLegalInformationSchemaFromServer(
	t *testing.T,
	server *sdk.Server,
) queryLegalInformationSchemaSnapshot {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
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
		t.Fatalf("MCP セッションを初期化できません: %v", err)
	}
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("tools/list error = %v", err)
	}
	var target *sdk.Tool
	for _, tool := range tools.Tools {
		if tool.Name == "query_legal_information" {
			target = tool
			break
		}
	}
	if target == nil {
		t.Fatal("query_legal_information が登録されていません")
	}
	input, err := json.Marshal(target.InputSchema)
	if err != nil {
		t.Fatalf("input schema を直列化できません: %v", err)
	}
	output, err := json.Marshal(target.OutputSchema)
	if err != nil {
		t.Fatalf("output schema を直列化できません: %v", err)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("MCP セッションを終了できません: %v", err)
	}
	select {
	case err := <-serverResult:
		if err != nil {
			t.Fatalf("MCP サーバーが正常終了しませんでした: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("MCP サーバーの終了を待機できません: %v", ctx.Err())
	}
	return queryLegalInformationSchemaSnapshot{
		input:  input,
		output: output,
	}
}
