# SOT-IF-058: `law.version.compare` capability v1

- 状態: 有効

## 規定

`law.version.compare@1` は、一つの法令参照と比較前後の版指定を受け取り、同じ
provider と情報源に属する二版の条単位比較を、出典付き共通モデルで返す内部の
型付き capability 契約とする。

## 能力識別子

| 項目 | 値 |
|---|---|
| `ProviderCapability.id` | `law.version.compare` |
| `ProviderCapability.majorVersion` | `1` |
| `ProviderCapability.level` | `core` |
| `ProviderCapability.stability` | `stable` |

## 型付き入力

`LawVersionCompareRequestV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `resource` | `SourceResourceRef` | はい | 比較対象法令と選択済み provider |
| `before` | `LawVersionSelectorV1` | はい | 比較前版の指定 |
| `after` | `LawVersionSelectorV1` | はい | 比較後版の指定 |

`LawVersionSelectorV1` は次の項目だけを持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `revisionId` | string | 条件付き | 正確に選択する法令履歴 ID |
| `asOf` | date | 条件付き | その日以前で最新の版を選ぶ基準日 |

## 入力制約

- `resource` は `SOT-MODEL-016` に従い、`resource.key.resourceType` は
  `law` とする。
- `resource.key.resourceId` は 1 文字以上、UTF-8 で 256 byte 以下とし、
  先頭又は末尾の U+0020 と ASCII 制御文字を許可しない。
- `resource.key.versionId` は使用せず、設定されている場合は
  `invalid_argument` とする。
- 各 selector は `revisionId` 又は `asOf` のどちらか一方だけを持つ。
- `revisionId` は 1 文字以上、UTF-8 で 512 byte 以下とし、先頭又は末尾の
  U+0020 と ASCII 制御文字を許可しない。
- `asOf` は実在する `YYYY-MM-DD` とする。provider 固有の収録開始日は
  共通入力制約にしない。
- 同じ selector 又は同じ版へ解決し得る二つの selector も有効とする。
- 未知の provider、descriptor と source metadata の不一致、空値、上限超過、
  selector の欠落又は相互排他違反は、外部呼出し前に `invalid_argument` とする。

## 比較意味

- 利用者が指定した `before` と `after` を保持し、日付又は履歴 ID から
  時系列順へ並べ替えない。
- `revisionId` はその版を正確に選択し、`asOf` はその日以前で最新の版を
  選択する。
- 選択された二版は同じ `lawId`、provider 及び source を持たなければならない。
- 条の対象範囲、同一性、変更分類及び結果構造は `SOT-MODEL-033` に従う。

## 型付き出力

結果は `SourcedResource<LawVersionComparison>` とする。

- `data` は `SOT-MODEL-033` に従う。
- `ref` は比較後に確定した法令版を指し、`resourceType: law`、
  `resourceId: data.lawId`、`versionId: data.after.law.revisionId` とする。
- 最後の `provenance` は `transformation: derived` とし、確定した比較前後の
  `SourceResourceKey` をこの順で `inputKeys` に持つ。
- `provenance.methodId` は比較方法を定める provider mapping SOT とし、前後版の
  `Citation` と同じ公式 source へ到達できなければならない。

能力別ポートは
`Compare(context.Context, Request) (SourcedResource<LawVersionComparison>, error)`
とし、外部レスポンス型、provider 固有 DTO 又は比較途中の型を公開しない。

## 欠落

- 比較対象法令又は指定したどちらかの版が存在しない場合は `not_found` とする。
- 同じ版へ解決された場合は、全対象条を `unchangedCount` に数えた成功結果とし、
  変更一覧を空にする。
- 一方だけ取得できた結果を成功として返さない。

## 到達し得る失敗

共通入力の不正は `invalid_argument`、対象法令又は版の不存在は
`not_found` とする。provider 境界からは `unsupported_capability`、
`unsupported_query`、`configuration_required`、`source_auth_failed`、
`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、
`source_contract_changed`、`invalid_source_response`、
`source_response_too_large`、`source_processing_limit` 及び
`unsafe_source_content` が到達し得る。

## 採用境界

この共通契約だけから既存 provider の descriptor、binding、route 又は公開ツールへ
能力を追加しない。provider mapping と組込み採用は provider ごとの後続 SOT で
決定する。

## 確認

各 selector の相互排他、同じ selector、`resource.key.versionId` の拒否、
入力上限、同じ法令と source の検証、同版比較、モデルの件数、出典、重複同一性、
`ref`、derived provenance と二つの `inputKeys`、及び入力と結果の不変性を
契約テストで確認する。

## 関連

- [SOT-SCN-013: 法令の二つの版を比較する](../10-scenarios/13-compare-law-versions.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-MODEL-033: LawVersionComparison](../20-model/33-law-version-comparison.md)
- [SOT-ARCH-040: 法令版間比較の境界](../30-architecture/40-law-version-comparison-boundary.md)
- [SOT-IF-015: 情報源操作の共通契約](15-source-operation-contract.md)
- [SOT-IF-017: 情報源エラーの正規化](17-source-error-normalization.md)
