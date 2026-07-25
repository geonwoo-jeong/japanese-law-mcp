# SOT-IF-006: エラー契約

- 状態: 廃止

## 規定

MCP ツールの実行開始後に判明した入力、該当結果、情報源または内部処理の失敗は、プロジェクトが定義する `ErrorResult` で表し、異なる失敗状態を同じエラーコードで混同しない。

この規定は `SOT-IF-027` に置き換えられた。公開エラーの現行定義には `SOT-IF-027` を使用する。

## エラーコード

| コード | 意味 | `retryable` | 許可する `details` |
|---|---|---:|---|
| `invalid_argument` | 入力がツール契約を満たさない | `false` | `field`, `reason` |
| `not_found` | 指定条件に該当する情報がない | `false` | なし |
| `ambiguous_location` | 法令内の位置を一意に決定できない | `false` | `candidates` |
| `source_unavailable` | 外部情報源へ一時的に到達できない | `true` | `sourceId` |
| `invalid_source_response` | 外部レスポンスが期待する仕様を満たさない | `false` | `sourceId`, `operation` |
| `internal_error` | 内部処理を完了できない | `false` | なし |

エラーメッセージは、保存済みの診断情報を参照しなくても、利用者が入力の修正または再試行の要否を判断できる内容とする。

MCP または JSON-RPC のプロトコルエラーは `ErrorResult` へ変換せず、`SOT-IF-007` に従う。

## 関連

- [SOT-MODEL-005: ErrorResult](../20-model/05-error-result.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-ENG-003: 明示的なエラー処理](../50-engineering/03-explicit-error-handling.md)
