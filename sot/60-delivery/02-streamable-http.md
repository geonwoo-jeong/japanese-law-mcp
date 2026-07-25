# SOT-DEL-002: Streamable HTTP

- 状態: 有効

## 規定

Japanese Law MCP は、独立して起動したプロセスへ複数の MCP クライアントが接続できる、無状態の Streamable HTTP 実行方式を公式に提供する。

## 動作

- MCP 仕様 `2025-11-25` の Streamable HTTP に従い、`/mcp` を単一の MCP エンドポイントとする。
- 一つの JSON-RPC メッセージを HTTP POST で受け付け、JSON-RPC リクエストには `application/json` の結果、通知またはレスポンスには本文のない `202 Accepted` を返す。
- 初期化後の HTTP リクエストでは `MCP-Protocol-Version` を検証する。
- サーバーから開始する SSE ストリームを提供せず、HTTP GET には `405 Method Not Allowed` を返す。
- `MCP-Session-Id` を発行せず、セッション再開とメッセージ再送を提供しない。
- ローカル実行時の既定公開先はループバックアドレスとする。
- `Origin` ヘッダーがある場合は `allowedOrigins` と照合し、一致しない値には `403 Forbidden` を返す。
- ループバック以外へ公開する場合は、TLS と `SOT-DEL-006` の認可を適用する。

## 完了条件

公式 HTTP 配布物またはコンテナを起動し、stdio と同じ MCP ツール契約を利用できる。

## 関連

- [SOT-ARCH-002: MCP トランスポート境界](../30-architecture/02-transport-boundary.md)
- [SOT-ARCH-005: リクエスト情報の一時性](../30-architecture/05-ephemeral-request-lifecycle.md)
- [SOT-IF-005: 実行設定](../40-interfaces/05-runtime-configuration.md)
- [SOT-DEL-006: HTTP 認可](06-http-authorization.md)
- [SOT-DEL-008: HTTP リソース制限](08-http-resource-limits.md)
