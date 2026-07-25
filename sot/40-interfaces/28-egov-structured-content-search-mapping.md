# SOT-IF-028: e-Gov 構造化本文検索マッピング

- 状態: 有効

## 規定

e-Gov 法令 API Version 2 のアダプターが `law.content.search@1` を実装する場合は、構造化された検索語だけから e-Gov のキーワード検索式を決定的に生成し、外部検索式を入力から注入させない。

## リクエスト

`LawContentSearchRequestV1` を検証した後、次の順で `keyword` を生成する。

1. `allTerms` の各語を入力順に並べ、半角空白で連結する。
2. `anyTerms` が一件の場合はその語を使用し、二件以上の場合は入力順に `|` で連結して半角の `(` と `)` で囲む。
3. `excludeTerms` の各語には半角の `!` を一つ付け、入力順に並べる。
4. 上記の空でない三つの部分を、`allTerms`、`anyTerms`、`excludeTerms` の順に半角空白で連結する。

各語は `SOT-IF-023` により e-Gov の演算子と空白を含まないため、アダプターは語を検索式として再解析、引用または escape しない。完成した `keyword` は HTTP query parameter の値として UTF-8 から percent-encoding し、文字列連結によって URL へ直接埋め込まない。

| `law.content.search@1` | `GET /keyword` |
|---|---|
| 生成した検索式 | `keyword` |
| `asOf` | `asof` |
| `limit` | `limit` |
| 継続トークン内の取得位置 | `offset` |

`response_format` は `json`、`order` は `+law_info.law_id`、`highlight_tag` は `mark` とする。`sentences_limit` は送信しない。送信する `asof` は、後述する実効値を使用する。

e-Gov の公式検索文法で表現できない完全一致、近接検索、語形変化、検索順位またはワイルドカードを、この mapping で近似しない。

`asOf` が `2017-04-01` より前の場合は、外部呼出し前に `unsupported_query` とする。

初回取得では、`asOf` がない場合に限り、リクエスト開始時点を `Asia/Tokyo` で暦日にした値を実効 `asof` として送る。入力に `asOf` がある場合はその値を実効 `asof` とする。継続トークンの `snapshot` は `{"asOf":"YYYY-MM-DD"}`、`sort` は `{"order":"+law_info.law_id"}`、`position` は `{"offset":<next_offset>}` とする。再開時は token の実効 `asof` と order を再利用し、どちらかを変更しない。この snapshot と決定的な order を適用できない場合は `nextToken` を発行しない。

## レスポンス

レスポンスの各 `sentences` を一件の `SourcedResource<LawContentMatch>` へ変換する。

- `ref.providerId` と `ref.key.sourceId` は `e-gov-law-api-v2` とする。
- `ref.key.resourceType` は `law` とする。
- `ref.key.resourceId` は `law_info.law_id`、`ref.key.versionId` は `revision_info.law_revision_id` とする。
- 同じ法令リビジョンに複数の一致箇所がある場合は同じ `ref` を使用し、各一致位置は `Provenance.location` と `LawContentMatch.location` で区別する。
- `Provenance.transformation` は `extracted`、`methodId` は `SOT-IF-028` とする。
- `sentences[].text` から API が挿入した `<mark>` と `</mark>` だけを除去し、それ以外を変更しない。

`total_count` は `SourcePage.totalCount`、`sentence_count` は `SourcePage.returnedCount`、`next_offset` は署名された継続トークンの外部取得位置へ対応させる。`total_count` は公式仕様が示す一致位置の総数として `totalRelation: exact` を使用する。

`SOT-IF-010` のページ不変条件を同じ parser 結果へ適用し、全 `sentences` の展開件数と `sentence_count` の一致、要求 `limit` 以下、非負の件数、`offset + sentence_count` と一致して前進する int32 範囲の `next_offset`、および残件と `null` の整合を確認する。不一致を切り捨て、再計数または独自 offset の合成で隠さず、同 SOT の分類で失敗させる。

## エラー

- `429` は `rate_limited` とし、外部情報源が示した場合だけ `retryAfter` を保持する。
- 期限超過は `source_timeout`、接続失敗、公式 OpenAPI が server 内部の失敗と定義する `500`、または一時的な `502`、`503`、`504` は `source_unavailable` とする。
- 公式スキーマの必須項目または型が確認済み fixture と一致しない場合は `source_contract_changed` とする。
- 値が契約を満たさないが公式スキーマ自体の変更と確認できない場合は `invalid_source_response` とする。
- 資源予算の超過は `SOT-ENG-016` に従う。

## 確認

空でない各入力組合せについて、生成する検索式を golden test で固定する。少なくとも `allTerms` だけ、`anyTerms` 一件、`anyTerms` 複数、正の条件と `excludeTerms` の組合せ、percent-encoding、演算子を含む入力の事前拒否、対象期間外の `unsupported_query`、および continuation の往復を確認する。

e-Gov の公式例と固定 fixture を使用し、入力語が意図しない演算子へ変わらないこと、`sentence_count` と返却 item 数が一致すること、`next_offset` が `offset + sentence_count` と一致すること、および同じ法令内の複数一致を失わないことを確認する。

## 関連

- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-IF-010: e-Gov 本文検索マッピング](10-egov-content-search-mapping.md)
- [SOT-IF-015: 情報源操作の共通契約](15-source-operation-contract.md)
- [SOT-IF-016: 情報源の継続取得](16-source-continuation-contract.md)
- [SOT-IF-017: 情報源エラーの正規化](17-source-error-normalization.md)
- [SOT-IF-023: law.content.search capability v1](23-law-content-search-capability.md)
- [SOT-ENG-016: プロバイダー資源予算](../50-engineering/16-provider-resource-budgets.md)
