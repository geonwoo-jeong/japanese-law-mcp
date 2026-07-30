# SOT-MODEL-006: LawSearchResult

- 状態: 有効

## 規定

`LawSearchResult` は、法令名検索の一ページを、該当総数、返却した法令および次の取得位置によって表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `totalCount` | integer | はい | 検索条件に該当する法令の総数 |
| `items` | `LawSummary[]` | はい | 現在のページに含まれる法令 |
| `nextOffset` | integer | いいえ | 次のページを取得するための開始位置 |

## 制約

`totalCount` と `nextOffset` は公式情報源の検索結果から取得する。次のページがない場合は `nextOffset` を省略する。

該当する法令がない場合は `totalCount` を `0`、`items` を空の配列とする。

## 関連

- [SOT-MODEL-001: LawSummary](01-law-summary.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
- [SOT-IF-053: MCP `search_laws` v3](../40-interfaces/53-mcp-search-laws-v3.md)
