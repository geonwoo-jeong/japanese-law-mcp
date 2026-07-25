# SOT-DEL-013: ローカル Streamable HTTP

- 状態: 有効

## 規定

Japanese Law MCP は、利用者の同じ host で起動した複数の MCP クライアントが接続できる、loopback 限定かつ無状態の Streamable HTTP 実行方式を補助的に提供する。

## 動作

- MCP 仕様 `2025-11-25` の Streamable HTTP に従い、`/mcp` を単一の MCP endpoint とする。
- 一つの JSON-RPC message を HTTP POST で受け付け、JSON-RPC request には `application/json` の結果、notification または response には本文のない `202 Accepted` を返す。
- 初期化後の HTTP request では `MCP-Protocol-Version` を検証する。
- server から開始する SSE stream を提供せず、HTTP GET には `405 Method Not Allowed` を返す。
- `MCP-Session-Id` を発行せず、session 再開と message 再送を提供しない。
- `listenAddress` は IP literal の `127.0.0.0/8` または `[::1]` と port の組だけを許可する。hostname、`0.0.0.0`、`::`、Unix domain socket、private address、link-local address および外部 address は起動エラーとする。
- `Origin` header がある場合は `allowedOrigins` と照合し、一致しない値には `403 Forbidden` を返す。
- `Origin` header がない接続は、同じ host の非 browser MCP client として受け付ける。したがって、この方式は同じ OS 利用者の非信頼 process を分離する認可境界ではない。
- health endpoint、`liveness`、`readiness`、稼働率収集および運用障害検知を提供しない。

## 完了条件

ローカル実行ファイルを loopback で起動し、stdio と同じ MCP ツール契約を利用できる。非 loopback の各待受け指定が外部接続の開始前に失敗する。

## 関連

- [SOT-ARCH-002: MCP トランスポート境界](../30-architecture/02-transport-boundary.md)
- [SOT-ARCH-005: リクエスト情報の一時性](../30-architecture/05-ephemeral-request-lifecycle.md)
- [SOT-IF-029: ローカル実行設定](../40-interfaces/29-local-runtime-configuration.md)
- [SOT-DEL-008: HTTP リソース制限](08-http-resource-limits.md)
- [SOT-DEL-012: ローカル実行経路](12-local-execution-paths.md)
