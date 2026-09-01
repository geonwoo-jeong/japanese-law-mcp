# SOT-IF-047: MCP `search_judicial_cases`

- 状態: 有効

## 規定

`search_judicial_cases` は、`judicial-cases` が有効な場合に限り `judicial-decision.search@1` を公開し、公式掲載裁判例を取得経路および収録範囲の注意とともに返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 |
|---|---|---:|---|
| `query` | string | はい | `SOT-IF-041` の検索語 |
| `limit` | integer | いいえ | 既定 20、1 以上 30 以下 |
| `continuationToken` | string | いいえ | 4096 byte 以下 |

欠落、`null`、整数へ正確に変換できない number、未知の項目および能力契約に反する値は、外部呼出し前に `invalid_argument` とする。

## 出力

成功時の `structuredContent` は次を持つ。

| 名前 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `coverageNotice` | string | はい | `SOT-PROD-010` の固定された収録範囲の注意 |
| `items` | `SourcedResource<JudicialDecisionSummary>[]` | はい | `SOT-IF-041` の item |
| `page` | `SourcePage` | はい | `SOT-IF-041` の page |

`items` は内部の `data`、`ref` および `provenance` を変更せず公開する。これにより `items[].ref` を `get_judicial_case` へそのまま渡せるようにする。`content`、`structuredContent`、省略可能値および JSON 表現は `SOT-IF-007` と `SOT-MODEL-009` に従う。

`coverageNotice` は次の文字列とする。

> 裁判所の裁判例検索には、すべての判決等が掲載されているわけではありません。掲載情報だけから先例性、拘束力、確定性または現在の有効性を判断できません。

## エラー

`invalid_argument`、`unsupported_capability`、`unsupported_query`、`configuration_required`、`source_auth_failed`、`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit`、`unsafe_source_content` および `internal_error` が到達し得る。公開 code、`retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 確認

pack 無効時に専門操作を利用可能にしないこと、有効時に `compact` の発見と実行および `full` の直接呼出しで同じ schema、入力検証、固定 notice、空結果、`ref` と provenance の公開、検索から詳細への往復および全エラー対応になることを MCP 契約テストで確認する。

## 関連

- [SOT-PROD-010: 裁判例拡張パック](../00-product/10-judicial-cases-extension-pack.md)
- [SOT-IF-077: MCP ツール公開方式と拡張パック有効化](77-mcp-tool-exposure-and-extension-packs.md)
- [SOT-IF-041: `judicial-decision.search` capability v1](41-judicial-decision-search-capability.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
