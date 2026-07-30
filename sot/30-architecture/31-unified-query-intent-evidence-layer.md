# SOT-ARCH-031: 統合照会の意図根拠レイヤ

- 状態: 有効

## 規定

統合照会は、日本語の法情報照会を LLM や自由推論で分類せず、共通前処理が抽出した位置付き事実を、固定順の意図根拠レイヤへ当てる決定的な手順で解釈する。

このレイヤは、どの根拠をどの順で意味解釈へ使えるかを定める内部規則である。score、pack の有効状態、外部結果、通信状態または利用者履歴を入力にせず、`SOT-ARCH-023` の選択条件、`SOT-MODEL-023` の decision または `SOT-IF-051` の公開 status をここで再定義しない。

五層は新しい公開 model または永続する中間 DTO を追加するものではない。
各 query profile が、同じ `LegalQueryPreprocessResult` から候補 draft、
`QueryProfileSignal`、`selectionMode` および既存の制約を作る際の評価順を
表す。各層の一時的な判定を `QueryProfileContribution` へ新しい field として
保存しない。

## 意図根拠レイヤ

一つの照会文に対する意図解釈は、次の五層を上から順に評価する。

| レイヤ | 主な根拠 | 役割 |
|---|---|---|
| `boundary` | `standalone_structured_query`、`non_japanese_query`、入力 `ref`、`SOT-ENG-028` の対象外 relation | 実行対象外、構造化入力または非実行境界を先に確定する |
| `explicit_task_resource` | task cue relation、`SOT-ENG-031` が定める明示 resource cue、明示 task cue | 採用済み task/resource の候補を決める |
| `target_anchor` | 公式識別子、法令名、条項、事件番号、日付、明示検索対象 | task/resource に対応する取得対象を束縛する |
| `semantic_expansion` | 法概念、一般検索語、弱い一般語 | 明示根拠が不足する候補を補助的に広げる |
| `clarification_or_reject` | 略称衝突、弱い一般語だけの競合、五 step 以上、対象外との混在 | 非実行または明確化へ落とす |

下位レイヤは、上位レイヤが確定した非実行境界、task/resource、対象束縛または明確化要求を上書きしない。

一つの照会文が複数主題へ分かれる場合は、`boundary` を照会全体へ先に確認した後、`SOT-ARCH-025` が分離した各主題に対して `explicit_task_resource` から `clarification_or_reject` までを独立に適用する。

## 評価順序

1. `boundary` で `standalone_structured_query`、非日本語境界、入力 `ref` の read 制約、および対象外 relation を確認する。
2. `explicit_task_resource` で、同じ節または同じ構造化参照に結び付く明示 task/resource を決める。
3. `target_anchor` で、どの法令、条文、裁判例、事件番号、日付または検索語が、その task/resource の対象かを束縛する。
4. `semantic_expansion` は、上記だけでは一意化しない場合に限り、法概念辞書または一般検索語から候補を追加する。
5. `clarification_or_reject` で、残った競合、対象外との混在および非採用
   task/resource を、`SOT-MODEL-026` の `selectionMode`、signal または既存の
   constraint へ変換する。公開 decision と reason への変換は
   `SOT-ARCH-023` と `SOT-MODEL-023` に従い、この層が直接決めない。

現行の固定 profile set における positive task の role 対応と relation 成立条件は
`SOT-ENG-032` を定義元とし、本規定では profile ごとの現在値を重複して定義しない。

`explicit_task_resource` がない場合でも、法概念辞書または一般検索語だけから候補を作ることはできる。ただし、その候補は明示 task/resource による候補より弱く、上位レイヤの境界を破れない。`boundary` が非実行を確定した場合でも、`SOT-MODEL-026` が許可する内部候補保持に限り、下位レイヤは監査用の `LegalQueryCandidate` を組み立てられる。これらは選択または実行の対象にせず、非実行理由を覆さない。

この評価順序は profile 共通の解釈境界であり、ある profile が下位レイヤの一般語だけで明示 task/resource を捏造したり、別 profile の cue relation を借りて自分の候補を強化したりしてはならない。

## 適用責任と一時データ

共通前処理は、位置付き出現、構造化参照、節、token および
`CueTaskRelation` までを返し、各出現を本規定の五層へ分類しない。

各 query profile は、自身の候補 draft を作る過程で、五層を一回だけ順番に
適用する。profile は、各 draft の各 step と、その step を成立させた原文 span、
evidence code および意図根拠レイヤを、profile 内の一時的な対応として保持できる。
この対応は、候補の根拠検証、対象外との混在時の候補 scope、および
`SOT-ARCH-032` の evidence cluster を確定するためだけに使用する。

一時的な対応は一 request の profile 評価中だけ保持し、
`LegalQueryPreprocessResult`、`LegalQueryCandidate`、
`QueryProfileContribution` または profile set result に新しい field として
保存しない。contribution を構築した時点で破棄し、別 profile、candidate
composer、selector または executor へ渡さない。

candidate composer と selector は、完成した contribution の候補、signal、
selection mode、composition member および constraint だけを検証する。
原文を再解析したり、失われた一時対応を score、候補近接または別 profile の
根拠から復元したりして、五層を再適用しない。

## 実行適格な同じ語の複数解釈

同じ原文表記が複数の採用済み意味へ対応し得る場合も、profile は意味ごとに
五層を独立して適用し、ある意味の `target_anchor` または
`semantic_expansion` を別の意味へ流用しない。

五層の評価後に、複数の候補 draft のうちどれを実行適格な分岐として保持するかは
`SOT-ARCH-032` を定義元とする。本規定は保持 margin、保持件数、
`selectionMode` または実行可否を重複して定義しない。

`SOT-MODEL-026` が対象外との混在を監査するために保持できる内部候補は、同 SOT の
候補・step ごとの強い根拠条件を満たすものに限り、選択または実行の候補へ
戻さない。

単純な語の列挙、長い照会全体の広い検索、または provider 固有の知識だけから、別の task/resource を補ってはならない。

## 節と主題の境界

意図根拠レイヤは、`SOT-ARCH-021` の節境界と、各 profile の採用済み
task/resource に適用できる `SOT-ARCH-025` の複数主題分離を前提に評価する。
同じ語が別の節へ裸で現れただけでは、先行する task/resource へ接続しない。
二つ以上の主題を同規定で分離できる照会は、まず主題の分離を保ち、その後に
各主題へ意図根拠レイヤを適用する。

## 禁止する解釈

- 日本語以外の自然言語を翻訳して採用済み意図へ変換すること
- LLM、外部分類器または利用者履歴で task/resource を補うこと
- 対象外 relation がある照会を「取得意図が少しでもあるから」と部分実行へ読み替えること
- 裸の法令名、裸の裁判例語または弱い一般語だけで read step を作ること
- 事件番号、題名、URL または provider 固有 ID だけから裁判例 read を推測すること
- ある profile の cue 近接だけから、別 profile の task/resource を補うこと

## 確認

少なくとも次を、profile test、planner test および統合評価 fixture で確認する。

- `民法第709条を見せてください。` は、明示 read と条文対象が `explicit_task_resource` と `target_anchor` で先に確定する
- `永住許可と帰化について教えてください。` は、二主題分離後にそれぞれの検索候補へ解釈される
- `営業秘密` のような法概念だけの語は、`semantic_expansion` の弱い候補として保持できても、上位の明示 read を上書きしない
- `民法第103条を引用する裁判例の影響グラフを作成してください。` は、対象外 relation により非実行境界が先に立ち、取得候補を実行しない
- `qznnsidvcvfxqirm` のような非日本語入力は、法情報照会へ翻訳または再分類しない
- `この ref を読んでください。` は、入力 `ref` と read cue が一致する場合だけ read 候補になる

## 関連

- [SOT-PROD-011: 統合法情報照会の製品範囲](../00-product/11-unified-legal-query-scope.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](../20-model/25-legal-query-preprocess-result.md)
- [SOT-MODEL-026: QueryProfileContribution](../20-model/26-query-profile-contribution.md)
- [SOT-MODEL-028: QueryCandidateCompositionMember](../20-model/28-query-candidate-composition-member.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](21-provider-independent-query-preprocessing.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](22-unified-query-planning-pipeline.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](27-unified-query-cross-profile-composition.md)
- [SOT-ARCH-032: 統合照会の限定分岐保持](32-unified-query-bounded-branch-retention.md)
- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](33-unified-query-profile-set-adoption-boundary.md)
- [SOT-ENG-028: 統合照会の対象外意図 cue セット](../50-engineering/28-unified-query-unsupported-intent-cues.md)
- [SOT-ENG-031: 統合照会の採用済み意図 cue セット](../50-engineering/31-unified-query-adopted-intent-cues.md)
- [SOT-ENG-032: 統合照会の positive cue role 対応](../50-engineering/32-unified-query-positive-cue-role-mapping.md)
