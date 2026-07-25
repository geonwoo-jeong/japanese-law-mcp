# SOT-DEL-001: stdio

- 状態: 有効

## 規定

Japanese Law MCP は、MCP クライアントが利用者の環境で子プロセスとして起動できる stdio 実行方式を公式に提供する。

## 動作

- 標準入力から MCP メッセージを受け取る。
- 標準出力には MCP メッセージだけを書き込む。
- 一時診断を有効にした場合だけ、標準エラー出力を使用する。
- MCP の HTTP 認可方式は使用しない。

## 完了条件

公式配布物を MCP クライアントから子プロセスとして起動し、すべての公式 MCP ツールを利用できる。

## 関連

- [SOT-ARCH-002: MCP トランスポート境界](../30-architecture/02-transport-boundary.md)
- [SOT-ARCH-005: リクエスト情報の一時性](../30-architecture/05-ephemeral-request-lifecycle.md)
- [SOT-ARCH-008: 一時的な診断](../30-architecture/08-ephemeral-diagnostics.md)
- [SOT-IF-029: ローカル実行設定](../40-interfaces/29-local-runtime-configuration.md)
- [SOT-DEL-012: ローカル実行経路](12-local-execution-paths.md)
