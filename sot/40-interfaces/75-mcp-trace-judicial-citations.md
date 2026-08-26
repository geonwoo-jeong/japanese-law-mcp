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

組込み route では `ref.providerId=courts-hanrei-html`、`ref.key.sourceId=courts-hanrei`、`resourceType=judicial-decision`、未設定の `versionId` および採用済みの canonical resource ID を外部呼出し前に確認する。過去に同じ process が発行したことは要求しないが、provider/source/type の不一致を別 provider へ fallback しない。

## 出力

成功時の `structuredContent` は `SOT-MODEL-035` の `JudicialCitationGraphResult` とする。

- `status` は `complete` または `partial`
- `coverageNotice` は `SOT-PROD-016` の固定文
- `graph.rootNodeId`、`graph.nodes`、`graph.edges`、`graph.unresolvedMentions`、`graph.summary` および `graph.coverage`
- `issues`

graph には `impactScore`、`goodLaw`、`overruled`、治療的評価、拘束力評価または自然文要約を追加しない。

`coverageNotice` は次の固定文字列とする。

> 裁判所の裁判例検索には、すべての判決等が掲載されているわけではありません。引用関係と件数は、取得して解析できた公表資料で確認した明示的な参照又は検索候補に限られます。被引用候補数は、現在の公式検索範囲で観測した候補数であり、実際の全引用回数ではありません。結果がないことは引用の不存在を示さず、先例性、拘束力、確定性、評価、判例変更又は現在の有効性を判断するものではありません。

## 実行

このツールは入力検証後、ルート詳細 HTML 一回、参照法条・原審 metadata の正規化、要求時の PDF 一回、および要求時の候補検索二回の順に、合計四つの外向き request を上限として実行する。候補又は参照先の詳細・PDF、法令 APIその他を連鎖取得しない。

片方向だけ失敗した場合又は候補検索二回の一方だけ失敗した場合は、成功結果を捨てず `status=partial` と direction、stage、公開可能な code を持つ `issues` を返す。PDF text layer 不在は「引用なし」ではなく `document_text_unavailable` の issue と縮退 coverage にする。ルート詳細失敗、全要求方向失敗又は全体キャンセルは部分 graph を公開せず `isError: true` とする。

要求した全処理が採用済み上限内で成功した場合は edge が空でも `status=complete` とする。利用者の `incomingLimit` で正常に候補を切り詰めたことだけでは `partial` にせず、coverage の `truncated` で示す。

## エラー

`invalid_argument`、`not_found`、`unsupported_capability`、`unsupported_query`、`configuration_required`、`source_auth_failed`、`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit`、`unsafe_source_content` および `internal_error` が到達し得る。公開 code、`retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

エラー、issue、診断又はログへ入力 ref 全文、検索語、事件番号、判例集表記、URL query、HTML 本文、PDF bytes、抽出 text、evidence excerpt、未解決 mention 又は内部 path を含めない。公開 graph の短い excerpt と mention だけは `SOT-MODEL-035` の上限に従う。

## 確認

pack 無効時の未登録、有効時の原子的登録、schema、`ref` の事前検証、`direction` と `incomingLimit` の境界、固定 `coverageNotice`、最大四 request、片方向 partial、候補一検索 partial、全方向失敗、全体取消、PDF text layer 不在、graph 閉包、未解決言及、原文非露出、非採用項目の拒否、既存二裁判例ツールの JSON 非変更、`query_legal_information` の非変更、および stdio/Streamable HTTP の schema と結果の一致を MCP 契約テストで確認する。

## 関連

- [SOT-SCN-015: 一件の公表裁判例から引用関係を追跡する](../10-scenarios/15-trace-judicial-citations.md)
- [SOT-MODEL-035: JudicialCitationGraph](../20-model/35-judicial-citation-graph.md)
- [SOT-IF-067: `judicial-citations` 拡張パックの有効化](67-judicial-citations-pack-activation.md)
