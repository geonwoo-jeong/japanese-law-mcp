# SOT-SCN-015: 一件の公表裁判例から引用関係を追跡する

- 状態: 有効

## 規定

利用者は、一件の公表裁判例 `ref` と追跡方向を指定して、その裁判例に関する 1-hop の判例引用関係、参照法条および原審関係を一つの結果グラフとして取得できる。

## 開始条件

`ref` が `judicial-decision` の有効な `SourceResourceRef` であり、`SOT-IF-067` に従って `judicial-cases` と `judicial-citations` の両方が有効である。

`direction` は `outgoing`、`incoming` または `both` である。`incomingLimit` は 1 以上 10 以下であり、省略時は 5 とする。

## 基本フロー

1. MCP クライアントが `ref`、任意の `direction` および任意の `incomingLimit` を送信する。
2. Japanese Law MCP が `judicial-decision.read@1` でルート裁判例の詳細を一度取得する。
3. 追跡方向にかかわらず、詳細 HTML の `参照法条` と原審メタデータを正規化してルートの公式 metadata 関係として graph へ加える。
4. `direction` に `outgoing` が含まれる場合は、詳細 HTML が直接示す `full_text` PDF を一度だけ解析し、本文中で明示的に確認できた公表裁判例参照を graph へ加える。
5. `direction` に `incoming` が含まれる場合は、事件番号と存在する場合の判例集表記を使って、公式検索を最大二回だけ実行し、ルート裁判例を指している可能性がある公表裁判例候補を graph へ加える。
6. 成功した方向の結果、固定の `coverageNotice`、件数要約、coverage および issue を `JudicialCitationGraph` として返す。

## 分岐

- `ref`、`direction` または `incomingLimit` が制約を満たさない場合は、外部情報源を呼び出さず `invalid_argument` とする。
- `judicial-citations` だけが有効、または `judicial-cases` が無効な場合は、起動時設定エラーとする。
- ルート裁判例の詳細取得に失敗した場合、または要求したすべての方向が失敗した場合は、MCP ツールエラーとする。
- `both` で片方向だけ成功した場合、または被引用候補の二検索の一方だけ成功した場合は、成功した結果を捨てず、失敗した処理の `issues` を含む `status=partial` を返す。
- PDF に text layer がない場合、確認済み外向き引用を空と断定せず、`document_text_unavailable` を coverage と issue に記録する。
- あいまいな判例参照、あいまいな法条、自己参照だけに依存する候補、または fuzzy 一致に依存する候補は edge に昇格させず、`unresolvedMentions` に残す。

## 完了条件

返された graph はルートノードを一つ持ち、確認済み引用、候補、参照法条および原審関係を閉じた relation 種別だけで表す。confirmed と candidate の関係が混在せず、各 edge の根拠は公式 metadata、PDF text layer または公式検索候補のいずれかへ到達できる。

## 関連

- [SOT-PROD-016: 判例引用追跡拡張パック](../00-product/16-judicial-citations-extension-pack.md)
- [SOT-MODEL-035: JudicialCitationGraph](../20-model/35-judicial-citation-graph.md)
- [SOT-IF-075: MCP `trace_judicial_citations`](../40-interfaces/75-mcp-trace-judicial-citations.md)
