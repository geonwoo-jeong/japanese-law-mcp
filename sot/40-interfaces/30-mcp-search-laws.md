# SOT-IF-030: MCP `search_laws`

- 状態: 廃止
- 後継: [SOT-IF-049: MCP `search_laws` v2](49-mcp-search-laws-v2.md)

## 廃止理由

検証済みの原検索を優先しながら、出典を持つ略称、表記揺れ、自然文および一意な軽微誤記を解決する公開契約へ置き換える。

## 規定

`search_laws` は、境界を明示した法令名または略称の一部を受け取り、公式情報源で確認できる `LawSearchResult` を返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `query` | string | はい | 正規化後 1 文字以上、UTF-8 で 512 byte 以下 | 法令名または略称に含まれる文字列 |
| `asOf` | string | いいえ | `2017-04-01` 以降の `YYYY-MM-DD` | 指定日以前で最新のリビジョンを検索する基準日 |
| `limit` | integer | いいえ | 1 以上 100 以下、既定値 20 | 一ページに返す法令数 |
| `offset` | integer | いいえ | 0 以上 2147483647 以下、既定値 0 | 取得を開始する法令位置 |

`query` は、先頭と末尾に連続する U+0020 を除いた値を正規化済み入力として検索と byte 数の判定に使用する。U+0000 から U+001F および U+007F の ASCII 制御文字を含む値は `invalid_argument` とする。

正規化済み `query` の先頭と末尾がともに U+002F `/` である値は、e-Gov の正規表現指定と区別できないため受け付けない。この公開ツールは正規表現検索を契約にせず、該当値を `invalid_argument` とする。

欠落、`null`、正規化後の空値、上限超過、日付不正および定義していない入力項目は受け付けない。

## 出力

`LawSearchResult` を返す。結果がない場合の表現は `SOT-MODEL-006` に従う。

`nextOffset` は 0 以上 2147483647 以下で、同じ条件の次の法令位置を e-Gov が `next_offset` として返した場合、または `SOT-IF-009` に従い e-Gov が返した `count` と `total_count` から `offset + count` を導出できる場合だけ返す。範囲外、現在の `offset` 以下、結果が残るのに前進できない値、および `count`、配列長または `total_count` と矛盾する値は `invalid_source_response` とする。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 外部情報源の制限、一時障害または現在のローカル同時実行上限は、原因に応じて `rate_limited`、`source_timeout`、`source_unavailable` または `source_busy` を返す。
- 外部契約、応答または安全上限の問題は、原因に応じて `source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` または `unsafe_source_content` を返す。
- 上記へ分類できない内部処理の失敗は `internal_error` を返す。

各コードの `retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 互換 facade

このツールは、e-Gov の数値 `offset` を維持する provider 固有の互換 facade とする。`SOT-IF-022` の内部 capability と同じ正規化済み `query`、明示された `asOf`、`limit`、応答 parser、`LawSummary` mapping およびページ不変条件を再利用するが、公開 `offset` を `continuationToken` に変換して内部 capability を直接呼び出さない。

公開入力と内部 capability に共通する `query`、`asOf` および `limit` の境界条件は同じ fixture で検証し、公開側で受理した値が内部の共通入力検証だけを理由に拒否されないことを確認する。

`asOf` を省略した公開 facade は `SOT-IF-009` に従い `asof` を送信しない。内部 capability は continuation の snapshot を固定するため、同じ省略を `Asia/Tokyo` の実効日へ変換する。この違いを隠すために facade の任意 `offset` から内部 token を合成しない。

同じ検証済み e-Gov 応答に対する公開 `nextOffset` と内部 continuation position は、`SOT-IF-009` の同じ取得位置から生成する。`next_offset` の欠落時だけ一方が継続できる、または両者でページ単位が異なる実装を許可しない。

## 関連

- [SOT-SCN-001: 法令名から法令を検索する](../10-scenarios/01-search-laws.md)
- [SOT-MODEL-006: LawSearchResult](../20-model/06-law-search-result.md)
- [SOT-IF-009: e-Gov 法令名検索マッピング](09-egov-law-search-mapping.md)
- [SOT-IF-022: law.search capability v1](22-law-search-capability.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
