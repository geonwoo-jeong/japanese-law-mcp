# SOT-IF-044: 裁判所の裁判例検索マッピング

- 状態: 有効

## 規定

裁判所「裁判例検索」の統合検索 HTML を、安全な DOM 解析によって `JudicialDecisionSearchPageV1` へ決定的に対応させる。

## 要求

`JudicialDecisionSearchRequestV1.query` を `query1` の値として一回だけ percent-encode し、次の GET を行う。

```text
https://www.courts.go.jp/hanrei/search1/index.html?query1={query}
```

`limit` は外部 query parameter に変換せず、検証済みの先頭結果から返す件数の上限としてだけ使用する。`continuationToken`、`offset`、並び順、カテゴリー、裁判年月日、事件番号、裁判所その他の未採用条件を送信しない。

## HTML の識別

次を検索結果の契約識別子とする。

- page title が `裁判例検索` を含む
- 結果がある場合は `table.search-result-table` の行を使用する
- 各結果行は、`th` 内の `/hanrei/{id}/detail{2..8}/index.html` への一つ以上の公式詳細リンクと、情報を持つ `td` を持つ
- 同じ結果行に複数の公式詳細リンクがある場合は、すべての `{id}` が一致し、正規化後の detail path が重複しないことを確認する。各リンクを DOM 順の別 item とし、同じ行の裁判情報と公式文書をそれぞれへ対応させる
- 空結果は、`p#searched` に `該当する裁判例がありませんでした。` の表示が一つあり、契約を満たす結果行がない状態とする。この場合の総件数は 0 とする
- 取得結果が 2000 件を超える場合は、表示されている一つの `ul.errorMessage` が `検索結果が2000件を超えました。「全文検索」欄の検索語を追加・変更してください。` で始まり、契約を満たす結果行と空結果表示がない状態とする。この場合は、公式情報源が検索語の絞り込みを求めた `unsupported_query` とする
- 空結果でない総件数は、`{total}件中` の ASCII または全角数字を一つだけ解釈する

同じ行の詳細 path に異なる `{id}` がある、正規化後の詳細 path が重複する、詳細 path を識別できない、必須の事件番号、裁判年月日若しくは裁判所名がない、件数表示と結果行数が矛盾する、2000 件超過表示が複数ある、または表示されている `ul.errorMessage` が結果行若しくは空結果表示と併存する場合は `invalid_source_response` とする。識別子そのものがない場合は `source_contract_changed` とする。

script を実行せず、CSS、非表示の選択肢、modal、analytics、画像および外部 resource を取得しない。閉じた `details` は最初の `summary` の子孫だけを表示対象とする。`ul.errorMessage`、その祖先または表示対象の子孫に inline `style` が一つでもある場合は、その内容にかかわらず CSS を実行しない条件では表示状態を確定できないものとして `invalid_source_response` とする。

## 資源参照

詳細 URL `/hanrei/{decisionId}/detail{categoryNumber}/index.html` から、次を作る。

| 項目 | 値 |
|---|---|
| `ref.providerId` | `courts-hanrei-html` |
| `ref.key.sourceId` | `courts-hanrei` |
| `ref.key.resourceType` | `judicial-decision` |
| `ref.key.resourceId` | `{decisionId}/detail{categoryNumber}` |
| `ref.key.versionId` | 省略 |

`decisionId` の先頭のゼロを除かず、detail path の数字を別カテゴリーへ変更しない。`data.decisionId` は URL の `{decisionId}` と一致させる。

`provenance.url` は実際に取得した検索 URL、`mediaType` は `text/html`、`transformation` は `normalized`、`methodId` は `SOT-IF-044`、`location` は該当結果行を識別できる値とする。最後の `resourceKey` は上表の `ref.key` と一致させる。

## 項目対応

| HTML 表示 | `JudicialDecisionSummary` |
|---|---|
| 詳細 URL の `{decisionId}` | `decisionId` |
| `th` の詳細リンク表示 | `sourceCategoryLabel` |
| 詳細リンクの `detail{categoryNumber}` と表示名 | `publicationCategory` |
| 結果本文の一行目にある事件番号 | `caseNumber` |
| 事件番号に続く表示 | `caseName`。空なら省略 |
| 裁判年月日 | `decisionDate` |
| 裁判年月日の直後に表示された裁判所または法廷 | `courtName` |
| 明示された支部 | `branchName` |
| 明示された部または法廷 | `divisionName` |
| 明示された裁判種別 | `decisionType` |
| 明示された結果 | `outcome` |
| 詳細リンク | `detailUrl` |
| `file-col` の公式 PDF link | `documents` |
| `SOT-IF-072` の情報源 | `source` |

HTML 上で位置を一意に識別できない省略可能項目は設定しない。周辺文字列の語尾または既知語彙だけから、裁判種別、結果、支部若しくは部を推測しない。

## カテゴリー対応

| source 表示または detail path | `publicationCategory` |
|---|---|
| 最高裁判例、`detail2` | `supreme_court` |
| 高裁判例、`detail3` | `high_court` |
| 下級裁裁判例、`detail4` | `lower_court` |
| 行政事件裁判例、`detail5` | `administrative` |
| 労働事件裁判例、`detail6` | `labor` |
| 知的財産裁判例、知財高裁裁判例、`detail7`、`detail8` | `intellectual_property` |

表示名と detail path の分類が矛盾する場合は `invalid_source_response` とする。

## 日付と文書

裁判年月日の `明治`、`大正`、`昭和`、`平成` または `令和` と、年または `元年`、月、日を Gregorian calendar の完全な `YYYY-MM-DD` へ変換する。各元号の開始日と終了日に反する値、存在しない日付、年・月・日の欠落を拒否する。

`全文` は `full_text`、`要旨` は `summary`、その他の同一行の公式 PDF は `attachment` とする。link の表示名を `label` に保持し、media type は `application/pdf` とする。URL は HTML の base URL から解決した後、`https://www.courts.go.jp/` であることを検証する。

## 件数

結果行の DOM 順と、同じ行にある公式詳細リンクの DOM 順を変更せず、掲載カテゴリー単位の先頭から `limit` 件まで返す。`SourcePage.returnedCount` は返した item 数とする。

公式表示の総件数は結果行の件数であり、一つの結果行から複数の掲載カテゴリー item が作られ得るため、`SourcePage` と同じ単位の総件数ではない。したがって、この provider は `totalCount` と `totalRelation` を設定しない。空結果でも `returnedCount: 0` だけを設定する。取得行数または展開後の item 数が `limit` より多いことだけを理由に error とせず、`nextToken` は発行しない。

この provider は現在の adapter contract では継続位置を発行しないため、空でない `continuationToken` を予約入力として外部呼出し前に `invalid_argument` とする。

## 確認

六カテゴリー、知財高裁表示、空結果、2000 件超過表示、上限切詰め、複数カテゴリー行の DOM 順展開と総件数の省略、欠落可能項目、複数 PDF、相対 URL、和暦境界、重複または矛盾した詳細 URL、DOM 深さ・node 数および script 非実行を fixture で確認する。

## 関連

- [SOT-MODEL-020: JudicialDecisionSummary](../20-model/20-judicial-decision-summary.md)
- [SOT-IF-041: `judicial-decision.search` capability v1](41-judicial-decision-search-capability.md)
- [SOT-IF-072: 裁判所「裁判例検索」HTML 情報源 v2](72-source-courts-hanrei-html-v2.md)
