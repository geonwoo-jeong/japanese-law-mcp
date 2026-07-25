# SOT-IF-011: e-Gov 法令本文取得マッピング

- 状態: 有効

## 規定

`get_law` は e-Gov 法令 API Version 2 の `GET /law_data/{law_id_or_num_or_revision_id}` を XML 形式で呼び出し、選択された法令リビジョンと `Law` 要素を `LawDocument` へ変換する。

## リクエスト

- パスの `law_id_or_num_or_revision_id` には `get_law.lawId` を設定する。
- `asOf` がある場合だけ `asof` を設定する。
- `response_format` と `law_full_text_format` は `xml` とする。
- `elm`、`json_format` および添付ファイルの内容は要求しない。

## レスポンス

`law_info` と `revision_info` は `SOT-IF-009` と同じフィールド対応で `LawSummary` に変換する。

法令本文の `Law` 要素を UTF-8 XML として内容を変更せずにシリアライズし、`LawDocument.content` に設定する。`LawDocument.format` は `xml` とする。入力に `asOf` がある場合だけ `LawDocument.asOf` に設定する。

`Citation.url` は、`revisionId` から先頭の `{lawId}_` を除いたリビジョン部分を使い、次の形式で生成する。

```text
https://laws.e-gov.go.jp/law/{lawId}/{revisionPart}
```

`revisionId` が `{lawId}_` で始まらない場合は、確認可能な URL を推測せず `invalid_source_response` とする。

## エラー

- 対象法令または指定日以前のリビジョンが存在しない場合は `not_found` とする。
- その他の情報源エラーは `SOT-IF-009` と同じ規則で変換する。

## 関連

- [SOT-IF-002: MCP `get_law`](02-mcp-get-law.md)
- [SOT-IF-009: e-Gov 法令名検索マッピング](09-egov-law-search-mapping.md)
- [SOT-MODEL-002: LawDocument](../20-model/02-law-document.md)
