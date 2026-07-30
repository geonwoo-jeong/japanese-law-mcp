# SOT-MODEL-020: JudicialDecisionSummary

- 状態: 有効

## 規定

`JudicialDecisionSummary` は、公式情報源に掲載された一つの裁判例の一つの掲載カテゴリーについて、情報源内の識別、裁判情報、公式詳細ページおよび公式文書への参照を表す。同じ裁判例が複数の掲載カテゴリーを持つ場合は、カテゴリーごとに別の summary として表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `decisionId` | string | はい | 公式詳細 URL が示す裁判例識別子 |
| `publicationCategory` | string | はい | 正規化した掲載カテゴリー |
| `sourceCategoryLabel` | string | はい | 情報源が表示した掲載カテゴリー名 |
| `caseNumber` | string | はい | 情報源が表示した事件番号 |
| `caseName` | string | いいえ | 情報源が表示した事件名 |
| `decisionDate` | date | はい | 裁判年月日 |
| `courtName` | string | はい | 裁判所または法廷の名称 |
| `branchName` | string | いいえ | 情報源が独立して示した支部名 |
| `divisionName` | string | いいえ | 情報源が独立して示した部または法廷名 |
| `decisionType` | string | いいえ | 判決、決定その他の裁判種別 |
| `outcome` | string | いいえ | 情報源が表示した結果 |
| `detailUrl` | string | はい | 裁判所の公式詳細ページ |
| `documents` | `JudicialDocumentLink[]` | はい | 情報源が直接示した公式文書。存在しない場合は空配列 |
| `source` | `InformationSource` | はい | 裁判例情報を取得した情報源 |

`publicationCategory` は次のいずれかとする。

- `supreme_court`
- `high_court`
- `lower_court`
- `administrative`
- `labor`
- `intellectual_property`

`JudicialDocumentLink` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `kind` | string | はい | `full_text`、`summary` または `attachment` |
| `label` | string | はい | 情報源が表示した文書名 |
| `mediaType` | string | はい | 初期版では `application/pdf` |
| `url` | string | はい | 裁判所が直接示した公式 HTTPS URL |

## 制約

- `decisionId`、`sourceCategoryLabel`、`caseNumber` および `courtName` は空文字にしない。
- `decisionDate` は、情報源が示した完全な年月日を検証して `YYYY-MM-DD` へ変換する。日を確認できない値を補完しない。
- 事件番号、裁判所名、事件名、裁判種別および結果は、語彙、全角半角、括弧または符号を別体系へ変換せず、表示上の空白だけを正規化する。
- `detailUrl` と `documents[].url` は、認証情報を含まない `https://www.courts.go.jp/` の URL とする。
- 情報源にない省略可能項目を空文字、別項目または推測値で補わない。
- JSON はこの表の camelCase 項目名を使用し、省略可能な値がない項目は `SOT-MODEL-009` に従って省略する。

## 確認

六つのカテゴリー、和暦から完全な Gregorian date への変換、欠落可能項目、複数文書、URL 制約、入力値の意味の保持および JSON 表現を単体テストで確認する。

## 関連

- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
- [SOT-MODEL-010: InformationSource](10-information-source.md)
- [SOT-MODEL-021: JudicialDecisionDetails](21-judicial-decision-details.md)
- [SOT-ARCH-018: 拡張パック単位の正規化境界](../30-architecture/18-pack-scoped-normalization-boundary.md)
