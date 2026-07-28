# SOT-MODEL-023: LegalQueryPlan

- 状態: 有効

## 規定

`LegalQueryPlan` は、照会文に対する意味候補の順位、pack の実行可否、選択結果、能力別 request の materialization および一リクエストの固定予算を表す内部モデルとする。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `language` | string | はい | 固定値 `ja` |
| `profileVersion` | string | はい | 重み、閾値、根拠規則および辞書版を特定する不透明な値 |
| `decision` | string | はい | `single`、`hedged`、`needs_clarification`、`capability_unavailable` または `unsupported` |
| `rankedCandidates` | `LegalQueryCandidate[]` | はい | 意味 score 順の内部候補。十六件以下 |
| `selected` | `LegalQueryPlanSelection[]` | はい | 実行または非実行の判断対象とした候補 |
| `reasonCodes` | `string[]` | はい | 非実行または選択方法を表す決定的な理由 |
| `budget` | `LegalQueryBudget` | はい | リクエスト全体の上限と step ごとの確定値 |

## 選択と実行可能性

`profileVersion` は有効な UTF-8 で 1 byte 以上 128 byte 以下とし、先頭若しくは末尾の Unicode White_Space および位置を問わず Unicode control character を含めない。構造を解釈せず、同じ `profileVersion` の候補間だけで score を比較する。

この値は `SOT-MODEL-026` の active profile set 全体を特定する不透明な版とする。各 query profile は独立した `profileVersion` を持つことができるが、同じ `rankingVersion` と校正値を持つ contribution だけを一つの set へ集約する。plan の `profileVersion` は固定順の全 profile 版、ranking version、辞書版および校正値のいずれかが変われば別の値とし、個別 profile の score を版の異なる set 間で比較しない。

`LegalQueryPlanSelection` は `candidateId`、`availability` および `requiredPacks` を持つ。`requiredPacks` は参照先候補の同名配列と完全に一致し、空配列を許す。`availability` は次のいずれかとする。

- `available`: `requiredPacks` の全 pack が有効
- `pack_disabled`: `requiredPacks` が一件以上あり、そのうち一件以上の採用済み拡張パックが無効

候補を意味 score で順位付けした後に pack の実行可否を付与する。pack の状態、ネットワーク状態、応答速度または過去の件数を `semanticScore` へ加えず、利用できない上位候補を理由に意味の弱い別 resource を繰り上げない。

`decision=single` は一つ、`decision=hedged` は二つの `available` な候補だけを実行対象とする。上位二候補のどちらかが `pack_disabled` で安全な一意解釈を選べない場合は、利用できる候補だけを実行せず `capability_unavailable` とする。残りの decision は外部情報源を呼び出さない。

`needs_clarification` の `selected` は、利用者へ示す候補を零件以上二件以下の `available` な selection として持つことができる。`capability_unavailable` は一件以上二件以下の `pack_disabled` な selection を持つ。`unsupported` は採用済みの task/resource 候補を `selected` にせず空にし、対象外、言語境界違反または混在要求を `reasonCodes` で表す。混在要求の判定根拠を内部で検査できるように、`unsupported` の `rankedCandidates` は、取得 span から既に生成された採用済み task/resource 候補に限って零件以上十六件以下を保持できる。この内部候補を選択または部分実行してはならない。

一つの照会文に、法的助言、未採用 task/resource または翻訳と、実行可能な取得意図が混在する場合は、実行可能な部分だけを抜き出さず `unsupported` とする。一つの複数 step 候補に無効な pack の step が含まれる場合も、他 step を部分実行せず `capability_unavailable` とする。

法的助言または翻訳だけを求め、採用済みの取得候補を生成できない場合は、採用範囲外の task として `unsupported_task_or_resource` を使用する。`mixed_unsupported_intent` は、採用済みの取得候補を内部順位に保持できる場合だけ使用する。

公開する法令コア route と、有効な pack が必要とする route・binding・materializer は起動時にすべて検証する。欠落または不整合があれば transport を開始せず、正常起動後の候補選択で route 不備を availability として扱わない。

## 決定理由

`reasonCodes` は次の値だけを、表の順序で重複なく保持する。

| 順序 | 値 | 意味 |
|---:|---|---|
| 1 | `single_clear_candidate` | 単独選択の score と margin を満たした |
| 2 | `hedged_close_candidates` | 実行可能な上位二候補が hedge 条件を満たした |
| 3 | `below_execution_threshold` | 候補が最低実行閾値を満たさない |
| 4 | `ambiguous_candidates` | 安全に一意化または二候補化できない |
| 5 | `required_pack_disabled` | 必要な採用済み拡張パックが無効 |
| 6 | `non_japanese_query` | 日本語入力境界を満たさない |
| 7 | `mixed_unsupported_intent` | 取得意図と対象外意図が混在する |
| 8 | `unsupported_task_or_resource` | task または resource が採用範囲外である |

decision ごとの組合せは次に限定する。

- `single`: `single_clear_candidate` 一件
- `hedged`: `hedged_close_candidates` 一件
- `needs_clarification`: `below_execution_threshold` または `ambiguous_candidates` を一件以上二件以下
- `capability_unavailable`: `required_pack_disabled` 一件
- `unsupported`: `non_japanese_query`、`mixed_unsupported_intent` または `unsupported_task_or_resource` を一件以上三件以下

## 固定予算

`LegalQueryBudget` は次の上限を持ち、profile や provider によって拡張しない。

| 項目 | 値 | 意味 |
|---|---:|---|
| `maxRankedCandidates` | `16` | 評価して plan に保持できる意味候補数 |
| `maxSelectedCandidates` | `2` | 選択できる代替解釈数 |
| `maxParallelCandidates` | `2` | 同時に実行できる独立候補数 |
| `maxCapabilityCalls` | `4` | 一リクエストで行う論理 capability 呼出し数 |
| `maxItemsPerCollectionStep` | `20` | 一つの検索または一覧 step の公開上限 |
| `maxReturnedItems` | `40` | read item を含む全 step の公開 item 数 |
| `firstPageOnly` | `true` | 統合照会内で継続取得しないこと |

この七項目に加えて、`LegalQueryBudget` は request で既定値適用後の `limitPerAttempt`、`readStepCount`、`collectionStepCount` および計画順の `stepBudgets` を持つ。

`LegalQueryStepBudget` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `candidateId` | string | はい | selection が参照する候補 |
| `stepId` | string | はい | 候補内の step |
| `reservedItems` | integer | はい | read は `1`、collection は `0` |
| `effectiveLimit` | integer | 条件付き | collection だけが持つ確定上限 |

`single` と `hedged` の `stepBudgets` は、`selected` の候補順と各候補の step 順で全実行 step を一度ずつ持つ。外部呼出しを行わない三 decision では、`readStepCount` と `collectionStepCount` を零、`stepBudgets` を空にし、selection を実行予約へ読み替えない。

## item 予算の配分

選択した plan を実行する前に、計画順を固定し、read step 数を `R`、検索または一覧 step 数を `C` とする。各 read step は成功時の一 item を先に予約する。

`C > 0` の場合、全 collection step に同じ次の `effectiveLimit` を割り当てる。

```text
effectiveLimit = min(
  limitPerAttempt,
  floor((maxReturnedItems - R) / C),
  maxItemsPerCollectionStep
)
```

step は最大四つなので `effectiveLimit` は一以上となる。`C == 0` の場合は collection limit を作らない。

検索 capability request は `effectiveLimit` を使用し、continuation を省略する。`law.update.list@1` は完全一覧を取得する既存契約を変更せず、executor の能力結果 mapping が計画順の先頭から `effectiveLimit` 件だけを attempt の公開 preview へ投影し、残件を `hasMore` と情報源の正確な `totalCount` で示す。最終 result assembler は検証済み attempt を再切出しせず保持する。

空結果、失敗または実際の返却件数が上限未満でも、未使用分を後続 step へ再配分しない。これにより並列の完了順と結果件数で公開結果が変わらない。

retry、adapter 内の通信および provider ごとの同時実行制御は、各能力と情報源の既存予算に従う。統合照会はそれらの上限を緩和しない。

## request materialization

計画確定後、executor は計画順に各 step の確定済み `LegalQueryStepBudget` を渡し、`SOT-ARCH-026` に従って binding の選択と既存 capability request の materialization を行う。plan は provider metadata、provider DTO または materialized request を保持しない。

## 不変条件

- `rankedCandidates` の `candidateId` と全候補を横断した `stepId` は、それぞれ plan 内で重複しない。
- `rankedCandidates` は `semanticScore` の非増加順とする。同点の完全順序は版付き profile が所有し、plan は受け取った順序を保持して再整列しない。
- `selected` は `rankedCandidates` の要素だけを重複なく参照し、意味順位と `requiredPacks` を保持する。
- 選択した全候補の step 数は四つ以下とする。
- `hedged` は互いに独立して実行でき、両方とも `available` な候補だけに使用する。
- 空結果または情報源エラーを受けて、候補、step、上限または `effectiveLimit` を変更しない。
- plan と profile はリクエスト中に変更せず、別リクエストの入力や結果を保持しない。

## 確認

一候補、二候補、明確化、pack 無効、対象外との混在、十六候補上限、四 step 上限、選択 step と materialization へ渡す予算の対応、item 予算の各 `R/C` 組合せ、同点の決定性および計画確定後の追加・再配分禁止をモデルテストで確認する。`available` で空でない `requiredPacks`、複数の unsupported 理由、混在要求で保持した内部候補の非選択、および同点候補を profile の順序のまま保持することも確認する。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](22-legal-query-candidate.md)
- [SOT-MODEL-026: QueryProfileContribution](26-query-profile-contribution.md)
- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](../30-architecture/23-unified-query-selection-and-hedging.md)
- [SOT-ARCH-026: 統合照会の request materialization](../30-architecture/26-unified-query-request-materialization.md)
- [SOT-ARCH-019: 拡張パックの有効化境界](../30-architecture/19-extension-pack-activation-boundary.md)
- [SOT-IF-026: プロバイダールーティング設定](../40-interfaces/26-provider-routing-configuration.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](../50-engineering/24-unified-query-evaluation-gate.md)
