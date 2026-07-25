# SOT-IF-008: MCP `search_law_content`

- 状態: 有効

## 規定

`search_law_content` は、法令本文の検索式を受け取り、公式情報源で確認できる `LawContentSearchResult` を返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `query` | string | はい | 前後の空白を除いて 1 文字以上 | 法令本文へ適用する e-Gov の検索式 |
| `asOf` | string | いいえ | `2017-04-01` 以降の `YYYY-MM-DD` | 指定日以前で最新のリビジョンを検索する基準日 |
| `limit` | integer | いいえ | 1 以上 100 以下、既定値 20 | 一ページに返す一致位置の上限 |
| `offset` | integer | いいえ | 0 以上、既定値 0 | 取得を開始する一致位置 |

`query` は、e-Gov が定義するワイルドカード検索または AND、OR、NOT 検索を受け付ける。ワイルドカードと AND、OR、NOT を同じ検索式で併用しない。

定義していない入力項目は受け付けない。

## 出力

`LawContentSearchResult` を返す。結果がない場合の表現は `SOT-MODEL-008` に従う。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 情報源を利用できない場合は `source_unavailable` を返す。
- 情報源のレスポンスを解釈できない場合は `invalid_source_response` を返す。

## 関連

- [SOT-SCN-004: 法令本文を検索する](../10-scenarios/04-search-law-content.md)
- [SOT-MODEL-008: LawContentSearchResult](../20-model/08-law-content-search-result.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-010: e-Gov 本文検索マッピング](10-egov-content-search-mapping.md)
