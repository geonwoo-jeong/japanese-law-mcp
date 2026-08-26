# SOT-IF-048: MCP `get_judicial_case`

- 状態: 有効

## 規定

`get_judicial_case` は、`judicial-cases` が有効な場合に限り検索結果の `SourceResourceRef` を受け取り、同じ provider の公式裁判例詳細を収録範囲の注意とともに返す MCP ツールとする。

## 入力

```json
{
  "ref": {
    "providerId": "courts-hanrei-html",
    "key": {
      "sourceId": "courts-hanrei",
      "resourceType": "judicial-decision",
      "resourceId": "95570/detail2"
    }
  }
}
```

`ref` は `SOT-MODEL-016` と `SOT-IF-042` に従う。欠落、`null`、未知の項目、空の識別子、version の指定、provider と source の不一致および異なる resource type は、外部呼出し前に `invalid_argument` とする。

## 出力

成功時の `structuredContent` は次を持つ。

| 名前 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `coverageNotice` | string | はい | `SOT-PROD-010` の固定された収録範囲の注意 |
| `item` | `SourcedResource<JudicialDecisionDetails>` | はい | `SOT-IF-042` の取得結果 |

`item.ref` は入力と同じ値とし、`data` と provenance を変更せず公開する。`coverageNotice` は `SOT-IF-047` と同じ固定文字列とする。`content`、`structuredContent`、省略可能値および JSON 表現は `SOT-IF-007` と `SOT-MODEL-009` に従う。

## エラー

`invalid_argument`、`not_found`、`unsupported_capability`、`configuration_required`、`source_auth_failed`、`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit`、`unsafe_source_content` および `internal_error` が到達し得る。公開 code、`retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 確認

pack 無効時の未登録、有効時の schema、検索結果からの参照往復、不正参照の外部呼出し前拒否、同一 provider の保持、not found、固定 notice および全エラー対応を MCP 契約テストで確認する。

## 関連

- [SOT-PROD-010: 裁判例拡張パック](../00-product/10-judicial-cases-extension-pack.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-MODEL-021: JudicialDecisionDetails](../20-model/21-judicial-decision-details.md)
- [SOT-IF-067: `judicial-citations` 拡張パックの有効化](67-judicial-citations-pack-activation.md)
- [SOT-IF-042: `judicial-decision.read` capability v1](42-judicial-decision-read-capability.md)
- [SOT-IF-047: MCP `search_judicial_cases`](47-mcp-search-judicial-cases.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
