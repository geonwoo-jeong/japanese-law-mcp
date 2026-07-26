# SOT-IF-036: e-Gov 更新法令一覧マッピング

- 状態: 有効

## 規定

e-Gov 法令 API Version 1 の `DataRoot/ApplData/LawNameListInfo` を、`law.update.list@1` の `LawUpdate` 完全一覧へ変換する。

## 結果と対象日

HTTP `200` かつ `Result/Code` が `0` の場合だけ通常の成功とする。公式 endpoint が収録範囲内の該当なし日について返す HTTP `404` は、XML が次のすべてを満たす場合だけ成功した空一覧とする。

- `Result/Code` が `1`
- `ApplData/Date` が要求日と一致
- `LawNameListInfo` がない

他の status と code の組合せ、要求日と `ApplData/Date` の不一致、または `404` に項目がある応答は `invalid_source_response` とする。`Date` と各日付は `yyyyMMdd` から共通 `date` へ変換し、存在しない日を受理しない。

## 項目対応

| e-Gov XML | `LawUpdate` |
|---|---|
| 要求日および `ApplData/Date` | `updatedOn` |
| `LawId` | `lawId` |
| `LawName` | `title` |
| `LawTypeName` | `lawType` |
| `LawNo` | `lawNumber` |
| `LawNameKana` | `titleKana` |
| `OldLawName` | `previousTitle` |
| `PromulgationDate` | `promulgationDate` |
| `AmendName` | `amendmentTitle` |
| `AmendNo` | `amendmentLawNumber` |
| `AmendPromulgationDate` | `amendmentPromulgationDate` |
| `EnforcementDate` | `effectiveDate` |
| `EnforcementComment` | `effectiveDateNote` |
| `LawUrl` | `documentUrl` |
| `EnforcementFlg` | `enforcementPending` |
| `AuthFlg` | `authorityReviewPending` |

空の省略可能 XML 項目は存在しない省略可能値とし、別の値で補わない。二つの flag は空を不在、`0` を `false`、`1` を `true` とし、他の値を `invalid_source_response` とする。`LawId` と `LawName` は非空を必須とし、`LawUrl` がある場合は認証情報を含まない HTTPS URL とする。

`LawUpdate.source` は `SOT-IF-035` の情報源とする。

## 参照、出典および件数

各 item は次の共通情報を持つ。

- `ref.providerId` と `ref.key.sourceId`: `e-gov-law-api-v1`
- `ref.key.resourceType`: `law-update-list`
- `ref.key.resourceId`: 要求日の `YYYY-MM-DD`
- `ref.key.versionId`: なし
- `provenance.url`: 呼び出した固定 endpoint
- `provenance.mediaType`: `text/xml`
- `provenance.retrievedAt`: この呼出しで取得を完了した時刻
- `provenance.location`: 対応する `DataRoot/ApplData/LawNameListInfo`
- `provenance.transformation`: `normalized`
- `provenance.methodId`: `SOT-IF-036`

同じ対象日の item は同じ一覧資源 ref を持てる。item の同一性をこの ref だけで判断しない。

`LawUpdateListPageV1.date` は要求日とし、`returnedCount`、`totalCount` および item 数を一致させ、`totalRelation` は `exact`、継続トークンはなしとする。

## 確認

公式の五件応答、該当なしの `404`、対象日不一致、必須値欠落、不正な日付・URL・flag・result code、未知または重複した XML 構造、全資源予算、取消および秘密非露出を fixture と合成応答で確認する。

## 関連

- [SOT-IF-034: `law.update.list` capability v1](34-law-update-list-capability.md)
- [SOT-IF-035: e-Gov 法令 API Version 1 更新一覧](35-source-egov-law-api-v1.md)
- [SOT-MODEL-019: LawUpdate](../20-model/19-law-update.md)
