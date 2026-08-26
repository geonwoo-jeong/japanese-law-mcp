# SOT-IF-073: 裁判所検索の被引用候補マッピング

- 状態: 有効

## 規定

`courts-hanrei-html` は、公式統合検索 HTML の結果行を使って、被引用候補を `SOT-IF-069` の共通出力へ対応させる。

## 検索実行

- 第一検索語はルート裁判例の `caseNumber` とする。
- `reporterCitation` が存在し、空でない場合に限り、第二検索語として追加検索できる。
- それ以外の自然文、全文、法令名、事件名または自由検索式を追加しない。

## 候補対応

- 各結果行は既存 `judicial-decision.search@1` の mapping を再利用して `JudicialDecisionSummary` と `ref` を構成する。
- ルート裁判例と同じ `ref` は除外する。
- 同じ `ref` を持つ候補は一件へ統合し、先に観測した `matchedQuery` と DOM 順を保持する。
- `limit` で切り詰める前に自己参照除外と `ref` 基準の重複排除を行う。

## 根拠

各候補には `evidenceLevel=official_search_candidate` の evidence を一件以上付ける。evidence の provenance は、当該検索 HTML、取得時刻、検索 URL および `SOT-IF-073` の method ID を持つ。

## 確認

ケース番号検索、判例集表記付き二回検索、自己参照除外、重複排除順序、DOM 順保持、上限適用、空結果および第三能力の provider 固有 mapping を単体テストで確認する。

## 関連

- [SOT-IF-069: `judicial-decision.citing-candidate.search` capability v1](69-judicial-citing-candidate-search-capability.md)
- [SOT-IF-072: 裁判所「裁判例検索」HTML 情報源 v2](72-source-courts-hanrei-html-v2.md)
