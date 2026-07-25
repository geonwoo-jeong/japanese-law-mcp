# SOT-IF-033: MCP `search_law_content`

- 状態: 有効

## 規定

`search_law_content` は、境界を明示した e-Gov 法令 API Version 2 の本文検索式を受け取り、公式情報源で確認できる `LawContentSearchResult` を返す provider 固有の MCP 互換ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `query` | string | はい | 正規化後 1 文字以上、UTF-8 で 2048 byte 以下 | 法令本文へ適用する e-Gov の検索式 |
| `asOf` | string | いいえ | `2017-04-01` 以降の `YYYY-MM-DD` | 指定日以前で最新のリビジョンを検索する基準日 |
| `limit` | integer | いいえ | 1 以上 100 以下、既定値 20 | 一ページに返す一致位置の上限 |
| `offset` | integer | いいえ | 0 以上 2147483647 以下、既定値 0 | 取得を開始する一致位置 |

`query` は、先頭と末尾に連続する U+0020 を除いた値を正規化済み入力として検索と byte 数の判定に使用する。U+0000 から U+001F および U+007F の ASCII 制御文字を含む値は `invalid_argument` とする。

`query` は、e-Gov が定義するワイルドカード検索または AND、OR、NOT 検索のうち、次に定義する決定的な部分集合を受け付ける。入力検証後の文字列は、演算子の追加、削除、escape または並べ替えを行わず、`SOT-IF-010` の `keyword` へそのまま渡す。

### 検索式の構文

正規化済み `query` に `*` または `?` が一つでもある場合はワイルドカード式、それ以外は論理式として検証する。

ワイルドカード式は、次のすべてを満たす。

- `*` は 0 文字以上、`?` は 1 文字の任意文字に対応する e-Gov の演算子として扱う。
- U+0020、`|`、`!`、`(` または `)` を含めない。
- `*` と `?` 以外の文字を一文字以上含む。

論理式は、次の ABNF 風記法に一致しなければならない。`SP` は U+0020 一文字、`term-char` は U+0020、`|`、`!`、`(`、`)`、`*`、`?` および ASCII 制御文字以外の一 Unicode scalar value とする。

```text
expression  = conjunction *("|" conjunction)
conjunction = factor *(1*SP factor)
factor      = ["!"] (term / "(" expression ")")
term        = 1*term-char
```

この構文では、U+0020 を AND、`|` を OR、factor の先頭に一つだけ置く `!` を NOT、丸括弧を group として受理する。空の term、空の group、前後または連続する `|`、factor のない `!`、対応しない丸括弧、およびワイルドカードと論理演算子の併用は `invalid_argument` とする。構文検証は受理範囲だけを決めるものであり、演算子の評価、検索結果の順位または e-Gov が明示していない意味を本製品が独自に定義するものではない。

欠落、`null`、正規化後の空値、上限超過、日付不正および定義していない入力項目は受け付けない。

## 出力

`LawContentSearchResult` を返す。結果がない場合の表現は `SOT-MODEL-008` に従う。

`limit`、`offset`、`items`、`totalCount` および `nextOffset` の単位は、すべて一つの `sentences[].position` に対応する一致位置とする。`nextOffset` は `SOT-IF-010` に従い、`offset + items.length` と一致し、0 以上 2147483647 以下で現在の `offset` より大きく、残件がある場合だけ返す。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 外部情報源の制限、一時障害または現在のローカル同時実行上限は、原因に応じて `rate_limited`、`source_timeout`、`source_unavailable` または `source_busy` を返す。
- 外部契約、応答または安全上限の問題は、原因に応じて `source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` または `unsafe_source_content` を返す。
- 上記へ分類できない内部処理の失敗は `internal_error` を返す。

各コードの `retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 共通 capability との境界

このツールは e-Gov 固有の検索式を維持する互換 facade であり、`law.content.search@1` の構造化入力へ再解釈しない。e-Gov adapter は両方の入口で同じ応答 parser と資源予算を再利用できるが、入力 mapping と契約試験を混同しない。

## 確認

論理式は少なくとも `情報 公開`、`情報公開|個人情報`、`!個人情報`、`(情報 公開)|個人` および `情報 !個人情報` を受理する。ワイルドカード式は `であって*として*定める` と `第?条` を受理する。`*` だけ、`情報* 公開`、`情報||公開`、`情報|`、`!`、`()`、`(情報` および ASCII 制御文字を含む値は、ネットワーク呼出し前に `invalid_argument` とする。

`limit: 1` の応答に一つの法令と複数の `sentences` を含む不正 fixture を使い、返却 item を黙って切り詰めず `invalid_source_response` とすることを確認する。正常 fixture では `sentence_count`、展開後の item 数および `limit` の関係と、`offset` から `nextOffset` への一致位置単位の継続を確認する。`next_offset` が `offset + sentence_count` と一致しない場合、および残件があるのに欠落または `null` の場合は `invalid_source_response` とする。

## 関連

- [SOT-SCN-004: 法令本文を検索する](../10-scenarios/04-search-law-content.md)
- [SOT-MODEL-008: LawContentSearchResult](../20-model/08-law-content-search-result.md)
- [SOT-IF-010: e-Gov 本文検索マッピング](10-egov-content-search-mapping.md)
- [SOT-IF-023: law.content.search capability v1](23-law-content-search-capability.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [e-Gov 法令検索ヘルプ「検索式の書き方」](https://laws.e-gov.go.jp/help/#how-to-write-a-search-expression)
