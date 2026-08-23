# SOT-ENG-017: プロバイダー適合性 matrix

- 状態: 有効

## 規定

リリース対象の各 provider capability について、対応する SOT、実装、fixture、資源予算および契約テストを、機械可読の conformance matrix で結び付ける。

matrix は実装の内部構造を完全に証明するためのものではない。共通 interface から観測できる振る舞いと、provider ごとの実装範囲および検証対象を追跡するために使用する。

## matrix

canonical artifact は、repository root から見た次の二箇所に固定する。

```text
conformance/provider-capability.schema.json
conformance/providers/{providerId}.yaml
```

schema は JSON Schema Draft 2020-12 とする。provider ごとの matrix は UTF-8 の YAML 1.2 を一ファイルだけ持ち、ファイル名の `{providerId}` と各 row の `providerId` を一致させる。

YAML の最上位は `schemaVersion: 1` と `rows` を持つ object とする。alias、anchor、merge key、custom tag、重複 key および未知の最上位 key を拒否する。各ファイルの `rows` は `capabilityId`、`majorVersion`、`operation`、`budgetKey` の順で昇順に並べ、同じ `(providerId, capabilityId, majorVersion, operation)` を重複させない。

`internal/providerconformance` の共通 loader が schema と `conformance/providers/*.yaml` を読み、通常の test と `provider-onboarding-fit` はこの loader を再利用する。

各 row は少なくとも次の列を持つ。

| 列 | 内容 |
|---|---|
| `providerId` | `SOT-IF-014` に従う provider 識別子 |
| `capabilityId` | `SOT-MODEL-013` に従う能力 ID |
| `majorVersion` | 能力の互換性境界 |
| `operation` | provider 内部の取得または解析 operation 名 |
| `interfaceSotIds` | 対応する provider、capability および mapping SOT ID |
| `budgetSotId` | 資源予算を定義した SOT ID |
| `budgetKey` | `budgetSotId` 内の budget row |
| `concurrencyGroup` | 同じ provider 内で共有する同時実行枠 |
| `artifactType` | `SOT-ENG-016` の artifact 種別 |
| `fixtureSet` | 正常系と異常系に使う provider 固有 fixture 群 |
| `requiredCases` | 必須の共通契約 test case |
| `notApplicableCases` | 適用しない標準 case と理由 |
| `supportsContinuation` | continuation 契約の適用有無 |
| `supportsAuth` | provider 設定または認証の適用有無 |
| `publicErrorSet` | 到達し得る公開エラー code |
| `parserContractVersion` | parser contract の版または確認日 |
| `implementedBy` | 実装を所有する Go package |
| `conformanceTarget` | この row の契約テスト target |
| `status` | `planned`、`implemented` または `retired` |

`interfaceSotIds` と `budgetSotId` は、`planned` と `implemented` では有効な SOT を参照する。`retired` では、実装当時の契約を追跡するため廃止済み SOT を参照してよい。

`implementedBy` は provider ごとに分離した package とする。一つの provider package から他の provider package を import しない。共通 model、能力別 port、共通 HTTP・継続取得・予算 helper は provider-neutral package に置き、外部 API の DTO、request builder、parser および mapping は各 provider package が所有する。

## fixture と共通 case

fixture は `implementedBy` の package 配下に provider ごと、`fixtureSet` ごとに分離して置く。fixture は外部 API の request と未加工 response、または同等の外部 artifact を表し、共通 model の完成値を返す fake port として使わない。

同じ `(capabilityId, majorVersion)` の provider は、同じ能力別 port と共通 conformance suite を使用する。provider 固有の応答形式や境界値は、共通 case の fixture variation として追加する。provider 固有の test logic が必要な場合は provider package の通常の test として追加できるが、共通 interface の意味を provider ごとに変更しない。

標準 case は、適用可能な範囲で次を含む。

- `descriptor`
- `capability-binding`
- `outbound-request`
- `authentication`
- `provenance`
- `resource-ref-roundtrip`
- `empty-vs-not-found`
- `unsupported-query`
- `page-invariants`
- `continuation-roundtrip`
- `continuation-tamper`
- `continuation-expired`
- `error-normalization`
- `secret-non-exposure`
- `response-bytes-limit`
- `decompressed-bytes-limit`
- `entries-or-objects-limit`
- `depth-limit`
- `parse-timeout`
- `concurrency-limit`
- `cancellation`
- `contract-changed`

適用しない case は `notApplicableCases` に理由を記録する。case の適用可否は、能力別 SOT、共通 conformance suite、`supportsContinuation` および `supportsAuth` から決定し、provider 固有の理由だけで共通 capability の必須 case を適用外にしない。`outbound-request`、`secret-non-exposure`、`concurrency-limit` および `cancellation` は、実装済みの外部 provider operation では省略しない。

`supportsContinuation: true` の row は continuation の roundtrip、改変、期限切れおよび `page-invariants` を検証する。`supportsAuth: true` の row は credential の欠落、正常な付与、外部の認証拒否および秘密非露出を検証する。認証が任意の場合の欠落時動作は provider SOT に従う。

一覧または検索 operation の `page-invariants` は、能力別 SOT と mapping SOT が定義する返却件数、配列長、総件数、次位置、末尾、空結果および前進性を、正常 fixture と不正 fixture の両方で確認する。

e-Gov 法令 API Version 2 の `law.search@1` は、少なくとも `count` と `laws.length`、`total_count`、`next_offset`、公開 `nextOffset` および内部 continuation position の整合を検証する。`law.content.search@1` は、全 `sentences` の展開件数と `sentence_count`、`total_count`、`next_offset` および continuation position の整合を検証する。`law.revision.list@1` は `revisions.length` と `returnedCount` および exact な `totalCount` の一致を検証する。不正応答を切捨て、再計数または独自 offset の合成で成功へ変換しない。

## status

- `planned`: SOT と実装予定を採用済みだが、runtime の binding または route から到達できない。後続作業を小さく分けるため、到達不能な provider package、fixture および test を先に追加してよい。
- `implemented`: 製品へ組み込まれた binding が存在し、対応する conformance target が成功する。実際の runtime registry へ含めるかは起動時の `enabled` 設定と route に従う。
- `retired`: 過去の記録として row を残すが、runtime の binding と route から到達できない。

一つの `ProviderDescriptor` が複数 capability を宣言する場合は、各 capability の実装と test を順番に準備してよい。planned の準備段階では descriptor の定義を test 用に置けるが、production へ公開しない。製品へ組み込むときは、descriptor が宣言する capability、`implemented` row および compiled binding inventory が互いに一致する単位で有効化する。

runtime composition は、`enabled: true` の provider だけについて設定と credential を解決し、その provider の implemented binding を registry へ登録する。disabled provider の factory を呼ばず、credential の欠落を起動失敗にしない。runtime route は、enabled かつ implemented である binding だけを参照する。

新しい provider は明示設定がない限り disabled とし、既存の組込み route を変更しない。e-Gov 法令 API Version 2 は `SOT-IF-026` の既存の組込み既定値で enabled になる provider であり、descriptor が宣言する五 capability と五つの compiled binding および既定 route を一致させる。

planned row の採用を取り消す場合は、その row と未到達の成果物を同じ変更で削除し、関係する SOT または Wiki に理由を記録する。implemented row を廃止する場合は `retired` へ変更し、binding、route および不要になった provider 固有成果物を除去する。

外部 provider の仕様、mapping または parser contract が変わる場合は、`SOT-ENG-007` と `SOT-ENG-008` の lifecycle に従い、provider SOT、matrix、fixture、parser および影響を受ける test を同じ契約変更として更新する。共通 capability の意味または major version を変更する場合は、provider 固有変更と分離する。

## 検証ゲート

CI の権威ある品質ゲートは次を確認する。ローカルでは対象を一つに限定した
回帰テストを任意に実行できるが、この一覧の全検査を繰り返さない。

- schema、file 名、row の順序、重複および必須列
- `planned` と `retired` の operation が runtime の binding または route から到達できないこと
- `implemented` row の package、fixture および conformance target が存在し、test が成功すること
- `ProviderDescriptor`、compiled binding inventory および `implemented` row の capability 集合が一致すること
- runtime registry と route が、enabled かつ implemented である binding の有効な部分集合だけを参照すること
- `budgetSotId + budgetKey` が一つの budget row を参照し、`SOT-ENG-016` の上限を満たすこと
- 同じ `providerId + concurrencyGroup` が同じ同時実行上限を共有すること
- `publicErrorSet` と公開エラー契約が矛盾しないこと
- provider package が他の provider package を import しないこと

これらは schema 検証、Go の型検査、通常の unit・integration test および必要な fixture test で確認する。関数 object、SSA、全 call graph または Go test event を完全一致させることは、この規定の必須条件としない。

## 関連

- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-007: SOT 識別子](07-sot-identifiers.md)
- [SOT-ENG-008: SOT lifecycle](08-sot-lifecycle.md)
- [SOT-ENG-013: プロバイダー契約の検証](13-provider-contract-verification.md)
- [SOT-ENG-016: プロバイダー資源予算](16-provider-resource-budgets.md)
- [SOT-ENG-018: プロバイダー追加 fitness gate](18-provider-onboarding-fitness-gate.md)
- [SOT-IF-014: ProviderDescriptor](../40-interfaces/14-provider-descriptor.md)
- [SOT-MODEL-013: ProviderCapability](../20-model/13-provider-capability.md)
