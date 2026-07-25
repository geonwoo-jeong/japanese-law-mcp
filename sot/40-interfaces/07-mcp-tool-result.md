# SOT-IF-007: MCP ツール結果

- 状態: 有効

## 規定

すべての MCP ツールは、成功を構造化された `CallToolResult` として返し、ツール実行中の失敗を `isError: true` のツール結果として返す。

## 成功

- `isError` は `false` とする。
- `structuredContent` は、各ツールが出力として参照する情報モデルに適合する JSON object とする。
- `content` には、`structuredContent` と同じ JSON をシリアライズした一つの `TextContent` を含める。
- ツール定義に `outputSchema` を含め、`structuredContent` を検証できるようにする。

## ツール実行エラー

- 入力値の検証、情報源の呼び出し、結果の変換またはユースケースの失敗は、`isError: true` とする。
- `content` には、`ErrorResult` を JSON としてシリアライズした一つの `TextContent` を含める。
- 成功時の `outputSchema` と異なるため、`structuredContent` は含めない。

## プロトコルエラー

存在しないツール、`tools/call` の形式不正、JSON-RPC の形式不正、および MCP の初期化失敗は、MCP または JSON-RPC が定めるプロトコルエラーとして返す。これらを成功したツール結果や `ErrorResult` に変換しない。

## 確認

各ツールについて、成功、入力値エラー、ユースケースエラーおよび形式不正の結果が、この規定の異なる経路を通ることを契約テストで確認する。

## 関連

- [SOT-MODEL-005: ErrorResult](../20-model/05-error-result.md)
- [SOT-MODEL-009: JSON シリアライズ](../20-model/09-json-serialization.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-013: MCP プロトコル基準](13-mcp-protocol-version.md)
