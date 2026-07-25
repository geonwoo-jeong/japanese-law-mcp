# SOT-MODEL-008: LawContentSearchResult

- 状態: 有効

## 規定

`LawContentSearchResult` は、法令本文検索の一ページを、一致箇所の総数、返却した一致箇所および次の取得位置によって表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `totalCount` | integer | はい | 検索条件に該当する位置の総数 |
| `items` | `LawContentMatch[]` | はい | 現在のページに含まれる一致箇所 |
| `nextOffset` | integer | いいえ | 次のページを取得するための開始位置 |

## 制約

次のページがない場合は `nextOffset` を省略する。

一致箇所がない場合は `totalCount` を `0`、`items` を空の配列とする。

## 関連

- [SOT-MODEL-007: LawContentMatch](07-law-content-match.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
- [SOT-IF-008: MCP `search_law_content`](../40-interfaces/08-mcp-search-law-content.md)
