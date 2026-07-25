# SOT-IF-032: MCP `get_article`

- 状態: 有効

## 規定

`get_article` は、境界を明示した法令識別子、本則または原始附則の区分、条番号、任意指定の項番号、および任意指定の検索基準日を受け取り、該当部分の XML と `Citation` を返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `lawId` | string | はい | 正規化後 1 文字以上、UTF-8 で 256 byte 以下 | 公式情報源の法令識別子 |
| `provision` | string | いいえ | `main` または `supplementary`、既定値 `main` | 本則または原始附則 |
| `article` | string | はい | `SOT-MODEL-018` の正規形、UTF-8 で 64 byte 以下 | 公式に付された条番号 |
| `paragraph` | integer | いいえ | 1 以上 | 公式に付された項番号 |
| `asOf` | string | いいえ | `2017-04-01` 以降の `YYYY-MM-DD` | 指定日以前で最新の本文を選ぶ基準日 |

`lawId` は、先頭と末尾に連続する U+0020 を除いた値を正規化済み入力として取得と byte 数の判定に使用する。`lawId` または `article` に U+0000 から U+001F および U+007F の ASCII 制御文字を含めない。

欠落、`null`、正規化後の空値、上限超過、形式不正、日付不正および定義していない入力項目は `invalid_argument` とする。

`paragraph` を省略した場合は指定した条全体を返し、指定した場合はその条に属する該当項だけを返す。

## 出力

| 名前 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `format` | string | はい | `xml` |
| `content` | string | はい | 該当する条または項をシリアライズした UTF-8 XML |
| `citation` | `Citation` | はい | 法令と条文位置を確認するための情報 |

内部結果が `format: xml` であり、選択した要素が入力から作った `LawArticleLocation` と一致する場合だけ公開結果へ投影する。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 該当する法令または条文がない場合は `not_found` を返す。
- 条文位置を一意に決定できない場合は `ambiguous_location` を返す。
- 外部情報源の制限、一時障害または現在のローカル同時実行上限は、原因に応じて `rate_limited`、`source_timeout`、`source_unavailable` または `source_busy` を返す。
- 外部契約、応答または安全上限の問題は、原因に応じて `source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` または `unsafe_source_content` を返す。
- 上記へ分類できない内部処理の失敗は `internal_error` を返す。

各コードの `retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 互換 facade

このツールは `SOT-IF-025` の内部 capability へ接続する互換 facade とする。正規化済み `lawId` から `SourceResourceRef`、`provision` または既定値、`article` および `paragraph` から `LawArticleLocation` を組み立てる。

公開入力と内部 capability に共通する識別子、位置および日付の境界条件は同じ fixture で検証し、公開側で受理した値が内部の共通入力検証だけを理由に拒否されないことを確認する。

## 関連

- [SOT-SCN-003: 条文を取得する](../10-scenarios/03-get-article.md)
- [SOT-MODEL-004: Citation](../20-model/04-citation.md)
- [SOT-MODEL-018: LawArticleLocation](../20-model/18-law-article-location.md)
- [SOT-IF-012: e-Gov 条文取得マッピング](12-egov-article-mapping.md)
- [SOT-IF-025: law.article.read capability v1](25-law-article-read-capability.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
