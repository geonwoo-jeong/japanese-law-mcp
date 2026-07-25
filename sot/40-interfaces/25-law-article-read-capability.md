# SOT-IF-025: `law.article.read` capability v1

- 状態: 有効

## 規定

`law.article.read@1` は、法令識別子と条文位置を provider 非依存の型付き入力で受け取り、該当する法令部分を返す、内部の共通 capability 契約とする。

## 能力識別子

| 項目 | 値 |
|---|---|
| `ProviderCapability.id` | `law.article.read` |
| `ProviderCapability.majorVersion` | `1` |
| `ProviderCapability.level` | `core` |
| `ProviderCapability.stability` | `stable` |

## 型付き入力

`LawArticleReadRequestV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `resource` | `SourceResourceRef` | はい | 対象法令と、検索または既存 facade が選択したプロバイダー |
| `asOf` | date | いいえ | 指定日以前で最新の本文を選ぶ基準日 |
| `location` | `LawArticleLocation` | はい | 表現方法から独立した本則または原始附則の条と任意の項 |

## 入力制約

- `resource` は `SOT-MODEL-016` に従い、`resource.key.resourceType` は `law` とする。
- `resource.key.resourceId` は 1 文字以上、UTF-8 で 256 byte 以下とし、先頭または末尾の U+0020、および U+0000 から U+001F と U+007F の ASCII 制御文字を許可しない。不透明な識別子を trimming または正規化しない。
- `resource.key.versionId` を指定する場合は 1 文字以上、UTF-8 で 512 byte 以下とし、`resourceId` と同じ文字制約を適用する。
- `resource.key.versionId` と `asOf` は同時に指定してはならない。
- `asOf` を指定する場合は実在する暦日の `YYYY-MM-DD` でなければならない。プロバイダー固有の収録開始日は共通入力制約にしない。
- `location` は `SOT-MODEL-018` に従う。`location.articleNumber` は UTF-8 の 64 byte を超えてはならない。
- 必須項目の欠落、`null`、空値、形式不正または上限超過は `invalid_argument` とする。

## 取得意味

- `resource.key.versionId` がある場合は、そのリビジョンの法令本文から位置を解決する。
- `resource.key.versionId` がなく `asOf` がある場合は、その日以前で最新の法令本文から位置を解決する。
- 両方ない場合は、情報源が最新として返す法令本文から位置を解決する。
- `location.paragraphNumber` がない場合は指定した条全体を返し、ある場合はその条に属する指定項だけを返す。
- 公式資料の別の条、項、表、引用または改正法令附則に含まれる同じ見かけの番号を、指定した位置の候補へ混在させない。
- 結果の `ref.providerId`、`ref.key.sourceId`、`ref.key.resourceType` および `ref.key.resourceId` は入力と同じ値とし、`ref.key.versionId` は実際に選択された `LawArticleFragment.law.revisionId` とする。
- 結果の `LawArticleFragment.location` は、正規化済みの入力 `location` と同じ値とする。

## 型付き出力

結果は `SourcedResource<LawArticleFragment>` とする。`LawArticleFragment` の構造と制約は `SOT-MODEL-015` に従う。各 provider mapping SOT は、条または項を一意に選択できる公式構造と、返す `xml`、`html` または `text` の表現を定義する。e-Gov 法令 API Version 2 は `SOT-IF-012` に従い `xml` を返す。

## 欠落と曖昧性

- 候補が存在しない場合は `not_found` とする。
- 候補が複数あり一意に決定できない場合は `ambiguous_location` とする。
- 存在しない条文位置、項番号または URL を推測して補わない。

## 継続取得

この capability は一つの資源を返すため、`continuationToken` と `SourcePage` を使用しない。

## 到達し得る失敗

### 公開能力の入力・結果として扱う失敗

- `invalid_argument`: `resource` または必須項目の欠落、不正な provider・source・resource type、空値、形式不正、byte 数超過、`versionId` と `asOf` の同時指定
- `not_found`: 対象法令または対象位置が存在しない
- `ambiguous_location`: 条文位置を一意に解決できない

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

## 既存 MCP ツールとの対応

既存の `get_article` facade は、次の対応でこの capability と接続できる。

| MCP `get_article` | `law.article.read@1` |
|---|---|
| `lawId` | `resource.key.resourceId` |
| `asOf` | `asOf` |
| `provision` または既定値 `main` | `location.provision` |
| `article` | `location.articleNumber` |
| `paragraph` | `location.paragraphNumber` |
| primary route | `resource.providerId` |
| primary provider の source | `resource.key.sourceId` |
| 固定値 `law` | `resource.key.resourceType` |
| なし | `resource.key.versionId` は省略する |

- facade は primary route から `SourceResourceRef` を組み立て、`SourcedResource<LawArticleFragment>.data` から `format`、`content` および `citation` を公開する。
- 内部結果の `law`、`ref` および `provenance` は既存の公開出力へ追加しない。
- `versionId` による正確取得は検索結果の `SourceResourceRef` を使う内部 capability の契約であり、既存の公開入力へ自動追加しない。

## 既定プロバイダー

`primary` の既定プロバイダーは、`SOT-IF-004` が定義する `providerId: e-gov-law-api-v2` とする。

正確な法令資源の取得は `SOT-ARCH-013` に従い、別の情報源へ暗黙に fallback しない。

## 確認

少なくとも次を契約テストで確認する。

- `ProviderDescriptor` が `law.article.read@1` を宣言し、対応する型付きポートを実装すること
- `resource`、`location`、`asOf` および `versionId` の境界条件を検証すること
- 共通の date として有効だが provider の収録範囲外である `asOf` を、`not_found` または `invalid_argument` ではなく `unsupported_query` とすること
- `not_found` と `ambiguous_location` を区別すること
- XML、HTML および text の test provider で、別の条、項、表、引用および改正法令附則を誤って選択しないこと
- `LawArticleFragment` の `law`、`location`、`content` および `citation` を保持すること
- e-Gov Version 2 fixture に対して、既存 `get_article` facade と互換の `content` と `citation` を得られること
- provider が宣言した各 format の unsafe content、過大応答および契約変更を検出すること

## 関連

- [SOT-SCN-003: 条文を取得する](../10-scenarios/03-get-article.md)
- [SOT-MODEL-001: LawSummary](../20-model/01-law-summary.md)
- [SOT-MODEL-004: Citation](../20-model/04-citation.md)
- [SOT-MODEL-015: LawArticleFragment](../20-model/15-law-article-fragment.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-MODEL-018: LawArticleLocation](../20-model/18-law-article-location.md)
- [SOT-ARCH-013: 情報源の選択と組合せ](../30-architecture/13-source-composition.md)
- [SOT-IF-032: MCP `get_article`](32-mcp-get-article.md)
- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-IF-012: e-Gov 条文取得マッピング](12-egov-article-mapping.md)
- [SOT-IF-015: 情報源操作の共通契約](15-source-operation-contract.md)
- [SOT-IF-017: 情報源エラーの正規化](17-source-error-normalization.md)
