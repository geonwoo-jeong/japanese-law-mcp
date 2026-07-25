# SOT-IF-010: e-Gov 本文検索マッピング

- 状態: 有効

## 規定

`search_law_content` は e-Gov 法令 API Version 2 の `GET /keyword` を呼び出す。ツール入力を e-Gov の本文検索条件へ変換し、e-Gov のレスポンスを `LawContentSearchResult` へ変換する。

## リクエスト

| `search_law_content` | `GET /keyword` |
|---|---|
| `query` | `keyword` |
| `asOf` | `asof` |
| `limit` | `limit` |
| `offset` | `offset` |

`response_format` は `json`、`order` は `+law_info.law_id`、`highlight_tag` は `mark` とする。`asOf` がない場合は `asof` を送信しない。

## レスポンス

`items` 内の各法令を `LawSummary` に変換し、その法令の `sentences` を一件ずつ次のように `LawContentMatch` へ展開する。

| e-Gov | Japanese Law MCP |
|---|---|
| 法令の `law_info` と `revision_info` | `LawContentMatch.law` |
| `sentences[].position` | `LawContentMatch.location` |
| `sentences[].text` | `LawContentMatch.text` |
| `total_count` | `LawContentSearchResult.totalCount` |
| `next_offset` | `LawContentSearchResult.nextOffset` |

`sentences[].text` から、検索 API が挿入した `<mark>` と `</mark>` だけを除去する。それ以外の文字列は変更しない。

`LawContentMatch.citation` は、`LawSummary` の法令 ID とリビジョン ID、`position` および `SOT-IF-011` の URL 規則から生成する。

## エラー

エラーの変換は `SOT-IF-009` と同じ規則を使用する。

## 関連

- [SOT-IF-008: MCP `search_law_content`](08-mcp-search-law-content.md)
- [SOT-IF-009: e-Gov 法令名検索マッピング](09-egov-law-search-mapping.md)
- [SOT-IF-011: e-Gov 法令本文取得マッピング](11-egov-law-document-mapping.md)
- [SOT-MODEL-008: LawContentSearchResult](../20-model/08-law-content-search-result.md)
