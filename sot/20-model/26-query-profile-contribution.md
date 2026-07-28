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

`profileId`、`profileVersion` および `rankingVersion` は、profile の起動時検証済み metadata と完全に一致しなければならない。profile 固有の版は独立して変更できるが、異なる `rankingVersion` の contribution 間で `semanticScore` を比較してはならない。

`candidates` は `semanticScore` の非増加順とし、同点では profile が定めた evidence、step 数、意味署名および原文位置の順をすでに確定しているものとする。selector は一 contribution 内の同点を再整列しない。

## 選択関係

`selectionMode=automatic` は、selector が score、閾値、margin および `hedgePairs` によって実行可否を決められることを表す。

`selectionMode=clarification_required` は、候補の score 差だけでは安全に意味を一意化できないことを profile が確認した状態とする。selector はこの contribution の候補を `single` または `hedged` として実行せず、外部情報源を呼ばない明確化へ渡す。

少なくとも次の場合は `clarification_required` とする。

- 同じ略称または別名が複数の法令へ衝突する
- 辞書 entry が自動実行しない複数 resource 候補を定める
- 弱い一般語から候補 resource を一意化できない
- 一候補の四 step 上限を超える複数主題を検出し、候補を切り捨てず非実行にする

`CandidateHedgePair` は `firstCandidateId` と `secondCandidateId` を持つ。両 ID は同じ contribution 内の相異なる候補を profile 順で参照し、逆順を含めて重複させない。二候補の step 合計は四件以下とする。

profile は、照会文が二つの代替検索を明示した場合、または独立した二つの出典付き概念候補を同時に取得しても意味を変えないと確認した場合だけ hedge pair を作る。略称衝突、自動実行しない辞書候補、同じ候補内の複数主題または検索結果に依存する候補を hedge pair にしない。

selector は上位二候補が一つの hedge pair と完全に一致する場合だけ `hedged` を検討する。異なる profile の候補を実行時に即席の hedge pair として組み合わせない。

## 信号と候補の保存

信号は次の値だけを固定順かつ重複なく持つ。

1. `non_japanese_query`
2. `unsupported_legal_advice`
3. `unsupported_translation`
4. `unsupported_task_or_resource`
5. `reserved_pack_request`

非日本語入力は意味解釈を行わず候補を空にする。

法的助言、翻訳または未採用 task/resource が採用済みの取得意図と混在する場合、profile は取得意図から生成できた候補を `candidates` に保持し、対象外信号も返す。`unsupported_legal_advice` または `unsupported_translation` だけを理由に候補の根拠強度を再評価してはならず、明示された取得 task と形態素文脈または一般語から生成できた候補も保持する。

`unsupported_task_or_resource` がある場合は、対象外 task/resource の説明中に現れただけの一般語を採用済みの取得意図とみなさない。profile は `official_identifier`、`structured_reference`、`explicit_resource`、`official_alias` または `legal_concept` のいずれかで採用済み取得対象を独立に根拠付けられる候補だけを保持し、それ以外の偶発的な候補を除く。

selector は保持された候補を内部順位として保存するが選択または部分実行しない。対象外意図しかない場合は候補を空にできる。

`reserved_pack_request` は採用済み pack への明示要求を表す非実行信号であり、それだけから selector が候補を捏造してはならない。`capability_unavailable` に必要な候補は、その意味を所有する採用済み profile contribution が `requiredPacks` と型付き step を付けて生成する。

## profile set への集約

profile set は contribution を composition root の固定順で集約する。すべての contribution は同じ `rankingVersion`、score policy、selection policy および tie-break を持たなければならず、不一致は起動または plan 作成を失敗させる。

候補は profile 順と profile 内順を保持して stable に score の非増加順へ統合する。同点ではその stable 順を完全順序とする。全 profile の候補が十六件を超える場合は切り捨てず失敗する。

profile set は、固定順の profile ID、各 profile version、ranking version、辞書版および実際の校正値を含む決定的な入力から、`SOT-MODEL-023` の不透明な `profileVersion` を作る。同じ set では常に同じ値とし、いずれかが変われば別の値とする。

一つでも `clarification_required` の contribution がある場合、集約結果も自動実行しない。signals は本規定の順序で和集合にする。候補 ID、step ID または意味署名が profile 間で衝突した場合は黙って統合せず失敗する。

## 不変条件

- getter は候補、信号および hedge pair の深い複製を返す。
- profile、contribution および集約結果は照会中に変更しない。
- pack の有効状態、provider route、外部件数、応答速度または実行結果を contribution に持たない。
- selector、executor または adapter が profile の代わりに hedge pair を補わない。

## 確認

自動選択、明確化必須、明示二候補、略称衝突、独立した概念候補、二主題、四主題、五主題、対象外との混在および予約済み pack を profile test で確認する。

存在しない候補を参照する hedge pair、自己参照、逆順重複、五 step 以上、異なる ranking version、score policy 不一致、十七候補、profile 間 ID・意味衝突および同点順の変更を拒否することを model test で確認する。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](22-legal-query-candidate.md)
- [SOT-MODEL-023: LegalQueryPlan](23-legal-query-plan.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](../30-architecture/22-unified-query-planning-pipeline.md)
- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](../30-architecture/23-unified-query-selection-and-hedging.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](../30-architecture/25-unified-query-multi-topic-separation.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](../50-engineering/24-unified-query-evaluation-gate.md)
