# SOT-ENG-017: プロバイダー適合性 matrix

- 状態: 有効

## 規定

リリース対象の各 provider capability について、要求される契約テスト、fixture、資源予算および SOT の対応を、機械可読の conformance matrix で管理し、欠落または不一致がある変更を検証ゲートで失敗させる。

## matrix

matrix の canonical artifact は、repository root から見た次の二箇所に固定する。

```text
conformance/provider-capability.schema.json
conformance/providers/{providerId}.yaml
```

schema は JSON Schema Draft 2020-12 の一ファイルだけとする。provider ごとの matrix は UTF-8 の YAML 1.2 を一ファイルだけ持ち、ファイル名の `{providerId}` と全 row の `providerId` を一致させる。JSON、TOML、CSV、Go の静的データ、別の schema または別 path を canonical matrix として併用しない。

YAML の最上位は `schemaVersion: 1` と `rows` の二項目だけを持つ object とする。alias、anchor、merge key、custom tag、重複 key、未知の最上位 key および JSON 互換型以外の scalar を拒否する。各ファイルの `rows` は `capabilityId`、`majorVersion`、`operation`、`budgetKey` の順で byte 単位の昇順に並べ、同じ組を重複させない。

`go test ./internal/providerconformance` は schema と `conformance/providers/*.yaml` を同じ loader で読み、0 件の provider file、schema 違反、file 名不一致、並び順不一致および未知ファイルを失敗させる。`go test ./...` と `provider-onboarding-fit` はこの test と loader を再利用し、独自の matrix parser を持たない。

各 row は、一つの `(providerId, capabilityId, majorVersion, operation)` を表し、少なくとも次の列を持つ。

| 列 | 内容 |
|---|---|
| `providerId` | `SOT-IF-014` に従う provider 識別子 |
| `capabilityId` | `SOT-MODEL-013` に従う能力 ID |
| `majorVersion` | 能力の互換性境界 |
| `operation` | provider 内部の取得または解析 operation 名 |
| `interfaceSotIds` | 対応する interface / mapping SOT ID の配列 |
| `budgetSotId` | 資源予算を定義した SOT ID |
| `budgetKey` | `budgetSotId` 内の一つの budget row を一意に示す識別子 |
| `concurrencyGroup` | 同じ provider 内で共有する同時実行枠 |
| `artifactType` | `SOT-ENG-016` の artifact 種別 |
| `fixtureSet` | 正常系と異常系に使う fixture 群の識別子 |
| `requiredCases` | 必須 test case 名の配列 |
| `supportsContinuation` | continuation 契約の適用有無 |
| `supportsAuth` | provider 設定または認証の適用有無 |
| `publicErrorSet` | 到達し得る公開エラー code の集合 |
| `parserContractVersion` | parser contract の版または確認日 |
| `implementedBy` | 実装を所有する Go package または test target |
| `status` | `planned`、`implemented` または `retired` |

`requiredCases` には、少なくとも次のカテゴリから適用可能な case を列挙する。

- descriptor
- capability-binding
- provenance
- resource-ref-roundtrip
- empty-vs-not-found
- unsupported-query
- page-invariants
- continuation-roundtrip
- continuation-tamper
- continuation-expired
- error-normalization
- secret-non-exposure
- response-bytes-limit
- decompressed-bytes-limit
- entries-or-objects-limit
- depth-limit
- parse-timeout
- concurrency-limit
- cancellation
- import-isolation
- contract-changed
- incremental-onboarding-fit

適用しない case は省略せず、row ごとに `n/a` と理由を持つ補助列または同等の表現で明示する。

`page-invariants` は、一覧または検索 operation について、能力別 SOT と mapping SOT が定義する返却件数、配列長、総件数、次位置、末尾、空結果および前進性の不変条件を、正常 fixture と不正 fixture の両方で確認する case とする。`supportsContinuation: true` の row、および `law.search@1` または `law.content.search@1` の row は `page-invariants` を `n/a` にできない。

e-Gov 法令 API Version 2 の `law.search@1` row では、少なくとも `count` と `laws.length` の一致、`total_count` との整合、非 `null` の `next_offset == offset + count`、明示された `null` と欠落の区別、欠落時の安全な導出、および公開 `nextOffset` と内部 continuation position の同値を検証する。

e-Gov 法令 API Version 2 の `law.content.search@1` row では、少なくとも全 `sentences` の展開件数と `sentence_count` の一致、`total_count` との整合、非 `null` の `next_offset == offset + sentence_count`、残件がある場合の `null` または欠落の失敗、および公開 `nextOffset` と内部 continuation position の同値を検証する。

上記の各不正 fixture は、item の切捨て、再計数、独自 offset の合成または末尾への読み替えで成功させず、mapping SOT が定義するエラーとなることを確認する。test 名だけが `page-invariants` で内容を満たさない場合は gate failure とする。

`status` は、runtime の登録状態と次のように一致させる。

- `planned`: 能力別 SOT は採用済みだが、コンパイルされた binding がなく、route から到達できない。
- `implemented`: コンパイルされた registry に binding が存在する。
- `retired`: 過去の検証記録としてだけ残り、コンパイルされた binding と route の参照がない。

公開ツール、組込み既定 route または利用者設定で有効化できる route から到達し得る binding は、対応する row をちょうど一つ持ち、その `status` を `implemented` とする。`ProviderDescriptor` の capability 宣言、registry binding、route の到達性および matrix status の四者を同じ検証で照合する。

## 検証ゲート

ローカルと CI の検証ゲートは、matrix を読み取り、各 row について次を確認する。

- `status=implemented` の row に必須列の欠落がない
- コンパイルされた各 binding に `status=implemented` の row がちょうど一つあり、`planned` または `retired` の row が binding や route から到達できない
- 参照する SOT ID が存在し、`status=implemented` の実装範囲と矛盾しない
- `requiredCases` の各 case に対応するテストが存在し、成功する
- `supportsContinuation: true` または検索 capability の row が `page-invariants` を持ち、対応する mapping SOT の全ページ不変条件を正常・不正 fixture で検証する
- `budgetSotId + budgetKey` が一つの budget row を参照し、`artifactType`、`concurrencyGroup` と各数値が `SOT-ENG-016` の必須予算を満たす
- 同じ `providerId + concurrencyGroup` の row が同じ `concurrency` を参照し、異なる operation 間の共有上限 test を持つ
- `publicErrorSet` が公開エラー契約と矛盾しない
- `implementedBy` が存在し、他 provider package への依存禁止に違反しない

次のいずれかに該当する変更は release gate failure とする。

- 実装済み capability に row がない
- 実装済みまたは route から到達可能な capability の row が `planned` または `retired` である
- 一つの binding に複数の active row がある、または `ProviderDescriptor`、registry、route および matrix の宣言が一致しない
- row はあるが必須列、必須 case または対応 SOT ID が欠ける
- test 名だけが存在し、matrix が要求する case を実際には検証していない
- `status=implemented` の row が `planned` 相当の fixture または budget を参照する
- 実装を追加または変更したのに matrix を更新していない

missing case は warning にせず failure とし、80% coverage または個別 package coverage が十分でも通過させない。

## 関連

- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-013: プロバイダー契約の検証](13-provider-contract-verification.md)
- [SOT-ENG-016: プロバイダー資源予算](16-provider-resource-budgets.md)
- [SOT-IF-014: ProviderDescriptor](../40-interfaces/14-provider-descriptor.md)
- [SOT-MODEL-013: ProviderCapability](../20-model/13-provider-capability.md)
