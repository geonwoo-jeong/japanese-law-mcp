# SOT-IF-002: MCP `get_law`

- 状態: 有効

## 規定

`get_law` は、法令識別子と任意の検索基準日を受け取り、公式情報源から取得した一つの `LawDocument` を返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `lawId` | string | はい | 空の値を受け付けない | 公式情報源の法令識別子 |
| `asOf` | string | いいえ | `2017-04-01` 以降の `YYYY-MM-DD` | 指定日以前で最新の本文を選ぶ基準日 |

定義していない入力項目は受け付けない。

`asOf` を省略した場合は、情報源がリクエスト処理時点で最新として返すリビジョンを取得する。

## 出力

該当する法令が存在する場合は、`LawDocument` を返す。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 該当する法令または基準時点の本文がない場合は `not_found` を返す。
- 情報源を利用できない場合は `source_unavailable` を返す。
- 情報源の応答を解釈できない場合は `invalid_source_response` を返す。

## 関連

- [SOT-SCN-002: 法令本文を取得する](../10-scenarios/02-get-law.md)
- [SOT-MODEL-002: LawDocument](../20-model/02-law-document.md)
- [SOT-IF-006: エラー契約](06-error-contract.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-011: e-Gov 法令本文取得マッピング](11-egov-law-document-mapping.md)
