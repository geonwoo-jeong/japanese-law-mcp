# SOT-IF-075: MCP `trace_judicial_citations`

- 状態: 有効

## 規定

`trace_judicial_citations` は、`SOT-IF-067` に従って `judicial-citations` の専門公開面が有効な場合だけ、一件の公表裁判例から 1-hop の引用関係 graph を返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 |
|---|---|---:|---|
| `ref` | `SourceResourceRef` | はい | `judicial-decision` の検証済み参照 |
| `direction` | string | いいえ | `outgoing`、`incoming` または `both`。既定 `both` |
| `incomingLimit` | integer | いいえ | 既定 5、1 以上 10 以下 |

`maxDepth`、任意 PDF URL、検索式、score 閾値、graph filter、cursor、offset または provider 指定は受け付けない。未知の項目、`null`、型不一致または能力契約に反する値は、外部呼出し前に `invalid_argument` とする。

## 出力

成功時の `structuredContent` は `SOT-MODEL-035` の `JudicialCitationGraphResult` とする。

- `status` は `complete` または `partial`
- `coverageNotice` は `SOT-PROD-016` の固定文
- `graph.rootNodeId`、`graph.nodes`、`graph.edges`、`graph.unresolvedMentions`、`graph.summary` および `graph.coverage`
- `issues`

graph には `impactScore`、`goodLaw`、`overruled`、治療的評価、拘束力評価または自然文要約を追加しない。

## 実行

このツールは、ルート詳細 HTML 一回、PDF 一回、および候補検索二回を上限として実行する。片方向だけ失敗した場合は成功方向の graph と `issues` を返せる。ルート詳細失敗、全方向失敗または全体キャンセルは `isError: true` のツール結果とする。

## エラー

`invalid_argument`、`not_found`、`unsupported_capability`、`configuration_required`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit`、`unsafe_source_content` および `internal_error` が到達し得る。公開 code、`retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 確認

pack 無効時の未登録、有効時の登録、schema、`ref` の検証、`direction` と `incomingLimit` の境界、固定 `coverageNotice`、片方向 partial、PDF text layer 不在、graph 閉包、未解決言及、非採用項目の拒否、および stdio/Streamable HTTP の schema 一致を MCP 契約テストで確認する。

## 関連

- [SOT-SCN-015: 一件の公表裁判例から引用関係を追跡する](../10-scenarios/15-trace-judicial-citations.md)
- [SOT-MODEL-035: JudicialCitationGraph](../20-model/35-judicial-citation-graph.md)
- [SOT-IF-067: `judicial-citations` 拡張パックの有効化](67-judicial-citations-pack-activation.md)
