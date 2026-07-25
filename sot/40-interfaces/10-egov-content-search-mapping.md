# SOT-IF-010: e-Gov 本文検索マッピング

- 状態: 有効

## 規定

`search_law_content` は e-Gov 法令 API Version 2 の `GET /keyword` を呼び出す。ツール入力を e-Gov の本文検索条件へ変換し、e-Gov のレスポンスを `LawContentSearchResult` へ変換する。

## リクエスト

| `search_law_content` | `GET /keyword` |
|---|---|
| `query` | `keyword` |
| `asOf` | `asof` |
| `limit` | `limit` |
| `offset` | `offset` |

`response_format` は `json`、`order` は `+law_info.law_id`、`highlight_tag` は `mark` とする。`sentences_limit` は送信しない。`asOf` がない場合は `asof` を送信しない。

`query` は `SOT-IF-033` の正規化と検索式検証を外部呼出し前に完了した値だけを受け取る。検証済みの文字列を `keyword` の値として UTF-8 から percent-encoding し、検索演算子を escape、追加、削除または並べ替えない。文字列連結によって URL へ直接埋め込まない。

## レスポンス

`items` 内の各法令を `LawSummary` に変換し、その法令の `sentences` を一件ずつ次のように `LawContentMatch` へ展開する。

| e-Gov | Japanese Law MCP |
|---|---|
| 法令の `law_info` と `revision_info` | `LawContentMatch.law` |
| `sentences[].position` | `LawContentMatch.location` |
| `sentences[].text` | `LawContentMatch.text` |
| `total_count` | `LawContentSearchResult.totalCount` |
| `next_offset` | `LawContentSearchResult.nextOffset` |

`sentences[].text` から、検索 API が挿入した `<mark>` と `</mark>` だけを除去する。それ以外の文字列は変更しない。

`LawContentMatch.citation` は、`LawSummary` の法令 ID とリビジョン ID、`position` および `SOT-IF-011` の URL 規則から生成する。

## ページ不変条件

e-Gov の公式 OpenAPI では、`limit` は応答内の `sentences[].position` 数の総和の上限、`sentence_count` は現在の応答に含まれる `sentences` 数の総和、`next_offset` は次に指定する `offset` と定義されている。

したがって、次のすべてを検証してから公開結果を作る。

- `total_count`、`sentence_count` および `items` が存在する。
- 全 `items[].sentences` を展開した件数が `sentence_count` と一致する。
- `sentence_count` は 0 以上で、要求した `limit` 以下である。
- `total_count` と `sentence_count` は 0 以上である。
- `offset + sentence_count` は integer overflow を起こさず、`total_count` 以下である。`offset` が `total_count` より大きく `sentence_count: 0` の場合だけ、`offset + sentence_count > total_count` を許容する。
- `next_offset` が非 `null` の場合は `offset + sentence_count` と一致し、現在の `offset` より大きく、0 以上 2147483647 以下かつ `total_count` 以下であり、`offset + sentence_count < total_count` である。
- `next_offset` が欠落または `null` の場合は、`offset + sentence_count >= total_count` でなければならない。

各 `items` は `law_info`、`revision_info` および一件以上の `sentences` を持ち、各 sentence は `position` と `text` を持たなければならない。公開契約に必要な項目の欠落または上記の値の不変条件への違反は `invalid_source_response` とし、公式 schema の必須項目または型自体が確認済み仕様から変わった場合だけ `source_contract_changed` とする。全 `sentences` の一部を `limit` に合わせて切り捨てず、一つの法令 item を一件として数え直さない。

`LawContentSearchResult.items` の件数は `sentence_count`、`totalCount` は `total_count` とする。`nextOffset` は検証済みの `next_offset` だけを使用し、独自に加算して作らない。

`next_offset` が欠落または `null` なのに `total_count` が残件を示す場合は、独自の offset を合成せず `invalid_source_response` とする。

## エラー

エラーの変換は `SOT-IF-009` と同じ規則を使用する。

`SOT-IF-033` の構文に一致しない値は外部呼出し前に `invalid_argument` とする。事前検証を通過した値に e-Gov が `4xx` を返した場合は、利用者入力を別の意味へ再解釈せず `invalid_source_response` とする。

## 関連

- [SOT-IF-033: MCP `search_law_content`](33-mcp-search-law-content.md)
- [SOT-IF-009: e-Gov 法令名検索マッピング](09-egov-law-search-mapping.md)
- [SOT-IF-011: e-Gov 法令本文取得マッピング](11-egov-law-document-mapping.md)
- [SOT-MODEL-008: LawContentSearchResult](../20-model/08-law-content-search-result.md)
- [e-Gov 法令 API Version 2 OpenAPI](https://laws.e-gov.go.jp/api/2/swagger-ui/lawapi-v2.yaml)
