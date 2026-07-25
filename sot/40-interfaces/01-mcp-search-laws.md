# SOT-IF-001: MCP `search_laws`

- 状態: 廃止
- 後継: [SOT-IF-030: MCP `search_laws`](30-mcp-search-laws.md)

## 規定

`search_laws` は、法令名または略称の一部を受け取り、公式情報源で確認できる `LawSearchResult` を返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `query` | string | はい | 前後の空白を除いて 1 文字以上 | 法令名または略称に含まれる文字列 |
| `asOf` | string | いいえ | `2017-04-01` 以降の `YYYY-MM-DD` | 指定日以前で最新のリビジョンを検索する基準日 |
| `limit` | integer | いいえ | 1 以上 100 以下、既定値 20 | 一ページに返す法令数 |
| `offset` | integer | いいえ | 0 以上、既定値 0 | 取得を開始する位置 |

定義していない入力項目は受け付けない。

## 出力

`LawSearchResult` を返す。結果がない場合の表現は `SOT-MODEL-006` に従う。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 外部情報源の制限、一時障害または現在のローカル同時実行上限は、原因に応じて `rate_limited`、`source_timeout`、`source_unavailable` または `source_busy` を返す。
- 外部契約、応答または安全上限の問題は、原因に応じて `source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` または `unsafe_source_content` を返す。
- 上記へ分類できない内部処理の失敗は `internal_error` を返す。

各コードの `retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 関連

- [SOT-SCN-001: 法令名から法令を検索する](../10-scenarios/01-search-laws.md)
- [SOT-MODEL-006: LawSearchResult](../20-model/06-law-search-result.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-009: e-Gov 法令名検索マッピング](09-egov-law-search-mapping.md)
