# SOT-IF-009: e-Gov 法令名検索マッピング

- 状態: 廃止
- 後継: [SOT-IF-050: e-Gov 法令名検索マッピング v2](50-egov-law-search-mapping-v2.md)

## 廃止理由

公開入力を常に直接 `law_title` へ対応させる規定を、ユースケースが原文または確認済み正式名称から選択した検索語を対応させる mapping へ置き換える。

## 規定

`search_laws` は e-Gov 法令 API Version 2 の `GET /laws` を呼び出す。ツール入力を e-Gov の検索条件へ変換し、e-Gov のレスポンスを `LawSearchResult` へ変換する。

## リクエスト

| `search_laws` | `GET /laws` |
|---|---|
| `query` | `law_title` |
| `asOf` | `asof` |
| `limit` | `limit` |
| `offset` | `offset` |

`response_format` は `json`、`order` は `+law_info.law_id` とする。`asOf` がない場合は `asof` を送信しない。

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

e-Gov の公式 OpenAPI では、`count` は `limit` と `offset` の適用後に返した法令数、`total_count` は適用前の該当総数、`next_offset` は次に指定する `offset` と定義されている。現在の公開 `LawSearchResult` は `totalCount` を必須とするため、次のすべてを検証してから公開結果または共通 capability の結果を作る。

- `total_count`、`count` および `laws` が存在する。
- `total_count` と `count` は 0 以上で、`count` は要求した `limit` 以下である。
- `laws` の要素数は `count` と一致する。
- `offset + count` は integer overflow を起こさず、`total_count` 以下である。`offset` が `total_count` より大きく `count: 0` の場合だけ、`offset + count > total_count` を許容する。
- `next_offset` が非 `null` の場合は `offset + count` と一致し、現在の `offset` より大きく、0 以上 2147483647 以下かつ `total_count` 以下であり、`offset + count < total_count` である。

検証済みの非 `null` の `next_offset` は、そのまま次の取得位置とする。`next_offset` 自体が欠落し、`offset + count < total_count` の場合だけ、e-Gov が返した `count` と `total_count` から `offset + count` を導出し、それが現在の `offset` より大きく 2147483647 以下の場合に次の取得位置とする。導出値が前進しない場合または範囲を超える場合は `invalid_source_response` とする。

`next_offset: null` は公式仕様どおり末尾を示す値として扱い、`offset + count < total_count` と同時に現れた場合は合成で隠さず `invalid_source_response` とする。`next_offset` が欠落または `null` で `offset + count >= total_count` の場合は次の取得位置を作らない。

この不変条件に違反する値または公開契約に必要な項目の欠落は `invalid_source_response` とする。公式 schema の必須項目または型自体が確認済み仕様から変わった場合だけ `source_contract_changed` とする。法令配列を `count` に合わせて切り捨てず、`total_count` または次の取得位置を独自に推測しない。

## 共通 capability

e-Gov 法令 API Version 2 のアダプターが `law.search@1` を実装する場合は、同じ `GET /laws` の結果を次のように対応させる。

- 各 `SourcedResource<LawSummary>.ref.providerId` と `ref.key.sourceId` は `e-gov-law-api-v2` とする。
- `ref.key.resourceType` は `law`、`ref.key.resourceId` は `law_info.law_id`、`ref.key.versionId` は `revision_info.law_revision_id` とする。
- `provenance.url` は `SOT-IF-011` と同じ公式法令 URL、`mediaType` は `application/json`、`transformation` は `normalized`、`methodId` は `SOT-IF-009` とする。
- `count` は `SourcePage.returnedCount`、`total_count` は `SourcePage.totalCount` とし、`totalRelation` は `exact` とする。
- 上記の規則で検証または導出した次の取得位置は `SOT-IF-016` に従う継続トークンの外部取得位置へ格納し、利用者へ数値のまま公開しない。

`law.search@1` の入力は `SOT-IF-022` に従う。既存 `search_laws` の `offset` は、従来の公開 facade に限ってこの mapping の外部取得位置へ直接対応させる。

e-Gov の `law_title` は、値の先頭と末尾が `/` の場合に正規表現として解釈される。`law.search@1` は部分文字列検索であるため、正規化済み `query` の先頭と末尾がともに `/` の場合は外部呼出し前に `unsupported_query` とし、正規表現として送信、escape または近似しない。`SOT-IF-030` の公開 facade は同じ値を公開入力検証で `invalid_argument` とする。

`law.search@1` の `asOf` が `2017-04-01` より前の場合は、外部呼出し前に `unsupported_query` とする。

`law.search@1` の初回取得では、`asOf` がない場合に限り、リクエスト開始時点を `Asia/Tokyo` で暦日にした値を実効 `asof` として送る。入力に `asOf` がある場合はその値を実効 `asof` とする。継続トークンの `snapshot` は `{"asOf":"YYYY-MM-DD"}`、`sort` は `{"order":"+law_info.law_id"}`、`position` は `{"offset":<next_offset>}` とする。再開時は token の実効 `asof` と order を再利用し、どちらかを変更しない。この snapshot と決定的な order を適用できない場合は `nextToken` を発行しない。

既存 `search_laws` facade と `law.search@1` は、同じ e-Gov client、応答 parser、`LawSummary` mapping およびページ不変条件を使用するが、入力 pagination は別の境界とする。facade は公開された任意の数値 `offset` を直接 e-Gov へ送り、`asOf` がない場合は `asof` を送信しない。内部 capability は公開 `offset` を入力にせず、上記の実効 `asof` と署名済み continuation を使用する。facade の `offset` から内部 token を合成しない。

## エラー

- e-Gov に到達できない場合、公式 OpenAPI が server 内部の失敗と定義する `500`、または一時的な `502`、`503`、`504` は `source_unavailable`、期限超過は `source_timeout` とする。
- `429` は `rate_limited` とし、e-Gov が値を示した場合だけ `retryAfter` を保持する。
- 2xx レスポンスが確認済みの公式スキーマにある必須項目または型を満たさない場合は `source_contract_changed` とする。スキーマ変更と確認できない不正な値は `invalid_source_response` とする。
- 事前検証を通過した入力に対する 4xx レスポンスは `invalid_source_response` とし、情報源のレスポンス本文を公開しない。
- 取得または解析の上限超過は `SOT-ENG-016` に従う。

## 確認

部分文字列、内部にだけ `/` を含む文字列、および先頭と末尾が `/` の文字列を fixture にし、最後の case がネットワーク呼出し前に `unsupported_query` となることを確認する。公開 facade では同じ case を `invalid_argument` とし、e-Gov の正規表現検索が共通 capability または公開ツールへ暗黙に入らないことを確認する。

ページ fixture では、非 `null` の `next_offset`、末尾を示す `null`、欠落した `next_offset` からの安全な導出、残件があるのに明示された `null`、`count` と配列長の不一致、`limit` 超過、`offset + count` と一致しない次位置、非前進または int32 範囲外の次位置、および結果が残るのに `count: 0` で停止する応答を確認する。同じ fixture から公開 `nextOffset` と内部 continuation の取得位置が一致することを確認する。

## 関連

- [SOT-IF-030: MCP `search_laws`](30-mcp-search-laws.md)
- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-IF-016: 情報源の継続取得](16-source-continuation-contract.md)
- [SOT-IF-022: law.search capability v1](22-law-search-capability.md)
- [SOT-MODEL-006: LawSearchResult](../20-model/06-law-search-result.md)
- [SOT-ENG-016: プロバイダー資源予算](../50-engineering/16-provider-resource-budgets.md)
