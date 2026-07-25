# SOT-IF-031: MCP `get_law`

- 状態: 有効

## 規定

`get_law` は、境界を明示した法令識別子と任意の検索基準日を受け取り、公式情報源から取得した一つの XML `LawDocument` を返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `lawId` | string | はい | 正規化後 1 文字以上、UTF-8 で 256 byte 以下 | 公式情報源の法令識別子 |
| `asOf` | string | いいえ | `2017-04-01` 以降の `YYYY-MM-DD` | 指定日以前で最新の本文を選ぶ基準日 |

`lawId` は、先頭と末尾に連続する U+0020 を除いた値を正規化済み入力として取得と byte 数の判定に使用する。U+0000 から U+001F および U+007F の ASCII 制御文字を含む値は `invalid_argument` とする。

欠落、`null`、正規化後の空値、上限超過、日付不正および定義していない入力項目は受け付けない。

`asOf` を省略した場合は、情報源がリクエスト処理時点で最新として返すリビジョンを取得する。

## 出力

該当する法令が存在する場合は、`SOT-MODEL-002` の `LawDocument` を返す。内部の `LawDocumentRepresentation` が `format: xml` であり、同モデルの XML 制約を満たす場合だけ投影する。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 該当する法令または基準時点の本文がない場合は `not_found` を返す。
- 外部情報源の制限、一時障害または現在のローカル同時実行上限は、原因に応じて `rate_limited`、`source_timeout`、`source_unavailable` または `source_busy` を返す。
- 外部契約、応答または安全上限の問題は、原因に応じて `source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` または `unsafe_source_content` を返す。
- 上記へ分類できない内部処理の失敗は `internal_error` を返す。

各コードの `retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 互換 facade

このツールは `SOT-IF-024` の内部 capability へ接続する互換 facade とする。正規化済み `lawId` を `resource.key.resourceId`、`asOf` を同名項目へ渡し、primary route から残りの `SourceResourceRef` を組み立てる。

公開入力と内部 capability に共通する `lawId` と `asOf` の境界条件は同じ fixture で検証し、公開側で受理した値が内部の共通入力検証だけを理由に拒否されないことを確認する。

## 関連

- [SOT-SCN-002: 法令本文を取得する](../10-scenarios/02-get-law.md)
- [SOT-MODEL-002: LawDocument](../20-model/02-law-document.md)
- [SOT-MODEL-017: LawDocumentRepresentation](../20-model/17-law-document-representation.md)
- [SOT-IF-011: e-Gov 法令本文取得マッピング](11-egov-law-document-mapping.md)
- [SOT-IF-024: law.document.read capability v1](24-law-document-read-capability.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
