# SOT-MODEL-023: LegalQueryPlan

- 状態: 有効

## 規定

`LegalQueryPlan` は、照会文に対する意味候補の順位、pack の実行可否、選択結果、能力別 request の materialization および一リクエストの固定予算を表す内部モデルとする。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `language` | string | はい | 固定値 `ja` |
| `profileVersion` | string | はい | 重み、閾値、根拠規則および辞書版を特定する値 |
| `decision` | string | はい | `single`、`hedged`、`needs_clarification`、`capability_unavailable` または `unsupported` |
| `rankedCandidates` | `LegalQueryCandidate[]` | はい | 意味 score 順の内部候補。十六件以下 |
| `selected` | `LegalQueryPlanSelection[]` | はい | 実行または非実行の判断対象とした候補 |
| `reasonCodes` | `string[]` | はい | 非実行または選択方法を表す決定的な理由 |
| `budget` | `LegalQueryBudget` | はい | リクエスト全体の上限と step ごとの確定値 |

## 選択と実行可能性

`LegalQueryPlanSelection` は `candidateId`、`availability` および任意の `requiredPack` を持つ。`availability` は次のいずれかとする。

- `available`: 必要な pack が有効
- `pack_disabled`: 採用済みだが必要な拡張パックが無効

候補を意味 score で順位付けした後に pack の実行可否を付与する。pack の状態、ネットワーク状態、応答速度または過去の件数を `semanticScore` へ加えず、利用できない上位候補を理由に意味の弱い別 resource を繰り上げない。

`decision=single` は一つ、`decision=hedged` は二つの `available` な候補だけを実行対象とする。上位二候補のどちらかが `pack_disabled` で安全な一意解釈を選べない場合は、利用できる候補だけを実行せず `capability_unavailable` とする。残りの decision は外部情報源を呼び出さない。

`needs_clarification` の `selected` は、利用者へ示す候補を二件以下の `available` な selection として持つことができる。`capability_unavailable` は一件以上二件以下の `pack_disabled` な selection を持つ。`unsupported` は採用済みの task/resource 候補として表さず `selected` を空にし、対象外、言語境界違反または混在要求を `reasonCodes` で表す。

一つの照会文に、法的助言、未採用 task/resource または翻訳と、実行可能な取得意図が混在する場合は、実行可能な部分だけを抜き出さず `unsupported` とする。一つの複数 step 候補に無効な pack の step が含まれる場合も、他 step を部分実行せず `capability_unavailable` とする。

公開する法令コア route と、有効な pack が必要とする route・binding・materializer は起動時にすべて検証する。欠落または不整合があれば transport を開始せず、正常起動後の候補選択で route 不備を availability として扱わない。

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

検索 capability request は `effectiveLimit` を使用し、continuation を省略する。`law.update.list@1` は完全一覧を取得する既存契約を変更せず、result assembler が計画順の先頭から `effectiveLimit` 件だけを公開 preview へ投影し、残件を `hasMore` と情報源の正確な `totalCount` で示す。

空結果、失敗または実際の返却件数が上限未満でも、未使用分を後続 step へ再配分しない。これにより並列の完了順と結果件数で公開結果が変わらない。

retry、adapter 内の通信および provider ごとの同時実行制御は、各能力と情報源の既存予算に従う。統合照会はそれらの上限を緩和しない。

## request materialization

計画確定後、executor は計画順に、各 step の `(capabilityId, capabilityMajorVersion)` route と能力別 request materializer を解決する。

materializer は logical input、検証済みの任意の `ref`、選択した binding および `effectiveLimit` から既存 capability request を新しく作る。検索 request に continuation を設定せず、read request の `SourceResourceRef` は入力の `ref` を検証して保持するか、法令 ID と選択 binding から決定的に組み立てる。

materialization で型、resource、provider または route の不一致を検出した場合は外部呼出し前に失敗する。planner、profile または candidate が provider DTO を扱わない。

## 不変条件

- `rankedCandidates` は意味 score と固定 tie-break の決定的な順序を持つ。
- `selected` は `rankedCandidates` の要素だけを参照し、意味順位を保持する。
- 選択した全候補の step 数は四つ以下とする。
- `hedged` は互いに独立して実行でき、両方とも `available` な候補だけに使用する。
- 空結果または情報源エラーを受けて、候補、step、上限または `effectiveLimit` を変更しない。
- plan と profile はリクエスト中に変更せず、別リクエストの入力や結果を保持しない。

## 確認

一候補、二候補、明確化、pack 無効、対象外との混在、十六候補上限、四 step 上限、ID と `ref` の materialization、起動時 route 検証、item 予算の各 `R/C` 組合せ、同点の決定性および計画確定後の追加・再配分禁止をモデルテストで確認する。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](22-legal-query-candidate.md)
- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](../30-architecture/23-unified-query-selection-and-hedging.md)
- [SOT-ARCH-019: 拡張パックの有効化境界](../30-architecture/19-extension-pack-activation-boundary.md)
- [SOT-IF-026: プロバイダールーティング設定](../40-interfaces/26-provider-routing-configuration.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](../50-engineering/24-unified-query-evaluation-gate.md)
