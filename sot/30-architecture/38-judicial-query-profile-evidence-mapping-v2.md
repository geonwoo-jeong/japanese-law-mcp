# SOT-ARCH-038: 裁判例 query profile の ref 忠実な根拠対応

- 状態: 有効

## 規定

`judicial-cases` query profile は、共通前処理の位置付き事実を、裁判例検索と
裁判例読取りの step および private evidence mapping へ決定的に対応付け、
法令コアの列挙規則、provider route または pack の有効状態に依存せず、
各 step の主題と独立根拠を確定する。

## 所有境界

本規定は、`judicial-cases` profile における次だけを定義する。

- input kind ごとに利用できる前処理事実
- 事実と step の同一主題への束縛
- `topicOrdinal` と `evidenceSpan` を作るための profile 固有対応
- 追加分岐に必要な独立した正の根拠
- 法令コアの共有末尾列を利用しない境界

task、resource、capability および logical input の七組は `SOT-MODEL-022`、
位置付き事実と layer/evidence code の共通対応は `SOT-ARCH-031`、
裁判例に適用できる複数主題分離は `SOT-ARCH-025`、cluster key、同値 draft の
統合、margin および三件上限は `SOT-ARCH-037` を定義元とする。

`judicial-cases` profile metadata の `targets` は、次の input kind をこの順で
一件ずつ持つ。

1. `judicial_decision_search`
2. `judicial_decision_read`

両 input kind の候補は `requiredPacks=["judicial-cases"]` を持つ。pack の有効・
無効は候補、step、根拠、score、cluster または `requiredPacks` を変えず、
selector が実行可否だけを決める。

## private evidence mapping

private mapping の項目、request 内だけの寿命、公開禁止および競合時の扱いは
`SOT-ARCH-031` を定義元とする。本規定は、`judicial-cases` の各 step へどの
事実を対応させるかだけを定義する。

同じ候補の read と search、または複数の search step の間で根拠を複製しない。
候補 draft を結合する場合は、ordered logical input の意味署名と
`topicOrdinal` が一致する step 同士だけの mapping を和集合にし、配列 index、
共通 resource 名または一つの terminal cue だけで別 step の根拠を借りない。

順序と cluster 対応は `SOT-ARCH-037` に従う。範囲外 span または profile 固有の
禁止事実を含む mapping は不正とする。束縛が一意でない事実の除外後に
根拠のない主題が残る場合、および同じ事実の競合を検出した場合は contribution を
構築しない。

span を持たない入力 `ref` だけを target とする `judicial_decision_read` では、
照会全体の `direct_task` relation から、当該 logical input を成立させられる
`judicial-cases` の read relation が正確に一件だけあることを確認する。
resource は構造検証済み
`ref.resourceType=judicial-decision` から一意に定まるため、別の明示
judicial-decision resource cue を必須にしない。read relation と同じ節に
採用済み resource cue がある場合は、適用できる全 cue が
`judicial_decision` と両立しなければならない。
`resource_scope=legal_information` は明示 judicial-decision resource 根拠へ
昇格させず、これだけを read relation の一意性または cluster span に使わない。
一件の relation なら、その位置付き task cue を mapping と cluster に使用する。
零件または二件以上、競合 cue、若しくは複数意味へ残る cue がある場合は read
候補を作らず、配列の最初の cue、距離、照会全体または零長 span を代用しない。

contribution を構築する前に、read logical input の `ref` が候補生成 request の
入力 `ref` と `SOT-MODEL-022` の条件で完全一致することを確認する。欠落または
不一致時の失敗単位は同 SOT の候補 draft 全体とし、本規定で再定義しない。
`judicial-cases` 固有の制約として、事件番号、題名、URL、検索結果、
route metadata または既定 provider から代替 `ref` を合成しない。

## input kind ごとの対応

| input kind | 主題と `topicOrdinal` | 明示根拠 | target | semantic | 独立した正の根拠 |
|---|---|---|---|---|---|
| `judicial_decision_search` | `SOT-ARCH-025` の明示的な個別分離を適用した事件番号、完全な日付、検索語または法概念を原文順に数える。分離しない場合は全 step を `1` とする | 同じ節で対象を束縛する `judicial-cases` の search `direct_task` relation と judicial-decision resource cue | 完全な事件番号、完全な暦日および `quoted_phrase` | 公的 source を持つ法概念と `morphological_phrase` | 完全な事件番号、search/resource の明示根拠と同じ主題にある完全な日付若しくは明示検索対象、公的 source を持つ法概念、または `SOT-ARCH-025` で独立主題とした一般検索語。task/resource cue だけ、日付だけ、または broad な legal-information resource cue だけでは足りない |
| `judicial_decision_read` | 入力 `ref` は span を持たないため、それを束縛した一件の `judicial-cases` read `direct_task` relation を位置付き主題とする。同一候補に search がある場合は、read cue と search target の原文順で step を並べる。分離しない場合は `1` とする | 前節の閉じた例外に従う一件の `judicial-cases` read `direct_task` relation。resource cue は必須にしないが、同じ節に存在する全 cue の両立を要求する | span を持つ target は利用しない。入力 `ref` だけを `boundary/official_identifier` として利用する | 利用しない | 構造を検証した、`resourceType=judicial-decision` かつ `versionId` を持たない request と同一の `ref`、および一件の明示 read relation の両方が必要 |

裁判例検索の初期採用範囲では、法令名、law ID、revision ID、法令番号、article
または paragraph を検索対象へ直接変換しない。これらを裁判例検索へ採用する場合は、
法令との関係をどの検索語へ投影するかを定める新しい有効な SOT、capability
契約および評価 corpus を先に追加する。法令コアの target 対応を流用しない。

事件番号、題名、URL、日付、provider 固有 ID または検索結果の第一件から
`judicial_decision_read` の `ref` を捏造しない。read 用 `ref` の provider、
source および resource ID を profile が別値へ読み替えず、binding の照合は
registry と materializer に残す。

## 複数主題と共有末尾列

`judicial-cases` profile は、二件から四件の検索対象に明示的な `それぞれ`、
`個別に`、`一つずつ`または`各々`が対応するときだけ、
`SOT-ARCH-025` に従って別の `judicial_decision_search` step にする。
法令本文検索の `SharedTerminalSequence`、`all`、`any` または `exclude` の
演算子を裁判例検索へ流用しない。

同じ span から複数の事件番号、法概念または検索語の意味が得られる場合は
一件へ推測せず、同じ `topicOrdinal` と `evidenceSpan` を持つ代替 draft とする。
異なる span に同じ検索意味が反復し、明示的な個別分離が成立する場合は、法令コアの
共有末尾列における同値縮約を流用せず、位置の異なる独立 step として保持する。
各主題の第一候補と追加意味は、当該主題の型付き logical input、位置付き正の根拠
および同じ主題へ一意に束縛した task/resource 根拠だけを持つ topic-local draft を、
`SOT-ARCH-037` の正規化済み score と通常 `tieBreak` で比較して決める。
別主題の根拠、入力配列順または法令別名の条件付き順を使わない。

入力 `ref` の read と別の明示 search は、各 step が自身の独立根拠を満たす場合に、
同じ `judicial-cases` profile が原文順の複数 step を持つ一候補へまとめる。
この profile 内結合に `SOT-ARCH-027` の composer を使わない。合計五 step 以上は
切り捨てず `step_limit_exceeded` とする。

## cluster 対応

各 step の `evidenceSpan` は、上表で当該 step に利用可能かつ実際にその step を
生成した事実だけから、`SOT-ARCH-037` の
`explicit_task_resource`、`target_anchor`、`semantic_expansion` の優先順で
一件を選ぶ。

`judicial_decision_read` の入力 `ref` は span を持たないため、同じ read に
束縛した一件の read relation の位置付き task cue span を使う。同じ節の
resource cue は互換性確認に使えるが、それがないことだけを理由に拒否しない。
一件の read relation がなければ read 候補自体を作らず、零長 span、照会全体、
「参照」らしい一般語、最初の resource cue または provider ID を代用しない。

同じ cluster と完全な意味署名を持つ draft の統合、代表 draft、根拠と
`conceptSources` の和集合および source tuple 衝突時の失敗は
`SOT-ARCH-037` に従う。別の search subject、read `ref` または法令コア候補から
根拠を借りて margin 内へ昇格させない。

## provider と実行状態からの独立

候補生成、主題、根拠、score および cluster の入力に、provider ID、source ID、
canonical URL、HTML path、endpoint、query parameter、adapter DTO、検索結果、
route 優先、通信成否または稼働状態を使用しない。

入力 `SourceResourceRef` は共通の不透明参照として、`resourceType` と
`versionId` の構造だけを profile で確認する。provider/source の registry 対応、
capability binding、pack availability および materialization は後段へ残す。
この構造確認は、request と logical input の typed `ref` 完全一致確認を省略する
理由にしない。候補生成時に courts-hanrei その他の provider route を選ばない。

## 確認

外部 network を使わない profile、cluster および integration test で、少なくとも
次の固定 test ID を確認する。

- `judicial-evidence-mapping-input-kinds`
- `judicial-evidence-mapping-private-lifetime`
- `judicial-evidence-mapping-topic-positive`
- `judicial-evidence-mapping-ref-no-span`
- `judicial-shared-terminal-rejected`
- `judicial-evidence-mapping-pack-provider-invariant`
- `judicial-evidence-mapping-fail-closed`
- `judicial-multi-step-evidence-step-local-normalization`
- `judicial-bounded-non-cartesian-alternatives`

検索の許可事実と禁止事実、read `ref`、read と search の結合、一から四主題、
五主題、同一 span の別意味、異なる span の同一意味、法令名・条項の非採用、
別 profile cue、pack 有効・無効および provider metadata の差を fixture にする。

`judicial-evidence-mapping-ref-no-span` では、一件、零件および二件以上の
`judicial-cases` read relation、互換 resource cue なし、互換 cue あり、競合 cue
あり、`resource_scope=legal_information` だけ、別節だけの cue、および
複数意味へ残る cue を組み合わせる。一件の relation では位置付き task cue を
cluster に使用し、別の明示 resource cue を要求しない。
この固定検証 ID は、span、read relation、resource cue と `resourceType` の
閉じた束縛だけを所有し、request と logical input の `ref` 完全一致は
`judicial-evidence-mapping-fail-closed` で確認する。

一つの事実を同じ draft の read と search、二つの search step または複数
`topicOrdinal` へ束縛できる曖昧な節、同じ draft・step・事実へ競合する layer/code
または cluster span 可否を与えた mapping、範囲外 span および独立した正の根拠を
失った主題を fixture にし、一方を選んだ contribution または部分的な mapping を
作らないことを確認する。
`judicial-evidence-mapping-fail-closed` は、これらに加えて logical input と
request の `ref` 不一致を含み、別 `ref` への置換または read step だけの削除を
許可しない。request と同じ `ref` の正常系、request に `ref` がない場合、および
provider、source、resource、version 若しくは version の存在状態、大文字小文字、
先頭の零、Unicode byte または区切り文字だけが異なる構造上有効な `ref` を
fixture にする。

`judicial-multi-step-evidence-step-local-normalization` では、read と search
または複数 search を一候補へまとめる場合に、各 step の同値経路を先に
正規化してから候補全体へ和集合する。別 step の強い根拠を理由に独立 step の
弱い根拠を削除せず、最終順序が `SOT-ARCH-036` に一致することを確認する。

`judicial-bounded-non-cartesian-alternatives` では、明示的に個別分離した二つ以上の
主題がそれぞれ複数意味を持つ場合に、基準 draft と一主題ずつの置換だけを
原文順で評価し、複数主題を同時に置換した Cartesian 組合せを作らない。
限定代替列の後に、`SOT-ARCH-037` の同値縮約、margin、三件上限および
四件目の明確化を適用する。同一 topic に事件番号または明示検索語、法概念および
`morphological_phrase` の複数意味を与え、topic-local draft の正規化済み score と
通常 `tieBreak` から第一候補と全追加意味の順が一意に決まり、入力配列順を変えても
同じになることを確認する。

## 関連

- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](../20-model/25-legal-query-preprocess-result.md)
- [SOT-MODEL-027: JudicialCaseNumberMention](../20-model/27-judicial-case-number-mention.md)
- [SOT-MODEL-031: SharedTerminalSequence](../20-model/31-shared-terminal-sequence.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-036: 複数 step 候補の step 内根拠正規化と保持](36-multi-step-evidence-normalization.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](31-unified-query-intent-evidence-layer.md)
- [SOT-ARCH-037: 統合照会の正規化済み限定分岐保持](37-unified-query-normalized-branch-retention.md)
- [SOT-IF-042: `judicial-decision.read` capability v1](../40-interfaces/42-judicial-decision-read-capability.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](../50-engineering/24-unified-query-evaluation-gate.md)
- [SOT-ENG-035: 統合照会 profile metadata 成果物契約](../50-engineering/35-unified-query-profile-metadata-artifact-contract.md)
