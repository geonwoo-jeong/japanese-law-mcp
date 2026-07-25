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

`law_info` と `revision_info` は `SOT-IF-009` と同じフィールド対応で `LawSummary` に変換する。

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
- その他の情報源エラーは `SOT-IF-009` と同じ規則で変換する。

## 関連

- [SOT-IF-031: MCP `get_law`](31-mcp-get-law.md)
- [SOT-IF-009: e-Gov 法令名検索マッピング](09-egov-law-search-mapping.md)
- [SOT-IF-024: law.document.read capability v1](24-law-document-read-capability.md)
- [SOT-MODEL-002: LawDocument](../20-model/02-law-document.md)
- [SOT-MODEL-017: LawDocumentRepresentation](../20-model/17-law-document-representation.md)
