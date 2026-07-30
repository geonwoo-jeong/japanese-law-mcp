# SOT-IF-049: MCP `search_laws` v2

- 状態: 廃止
- 後継: [SOT-IF-053: MCP `search_laws` v3](53-mcp-search-laws-v3.md)

## 規定

原検索の非空 page に含まれる解決済み法令を優先できないため、本規定を
`SOT-IF-053` に置き換えた。

以下は廃止時点の履歴であり、現行の公開契約には適用しない。

`search_laws` は、境界を明示した法令名、略称、法令名を一つ含む自然文または軽微な誤記を受け取り、選択済みの公式情報源で確認できる `LawSearchResult` を返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `query` | string | はい | 正規化後 1 文字以上、UTF-8 で 512 byte 以下 | 法令を特定するための検索語 |
| `asOf` | string | いいえ | `2017-04-01` 以降の `YYYY-MM-DD` | 指定日以前で最新のリビジョンを検索する基準日 |
| `limit` | integer | いいえ | 1 以上 100 以下、既定値 20 | 一ページに返す法令数 |
| `offset` | integer | いいえ | 0 以上 2147483647 以下、既定値 0 | 取得を開始する法令位置 |

`query` は、先頭と末尾に連続する U+0020 を除いた値を検証済み原文として、byte 数の判定と最初の検索に使用する。U+0000 から U+001F および U+007F の ASCII 制御文字を含む値は `invalid_argument` とする。

検証済み `query` の先頭と末尾がともに U+002F `/` である値は、e-Gov の正規表現指定と区別できないため受け付けない。この公開ツールは正規表現検索を契約にせず、該当値を `invalid_argument` とする。

欠落、`null`、検証後の空値、上限超過、日付不正および定義していない入力項目は受け付けない。

## 検索語の解決

`offset` が 0 の場合は、検証済み原文を変更せず最初に検索する。原検索が非空の正常結果を返した場合は、その結果を変更せず返す。原検索がエラーを返した場合は、別の検索語を試さずそのエラーを返す。原検索が正常な空結果を返した場合だけ、次の順で一つの安全な候補を解決する。

1. 正式名称または出典を持つ別名との完全一致
2. Unicode NFKC、かな、空白および句読点を比較用に正規化した完全一致
3. Kagome の user dictionary が自然文から抽出した正式名称または別名との完全一致
4. rune 単位の Damerau-Levenshtein 距離が閉じた閾値内にあり、最小距離の法令 ID が一つだけである誤記候補

比較用の正規化値、読み、token または編集途中の文字列を情報源へ送らない。候補検索には辞書に記録した正式名称を使用し、`asOf`、`limit` および `offset` は原検索から変更しない。意味が似ているだけの語は、出典を持つ別名として辞書にない限り候補にしない。

3 rune 未満の語には誤記補正を行わない。3 rune 以上 9 rune 以下では距離 1、10 rune 以上 15 rune 以下では距離 2、16 rune 以上では距離 3 以下かつ長い方の 20% 以下を上限とする。同じ最小距離で複数の法令 ID が残る場合、自然文から複数の法令 ID が見つかる場合または一つの別名が複数の法令 ID に対応する場合は、自動解決しない。

安全な候補がない場合または候補検索も正常な空結果の場合は、最初の空結果を返す。候補検索がエラーを返した場合は、そのエラーを返す。原検索と候補検索を結合せず、総件数、順序または取得位置を独自に合成しない。

## offset

候補検索の可否は `items` の長さではなく、ページ位置と独立した `totalCount` が 0 であるかによって判定する。`offset` が 0 より大きく、現在のページの `items` が空であっても `totalCount` が 1 以上なら、原検索の正常結果をそのまま返し、別の検索語へ切り替えない。

候補検索は原検索と同じ `asOf`、`limit` および `offset` を使用する。すべての検索は一つの公開リクエスト期限と context cancellation に従い、原検索と候補検索を合わせて最大二回だけ情報源を呼び出す。

## 出力

`SOT-MODEL-006` の `LawSearchResult` を返す。結果がない場合は `totalCount: 0` と空の `items` を返す。

`nextOffset` は 0 以上 2147483647 以下で、選択した一つの検索条件に対して e-Gov が `next_offset` として返した場合、または `SOT-IF-050` に従い e-Gov が返した `count` と `total_count` から `offset + count` を導出できる場合だけ返す。前処理の確認検索から `nextOffset` を作らない。

検索語の解決方法、内部 token、辞書の出典または補正前後の検索語を出力へ追加しない。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 外部情報源の制限、一時障害または現在のローカル同時実行上限は、原因に応じて `rate_limited`、`source_timeout`、`source_unavailable` または `source_busy` を返す。
- 外部契約、応答または安全上限の問題は、原因に応じて `source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` または `unsafe_source_content` を返す。
- 起動時に検証済みの前処理がリクエスト中に予期せず失敗した場合を含み、上記へ分類できない内部処理の失敗は `internal_error` を返す。

各コードの `retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 互換 facade

このツールは、e-Gov の数値 `offset` を維持する provider 固有の互換 facade とする。`SOT-IF-022` の内部 capability と同じ query の検証、明示された `asOf`、`limit`、応答 parser、`LawSummary` mapping およびページ不変条件を再利用するが、公開 `offset` を `continuationToken` に変換して内部 capability を直接呼び出さない。

前処理で選択した各検索語は既存の公開 facade へ独立した `Request` として渡し、facade と e-Gov アダプターに辞書または Kagome の判断を持ち込まない。無設定起動の選択済みプロバイダーは e-Gov 法令 API Version 2 とする。

## 確認

原検索の非空結果とエラーを保存すること、正常な空結果だけが候補検索を許すこと、候補検索の結果を変更しないこと、および候補同士を集約しないことを偽の provider で確認する。

正式名称、公式略称、補足略称、自然文、互換表記、挿入、削除、置換、隣接文字の転置、短い語、衝突する略称、同率候補および複数法令を含む文を fixture にする。`offset` が 0 より大きく `items` が空でも `totalCount` が 1 以上なら候補検索を行わず、`totalCount` が 0 の場合だけ同じ `asOf`、`limit` および `offset` で候補検索することを確認する。

## 関連

- [SOT-SCN-001: 法令名から法令を検索する](../10-scenarios/01-search-laws.md)
- [SOT-SCN-008: 法令名検索語を解決する](../10-scenarios/08-resolve-law-name-query.md)
- [SOT-MODEL-006: LawSearchResult](../20-model/06-law-search-result.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-IF-022: law.search capability v1](22-law-search-capability.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-050: e-Gov 法令名検索マッピング v2](50-egov-law-search-mapping-v2.md)
- [SOT-ENG-022: 法令名検索辞書](../50-engineering/22-law-name-search-lexicon.md)
