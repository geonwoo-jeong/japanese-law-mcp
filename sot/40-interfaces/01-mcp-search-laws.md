# SOT-IF-001: MCP `search_laws`

- 状態: 有効

## 規定

`search_laws` は、法令名または略称の一部を受け取り、公式情報源で確認できる `LawSearchResult` を返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `query` | string | はい | 前後の空白を除いて 1 文字以上 | 法令名または略称に含まれる文字列 |
| `asOf` | string | いいえ | `2017-04-01` 以降の `YYYY-MM-DD` | 指定日以前で最新のリビジョンを検索する基準日 |
| `limit` | integer | いいえ | 1 以上 100 以下、既定値 20 | 一ページに返す法令数 |
| `offset` | integer | いいえ | 0 以上、既定値 0 | 取得を開始する位置 |

定義していない入力項目は受け付けない。

## 出力

`LawSearchResult` を返す。結果がない場合の表現は `SOT-MODEL-006` に従う。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 情報源を利用できない場合は `source_unavailable` を返す。
- 情報源の応答を解釈できない場合は `invalid_source_response` を返す。

## 関連

- [SOT-SCN-001: 法令名から法令を検索する](../10-scenarios/01-search-laws.md)
- [SOT-MODEL-006: LawSearchResult](../20-model/06-law-search-result.md)
- [SOT-IF-006: エラー契約](06-error-contract.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-009: e-Gov 法令名検索マッピング](09-egov-law-search-mapping.md)
