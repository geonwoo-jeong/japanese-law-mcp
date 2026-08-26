# SOT-IF-073: 裁判所検索の被引用候補マッピング

- 状態: 有効

## 規定

`courts-hanrei-html` は、公式統合検索 HTML の結果行を使って、被引用候補を `SOT-IF-069` の共通出力へ対応させる。

## 検索実行

- 第一検索語はルート裁判例の `caseNumber` とする。
- `reporterCitation` が存在し、空でない場合に限り、第二検索語として追加検索できる。
- それ以外の自然文、全文、法令名、事件名または自由検索式を追加しない。
- 二つの値が完全一致する場合は同じ検索を繰り返さない。元号、数字、括弧、符号、巻、号又は頁を別表記へ変更しない。

各値を UTF-8 の `query1` query component として一回だけ percent-encode し、統合検索へ `GET` を一回送る。自動再試行、継続 page、別カテゴリー検索又は追加条件を使用せず、外向き request は合計二回を超えない。

## 候補対応

- 各結果行は既存 `judicial-decision.search@1` の mapping を再利用して `JudicialDecisionSummary` と `ref` を構成する。
- ルート裁判例と同じ `ref` は除外する。
- 事件番号検索、次いで判例集表記検索の順に処理し、各結果内の公式 DOM 順を保持する。
- 同じ `ref` を持つ候補は一件へ統合し、初出位置を候補順に使用する。再出現時は evidence を検索順と DOM 順のまま既存 item の末尾へ加える。
- `limit` で切り詰める前に自己参照除外と `ref` 基準の重複排除を行う。
- 初出順の先頭から実効 `limit` 件、最大 10 件だけを返す。異なる掲載カテゴリー、事件番号、裁判所名、日付又は事件名を文字列類似で統合しない。
- 自己参照除外と重複排除後の候補数が実効 `limit` を超えた場合、又は公式 HTML が未取得の後続結果の存在を明示した場合だけ coverage の `truncated` を `true` にする。検索失敗は `truncated` ではなく partial と issue で表す。

## 根拠

各候補には `evidenceLevel=official_search_candidate` の evidence を一件以上付ける。evidence の provenance は当該検索 HTML、取得時刻および `SOT-IF-073` の method ID を持つ。検索語を item 又は coverage へ含めず、公式検索への出現を `exact_text_match`、確認済み引用又は引用回数へ読み替えない。

二検索の一方だけが失敗した場合は成功検索の候補を保持し、capability result を `partial`、失敗した attempt を issue とする。両検索が失敗した場合又は全域 cancellation だけを capability error とする。候補の詳細又は PDF を追加取得しない。

## 確認

事件番号検索、判例集表記付き二回検索、同一検索語の一回化、一回だけの percent-encoding、自己参照除外、重複 evidence 順、DOM 順、1 件と 10 件の上限、空結果、一検索失敗の部分結果、全検索失敗、候補 resource 非取得および検索内容の非露出を fixture で確認する。

## 関連

- [SOT-IF-069: `judicial-decision.citing-candidate.search` capability v1](69-judicial-citing-candidate-search-capability.md)
- [SOT-IF-072: 裁判所「裁判例検索」HTML 情報源 v2](72-source-courts-hanrei-html-v2.md)
