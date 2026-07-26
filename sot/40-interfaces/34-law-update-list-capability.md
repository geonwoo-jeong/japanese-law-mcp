# SOT-IF-034: `law.update.list` capability v1

- 状態: 有効

## 規定

`law.update.list@1` は、一つの暦日を受け取り、その日に更新一覧へ掲載された法令を `LawUpdate` の完全な一覧として返す、内部の共通 capability 契約とする。

## 能力識別子

| 項目 | 値 |
|---|---|
| `ProviderCapability.id` | `law.update.list` |
| `ProviderCapability.majorVersion` | `1` |
| `ProviderCapability.level` | `core` |
| `ProviderCapability.stability` | `stable` |

## 型付き入力

`LawUpdateListRequestV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `date` | date | はい | 更新一覧の対象日 |

`date` は実在する `YYYY-MM-DD` 形式の暦日とし、違反は `invalid_argument` とする。収録開始日、現在日その他のプロバイダー固有の提供範囲は共通入力制約にしない。共通入力として有効でも選択したプロバイダーの提供範囲外である日付は、アダプターが外部呼出し前に `unsupported_query` とする。

## 型付き出力

`LawUpdateListPageV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `items` | `SourcedResource<LawUpdate>[]` | はい | 対象日の法令更新情報 |
| `page` | `SourcePage` | はい | 返した件数と正確な総件数 |

各 item は次を満たす。

- `data.updatedOn` は入力の `date` と一致する。
- `ref.key.resourceType` は `law-update-list` とする。
- `ref.key.resourceId` は `data.updatedOn` の `YYYY-MM-DD` 文字列とする。
- `ref.key.versionId` は使用しない。
- `ref.key.sourceId` は `data.source.id` と一致する。
- `ref` は対象日の一覧資源を示す。同じ日の複数 item を `ref` だけで重複排除しない。

## 件数、継続取得および空結果

この v1 契約は一日分の完全な一覧を一回で返し、継続取得を使用しない。

- `page.nextToken` は省略する。
- `page.totalRelation` は `exact` とする。
- `page.totalCount`、`page.returnedCount` および `items` の件数は一致させる。
- 該当する法令がない場合は成功した空の結果とし、`items` は空の配列、二つの件数は `0` とする。`not_found` へ変換しない。
- 情報源に存在しない `LawUpdate` の省略可能項目を補完しない。

## ポート

能力別ポートは `List(context.Context, Request) (Page, error)` とし、外部レスポンス型またはプロバイダー固有 DTO を公開しない。

## 到達し得る失敗

共通入力の形式不正は `invalid_argument` とする。プロバイダー境界からは `unsupported_capability`、`unsupported_query`、`configuration_required`、`source_auth_failed`、`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` および `unsafe_source_content` が到達し得る。

この capability を MCP ツールへ公開する場合は、先に公開ツールと公開エラーの SOT を採用する。

## 確認

少なくとも次を契約テストで確認する。

- 有効な暦日を受理し、ゼロ値または形式不正日付を拒否すること
- プロバイダーの提供範囲外だが共通形式として有効な日付を `unsupported_query` とすること
- item の参照、対象日、情報源および `LawUpdate` の対応を保持すること
- 継続トークンを返さず、正確な総件数と返却件数が一致すること
- 空の一覧を成功として返すこと
- 入力、items および省略可能値を外部から変更できないこと

## 関連

- [SOT-MODEL-019: LawUpdate](../20-model/19-law-update.md)
- [SOT-MODEL-014: SourcePage](../20-model/14-source-page.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-ARCH-017: 採用可能な能力群](../30-architecture/17-approved-capability-families.md)
- [SOT-IF-015: 情報源操作の共通契約](15-source-operation-contract.md)
- [SOT-IF-017: 情報源エラーの正規化](17-source-error-normalization.md)
