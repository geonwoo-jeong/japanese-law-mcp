# SOT-MODEL-032: LawRevision

- 状態: 有効

## 規定

`LawRevision` は、一つの法令履歴 ID に対応する改正履歴の共通メタ情報と情報源を表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `lawId` | string | はい | 情報源が使用する法令 ID |
| `revisionId` | string | はい | 法令履歴 ID |
| `title` | string | はい | 当該履歴における法令名 |
| `lawType` | string | いいえ | 法令種別 |
| `lawNumber` | string | いいえ | 法令番号 |
| `titleKana` | string | いいえ | 法令名読み |
| `abbreviation` | string | いいえ | 法令略称 |
| `category` | string | いいえ | 法令分野分類 |
| `promulgationDate` | date | いいえ | 法令の公布日 |
| `sourceUpdatedAt` | datetime string | いいえ | 情報源における当該履歴データの更新日時 |
| `amendmentPromulgationDate` | date | いいえ | 改正法令の公布日 |
| `effectiveDate` | date | いいえ | 当該履歴に対応する改正の施行期日 |
| `effectiveDateNote` | string | いいえ | 施行期日規定等の参考情報 |
| `scheduledEffectiveDate` | date | いいえ | 情報源が示す暫定的又は擬似的な施行予定日 |
| `amendmentLawId` | string | いいえ | 改正法令の法令 ID |
| `amendmentLawTitle` | string | いいえ | 改正法令名 |
| `amendmentLawTitleKana` | string | いいえ | 改正法令名読み |
| `amendmentLawNumber` | string | いいえ | 改正法令番号 |
| `revisionKind` | enum | いいえ | 共通化した改正区分 |
| `repealStatus` | enum | いいえ | 共通化した廃止等の状態 |
| `repealRecordedDate` | date | いいえ | 情報源が廃止等の日として記録した日 |
| `remainInForce` | boolean | いいえ | 廃止等の後も効力を有するか |
| `currentStatus` | enum | いいえ | 現時点との関係を共通化した履歴状態 |
| `documentUrl` | string | いいえ | 当該履歴を確認できる公式 HTTPS URL |
| `source` | `LegalSource` | はい | 改正履歴を取得した情報源 |

## 列挙値

`revisionKind` は `enactment`、`partial_amendment`、`affected_law` または `repeal` とする。

`repealStatus` は `none`、`repealed`、`expired`、`suspended` または `loss_of_effectiveness` とする。

`currentStatus` は `current`、`future`、`previous` または `repealed` とする。

## 制約

- `lawId`、`revisionId` および `title` は空文字にしない。
- 省略せず設定した日付は実在する `YYYY-MM-DD` とする。
- `sourceUpdatedAt` を設定する場合は RFC 3339 の日時文字列とする。
- `documentUrl` を設定する場合は認証情報を含まない HTTPS URL とする。
- `scheduledEffectiveDate` は確定した `effectiveDate` の代用にしない。
- `repealRecordedDate` は情報源の意味を保持し、実際の法的効力発生日と推測して名前を変えない。
- `source` は `SOT-MODEL-003` に従う。
- 情報源にない省略可能項目を、空文字、別項目又は推測値で補わない。
- JSON はこの表の camelCase 項目名を使用し、省略可能な値がない項目は省略する。明示された boolean の `false` は欠落と区別して保持する。

## 確認

必須項目、列挙値、日付、日時、URL、boolean の `false` と欠落の区別、入力値の複製、および全項目の JSON 表現を単体テストで確認する。

## 関連

- [SOT-MODEL-003: LegalSource](03-legal-source.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
- [SOT-IF-055: `law.revision.list` capability v1](../40-interfaces/55-law-revision-list-capability.md)
