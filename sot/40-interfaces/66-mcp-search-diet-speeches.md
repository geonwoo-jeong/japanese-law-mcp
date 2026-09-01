# SOT-IF-066: MCP `search_diet_speeches`

- 状態: 有効

## 規定

`search_diet_speeches` は、`SOT-IF-077` に従って `legislative-history` の専門操作が
有効な場合だけ `parliament.speech.search@1` を公開し、公式国会発言を出典、正確な
件数および固定した利用上の注意とともに返す MCP ツールとする。

専門操作の有効化は、同じ pack 条件で `SOT-IF-065` の provider binding と primary route を構成できる
場合に限る。統合照会 contribution は第一段階の構成条件に含めず、この操作の有効化を
理由に `query_legal_information` の範囲を広げない。

## 入力

| 名前 | 型 | 必須 | 制約 |
|---|---|---:|---|
| `query` | string | いいえ | `SOT-IF-062` の発言検索語 |
| `speaker` | string | いいえ | `SOT-IF-062` の発言者名 |
| `meetingName` | string | いいえ | `SOT-IF-062` の会議名 |
| `house` | string | いいえ | `house_of_representatives`、`house_of_councillors`、`both_houses` または `conference_of_both_houses` |
| `fromDate` | string | いいえ | 実在する `YYYY-MM-DD` |
| `untilDate` | string | いいえ | 実在する `YYYY-MM-DD` |
| `limit` | integer | いいえ | 既定 20、1 以上 30 以下 |
| `continuationToken` | string | いいえ | 4096 byte 以下 |

`query`、`speaker`、`meetingName`、`house`、`fromDate` または `untilDate` のうち一つ以上を必須とする。欠落、`null`、未知の項目、整数へ正確に変換できない number、日付順序の違反および能力契約に反する値は、外部呼出し前に `invalid_argument` とする。

初期の `ndl-diet-speech-api` route は continuation を発行しないため、空でない `continuationToken` は外部呼出し前に `invalid_argument` とする。

## 出力

成功時の `structuredContent` は次を持つ。

| 名前 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `usageNotice` | string | はい | `SOT-PROD-014` の固定した著作権と法的意味の注意 |
| `items` | `SourcedResource<ParliamentSpeech>[]` | はい | `SOT-IF-062` の item |
| `page` | `SourcePage` | はい | 正確な総件数と最大 30 件の返却件数。`nextToken` は省略 |

`items` は内部の `data`、`ref` および `provenance` を変更せず公開する。`content`、`structuredContent`、省略可能値および JSON 表現は `SOT-IF-007` と `SOT-MODEL-009` に従う。

`usageNotice` は次の文字列とする。

> 国会会議録の発言は発言者等が著作権を有する場合があります。利用条件を確認してください。発言は現行法令または法的結論を示すものではありません。

ツールは発言の検索と原文の提示だけを行い、法的助言、立法趣旨、国会の統一見解、法令との対応または法的結論を出力へ加えない。

## エラー

`invalid_argument`、`unsupported_capability`、`unsupported_query`、`configuration_required`、`source_auth_failed`、`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit`、`unsafe_source_content` および `internal_error` が到達し得る。公開 code、`retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 確認

pack 無効時に専門操作を利用可能にしないこと、有効時に binding・route と原子的に構成し、`compact` の発見と実行および `full` の直接呼出しで同じ schema、少なくとも一つの検索条件、文字列・院名・日付・上限の検証、空でない token の事前拒否、固定 notice、空結果、最大 30 件、exact page、`ref` と provenance の無変更公開、法的結論の非生成および全エラー対応になることを MCP 契約テストで確認する。

## 関連

- [SOT-PROD-014: 立法過程拡張パックの国会発言検索](../00-product/14-legislative-history-extension-pack.md)
- [SOT-MODEL-034: ParliamentSpeech](../20-model/34-parliament-speech.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-077: MCP ツール公開方式と拡張パック有効化](77-mcp-tool-exposure-and-extension-packs.md)
- [SOT-IF-062: `parliament.speech.search` capability v1](62-parliament-speech-search-capability.md)
- [SOT-IF-065: 国立国会図書館の国会発言検索の組込み採用](65-ndl-diet-speech-built-in-adoption.md)
