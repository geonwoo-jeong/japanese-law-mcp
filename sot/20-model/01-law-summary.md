# SOT-MODEL-001: LawSummary

- 状態: 有効

## 規定

`LawSummary` は、検索結果に含まれる一つの法令と選択されたリビジョンを、公式識別子、名称、日付および情報源によって表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `lawId` | string | はい | 情報源が使用する法令識別子 |
| `revisionId` | string | はい | 選択された法令リビジョンの識別子 |
| `title` | string | はい | 選択されたリビジョンの法令名 |
| `lawNumber` | string | いいえ | 法令番号 |
| `promulgationDate` | date | いいえ | 法令の公布日 |
| `revisionEffectiveDate` | date | いいえ | 選択されたリビジョンの施行日 |
| `source` | `LegalSource` | はい | 情報を取得した情報源 |

## 制約

項目を確認できない場合は値を推測せず、省略可能な項目として扱う。

`lawId`、`revisionId` および `title` は、同じ情報源上の同じ法令リビジョンを指す。

## 関連

- [SOT-MODEL-003: LegalSource](03-legal-source.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
- [SOT-IF-053: MCP `search_laws` v3](../40-interfaces/53-mcp-search-laws-v3.md)
