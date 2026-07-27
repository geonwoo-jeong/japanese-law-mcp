# SOT-IF-045: 裁判所の裁判例詳細マッピング

- 状態: 有効

## 規定

`courts-hanrei-html` の `judicial-decision.read@1` は、検証済みの canonical resource ID から公式詳細 URL を組み立て、詳細 HTML のラベル付き項目を `JudicialDecisionDetails` へ決定的に対応させる。

## 要求と参照

入力の `ref` は次をすべて満たす。

- `providerId` が `courts-hanrei-html`
- `key.sourceId` が `courts-hanrei`
- `key.resourceType` が `judicial-decision`
- `key.resourceId` が ASCII の `{decisionId}/detail{categoryNumber}`。`decisionId` は一桁以上、`categoryNumber` は `2` から `8`
- `key.versionId` が未設定

不一致は外部呼出し前に `invalid_argument` とする。要求 URL は次の一つに固定する。

```text
https://www.courts.go.jp/hanrei/{decisionId}/detail{categoryNumber}/index.html
```

## HTML の識別

page title が `裁判例結果詳細` を含み、main 内に一つ以上の `dl` があり、各項目の `dt` をラベル、対応する `dd` を値として読む。

必須の事件番号、裁判年月日、`法廷名`、`裁判所名・部` または `裁判所名` のいずれか、および入力と同じ詳細 URL を識別できない場合は `invalid_source_response` とする。詳細画面ではない構造へ変更された場合は `source_contract_changed` とする。同じ出力項目へ対応するラベル群が異なる値で複数回現れる場合は `invalid_source_response` とする。

script を実行せず、HTML から参照される PDF、画像、stylesheet その他の resource を取得しない。

## 項目対応

`publicationCategory` は detail path から `SOT-IF-044` のカテゴリー対応によって導出する。detail path と詳細ページのカテゴリー見出しは次の組合せだけを受け入れ、矛盾する場合は `invalid_source_response` とする。

| detail path | カテゴリー見出し |
|---|---|
| `detail2` | `最高裁判所` |
| `detail3` | `高等裁判所` |
| `detail4` | `下級裁判所(速報)` |
| `detail5` | `行政事件` |
| `detail6` | `労働事件` |
| `detail7` | `知的財産事件` |
| `detail8` | `知的財産高等裁判所` |

`sourceCategoryLabel` は詳細ページに表示された上表のカテゴリー見出しとする。検索結果のリンク表示を保持する `SOT-IF-044` の `sourceCategoryLabel` とは表記が異なり得るため、検索から詳細への往復では `ref`、`decisionId` および `publicationCategory` の同一性を検証し、二画面の `sourceCategoryLabel` の文字列一致を要求しない。

その他は次のラベルを使用する。

| HTML の `dt` | 出力 |
|---|---|
| `事件番号` | `summary.caseNumber` |
| `事件名` | `summary.caseName` |
| `裁判年月日` | `summary.decisionDate` |
| `法廷名`、`裁判所名・部` または `裁判所名` | `summary.courtName` |
| `部名` | `summary.divisionName` |
| `裁判種別` | `summary.decisionType` |
| `結果` または `判決結果` | `summary.outcome` |
| `判例集等巻・号・頁` または `高裁判例集登載巻・号・頁` | `reporterCitation` |
| `原審裁判所名` | `lowerCourtName` |
| `原審事件番号` | `lowerCourtCaseNumber` |
| `原審裁判年月日` | `lowerCourtDecisionDate` |
| `判示事項` または `判示事項の要旨` | `holdingText` |
| `裁判要旨` または `要旨` | `summaryText` |
| `参照法条` | `referencedProvisionsText` |
| `全文`、`要旨` その他の PDF link | `summary.documents` |

`summary.decisionId`、`detailUrl` および `source` は、入力 ref、要求 URL および `SOT-IF-043` の固定値から作る。

支部を独立したラベルで確認できる場合だけ `summary.branchName` を設定する。`裁判所名・部`、法廷名または裁判所名の文字列から支部若しくは部を切り出さない。`裁判所名・部` は表示された結合値をそのまま `summary.courtName` へ保持する。

ラベルがない値を、表示順または別カテゴリーの欄から推測しない。`原審結果`、行政事件若しくは労働事件の `分野` および知的財産事件だけの `事件種別`、`事件種類`、`当事者`、`権利種別`、`訴訟類型`、`発明等の名称等`、`主な争点`、`上告提起等の有無` 若しくは `上告審の結果` は、近い名前の共通項目へ変換せず、`SOT-MODEL-021` に従い初期版の公開モデルへ混入させない。特に `事件種別` または `事件種類` を `summary.decisionType` へ変換しない。

## 日付、文字列および文書

和暦日付は `SOT-IF-044` と同じ元号境界検証で `YYYY-MM-DD` へ変換する。

文字列値は HTML entity を文字へ復元し、先頭末尾の空白行を除く。複数行の順序と改行を保持し、各行の先頭末尾の空白と行内の連続空白だけを一つへ正規化する。空になった省略可能値は設定しない。

PDF link は `SOT-IF-044` と同じ `JudicialDocumentLink` へ対応させる。`裁判要旨` または `要旨` の `dd` に文字列と PDF link が併存する場合は、link の表示文字列を除く本文を `summaryText`、PDF link を `summary.documents` としてそれぞれ保持する。PDF link の表示文字列だけの `要旨` を `summaryText` へ複製しない。

## 出典

出力 `ref` は入力と同じ値とする。`provenance.url` は要求した詳細 URL、`mediaType` は `text/html`、`transformation` は `normalized`、`methodId` は `SOT-IF-045` とする。最後の `resourceKey` は入力 `ref.key` と一致させる。

## 確認

全 detail path のカテゴリー、`裁判所名・部` の非分割、ラベルの差異、欠落可能項目、重複ラベル、原審日付、複数行文字列、共通化しないカテゴリー固有値、PDF link だけの要旨、文字列と PDF link が混在する要旨、404、入力 ref の改変、resource budget および検索からの往復を fixture で確認する。

## 関連

- [SOT-MODEL-021: JudicialDecisionDetails](../20-model/21-judicial-decision-details.md)
- [SOT-IF-042: `judicial-decision.read` capability v1](42-judicial-decision-read-capability.md)
- [SOT-IF-043: 裁判所「裁判例検索」HTML 情報源](43-source-courts-hanrei-html.md)
- [SOT-IF-044: 裁判所の裁判例検索マッピング](44-courts-hanrei-search-mapping.md)
