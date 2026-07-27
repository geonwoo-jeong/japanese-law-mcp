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

複数の独立した step を一つの公開結果として扱う複合ツールでは、一つの情報源呼出しの失敗だけで tool request 全体が失敗したとは限らない。各 step の結果と失敗を保持することをそのツールの有効なインターフェース SOT が明示し、少なくとも一つの step が型付き結果を返した場合に限り、別 step の公開可能な `ErrorResult` を成功時の `structuredContent` に含められる。

複合ツールでも、実行した全 step の失敗、入力検証、計画、結果変換または結果不変条件の失敗は `isError: true` とする。部分失敗を許可する規定を、単一能力ツールの情報源エラーを成功結果へ変える根拠にしない。

## プロトコルエラー

存在しないツール、`tools/call` の形式不正、JSON-RPC の形式不正、および MCP の初期化失敗は、MCP または JSON-RPC が定めるプロトコルエラーとして返す。これらを成功したツール結果や `ErrorResult` に変換しない。

## 確認

各ツールについて、成功、入力値エラー、ユースケースエラーおよび形式不正の結果が、この規定の異なる経路を通ることを契約テストで確認する。複合ツールでは、部分成功と全 step 失敗がそれぞれ成功結果とツールエラーへ分かれることも確認する。

## 関連

- [SOT-MODEL-005: ErrorResult](../20-model/05-error-result.md)
- [SOT-MODEL-009: JSON シリアライズ](../20-model/09-json-serialization.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-013: MCP プロトコル基準](13-mcp-protocol-version.md)
- [SOT-IF-051: MCP `query_legal_information`](51-mcp-query-legal-information.md)
