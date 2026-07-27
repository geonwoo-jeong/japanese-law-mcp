# SOT-IF-022: `law.search` capability v1

- 状態: 有効

## 規定

`law.search@1` は、法令名または略称に関する検索条件を受け取り、型付きの `LawSummary` 一覧と継続取得情報を返す、内部の共通 capability 契約とする。

この capability は、既存の MCP ツール `search_laws` と同じ公開契約ではない。内部の capability 契約として `continuationToken` と `SourcedResource<LawSummary>` を使用し、公開の `offset` と `LawSearchResult` は既存 facade の責務として維持する。

## 能力識別子

| 項目 | 値 |
|---|---|
| `ProviderCapability.id` | `law.search` |
| `ProviderCapability.majorVersion` | `1` |
| `ProviderCapability.level` | `core` |
| `ProviderCapability.stability` | `stable` |

## 型付き入力

`LawSearchRequestV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `query` | string | はい | 法令名または略称に含まれる検索語 |
| `asOf` | date | いいえ | 指定日以前で最新のリビジョンを選ぶ基準日 |
| `limit` | integer | いいえ | 今回返す上限。既定値 `20`、最大値 `100` |
| `continuationToken` | string | いいえ | 同じ条件の続きを取得する不透明な継続トークン |

## 入力制約

- `query` は先頭と末尾に連続する U+0020 を除いた値を正規化済み入力とし、その値が 1 文字以上でなければならない。
- 正規化済み `query` は UTF-8 の 512 byte を超えてはならず、U+0000 から U+001F および U+007F の ASCII 制御文字を含めない。違反は `invalid_argument` とする。
- `query` に `null` を使用しない。欠落または `null` は `invalid_argument` とする。
- `asOf` を指定する場合は実在する暦日の `YYYY-MM-DD` でなければならない。プロバイダー固有の収録開始日は共通入力制約にしない。
- `limit` を省略した場合は `20` とし、`1` 未満または `100` 超は `invalid_argument` とする。
- `continuationToken` を省略した初回取得では `query` と任意の `asOf` を使用する。
- `continuationToken` を指定した場合は、初回と同じ `query`、`asOf` および `limit` を使用しなければならない。条件不一致、期限切れまたは改変は `invalid_argument` とする。
- 条件 fingerprint の JSON object は、正規化済み `query`、`asOf` または `null`、既定値適用後の `limit` の三つの key を持つ。key の省略を許可しない。

## 型付き出力

検索結果は `LawSearchPageV1` とし、次の構造を返す。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `items` | `SourcedResource<LawSummary>[]` | はい | 現在のページに含まれる法令 |
| `page` | `SourcePage` | はい | 件数と継続取得情報 |

各 `SourcedResource<LawSummary>` は `SOT-IF-015` に従い、`ref`、`provenance` および `data` を必須とする。

各 item の `ref.key.resourceType` は `law` とし、`ref.key.resourceId` は `data.lawId`、`ref.key.versionId` は `data.revisionId`、`ref.key.sourceId` は `data.source.id` と一致させる。

## 欠落と空結果

- 該当する法令がない場合は成功した空の結果とし、`items` は空の配列、`page.returnedCount` は `0` とする。
- 正確な識別子による取得ではないため、検索結果なしを `not_found` にしない。
- 情報源に存在しない日付、名称または識別情報を補完しない。

## 継続取得

- `page.nextToken` は、次のページがある場合だけ返す。
- 継続トークンは `SOT-IF-016` に従い、利用者が内部値を解釈しない不透明な値とする。
- 継続トークンの有効期限は発行から 15 分以内とする。
- `SourcePage.totalCount` と `totalRelation` は、情報源が返す場合だけ使用する。
- 現在の製品範囲では primary route が単一の e-Gov 法令 API Version 2 であるため、内部の継続位置は e-Gov の検索位置から生成できる。

## 到達し得る失敗

### 公開能力の入力・結果として扱う失敗

- `invalid_argument`: 空文字、`null`、日付形式不正、上限超過、継続条件不一致

### 情報源エラーとして到達し得る失敗

- `unsupported_capability`
- `unsupported_query`
- `configuration_required`
- `source_auth_failed`
- `rate_limited`
- `source_timeout`
- `source_unavailable`
- `source_busy`
- `source_contract_changed`
- `invalid_source_response`
- `source_response_too_large`
- `source_processing_limit`
- `unsafe_source_content`

この capability を MCP ツールへ公開する場合は、上記の情報源エラーを `SOT-IF-017` と公開エラー契約の対応表に従って変換する。

## 既存 MCP ツールとの対応

既存の `search_laws` facade とこの capability は、次の意味を共有する。

| MCP `search_laws` | `law.search@1` |
|---|---|
| `query` | `query` |
| `asOf` | `asOf` |
| `limit` | `limit` |
| `offset` | e-Gov 固有 facade だけの取得位置。共通入力へ変換しない |

- facade は e-Gov adapter の同じ応答 parser と `LawSummary` mapping を使い、内部 capability の `ref`、`provenance` および `page.nextToken` を公開しない。
- facade の `nextOffset` は、現在の単一 e-Gov route に限り `SOT-IF-050` の同じページ不変条件で検証する。
- `offset` はこの capability の公開入力ではなく、既存互換 facade の責務として維持する。任意の公開 `offset` から内部 `continuationToken` を合成して capability を呼び出さない。
- `asOf` を省略した facade は e-Gov に `asof` を送信しない。内部 capability は継続 snapshot のため実効 `asOf` を固定する。この差を lossless な共通入力として扱わない。

## 既定プロバイダー

`primary` の既定プロバイダーは、`SOT-IF-004` が定義する `providerId: e-gov-law-api-v2` とする。

`aggregate` route は、この capability だけでは登録しない。複数情報源を横断する検索結果モデル、順序、部分失敗および情報源別継続位置を定義する後継 SOT を採用するまで使用しない。

## 確認

少なくとも次を契約テストで確認する。

- `ProviderDescriptor` が `law.search@1` を宣言し、対応する型付きポートを実装すること
- `query` の `null`、空文字、空白のみ、形式不正日付および 512 byte 超を拒否すること
- 共通の date として有効だが provider の収録範囲外である `asOf` を、空結果または `invalid_argument` ではなく `unsupported_query` とすること
- 空の検索結果を成功として返し、`not_found` へ変換しないこと
- `SourceResourceRef`、`Provenance` および `LawSummary` の必須項目を保持すること
- 継続トークンの往復、条件不一致、改変および期限切れを検出すること
- 4096 byte 超のトークン、再起動前の鍵で発行したトークンおよび `adapterContractVersion` または設定 scope が異なるトークンを拒否すること
- e-Gov Version 2 fixture に対して、既存 `search_laws` facade の `LawSearchResult` と互換の結果を得られること
- 情報源エラーが成功結果へ化けず、秘密情報と外部本文を露出しないこと

## 関連

- [SOT-SCN-001: 法令名から法令を検索する](../10-scenarios/01-search-laws.md)
- [SOT-MODEL-001: LawSummary](../20-model/01-law-summary.md)
- [SOT-MODEL-014: SourcePage](../20-model/14-source-page.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-ARCH-017: 採用可能な能力群](../30-architecture/17-approved-capability-families.md)
- [SOT-ARCH-013: 情報源の選択と組合せ](../30-architecture/13-source-composition.md)
- [SOT-IF-049: MCP `search_laws` v2](49-mcp-search-laws-v2.md)
- [SOT-IF-050: e-Gov 法令名検索マッピング v2](50-egov-law-search-mapping-v2.md)
- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-IF-015: 情報源操作の共通契約](15-source-operation-contract.md)
- [SOT-IF-016: 情報源の継続取得](16-source-continuation-contract.md)
- [SOT-IF-017: 情報源エラーの正規化](17-source-error-normalization.md)
