# SOT-ARCH-025: 統合照会の複数主題分離

- 状態: 有効

## 規定

統合照会は、一つの照会文で個別取得を求められた二つ以上の主題を、論理積による一検索へ暗黙に結合せず、一つの意味候補内の独立した logical step として原文順に保持する。

## 分離規則

法令コアの query profile は、共通前処理が位置付き事実として抽出した法令名、法概念または一般検索語について、次の優先順で論理条件を決める。

1. `all`、`any` または `exclude` の明示 cue がある場合は、その演算子に従う一つの検索条件とする。
2. `それぞれ`、`個別に`、`一つずつ`若しくは`各々`の明示 cue、または二つ以上の並列主題に係る`について`の cue がある場合は、主題ごとに独立した step とする。
3. 上記の根拠がない複数語は、個別検索であると推測せず、profile の評価済み既定規則に従う。

`all` は主題数ごとの別規則にしない。`すべて`、`全部`、`いずれも`、`両方`または数を表す語に続く`とも含む`などを同じ意味群として認識し、抽出した全検索語へ適用する。`二つとも`、`三つとも`のような個数別の語を上限なく列挙しない。

`教えて`または`教えてください`は、資料を根拠に情報を取得する検索 task の cue として扱う。回答文の合成、法的助言または法適用を求める語には読み替えない。

例えば`永住許可と帰化について教えてください`は、`永住許可`と`帰化`を別々の `law_content_search` step とする。`営業秘密と個人情報を両方含む条文を検索してください`は、二つの語を `allTerms` に持つ一つの `law_content_search` step とする。

明示されていない演算子を補うために、照会文全体を広い一検索として情報源へ渡したり、`any`へ読み替えたりしない。呼出し側の LLM は取得済み結果の説明と追加照会の判断を担えるが、固定上限のために返らなかった項目を判定できないため、取得意味の確定を後段へ委ねない。

本規定の `all`、`any` および `exclude` は、一つの
`law_content_search` 内の検索語関係を表す。法令名、条文、更新一覧および
裁判例のように異なる task/resource をそれぞれ明示した照会は、検索語演算へ
平坦化せず、`SOT-ARCH-027` に従って能力別 step を一つの意味候補へ合成する。
`含む` がなくても各 task/resource と取得対象が明示されていれば複数 step に
できるが、resource との対応が不明な単純列挙を無条件に fan-out しない。

## 上限と結果

一つの意味候補に保持できる独立 step は `SOT-MODEL-022` に従い四件以下とする。
五つ以上の主題を黙って切り捨てず、
`compositionConstraint=step_limit_exceeded` を持つ
`QueryProfileContribution` として外部情報源を呼ばない明確化へ渡す。
selector は通常の候補不足または曖昧性へ読み替えず、
`SOT-MODEL-023` の同名 reason 一件だけを返す。

一つの `law_content_search` に保持する各演算子の検索語数は `SOT-IF-023` の上限に従う。上限を超えた語を切り捨てたり、広い一検索へ置き換えたりせず、外部情報源を呼ばず失敗させる。

selector は、同じ照会文で明示された複数主題を、近接する別の意味候補に対する hedge として扱わない。executor は各 step の予算と結果を別々に保持し、結果を一つの件数、順位または本文へ統合しない。公開結果は、各主題がどの interpretation と step から得られたかを辿れる形で返す。

## 確認

二主題、四主題、法令名、法概念、一般検索語およびそれらの組合せについて、独立 step の原文順、四 step 上限、結果の分離および固定予算を確認する。

`について`による個別主題、明示的な`個別に`、`all`、`any`および`exclude`を fixture にし、明示演算子が個別分離より優先すること、五主題を切り詰めず `step_limit_exceeded` にすること、および executor が完了順で結果順を変えないことを確認する。

`二つとも含む`と`三つとも含む`が同じ `all` 規則になること、演算子の検索語上限を超えた場合に切捨てまたは広い検索へ変換しないことも確認する。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](../20-model/25-legal-query-preprocess-result.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](21-provider-independent-query-preprocessing.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](22-unified-query-planning-pipeline.md)
- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](23-unified-query-selection-and-hedging.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](27-unified-query-cross-profile-composition.md)
- [SOT-IF-023: `law.content.search` capability v1](../40-interfaces/23-law-content-search-capability.md)
