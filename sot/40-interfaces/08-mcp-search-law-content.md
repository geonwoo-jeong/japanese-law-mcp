# SOT-IF-008: MCP `search_law_content`

- 状態: 廃止
- 後継: [SOT-IF-033: MCP `search_law_content`](33-mcp-search-law-content.md)

## 規定

`search_law_content` は、法令本文の検索式を受け取り、公式情報源で確認できる `LawContentSearchResult` を返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `query` | string | はい | 前後の空白を除いて 1 文字以上 | 法令本文へ適用する e-Gov の検索式 |
| `asOf` | string | いいえ | `2017-04-01` 以降の `YYYY-MM-DD` | 指定日以前で最新のリビジョンを検索する基準日 |
| `limit` | integer | いいえ | 1 以上 100 以下、既定値 20 | 一ページに返す一致位置の上限 |
| `offset` | integer | いいえ | 0 以上、既定値 0 | 取得を開始する一致位置 |

`query` は、e-Gov が定義するワイルドカード検索または AND、OR、NOT 検索を受け付ける。ワイルドカードと AND、OR、NOT を同じ検索式で併用しない。

定義していない入力項目は受け付けない。

## 出力

`LawContentSearchResult` を返す。結果がない場合の表現は `SOT-MODEL-008` に従う。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 外部情報源の制限、一時障害または現在のローカル同時実行上限は、原因に応じて `rate_limited`、`source_timeout`、`source_unavailable` または `source_busy` を返す。
- 外部契約、応答または安全上限の問題は、原因に応じて `source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` または `unsafe_source_content` を返す。
- 上記へ分類できない内部処理の失敗は `internal_error` を返す。

各コードの `retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 関連

- [SOT-SCN-004: 法令本文を検索する](../10-scenarios/04-search-law-content.md)
- [SOT-MODEL-008: LawContentSearchResult](../20-model/08-law-content-search-result.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-010: e-Gov 本文検索マッピング](10-egov-content-search-mapping.md)
