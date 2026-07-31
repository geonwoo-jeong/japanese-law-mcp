# SOT-IF-054: e-Gov 法令名検索マッピング v3

- 状態: 有効

## 規定

e-Gov 法令 API Version 2 による法令名検索は `GET /laws` を呼び出し、
法令名検索ユースケースが選択した一つの検証済み検索語を e-Gov の検索条件へ
変換し、レスポンスを `LawSearchResult` または `law.search@1` の
情報源ページへ変換する。

本規定は `SOT-IF-050` を置き換える。個別の 2xx 応答が契約を満たさない場合と、
公式一次資料で確認した契約変更を、異なる情報源エラーとして扱う。

## 公式仕様との関係

本規定は、2026年7月30日に確認した
[法令 API Version 2 OpenAPI](https://laws.e-gov.go.jp/api/2/swagger-ui/lawapi-v2.yaml)
の `laws_response`、`law_info`、`revision_info`、公式 JSON 例および各項目の
説明を外部仕様の基準とする。

同 OpenAPI の `laws_response.required` は `count` だけを列挙し、
`law_info` と `revision_info` に required 項目を定義していない。
しかし、その指定だけでは `LawSearchResult` と `LawSummary` の必須項目を
作れない。そのため adapter は OpenAPI の required 配列を機械的な受理条件に
せず、共通モデルを安全に作るために必要な後述の意味上の必須項目を
provider contract とする。

公開 facade と `law.search@1` は一つの provider 固有 parser を共有し、
入口ごとに JSON shape、`null` または error 分類を実装し直さない。

## リクエスト

| Japanese Law MCP | `GET /laws` |
|---|---|
| ユースケースが選択した `query` | `law_title` |
| `asOf` | `asof` |
| `limit` | `limit` |
| 公開 facade の `offset` | `offset` |

`response_format` は `json`、`order` は `+law_info.law_id` とする。
公開 facade の `asOf` がない場合は `asof` を送信しない。

e-Gov アダプターは、受け取った `query` を追加で解析、略称展開、誤記補正
または token 分割せず、そのまま `law_title` へ対応させる。前処理の原検索、
確認検索および候補検索は、それぞれ独立した通常のリクエストとして同じ
mapping を使用する。

## 受理する JSON 構造

adapter は `response_format=json` を明示して要求する。成功応答は media type が
`application/json` である場合だけ、一つの top-level JSON object として解析する。

top-level では次を検証する。

| 位置 | 必須 | 受理する値 |
|---|---:|---|
| `total_count` | はい | 0 以上 9223372036854775807 以下の整数 |
| `count` | はい | 0 以上、要求した `limit` 以下の整数 |
| `next_offset` | いいえ | `null` または 0 以上 2147483647 以下の整数 |
| `laws` | はい | `null` ではない JSON array |

`laws` の各要素は object とし、`law_info` と `revision_info` を持つ。
両項目は `null` ではない object とする。共通モデルを識別する次の項目は、
`null` ではない空でない string とする。

- `law_info.law_id`
- `revision_info.law_revision_id`
- `revision_info.law_title`

`LawSummary` の省略可能項目へ対応する `law_info.law_num`、
`law_info.promulgation_date` および
`revision_info.amendment_enforcement_date` は、欠落または `null` の場合に
値を推測せず共通モデルから省略する。存在する `law_num` は空でない string、
二つの日付は実在する `YYYY-MM-DD` 形式の暦日でなければならない。

公式 schema に追加された未知の項目と、共通モデルへ対応させない
`current_revision_info` その他の既知項目は、上記の必須項目、既知項目の型または
`SOT-ENG-016` の資源予算を変えない限り無視する。無視した値を共通モデルの
拡張 map、provenance または公開 JSON へ移さない。

単一の law であっても object へ縮約した `laws` は受理しない。空結果は
`total_count: 0`、`count: 0` および空の `laws` を持つ成功として受理する。

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

`count` は `limit` と `offset` の適用後に返した法令数、`total_count` は
適用前の該当総数、`next_offset` は次に指定する `offset` として扱い、
次のすべてを検証する。

- `total_count`、`count` および `laws` が存在する。
- `total_count` と `count` は 0 以上で、`count` は要求した `limit` 以下である。
- `laws` の要素数は `count` と一致する。
- `offset + count` は integer overflow を起こさず、`total_count` 以下である。
  `offset` が `total_count` より大きく `count: 0` の場合だけ、
  この大小関係の例外を許容する。
- `next_offset` が非 `null` の場合は `offset + count` と一致し、
  現在の `offset` より大きく、0 以上 2147483647 以下かつ
  `total_count` 以下であり、`offset + count < total_count` である。

検証済みの非 `null` の `next_offset` は、そのまま次の取得位置とする。
`next_offset` 自体が欠落し、`offset + count < total_count` の場合だけ
`offset + count` を導出し、それが現在の `offset` より大きく
2147483647 以下の場合に次の取得位置とする。

`next_offset: null` は末尾として扱い、
`offset + count < total_count` と同時に現れた場合は合成で隠さず
`invalid_source_response` とする。結果が残るのに `count: 0`、配列長との
不一致、`limit` 超過、非前進、範囲外または矛盾する次位置を受理しない。

## 共通 capability

e-Gov 法令 API Version 2 のアダプターが `law.search@1` を実装する場合は、
同じ `GET /laws` の結果を次のように対応させる。

- 各 `SourcedResource<LawSummary>.ref.providerId` と `ref.key.sourceId` は
  `e-gov-law-api-v2` とする。
- `ref.key.resourceType` は `law`、`ref.key.resourceId` は
  `law_info.law_id`、`ref.key.versionId` は
  `revision_info.law_revision_id` とする。
- `provenance.url` は `SOT-IF-011` と同じ公式法令 URL、
  `mediaType` は `application/json`、`transformation` は `normalized`、
  `methodId` は `SOT-IF-054` とする。
- `count` は `SourcePage.returnedCount`、`total_count` は
  `SourcePage.totalCount` とし、`totalRelation` は `exact` とする。
- 検証または導出した次の取得位置は `SOT-IF-016` に従う継続トークンの
  外部取得位置へ格納し、利用者へ数値のまま公開しない。

`law.search@1` の入力は `SOT-IF-022` に従う。既存 `search_laws` の `offset` は、
公開 facade に限ってこの mapping の外部取得位置へ直接対応させる。

`law_title` は、値の先頭と末尾が `/` の場合に正規表現として解釈される。
`law.search@1` は該当値を外部呼出し前に `unsupported_query` とし、
正規表現として送信、escape または近似しない。`SOT-IF-053` の公開 facade は
同じ値を公開入力検証で `invalid_argument` とする。

`law.search@1` の `asOf` が `2017-04-01` より前の場合は、外部呼出し前に
`unsupported_query` とする。初回取得で `asOf` がない場合に限り、
リクエスト開始時点を `Asia/Tokyo` で暦日にした値を実効 `asof` として送る。
継続トークンの `snapshot` は `{"asOf":"YYYY-MM-DD"}`、
`sort` は `{"order":"+law_info.law_id"}`、
`position` は `{"offset":<next_offset>}` とし、再開時に変更しない。

公開 facade と `law.search@1` は、同じ e-Gov client、応答 parser、
`LawSummary` mapping およびページ不変条件を使用する。facade は任意の数値
`offset` を直接 e-Gov へ送り、`asOf` がない場合は `asof` を送信しない。
内部 capability は公開 `offset` を入力にせず、実効 `asof` と署名済み
continuation を使用する。facade の `offset` から内部 token を合成しない。

## エラー

- 到達不能、e-Gov が server 内部の失敗と定義する `500`、または一時的な
  `502`、`503`、`504` は `source_unavailable`、期限超過は
  `source_timeout` とする。
- `429` は `rate_limited` とし、e-Gov が値を示した場合だけ
  `retryAfter` を保持する。
- 一件の 2xx レスポンスが必須項目、型またはページ不変条件を満たさない場合は
  `invalid_source_response` とする。現在の公式 OpenAPI または公式例を
  provider contract 検証で確認し、記録済みの必須構造または型が変更された場合
  だけ `source_contract_changed` とし、個別応答の不正を同 code へ読み替えない。
- `response_format=json` に対する JSON 以外の成功時 media type、JSON として
  解析できない body、top-level の error object または trailing JSON は
  `invalid_source_response` とする。
- 事前検証を通過した入力に対する 4xx レスポンスは
  `invalid_source_response` とし、情報源のレスポンス本文を公開しない。
- 応答または展開後の byte 上限超過は `source_response_too_large`、
  構造上の危険または処理時間上限は `SOT-ENG-016` に従い
  `unsafe_source_content` または `source_processing_limit` とする。

外部応答本文、検索語、URL query、内部 decoder error および無視した項目の値を
公開 error details に含めない。

## 確認

少なくとも次の固定 test ID を provider fixture と facade/capability contract test で
確認する。

- `egov-laws-runtime-response-classification`: 個別 runtime 応答の構造違反を
  `invalid_source_response` にする
- `egov-laws-contract-change-separation`: 保存した公式契約の意図的な変更だけを
  `source_contract_changed` にする
- `egov-laws-facade-capability-parser-identity`: 同じ raw fixture を両入口で同じ
  parser 判定、item identity、順序および mapping にする
- `egov-laws-page-invariants`: count、array 長、offset、next offset、limit および
  total count の全境界を固定する

原文と解決済みの正式名称が、追加の変換なしで `law_title` へ一回だけ
encode されることを確認する。部分文字列、内部にだけ `/` を含む文字列、
および先頭と末尾が `/` の文字列を fixture にし、最後の case が
外部呼出し前に拒否されることを確認する。

ページ fixture では、非 `null` の `next_offset`、末尾を示す `null`、
欠落時の安全な導出、残件があるのに明示された `null`、`count` と配列長の
不一致、`limit` 超過、矛盾する次位置、非前進または int32 範囲外の次位置、
および結果が残るのに `count: 0` で停止する応答を確認する。

正常な空結果、単一 law、複数 law、省略可能三項目の欠落と `null`、
未知の追加項目、および無視する `current_revision_info` を fixture にする。
意味上の各必須項目と container について欠落、`null` および型不一致を別々に
確認し、いずれも `invalid_source_response` とする。省略可能三項目の欠落と
`null` は同じ共通モデルとなり、存在する空の `law_num` または不正な日付は
`invalid_source_response` となることを確認する。

`total_count` は 9223372036854775807 とその値を一つ超える値、`count` は
要求した `limit` とその値を一つ超える値、`next_offset` は 2147483647 と
その値を一つ超える値を fixture にする。
単一 object へ縮約した `laws`、2xx の非 JSON media type、malformed JSON、
top-level error object、trailing JSON および壊れた圧縮 body は
`invalid_source_response` とする。`responseBytes` と `decompressedBytes` を
一 byte 超える fixture は `source_response_too_large` とし、途中まで解析した
item を返さない。

保存した公式 schema の必須構造または型を意図的に変更する
provider contract fixture だけで `source_contract_changed` を確認する。
同じ runtime fixture を facade と capability の両入口へ渡し、同じ parser 判定、
同じ item 順および同じ `LawSummary` mapping になることを確認する。

## 関連

- [SOT-IF-050: e-Gov 法令名検索マッピング v2](50-egov-law-search-mapping-v2.md)
- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-IF-011: e-Gov 法令本文マッピング](11-egov-law-document-mapping.md)
- [SOT-IF-016: 情報源の継続取得](16-source-continuation-contract.md)
- [SOT-IF-022: law.search capability v1](22-law-search-capability.md)
- [SOT-IF-053: MCP `search_laws` v3](53-mcp-search-laws-v3.md)
- [SOT-MODEL-006: LawSearchResult](../20-model/06-law-search-result.md)
- [SOT-ENG-016: プロバイダー資源予算](../50-engineering/16-provider-resource-budgets.md)
- [e-Gov 法令 API Version 2 OpenAPI](https://laws.e-gov.go.jp/api/2/swagger-ui/lawapi-v2.yaml)
