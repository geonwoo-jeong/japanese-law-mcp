# SOT-IF-011: e-Gov 法令本文取得マッピング

- 状態: 有効

## 規定

e-Gov 法令 API Version 2 の法令本文取得は `GET /law_data/{law_id_or_num_or_revision_id}` を XML 形式で呼び出し、選択された法令リビジョンと `Law` 要素を内部の `LawDocumentRepresentation` へ変換する。既存 `get_law` は、その XML 結果だけを `LawDocument` へ投影する。

## リクエスト

- 既存 `get_law` facade では、パスの `law_id_or_num_or_revision_id` に `get_law.lawId` を設定する。
- `law.document.read@1` では、`resource.providerId` と `resource.key.sourceId` がともに `e-gov-law-api-v2`、`resource.key.resourceType` が `law` であることを確認する。
- `resource.key.versionId` がある場合はパスにその値を設定し、ない場合は `resource.key.resourceId` を設定する。
- `asOf` がある場合だけ `asof` を設定する。`resource.key.versionId` と `asOf` を同時に送信しない。
- `law.document.read@1` の `asOf` が `2017-04-01` より前の場合は、外部呼出し前に `unsupported_query` とする。
- `response_format` と `law_full_text_format` は `xml` とする。
- `elm`、`json_format` および添付ファイルの内容は要求しない。

## レスポンス

`law_info` と `revision_info` は `SOT-IF-054` と同じフィールド対応で
`LawSummary` に変換する。

法令本文の `Law` 要素を UTF-8 XML として内容を変更せずにシリアライズし、`LawDocumentRepresentation.content` に設定する。`LawDocumentRepresentation.format` は `xml` とする。入力に `asOf` がある場合だけ `LawDocumentRepresentation.asOf` に設定する。

`Citation.url` は、`revisionId` から先頭の `{lawId}_` を除いたリビジョン部分を使い、次の形式で生成する。

```text
https://laws.e-gov.go.jp/law/{lawId}/{revisionPart}
```

`revisionId` が `{lawId}_` で始まらない場合は、確認可能な URL を推測せず `invalid_source_response` とする。

## 共通 capability

`law.document.read@1` の結果は、次の `SourcedResource<LawDocumentRepresentation>` とする。

- `ref.providerId` と `ref.key.sourceId` は `e-gov-law-api-v2`、`ref.key.resourceType` は `law` とする。
- `ref.key.resourceId` は返された `law_info.law_id`、`ref.key.versionId` は返された `revision_info.law_revision_id` とする。
- `provenance.url` は `Citation.url`、`mediaType` は `application/xml`、`transformation` は `extracted`、`methodId` は `SOT-IF-011` とする。
- 入力に `resource.key.versionId` がある場合は、返された `revision_info.law_revision_id` と完全一致しなければ `invalid_source_response` とする。
- 入力の `resource.key.resourceId` と返された `law_info.law_id` が一致しない場合は `invalid_source_response` とする。

既存 `get_law` facade は、この内部結果の `data` が `format: xml` かつ `SOT-MODEL-002` の制約を満たすことを確認し、同じ `law`、`asOf`、`content` および `citation` を持つ `LawDocument` として公開する。

## エラー

- 対象法令または指定日以前のリビジョンが存在しない場合は `not_found` とする。
- HTTP status と transport error の変換は `SOT-IF-054` と同じ規則を使用する。
- 一件の 2xx 応答で XML を解析できない場合、`law_info`、`revision_info`
  若しくは `Law` 要素が欠落している場合、共通モデルの必須項目を作れない場合、
  または入力と応答の法令 ID 若しくはリビジョン ID が一致しない場合は
  `invalid_source_response` とする。
- DTD、entity declaration、外部 entity または解析中の外部参照は受理せず、
  `unsafe_source_content` とする。
- 現在の公式 OpenAPI、法令 XML 資料または公式例を provider contract 検証で
  確認し、記録済みの必須構造または型が変更された場合だけ
  `source_contract_changed` とする。個別応答の malformed XML、欠落または
  型不一致を同 code へ読み替えない。
- 応答または展開後の byte 上限超過、構造上限および解析時間上限は
  `SOT-ENG-016` に従う。

## 確認

正常な公式 XML 例、対象なし、`law_info`、`revision_info` または `Law` の欠落、
malformed XML、DTD、entity declaration、法令 ID とリビジョン ID の不一致、
byte・構造・解析時間の各上限、および外部本文を公開 error に含めない case を
固定 fixture で確認する。

個別 2xx 応答の必須項目の欠落、`null` 相当の空要素、型不一致または
malformed XML は `invalid_source_response`、DTD と entity は
`unsafe_source_content` とする。
保存した公式 schema または公式 XML 構造を意図的に変更する provider contract
fixture だけで `source_contract_changed` を確認する。外部ネットワークを使う
確認は `SOT-ENG-013` に従う任意の定期確認へ分離し、fixture test の代わりに
しない。

## 関連

- [SOT-IF-031: MCP `get_law`](31-mcp-get-law.md)
- [SOT-IF-054: e-Gov 法令名検索マッピング v3](54-egov-law-search-mapping-v3.md)
- [SOT-IF-024: law.document.read capability v1](24-law-document-read-capability.md)
- [SOT-MODEL-002: LawDocument](../20-model/02-law-document.md)
- [SOT-MODEL-017: LawDocumentRepresentation](../20-model/17-law-document-representation.md)
- [SOT-ENG-013: プロバイダー契約の検証](../50-engineering/13-provider-contract-verification.md)
- [SOT-ENG-016: プロバイダー資源予算](../50-engineering/16-provider-resource-budgets.md)
- [法令の条文構造と法令 XML](https://laws.e-gov.go.jp/docs/law-data-basic/8ebd8bc-law-structure-and-xml/)
