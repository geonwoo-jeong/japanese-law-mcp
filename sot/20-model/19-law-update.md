# SOT-MODEL-019: LawUpdate

- 状態: 有効

## 規定

`LawUpdate` は、一つの対象日に更新一覧へ掲載された法令の識別情報、改正・施行情報および情報源を表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `updatedOn` | date | はい | 更新一覧の対象日 |
| `lawId` | string | はい | 情報源が使用する法令識別子 |
| `title` | string | はい | 法令名 |
| `lawType` | string | いいえ | 法令種別 |
| `lawNumber` | string | いいえ | 法令番号 |
| `titleKana` | string | いいえ | 法令名の読み |
| `previousTitle` | string | いいえ | 旧法令名 |
| `promulgationDate` | date | いいえ | 法令の公布日 |
| `amendmentTitle` | string | いいえ | 改正法令名 |
| `amendmentLawNumber` | string | いいえ | 改正法令番号 |
| `amendmentPromulgationDate` | date | いいえ | 改正法令の公布日 |
| `effectiveDate` | date | いいえ | 施行日 |
| `effectiveDateNote` | string | いいえ | 施行日に関する注記 |
| `documentUrl` | string | いいえ | 個別の法令を確認できる公式 URL |
| `enforcementPending` | boolean | いいえ | 未施行であることを情報源が示したか |
| `authorityReviewPending` | boolean | いいえ | 所管課確認中であることを情報源が示したか |
| `source` | `LegalSource` | はい | 更新情報を取得した法令情報源 |

## 制約

- `updatedOn` と、省略せず設定した日付は、実在する `YYYY-MM-DD` 形式の暦日とする。
- `lawId` と `title` は空文字にしない。
- `documentUrl` を設定する場合は、認証情報を含まない HTTPS URL とする。
- 二つの boolean は `false` と値の欠落を区別する。情報源が値を示さない場合は推測せず項目を省略する。
- 情報源にない省略可能項目を、空文字、別項目または推測値で補わない。
- JSON はこの表の camelCase 項目名を使用し、省略可能な値がない項目は `SOT-MODEL-009` に従って省略する。

## 確認

必須項目、日付、URL、入力値の複製、boolean の `false` と欠落の区別、および全項目の JSON 表現を単体テストで確認する。

## 関連

- [SOT-MODEL-003: LegalSource](03-legal-source.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
- [SOT-IF-034: `law.update.list` capability v1](../40-interfaces/34-law-update-list-capability.md)
