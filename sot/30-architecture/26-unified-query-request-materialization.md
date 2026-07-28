# SOT-ARCH-026: 統合照会の request materialization

- 状態: 有効

## 規定

統合照会の request materializer は、選択済みの能力 binding metadata、検証済み logical input および確定済み step 予算から、対応する既存 capability request を新しく作る。情報源の選択と request の組立てを分離し、provider DTO、外部 query parameter、fallback、外部呼出しまたは結果変換を扱わない。

## binding の選択

選択済み binding metadata は、`providerId`、`sourceId`、`capabilityId` および `capabilityMajorVersion` だけを公開する `application/legalquery` 所有の最小 interface とする。実装は起動時に検証済みの registry と route から作った不変な snapshot とし、`application/legalquery` は `ProviderDescriptor` の具象型を受け取らない。

能力 binding の選択方法は logical input の対象形によって次のように分ける。

- 検索、更新一覧、および法令 ID から初めて `SourceResourceRef` を作る read は、能力の実効 primary route を使用する。
- 入力に `SourceResourceRef` がある read は `explicit` とし、`ref.providerId` と完全一致する有効な binding を使用する。primary route は参照せず、別の provider へ fallback しない。

registry adapter は選択した provider が要求された `capabilityId` と major version を宣言し、対応する型付き port を持つ場合だけ binding metadata を返す。未知の provider、未実装能力、無効な binding、未初期化 route または typed nil port から metadata を作らない。

materializer は受け取った binding metadata の capability ID と major version が、呼び出した能力別 materializer と完全一致することをもう一度確認する。入力 `ref` がある場合は、`ref.providerId` と binding の `providerId`、および `ref.key.sourceId` と binding の `sourceId` がそれぞれ一致することを確認する。resource type と version の制約は既存 capability request の constructor でも再検証する。入力値の不一致は外部呼出し前の `invalid_argument` とし、route または binding の内部不整合と区別する。

## 予算

能力別 materializer は `LegalQueryStepBudget` を受け取り、read では `reservedItems=1` かつ `effectiveLimit` なし、collection では `reservedItems=0` かつ `effectiveLimit` ありであることを確認する。

`law.search@1`、`law.content.search@1` および `judicial-decision.search@1` は、確定済み `effectiveLimit` を request の `limit` に設定し、`continuationToken` を空にする。logical input、利用者入力または capability の既定値から別の上限を補わない。

`law.update.list@1` は一日分の完全一覧を返す既存契約を保つため、request に上限または continuation を追加しない。collection 予算の存在だけを検証し、executor の能力結果 mapping で `SOT-MODEL-023` の公開 preview 規則を適用する。最終 result assembler はこの attempt を再切出ししない。

## 能力別の変換

| logical input | binding の選択 | 既存 capability request への対応 |
|---|---|---|
| `LawSearchIntentV1` | primary | `query` と任意の `asOf` を保持し、`limit=effectiveLimit`、continuation なし |
| `LawContentSearchIntentV1` | primary | `allTerms`、`anyTerms`、`excludeTerms` と任意の `asOf` を保持し、`limit=effectiveLimit`、continuation なし |
| `LawReadIntentV1` の ID 形 | primary | binding から法令 `ref` を作り、任意の `revisionId` を `versionId`、任意の `asOf` を request の `asOf` とする |
| `LawReadIntentV1` の ref 形 | explicit | 入力 `ref` を変更せず保持し、request の `asOf` は持たない |
| `LawArticleReadIntentV1` の ID 形 | primary | binding から版なしの法令 `ref` を作り、`location` と任意の `asOf` を保持する |
| `LawArticleReadIntentV1` の ref 形 | explicit | 入力 `ref`、`location` と任意の `asOf` を保持する |
| `LawUpdateListIntentV1` | primary | `date` を保持し、request に limit または continuation を追加しない |
| `JudicialDecisionSearchIntentV1` | primary | `query` を保持し、`limit=effectiveLimit`、continuation なし |
| `JudicialDecisionReadIntentV1` | explicit | 入力 `ref` を変更せず保持する |

法令 ID から作る `SourceResourceRef` は、次の対応だけを使用する。

| `SourceResourceRef` | 値 |
|---|---|
| `providerId` | binding の `providerId` |
| `key.sourceId` | binding の `sourceId` |
| `key.resourceType` | `law` |
| `key.resourceId` | logical input の `lawId` |
| `key.versionId` | `LawReadIntentV1.revisionId` がある場合だけ同じ値 |

`asOf` を `versionId` に変換せず、法令 ID、revision ID、provider ID、source ID または入力 `ref` を trim、正規化、推測若しくは置換しない。

## 型付き境界

request materializer の interface は七つの logical input variant に対応する能力別 method を持ち、各 method は `any`、open union または provider DTO ではなく、対応する既存 capability request の具象型を直接返す。共通 dispatch のためだけの新しい request schema または tagged wrapper を公開しない。

法令コアの五 method と `judicial-cases` の二 method は別の materializer interface として構成する。法令コア materializer は常に構成し、裁判例 materializer は `judicial-cases` pack が有効な場合だけ binding、route および profile と同時に構成する。実装内部の安全な補助関数は共有できるが、無効な pack の materializer を必須依存にしない。

materializer は候補 score、候補順位、pack 状態、provider の速度若しくは応答状態、外部結果または公開 MCP model を受け取らない。request constructor が返した検証エラーを成功、空 request または別能力へ読み替えない。

## 確認

ネットワークを使わない単体テストで、七 variant の lossless な変換、検索の固定上限と continuation なし、更新一覧の完全取得契約、法令 ID と revision ID からの ref 組立て、および入力 ref の保持を確認する。

primary と explicit の binding 選択、primary と異なる provider の有効な ref、provider・source・capability・major version の不一致、未知の provider、欠落 port、typed nil、未初期化 route、read と collection の予算形不一致、および zero-value materializer を確認する。すべての不一致について、provider port が呼ばれる前に失敗することを確認する。

## 関連

- [SOT-ARCH-012: プロバイダーの登録](12-provider-registry.md)
- [SOT-ARCH-013: 情報源の選択と組合せ](13-source-composition.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](22-unified-query-planning-pipeline.md)
- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-IF-022: law.search capability v1](../40-interfaces/22-law-search-capability.md)
- [SOT-IF-023: law.content.search capability v1](../40-interfaces/23-law-content-search-capability.md)
- [SOT-IF-024: law.document.read capability v1](../40-interfaces/24-law-document-read-capability.md)
- [SOT-IF-025: law.article.read capability v1](../40-interfaces/25-law-article-read-capability.md)
- [SOT-IF-026: プロバイダールーティング設定](../40-interfaces/26-provider-routing-configuration.md)
- [SOT-IF-034: law.update.list capability v1](../40-interfaces/34-law-update-list-capability.md)
- [SOT-IF-041: judicial-decision.search capability v1](../40-interfaces/41-judicial-decision-search-capability.md)
- [SOT-IF-042: judicial-decision.read capability v1](../40-interfaces/42-judicial-decision-read-capability.md)
- [SOT-ENG-025: 統合照会のパッケージ構成](../50-engineering/25-unified-query-package-layout.md)
