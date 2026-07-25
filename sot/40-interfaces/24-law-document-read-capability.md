# SOT-IF-024: `law.document.read` capability v1

- 状態: 有効

## 規定

`law.document.read@1` は、一つの法令本文を provider 非依存の取得条件で読み出し、型付きの `LawDocumentRepresentation` と出典を返す、内部の共通 capability 契約とする。

## 能力識別子

| 項目 | 値 |
|---|---|
| `ProviderCapability.id` | `law.document.read` |
| `ProviderCapability.majorVersion` | `1` |
| `ProviderCapability.level` | `core` |
| `ProviderCapability.stability` | `stable` |

## 型付き入力

`LawDocumentReadRequestV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `resource` | `SourceResourceRef` | はい | 取得する法令と、検索または既存 facade が選択したプロバイダー |
| `asOf` | date | いいえ | 指定日以前で最新の本文を選ぶ基準日 |

## 入力制約

- `resource` は `SOT-MODEL-016` に従い、`resource.key.resourceType` は `law` とする。
- `resource.key.resourceId` は 1 文字以上、UTF-8 で 256 byte 以下とし、先頭または末尾の U+0020、および U+0000 から U+001F と U+007F の ASCII 制御文字を許可しない。不透明な識別子を trimming または正規化しない。
- `resource.key.versionId` を指定する場合は 1 文字以上、UTF-8 で 512 byte 以下とし、`resourceId` と同じ文字制約を適用する。
- `resource.key.versionId` と `asOf` は同時に指定してはならない。両方ある場合は `invalid_argument` とする。
- `asOf` を指定する場合は実在する暦日の `YYYY-MM-DD` でなければならない。プロバイダー固有の収録開始日は共通入力制約にしない。
- `resource` の欠落、`null`、空の識別子、provider と source の不一致、resource type の不一致または上限超過は `invalid_argument` とする。

## 取得意味

- `resource.key.versionId` がある場合は、そのリビジョンを正確に取得する。
- `resource.key.versionId` がなく `asOf` がある場合は、その日以前で最新のリビジョンを取得する。
- `resource.key.versionId` と `asOf` がない場合は、情報源がリクエスト処理時点で最新として返すリビジョンを取得する。
- `resource.key.versionId` を指定した場合に、返された `LawDocumentRepresentation.law.revisionId` が一致しない結果を受け入れない。不一致は `invalid_source_response` とする。
- 結果の `ref.providerId`、`ref.key.sourceId`、`ref.key.resourceType` および `ref.key.resourceId` は入力と同じ値とし、`ref.key.versionId` は実際に選択された `LawDocumentRepresentation.law.revisionId` とする。

## 型付き出力

結果は `SourcedResource<LawDocumentRepresentation>` とする。

`LawDocumentRepresentation` は `SOT-MODEL-017` に従い、次を必須とする。

- `law`
- `format`
- `content`
- `citation`

`format` と `content` は `SOT-MODEL-017` の表現契約に従う。各 provider mapping SOT は、公式に提供される表現から `xml`、`html` または `text` のどれを返すか、選択単位、安全化または抽出方法、および `Provenance.transformation` を定義する。e-Gov 法令 API Version 2 は `SOT-IF-011` に従い `xml` を返す。

## 欠落

- 指定した法令、リビジョンまたは基準日以前の本文が存在しない場合は `not_found` とする。
- 取得できない日付、リビジョン ID、URL または本文を推測して補わない。

## 継続取得

この capability は一つの資源を返すため、`continuationToken` と `SourcePage` を使用しない。

## 到達し得る失敗

### 公開能力の入力・結果として扱う失敗

- `invalid_argument`: `resource` 欠落、不正な provider・source・resource type、空値、byte 数超過、`versionId` と `asOf` の同時指定、日付形式不正
- `not_found`: 対象法令または対象リビジョンが存在しない

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

既存の `get_law` facade は、次の対応でこの capability と接続できる。

| MCP `get_law` | `law.document.read@1` |
|---|---|
| `lawId` | `resource.key.resourceId` |
| `asOf` | `asOf` |
| primary route | `resource.providerId` |
| primary provider の source | `resource.key.sourceId` |
| 固定値 `law` | `resource.key.resourceType` |
| なし | `resource.key.versionId` は省略する |

- facade は primary route から `SourceResourceRef` を組み立て、`SourcedResource<LawDocumentRepresentation>.data` が `format: xml` かつ `SOT-MODEL-002` の制約を満たすことを確認して、公開の `LawDocument` へ投影する。
- `ref` と `provenance` は内部で保持し、既存公開結果へ追加しない。
- `versionId` による正確取得は検索結果の `SourceResourceRef` を使う内部契約であり、既存の `get_law` 公開入力へ自動追加しない。
- `html` または `text` の結果を既存の `LawDocument` へ投影せず、公開するには別の公開ツール契約を先に採用する。

## 既定プロバイダー

`primary` の既定プロバイダーは、`SOT-IF-004` が定義する `providerId: e-gov-law-api-v2` とする。

正確な法令資源の取得は `SOT-ARCH-013` に従い、別の情報源へ暗黙に fallback しない。

## 確認

少なくとも次を契約テストで確認する。

- `ProviderDescriptor` が `law.document.read@1` を宣言し、対応する型付きポートを実装すること
- `resource` の `null`、provider・source・resource type の不一致、空値、byte 数超過、`versionId` と `asOf` の同時指定、および日付不正を拒否すること
- 共通の date として有効だが provider の収録範囲外である `asOf` を、`not_found` または `invalid_argument` ではなく `unsupported_query` とすること
- `not_found` と情報源エラーを区別すること
- `versionId` 指定時に、返却された `LawSummary.revisionId` との一致を検証すること
- `SourcedResource` の `ref`、`provenance` および `LawDocumentRepresentation` を保持すること
- e-Gov Version 2 fixture に対して、既存 `get_law` facade と互換の `LawDocument` を得られること
- 既存 `get_law` facade が `html` と `text` を XML として公開しないこと
- provider が宣言した各 format の unsafe content、過大応答および契約変更を検出すること

## 関連

- [SOT-SCN-002: 法令本文を取得する](../10-scenarios/02-get-law.md)
- [SOT-MODEL-002: LawDocument](../20-model/02-law-document.md)
- [SOT-MODEL-017: LawDocumentRepresentation](../20-model/17-law-document-representation.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-ARCH-013: 情報源の選択と組合せ](../30-architecture/13-source-composition.md)
- [SOT-IF-031: MCP `get_law`](31-mcp-get-law.md)
- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-IF-011: e-Gov 法令本文取得マッピング](11-egov-law-document-mapping.md)
- [SOT-IF-015: 情報源操作の共通契約](15-source-operation-contract.md)
- [SOT-IF-017: 情報源エラーの正規化](17-source-error-normalization.md)
