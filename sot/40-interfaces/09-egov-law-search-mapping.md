# SOT-IF-009: e-Gov 法令名検索マッピング

- 状態: 有効

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

`next_offset` が返された場合は `LawSearchResult.nextOffset` に使用する。返されない場合は、`offset + count` が `total_count` より小さいときだけ、その値を `nextOffset` とする。

## エラー

- e-Gov に到達できない場合、タイムアウトした場合、または一時的な 5xx レスポンスは `source_unavailable` とする。
- 2xx レスポンスが必要な項目または型を満たさない場合は `invalid_source_response` とする。
- 事前検証を通過した入力に対する 4xx レスポンスは `invalid_source_response` とし、情報源のレスポンス本文を公開しない。

## 関連

- [SOT-IF-001: MCP `search_laws`](01-mcp-search-laws.md)
- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-MODEL-006: LawSearchResult](../20-model/06-law-search-result.md)
