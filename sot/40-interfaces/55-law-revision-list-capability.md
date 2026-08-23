# SOT-IF-055: `law.revision.list` capability v1

- 状態: 有効

## 規定

`law.revision.list@1` は、一つの法令 ID または法令番号を受け取り、その法令の完全な改正履歴を共通モデルで返す、内部の型付き capability 契約とする。

## 能力識別子

| 項目 | 値 |
|---|---|
| `ProviderCapability.id` | `law.revision.list` |
| `ProviderCapability.majorVersion` | `1` |
| `ProviderCapability.level` | `core` |
| `ProviderCapability.stability` | `stable` |

## 型付き入力

`LawRevisionListRequestV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `lawIdOrNumber` | string | はい | 情報源が受け付ける法令 ID または法令番号 |

`lawIdOrNumber` は端の U+0020 を除いた後に、一文字以上、UTF-8 で 256 byte 以下、有効な UTF-8、ASCII 制御文字なしとする。共通入力では法令 ID と法令番号の種別判定、補完又は推測を行わない。情報源固有の検索フィルター、改正種別コードおよび状態コードは共通入力へ公開しない。

## 型付き出力

`LawRevisionListPageV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `lawId` | string | はい | 改正履歴が属する法令 ID |
| `items` | `SourcedResource<LawRevision>[]` | はい | 情報源の新しい順を保持した完全な改正履歴 |
| `page` | `SourcePage` | はい | 正確な総件数 |

各 item は次を満たす。

- `data.lawId` は出力の `lawId` と一致する。
- `ref.key.resourceType` は `law` とする。
- `ref.key.resourceId` は `data.lawId` と一致する。
- `ref.key.versionId` は `data.revisionId` と一致する。
- `ref.key.sourceId` は `data.source.id` と一致する。
- 同じ `revisionId` を重複して返さない。

## 順序、件数および空結果

- `items` は情報源が返す新しい履歴からの順序を変えない。
- 一回の呼出しで完全な履歴一覧を返し、継続取得を使用しない。
- `page.returnedCount`、`page.totalCount` および `items` の件数は一致し、`page.totalRelation` は `exact` とする。
- 対象法令が存在し、履歴がない場合は成功した空配列とする。
- 対象法令が存在しない場合は `not_found` とする。

## ポート

能力別ポートは `List(context.Context, Request) (Page, error)` とし、外部レスポンス型またはプロバイダー固有 DTO を公開しない。

## 到達し得る失敗

共通入力の形式不正は `invalid_argument` とする。プロバイダー境界からは `not_found`、`unsupported_capability`、`unsupported_query`、`configuration_required`、`source_auth_failed`、`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` および `unsafe_source_content` が到達し得る。

## 確認

少なくとも次を契約テストで確認する。

- 有効な `lawIdOrNumber` だけを受理し、空値、不正 UTF-8、256 byte 超過および制御文字を拒否すること
- item の `lawId`、`revisionId`、`ref` および `source` が一致すること
- 一つの法令の重複しない履歴だけを返し、順序を保持すること
- 正確な件数、成功した空結果および `not_found` を区別すること
- 入力、items および省略可能値を外部から変更できないこと

## 関連

- [SOT-MODEL-014: SourcePage](../20-model/14-source-page.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-MODEL-032: LawRevision](../20-model/32-law-revision.md)
- [SOT-ARCH-017: 採用可能な能力群](../30-architecture/17-approved-capability-families.md)
- [SOT-IF-015: 情報源操作の共通契約](15-source-operation-contract.md)
- [SOT-IF-017: 情報源エラーの正規化](17-source-error-normalization.md)
