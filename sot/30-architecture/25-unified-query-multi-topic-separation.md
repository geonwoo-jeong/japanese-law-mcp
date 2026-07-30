# SOT-ARCH-025: 統合照会の複数主題分離

- 状態: 有効

## 規定

統合照会は、一つの照会文で個別取得を求められた二つ以上の主題を、論理積による一検索へ暗黙に結合せず、一つの意味候補内の独立した logical step として原文順に保持する。

## 分離規則

各 query profile は、共通前処理が保持する入力 `ref`、位置付きの法令名、法概念、
一般検索語、公式識別子、および同じ profile の明示 task/resource cue について、
次の優先順で論理条件を決める。入力 `ref` 自体には原文 span を補わない。

1. logical input が語の演算子を型として持ち、`all`、`any`または`exclude`の明示
   cue がある場合は、その演算子に従う一つの検索条件とする。対応する型を持たない
   task/resource では、演算子を削除して個別 step に読み替えず、
   `selectionMode=clarification_required`、`hedgePairs=[]` として
   `ambiguous_candidates` の明確化へ渡す。
2. 同じ profile の異なる task/resource が、それぞれ独立した明示 cue と
   取得対象を持つ場合は、能力別の別 step とする。
3. `それぞれ`、`個別に`、`一つずつ`若しくは`各々`の明示 cue、または後述する
   法令本文検索の共有末尾 cue の全条件を満たす場合は、主題ごとに独立した
   step とする。各主題は、同じ profile の採用済み task/resource と型付き
   logical input を独立に根拠付けられなければならない。
4. 上記の根拠がない複数語は、個別検索であると推測せず、profile の評価済み
   既定規則に従う。

## 法令本文検索の共有末尾 cue

同じ task/resource を示す末尾の cue を列挙全体へ共有できるのは、次の条件を
すべて満たす場合だけとする。

1. 同じ節に、法令名、法概念または一般検索語の位置付き出現が二件以上あり、
   原文順に重ならない
2. 隣接する主題間の byte 列は、Unicode White_Space と、`、`、`,`、`，`、
   `と`、`及び`、`および`、`並びに`または`ならびに`だけからなる
3. 最後の主題と末尾 cue の間は、Unicode White_Space と、任意の一つの
   `を`または`について`だけからなる
4. 末尾 cue は節末にあり、法令コア profile で一つの
   `law_content_search` task/resource に対応する
5. 列挙全体に `all`、`any`または`exclude`の明示 cue がなく、途中に別の
   task/resource cue または節境界がない

この確認は、共通前処理が返した原文、一回の token 列および位置付き出現の間を
閉じた separator 集合と照合するだけとする。新しい検索語、cue または relation を
発見する再 tokenization にはしない。条件を満たす場合は、`含む` の有無に
かかわらず列挙各項を同じ取得意図群として扱う。例えば
`永住許可、帰化を教えてください`は、共有された検索 task の cue を根拠に
二つの独立した `law_content_search` step とする。一条件でも満たさない単純列挙は、
同数の検索へ無条件に fan-out しない。

`all` は主題数ごとの別規則にしない。`すべて`、`全部`、`いずれも`、`両方`または数を表す語に続く`とも含む`などを同じ意味群として認識し、抽出した全検索語へ適用する。`二つとも`、`三つとも`のような個数別の語を上限なく列挙しない。

法令コア profile では、`教えて`または`教えてください`を、資料を根拠に情報を
取得する `law_content_search` の cue として扱う。ほかの profile へこの対応を
流用せず、回答文の合成、法的助言または法適用を求める語には読み替えない。

例えば`永住許可と帰化について教えてください`は、`永住許可`と`帰化`を別々の `law_content_search` step とする。`営業秘密と個人情報を両方含む条文を検索してください`は、二つの語を `allTerms` に持つ一つの `law_content_search` step とする。

明示されていない演算子を補うために、照会文全体を広い一検索として情報源へ渡したり、`any`へ読み替えたりしない。呼出し側の LLM は取得済み結果の説明と追加照会の判断を担えるが、固定上限のために返らなかった項目を判定できないため、取得意味の確定を後段へ委ねない。

法令コアにおける本規定の `all`、`any` および `exclude` は、一つの
`law_content_search` 内の検索語関係を表す。法令名、条文、更新一覧および
裁判例のように異なる task/resource をそれぞれ明示した照会は、検索語演算へ
平坦化しない。同じ profile が所有する task/resource は、その profile が先に
能力別 step を一候補へまとめる。異なる profile が所有する step だけを
`SOT-ARCH-027` で合成する。`含む` がなくても各 task/resource と取得対象が
独立に明示されていれば複数 step にできるが、resource との対応が不明な
単純列挙を無条件に fan-out しない。

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

`について`による法令本文の個別主題、明示的な`個別に`、`all`、`any`および
`exclude`を fixture にし、型が対応する明示演算子が個別分離より優先すること、
型が対応しない演算子を削除して別 step にしないこと、五主題を切り詰めず
`step_limit_exceeded` にすること、および executor が完了順で結果順を変えない
ことを確認する。

`二つとも含む`と`三つとも含む`が同じ `all` 規則になること、演算子の検索語上限を超えた場合に切捨てまたは広い検索へ変換しないことも確認する。

`永住許可、帰化を教えてください`と
`永住許可と帰化について教えてください`は共有された末尾 cue により二 step に
分離する。一方、末尾 cue がない単純列挙、途中に別 task cue がある列挙、
未知の接続表現を挟む列挙および明示 `all` cue がある列挙は、この規則で
fan-out しないことも確認する。

裁判例 profile では、二つから四つの独立検索対象と明示的な`それぞれ`または
`個別に`がある場合に、各対象を `judicial_decision_search` の別 step とする。
法令本文検索の共有末尾 cue、語の `all`、`any`または`exclude`を裁判例検索へ
流用しない。入力 `ref` の `judicial_decision_read` と別の明示
`judicial_decision_search` は、同じ `judicial-cases` profile が原文順の
複数 step を持つ一候補へまとめる。同じ profile 内の step 化に
`SOT-ARCH-027` の composer を使わない。法令コアと裁判例のように異なる
profile の組合せだけを、各 profile の step 化後に同 SOT で合成する。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](../20-model/25-legal-query-preprocess-result.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](21-provider-independent-query-preprocessing.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](22-unified-query-planning-pipeline.md)
- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](23-unified-query-selection-and-hedging.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](27-unified-query-cross-profile-composition.md)
- [SOT-IF-023: `law.content.search` capability v1](../40-interfaces/23-law-content-search-capability.md)
