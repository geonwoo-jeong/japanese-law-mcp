# SOT-IF-023: `law.content.search` capability v1

- 状態: 有効

## 規定

`law.content.search@1` は、法令本文に対する provider 非依存の構造化検索条件を受け取り、型付きの `LawContentMatch` 一覧と継続取得情報を返す、内部の共通 capability 契約とする。

この capability は e-Gov の検索式そのものを契約へ取り込まない。既存の MCP ツール `search_law_content` が受け付ける e-Gov 固有の AND、OR、NOT またはワイルドカード式は、互換 facade の責務に留め、`law.content.search@1` の入力値にはしない。

## 能力識別子

| 項目 | 値 |
|---|---|
| `ProviderCapability.id` | `law.content.search` |
| `ProviderCapability.majorVersion` | `1` |
| `ProviderCapability.level` | `core` |
| `ProviderCapability.stability` | `stable` |

## 型付き入力

`LawContentSearchRequestV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `allTerms` | `string[]` | いいえ | すべてを含む語 |
| `anyTerms` | `string[]` | いいえ | いずれかを含む語 |
| `excludeTerms` | `string[]` | いいえ | 含んではならない語 |
| `asOf` | date | いいえ | 指定日以前で最新のリビジョンを対象にする基準日 |
| `limit` | integer | いいえ | 今回返す上限。既定値 `20`、最大値 `100` |
| `continuationToken` | string | いいえ | 同じ条件の続きを取得する不透明な継続トークン |

## 入力制約

- `allTerms` または `anyTerms` のいずれか一つ以上で、正の検索条件を持たなければならない。
- 各語または語句は、先頭と末尾に連続する U+0020 を除いた値を正規化済み入力とし、その値が 1 文字以上でなければならない。
- 各正規化済み語は UTF-8 の 128 byte を超えてはならない。全配列の正規化済み語を合わせた UTF-8 の byte 数は 2048 byte を超えてはならない。超過は `invalid_argument` とする。
- `allTerms`、`anyTerms` および `excludeTerms` の各配列は `8` 件以下、三者の合計は `16` 件以下とする。超過は `invalid_argument` とする。
- `null` の配列、`null` の語、空文字および空白のみの語は `invalid_argument` とする。
- 各語に ASCII の空白または制御文字、`|`、`!`、`(`、`)`、`*` または `?` を含めない。これらを含む値は provider の検索演算子へ変換せず、`invalid_argument` とする。
- 正規化後の同じ語を同じ配列または複数の配列へ重複して指定しない。正の条件と `excludeTerms` の両方に同じ語がある場合も `invalid_argument` とする。
- 省略した配列と空の配列は、継続条件の正規化では同じ空配列として扱う。配列内の順序は変更せず、継続取得では初回と同じ順序を使用する。
- `asOf` を指定する場合は実在する暦日の `YYYY-MM-DD` でなければならない。プロバイダー固有の収録開始日は共通入力制約にしない。
- `limit` を省略した場合は `20` とし、`1` 未満または `100` 超は `invalid_argument` とする。
- `continuationToken` を指定した場合は、初回と同じ正規化済み検索条件、`asOf` および `limit` を使用しなければならない。
- 条件 fingerprint の JSON object は、正規化した `allTerms`、`anyTerms`、`excludeTerms`、`asOf` または `null`、既定値適用後の `limit` の五つの key を持つ。省略した配列は空の配列とし、key の省略を許可しない。

## 共通 capability が扱わないもの

- e-Gov の `AND`、`OR`、`NOT`、ワイルドカードまたは括弧の生文字列
- provider 固有の `highlight_tag`、offset 名、ページ番号または検索順位尺度
- 生のクエリ文字列を、そのまま別プロバイダーへ要求する契約

`AND`、`OR` および `NOT` という文字列は通常の検索語として扱い、演算子として解釈しない。演算子の意味づけは、構造化された各フィールドによってだけ表す。連続する文字列の完全一致またはワイルドカードは、複数プロバイダーで同じ意味を確認できる後継 capability まで扱わない。

## 型付き出力

検索結果は `LawContentSearchPageV1` とし、次の構造を返す。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `items` | `SourcedResource<LawContentMatch>[]` | はい | 現在のページに含まれる一致箇所 |
| `page` | `SourcePage` | はい | 件数と継続取得情報 |

`LawContentMatch.location` と `text` は、情報源が確認できる位置と範囲だけを返し、条項番号や前後文脈を推測して補わない。

各 item の `ref.key.resourceType` は `law` とし、`ref.key.resourceId` は `data.law.lawId`、`ref.key.versionId` は `data.law.revisionId`、`ref.key.sourceId` は `data.law.source.id` と一致させる。同じ法令リビジョンの複数の一致箇所は同じ `ref` を使用する。

この capability の item identity は、`ref.providerId`、`ref.key` および `data.location` の組とする。同じ組の重複だけを同一 item とみなし、`ref` だけによる重複排除を禁止する。`provenance.location` は `data.location` と同じ情報源上の位置を指す。

## 欠落と空結果

- 一致箇所がない場合は成功した空の結果とし、`items` は空の配列、`page.returnedCount` は `0` とする。
- 本 capability は検索であるため、結果なしを `not_found` にしない。
- 情報源が返さない位置、法令番号、リビジョン日付または引用 URL を推測して作らない。

## 継続取得

- `page.nextToken` は次の結果がある場合だけ返す。
- 継続トークンは `SOT-IF-016` に従う。
- 継続トークンの有効期限は発行から 15 分以内とする。
- `SourcePage.totalCount` は、情報源が総件数を返す場合だけ設定する。
- 現在の primary route は単一の e-Gov Version 2 であるため、内部継続位置は route 内で保持できるが、公開 `offset` 契約とは分離する。

## 到達し得る失敗

### 公開能力の入力・結果として扱う失敗

- `invalid_argument`: 条件が空、語の空値、件数超過、byte 数超過、日付不正、継続条件不一致

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

既存の `search_law_content` は、e-Gov Version 2 の検索式を公開入力とする provider 固有 facade とする。

- `search_law_content.query` は `law.content.search@1` の入力項目ではない。
- e-Gov 固有検索式から `allTerms`、`anyTerms` および `excludeTerms` への lossless な変換規則を別の SOT で採用するまでは、既存 facade を自動的にこの capability へ再解釈しない。
- したがって、既存の `search_law_content` が受け付ける e-Gov 固有 DSL は、`law.content.search@1` の共通契約へ漏れ出さない。
- 将来、構造化入力を公開する新しい MCP ツールを採用した場合は、そのツールが `law.content.search@1` の正規 facade となる。

## 既定プロバイダー

`primary` の既定プロバイダーは、`SOT-IF-004` が定義する `providerId: e-gov-law-api-v2` とする。

現在の製品範囲では、既存の公開本文検索 facade は e-Gov 専用であり、集約検索 route は登録しない。

## 確認

少なくとも次を契約テストで確認する。

- `ProviderDescriptor` が `law.content.search@1` を宣言し、対応する型付きポートを実装すること
- 空条件、`null`、空語、配列件数超過、byte 数超過および形式不正日付を拒否すること
- 共通の date として有効だが provider の収録範囲外である `asOf` を、空結果または `invalid_argument` ではなく `unsupported_query` とすること
- e-Gov 固有 DSL が共通 capability の入力型に現れないこと
- `LawContentMatch` の `location`、`text`、`citation` および `LawSummary` を維持すること
- 同じ法令リビジョンに複数の一致位置を持つ fixture で、`ref` が同じでも全 item を保持し、`ref + location` の同一項目だけを重複として判定すること
- 空の検索結果を成功として返すこと
- 継続トークンの往復、条件不一致、改変および期限切れを検出すること
- 4096 byte 超のトークン、再起動前の鍵で発行したトークンおよび `adapterContractVersion` または設定 scope が異なるトークンを拒否すること
- e-Gov Version 2 fixture に対して、既存 facade と独立に構造化条件から結果を得られること
- 情報源エラーが成功結果へ化けず、秘密情報と外部本文を露出しないこと

## 関連

- [SOT-SCN-004: 法令本文を検索する](../10-scenarios/04-search-law-content.md)
- [SOT-MODEL-007: LawContentMatch](../20-model/07-law-content-match.md)
- [SOT-MODEL-014: SourcePage](../20-model/14-source-page.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-ARCH-017: 採用可能な能力群](../30-architecture/17-approved-capability-families.md)
- [SOT-ARCH-013: 情報源の選択と組合せ](../30-architecture/13-source-composition.md)
- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-IF-033: MCP `search_law_content`](33-mcp-search-law-content.md)
- [SOT-IF-010: e-Gov 本文検索マッピング](10-egov-content-search-mapping.md)
- [SOT-IF-015: 情報源操作の共通契約](15-source-operation-contract.md)
- [SOT-IF-016: 情報源の継続取得](16-source-continuation-contract.md)
- [SOT-IF-017: 情報源エラーの正規化](17-source-error-normalization.md)
- [SOT-IF-028: e-Gov 構造化本文検索マッピング](28-egov-structured-content-search-mapping.md)
