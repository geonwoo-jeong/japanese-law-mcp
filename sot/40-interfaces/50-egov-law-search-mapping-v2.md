# SOT-IF-050: e-Gov 法令名検索マッピング v2

- 状態: 有効

## 規定

e-Gov 法令 API Version 2 による法令名検索は `GET /laws` を呼び出し、法令名検索ユースケースが選択した一つの検証済み検索語を e-Gov の検索条件へ変換し、レスポンスを `LawSearchResult` または `law.search@1` の情報源ページへ変換する。

## リクエスト

| Japanese Law MCP | `GET /laws` |
|---|---|
| ユースケースが選択した `query` | `law_title` |
| `asOf` | `asof` |
| `limit` | `limit` |
| 公開 facade の `offset` | `offset` |

`response_format` は `json`、`order` は `+law_info.law_id` とする。公開 facade の `asOf` がない場合は `asof` を送信しない。

e-Gov アダプターは、受け取った `query` を追加で解析、略称展開、誤記補正または token 分割せず、そのまま `law_title` へ対応させる。前処理の原検索、確認検索および候補検索は、それぞれ独立した通常のリクエストとして同じ mapping を使用する。

## レスポンス

| e-Gov | Japanese Law MCP |
|---|---|
| `law_info.law_id` | `LawSummary.lawId` |
| `revision_info.law_revision_id` | `LawSummary.revisionId` |
| `revision_info.law_title` | `LawSummary.title` |
| `law_info.law_num` | `LawSummary.lawNumber` |
| `law_info.promulgation_date` | `LawSummary.promulgationDate` |
| `revision_info.amendment_enforcement_date` | `LawSummary.revisionEffectiveDate` |
| `total_count` | `LawSearchResult.totalCount` |
| `laws` | `LawSearchResult.items` |

`LawSummary.source` は `SOT-IF-004` に定義した情報源とする。

## ページ不変条件

`count` は `limit` と `offset` の適用後に返した法令数、`total_count` は適用前の該当総数、`next_offset` は次に指定する `offset` として扱い、次のすべてを検証する。

- `total_count`、`count` および `laws` が存在する。
- `total_count` と `count` は 0 以上で、`count` は要求した `limit` 以下である。
- `laws` の要素数は `count` と一致する。
- `offset + count` は integer overflow を起こさず、`total_count` 以下である。`offset` が `total_count` より大きく `count: 0` の場合だけ、この大小関係の例外を許容する。
- `next_offset` が非 `null` の場合は `offset + count` と一致し、現在の `offset` より大きく、0 以上 2147483647 以下かつ `total_count` 以下であり、`offset + count < total_count` である。

検証済みの非 `null` の `next_offset` は、そのまま次の取得位置とする。`next_offset` 自体が欠落し、`offset + count < total_count` の場合だけ `offset + count` を導出し、それが現在の `offset` より大きく 2147483647 以下の場合に次の取得位置とする。

`next_offset: null` は末尾として扱い、`offset + count < total_count` と同時に現れた場合は合成で隠さず `invalid_source_response` とする。結果が残るのに `count: 0`、配列長との不一致、`limit` 超過、非前進、範囲外または矛盾する次位置を受理しない。

## 共通 capability

e-Gov 法令 API Version 2 のアダプターが `law.search@1` を実装する場合は、同じ `GET /laws` の結果を次のように対応させる。

- 各 `SourcedResource<LawSummary>.ref.providerId` と `ref.key.sourceId` は `e-gov-law-api-v2` とする。
- `ref.key.resourceType` は `law`、`ref.key.resourceId` は `law_info.law_id`、`ref.key.versionId` は `revision_info.law_revision_id` とする。
- `provenance.url` は `SOT-IF-011` と同じ公式法令 URL、`mediaType` は `application/json`、`transformation` は `normalized`、`methodId` は `SOT-IF-050` とする。
- `count` は `SourcePage.returnedCount`、`total_count` は `SourcePage.totalCount` とし、`totalRelation` は `exact` とする。
- 検証または導出した次の取得位置は `SOT-IF-016` に従う継続トークンの外部取得位置へ格納し、利用者へ数値のまま公開しない。

`law.search@1` の入力は `SOT-IF-022` に従う。既存 `search_laws` の `offset` は、公開 facade に限ってこの mapping の外部取得位置へ直接対応させる。

`law_title` は、値の先頭と末尾が `/` の場合に正規表現として解釈される。`law.search@1` は該当値を外部呼出し前に `unsupported_query` とし、正規表現として送信、escape または近似しない。`SOT-IF-049` の公開 facade は同じ値を公開入力検証で `invalid_argument` とする。

`law.search@1` の `asOf` が `2017-04-01` より前の場合は、外部呼出し前に `unsupported_query` とする。初回取得で `asOf` がない場合に限り、リクエスト開始時点を `Asia/Tokyo` で暦日にした値を実効 `asof` として送る。継続トークンの `snapshot` は `{"asOf":"YYYY-MM-DD"}`、`sort` は `{"order":"+law_info.law_id"}`、`position` は `{"offset":<next_offset>}` とし、再開時に変更しない。

公開 facade と `law.search@1` は、同じ e-Gov client、応答 parser、`LawSummary` mapping およびページ不変条件を使用する。facade は任意の数値 `offset` を直接 e-Gov へ送り、`asOf` がない場合は `asof` を送信しない。内部 capability は公開 `offset` を入力にせず、実効 `asof` と署名済み continuation を使用する。facade の `offset` から内部 token を合成しない。

## エラー

- 到達不能、e-Gov が server 内部の失敗と定義する `500`、または一時的な `502`、`503`、`504` は `source_unavailable`、期限超過は `source_timeout` とする。
- `429` は `rate_limited` とし、e-Gov が値を示した場合だけ `retryAfter` を保持する。
- 2xx レスポンスが公式スキーマの必須項目または型を満たさない場合は `source_contract_changed` とする。スキーマ変更と確認できない不正な値は `invalid_source_response` とする。
- 事前検証を通過した入力に対する 4xx レスポンスは `invalid_source_response` とし、情報源のレスポンス本文を公開しない。
- 取得または解析の上限超過は `SOT-ENG-016` に従う。

## 確認

原文と解決済みの正式名称が、追加の変換なしで `law_title` へ一回だけ encode されることを確認する。部分文字列、内部にだけ `/` を含む文字列、および先頭と末尾が `/` の文字列を fixture にし、最後の case が外部呼出し前に拒否されることを確認する。

ページ fixture では、非 `null` の `next_offset`、末尾を示す `null`、欠落時の安全な導出、残件があるのに明示された `null`、`count` と配列長の不一致、`limit` 超過、矛盾する次位置、非前進または int32 範囲外の次位置、および結果が残るのに `count: 0` で停止する応答を確認する。

## 関連

- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-IF-011: e-Gov 法令本文マッピング](11-egov-law-document-mapping.md)
- [SOT-IF-016: 情報源の継続取得](16-source-continuation-contract.md)
- [SOT-IF-022: law.search capability v1](22-law-search-capability.md)
- [SOT-IF-049: MCP `search_laws` v2](49-mcp-search-laws-v2.md)
- [SOT-MODEL-006: LawSearchResult](../20-model/06-law-search-result.md)
- [SOT-ENG-016: プロバイダー資源予算](../50-engineering/16-provider-resource-budgets.md)
