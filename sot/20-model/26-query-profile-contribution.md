# SOT-MODEL-026: QueryProfileContribution

- 状態: 有効

## 規定

`QueryProfileContribution` は、一つの query profile が位置付き前処理事実から生成した意味候補、安全信号および候補間の選択関係を、selector が provider を選ぶ前に検証できる内部モデルとする。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `profileId` | string | はい | contribution を生成した profile の ID |
| `profileVersion` | string | はい | profile 固有の規則と辞書を特定する版 |
| `rankingVersion` | string | はい | 複数 profile 間で共有する score、confidence、閾値、margin および tie-break の校正版 |
| `candidates` | `LegalQueryCandidate[]` | はい | profile が完全順位を確定した候補。十六件以下 |
| `signals` | `QueryProfileSignal[]` | はい | 候補とは分離して selector が扱う入力境界または対象外の信号 |
| `selectionMode` | string | はい | `automatic` または `clarification_required` |
| `hedgePairs` | `CandidateHedgePair[]` | はい | profile が独立実行しても意味を変えないと確認した候補対 |
| `compositionMembers` | `QueryCandidateCompositionMember[]` | はい | 別 profile の明示意図と結合できる必須候補の位置付き sidecar |
| `compositionConstraint` | string | はい | `none` または、保持すべき明示意図が四 step 上限を超えたことを表す `step_limit_exceeded` |

`profileId`、`profileVersion` および `rankingVersion` は、profile の起動時検証済み metadata と完全に一致しなければならない。profile 固有の版は独立して変更できるが、異なる `rankingVersion` の contribution 間で `semanticScore` を比較してはならない。

`candidates` は `semanticScore` の非増加順とし、同点では profile が定めた
evidence、step 数、意味署名および原文位置の順をすでに確定しているものとする。
`SOT-ARCH-028` の法令別名衝突群が複数あり、その合計が固定候補上限を
超える場合に限り、法令コア profile は evidence と step 数の後、
公式別名群の原文位置を群内の意味署名より先に比較する。
selector は一 contribution 内の同点を再整列しない。

## 選択関係

`selectionMode=automatic` は、selector が score、閾値、margin および `hedgePairs` によって実行可否を決められることを表す。

`selectionMode=clarification_required` は、候補の score 差だけでは安全に意味を一意化できないことを profile が確認した状態とする。selector はこの contribution の候補を `single` または `hedged` として実行せず、外部情報源を呼ばない明確化へ渡す。

少なくとも次の場合は `clarification_required` とする。

- 同じ略称または別名が複数の法令へ衝突する
- 辞書 entry が自動実行しない複数 resource 候補を定める
- 弱い一般語から候補 resource を一意化できない
- 一候補の四 step 上限を超える複数主題を検出し、候補を切り捨てず非実行にする

四 step 上限を超えた場合は `compositionConstraint=step_limit_exceeded`、
`selectionMode=clarification_required` とし、`candidates`、
`hedgePairs` および `compositionMembers` を空にする。上限内の一部候補を
保持または実行してはならない。それ以外は
`compositionConstraint=none` とする。`SOT-ARCH-027` の
`composition_ineligible` は profile set 内部の制約であり、
`QueryProfileContribution` が出力してはならない。

`CandidateHedgePair` は `firstCandidateId` と `secondCandidateId` を持つ。両 ID は同じ contribution 内の相異なる候補を profile 順で参照し、逆順を含めて重複させない。二候補の step 合計は四件以下とする。

profile は、照会文が二つの代替検索を明示した場合、または独立した二つの出典付き概念候補を同時に取得しても意味を変えないと確認した場合だけ hedge pair を作る。略称衝突、自動実行しない辞書候補、同じ候補内の複数主題または検索結果に依存する候補を hedge pair にしない。

selector は上位二候補が一つの hedge pair と完全に一致する場合だけ `hedged` を検討する。異なる profile の候補を実行時に即席の hedge pair として組み合わせない。

`compositionMembers` は `SOT-MODEL-028` に従い、同じ contribution の候補と
step だけを参照する。member 候補と hedge pair の重複は構造として受理するが、
合成適格とはみなさず `SOT-ARCH-027` の composer が
`composition_ineligible` へ変換する。
`selectionMode=clarification_required` の contribution は member を持たない。
profile は候補を別 profile の候補と直接結合せず、構成可能性と原文位置だけを
sidecar として返す。複数候補または hedge と併存する sidecar は、
composer が合成不適格を判定するための位置付き入力であり、単独では
合成可能性を保証しない。

一つの contribution が複数の member を持つ場合、または member 以外の
代替候補を持つ場合も、構造自体は有効とする。ただし composer はその
contribution を使って合成せず、元候補を通常の意味順位へ保持する。
ほかの二つ以上の profile に一意な member がある場合、その合成まで
中止しない。部分実行の禁止は `SOT-ARCH-027` に従う。

## 信号と候補の保存

信号は次の値だけを固定順かつ重複なく持つ。

1. `non_japanese_query`
2. `standalone_structured_query`
3. `unsupported_legal_advice`
4. `unsupported_translation`
5. `unsupported_task_or_resource`
6. `reserved_pack_request`

非日本語入力は意味解釈を行わず候補を空にする。

`standalone_structured_query` は、`SOT-IF-051` が日本語照会文として扱わない、決定的な識別子、事件番号または日付だけの入力を表す。非空白文字の全体が前処理で検証した公式識別子、事件番号若しくは日付の span、またはそれらを区切る句読点と記号だけからなり、task、resource、法令名、法概念若しくは構造化 span と異なる一般検索語がない場合にだけ生成する。同じ span の `quoted_phrase` は、決定的な構造だけという判定を変更しない。

この判定は言語信号より先に行い、該当する場合は候補を空にして `standalone_structured_query` だけを返す。したがって ISO 日付だけのように日本語 script もない入力を `non_japanese_query` と重複させない。

法的助言、翻訳または未採用 task/resource が採用済みの取得意図と混在する場合、
profile は取得意図から生成できた候補を `candidates` に保持し、対象外信号も返す。
`unsupported_legal_advice` または `unsupported_translation` だけを理由に候補の
根拠強度を再評価してはならず、対象外 relation と別に明示された取得 task と
形態素文脈または一般語から生成できた候補も保持する。

`unsupported_task_or_resource` がある場合は、対象外 task/resource の説明中に
現れただけの一般語を採用済みの取得意図とみなさない。候補の保持可否は
contribution または照会文全体の真偽値ではなく、一候補ずつ独立に判定し、
別候補の cue、relation または根拠を流用しない。

profile は `LegalQueryCandidate` を materialize する前の全候補 draft で、各 step と、
その step を実際に成立させた `SOT-MODEL-022` の evidence code に対応する位置付き
前処理出現を一時的に対応付ける。通常の候補では `legal_concept` または
`general_term` の出現も対応に使用できるが、その根拠強度と自動選択可否を
明示 task/resource と同等に引き上げない。

`unsupported_task_or_resource` を持つ contribution で内部監査用に保持する候補は、
各 step が `official_identifier`、`structured_reference`、`explicit_resource`、
`official_alias` または `legal_concept` のいずれかの位置付き出現により、
採用済み取得対象を独立に根拠付けられなければならない。対象外 relation の
subject、predicate または `general_term` だけからこの強い対応を作らない。
一 step でも条件を満たさない候補は候補全体を除き、強い step だけへ縮約しない。

この step ごとの対応は生成時検証と `SOT-ARCH-032` の一時的な evidence cluster
key にだけ使用する。最終の `LegalQueryCandidate` は `SOT-ARCH-029` に従う
根拠コードの和集合を持ち、`QueryProfileContribution` はこの対応または span を
新しい field として保存しない。`compositionMembers` が保持する位置 sidecar は
profile 横断合成のための別契約とする。

対象外 relation と同じ節にある候補は、上記の強い対象根拠が同じ節にある場合に
内部候補として保持できる。別の節にある候補は、その節自身に採用済みの明示取得
task と対象根拠がある場合だけ保持する。別の節に法令名、法概念、構造化値または
一般語が裸で現れただけでは保持しない。

selector は保持された候補を内部順位として保存するが選択または部分実行しない。対象外意図しかない場合は候補を空にできる。

`reserved_pack_request` は採用済み pack への明示要求を表す非実行信号であり、それだけから selector が候補を捏造してはならない。`capability_unavailable` に必要な候補は、その意味を所有する採用済み profile contribution が `requiredPacks` と型付き step を付けて生成する。

## profile set への集約

profile set は contribution を composition root の固定順で回収する。すべての contribution は同じ `rankingVersion`、score policy、selection policy および tie-break を持たなければならず、不一致は起動または plan 作成を失敗させる。

全 contribution の検証後、`SOT-ARCH-027` の composer が
`compositionMembers` を消費し、必要な場合だけ新しい合成候補へ置き換える。
その後、候補は profile 順と profile 内順を保持して stable に score の
非増加順へ統合する。同点ではその stable 順を完全順序とする。合成後の全
候補が十六件を超える場合は切り捨てず失敗する。

profile set は、固定順の profile ID、各 profile version、ranking version、
辞書版、実際の校正値および `compositionVersion` を含む決定的な入力から、
`SOT-MODEL-023` の不透明な `profileVersion` を作る。同じ set では常に
同じ値とし、いずれかが変われば別の値とする。

一つでも `clarification_required` の contribution がある場合、集約結果も自動実行しない。signals は本規定の順序で和集合にする。候補 ID、step ID または意味署名が profile 間で衝突した場合は黙って統合せず失敗する。

一つでも `compositionConstraint=step_limit_exceeded` の contribution が
ある場合、または composer が合成対象の五 step 以上を検出した場合、
profile set result も同じ constraint を保持する。selector は通常の score
選択と pack 可否より前にこれを適用し、`SOT-MODEL-023` の
`step_limit_exceeded` だけを持つ非実行 plan を作る。

composer が automatic contribution の合成不適格を検出した場合は、
profile set result だけに内部値 `composition_ineligible` を保持できる。
profile はこの値を `QueryProfileContribution` に出してはならない。
`step_limit_exceeded` は `composition_ineligible` より優先する。
selector の非実行変換と公開境界は `SOT-ARCH-027` に従う。

## 不変条件

- getter は候補、信号、hedge pair および composition member の深い複製と、
  検証済み `compositionConstraint` を返す。
- profile、contribution および集約結果は照会中に変更しない。
- pack の有効状態、provider route、外部件数、応答速度または実行結果を contribution に持たない。
- selector、executor または adapter が profile の代わりに hedge pair を補わない。

## 確認

自動選択、明確化必須、明示二候補、略称衝突、独立した概念候補、二主題、四主題、五主題、非日本語入力、決定的な構造だけの入力、対象外との混在および予約済み pack を profile test で確認する。

対象外 relation と同じ節にある強い法令・条文根拠、別の節に明示した取得 task、
別の節に裸で現れただけの法令名、一候補内の強い step と弱い step、および
二候補の片方だけが持つ強い根拠を profile fixture にする。候補 draft の
step ごとの位置対応、候補ごとの保持、候補全体の除外、別候補の根拠非共有、
最終モデルへ位置対応を残さないこと、および保持候補の非選択・非実行を確認する。
通常の `general_term` 候補が位置対応を持てることと、対象外 relation の説明中に
ある `general_term` だけでは監査用候補を保持できないことを別 fixture にする。

存在しない候補を参照する hedge pair、自己参照、逆順重複、不正な
composition member、`step_limit_exceeded` と候補または自動選択の併存、
異なる ranking version、score policy
不一致、十七候補、profile 間 ID・意味衝突および同点順の変更を拒否することを
model test で確認する。

member と hedge pair の構造的な併存、および複数候補の位置 sidecar を
保持できることは model test で確認し、その合成不適格化は
`SOT-ARCH-027` の composer test で確認する。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](22-legal-query-candidate.md)
- [SOT-MODEL-023: LegalQueryPlan](23-legal-query-plan.md)
- [SOT-MODEL-028: QueryCandidateCompositionMember](28-query-candidate-composition-member.md)
- [SOT-MODEL-029: CueTaskRelation](29-cue-task-relation.md)
- [SOT-ARCH-029: 複数 step 候補の根拠保持](../30-architecture/29-multi-step-evidence-preservation.md)
- [SOT-ARCH-032: 統合照会の限定分岐保持](../30-architecture/32-unified-query-bounded-branch-retention.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](../30-architecture/22-unified-query-planning-pipeline.md)
- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](../30-architecture/23-unified-query-selection-and-hedging.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](../30-architecture/25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](../30-architecture/27-unified-query-cross-profile-composition.md)
- [SOT-IF-051: MCP `query_legal_information`](../40-interfaces/51-mcp-query-legal-information.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](../50-engineering/24-unified-query-evaluation-gate.md)
- [SOT-ENG-028: 統合照会の対象外意図 cue セット](../50-engineering/28-unified-query-unsupported-intent-cues.md)
