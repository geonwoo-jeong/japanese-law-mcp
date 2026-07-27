# SOT-MODEL-021: JudicialDecisionDetails

- 状態: 有効

## 規定

`JudicialDecisionDetails` は、一つの `JudicialDecisionSummary` と、同じ公式詳細ページに掲載された裁判例固有の詳細を表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `summary` | `JudicialDecisionSummary` | はい | 検索結果と同じ意味を持つ裁判例概要 |
| `reporterCitation` | string | いいえ | 判例集等の巻、号および頁 |
| `lowerCourtName` | string | いいえ | 原審裁判所名 |
| `lowerCourtCaseNumber` | string | いいえ | 原審事件番号 |
| `lowerCourtDecisionDate` | date | いいえ | 原審裁判年月日 |
| `holdingText` | string | いいえ | 詳細ページに文字として掲載された判示事項または判示事項の要旨 |
| `summaryText` | string | いいえ | 詳細ページに文字として掲載された裁判要旨または要旨 |
| `referencedProvisionsText` | string | いいえ | 詳細ページに掲載された参照法条の原文 |

## 制約

- `summary` は `SOT-MODEL-020` に従う。
- 省略しない日付は実在する `YYYY-MM-DD` とする。
- 文字列の意味、列挙順序、条番号および改行を変更しない。HTML 表示に由来する先頭末尾の空白と空行だけを除き、行内の連続空白だけを一つへ正規化できる。
- PDF のリンクしかない本文または要旨を `holdingText` 若しくは `summaryText` へ設定しない。
- 参照法条を `Citation`、法令 ID または現行法令へ推測変換せず、公式表示を `referencedProvisionsText` として保持する。
- 原審結果、行政事件若しくは労働事件だけが持つ分野、および知的財産事件だけが持つ当事者、権利種別、訴訟類型、事件種別、事件種類、発明等の名称、主な争点若しくは上告情報は、共通項目へ意味を縮約せず初期版の公開モデルへ含めない。
- JSON はこの表の camelCase 項目名を使用し、省略可能な値がない項目は `SOT-MODEL-009` に従って省略する。

## 確認

共通概要の保持、判示事項と要旨のラベル差異、詳細文字列、原審日付、欠落可能項目、カテゴリー固有値の非混入、PDF リンクだけの場合の文字列省略および JSON 表現を単体テストで確認する。

## 関連

- [SOT-MODEL-020: JudicialDecisionSummary](20-judicial-decision-summary.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
- [SOT-IF-042: `judicial-decision.read` capability v1](../40-interfaces/42-judicial-decision-read-capability.md)
