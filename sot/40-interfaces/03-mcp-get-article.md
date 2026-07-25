# SOT-IF-003: MCP `get_article`

- 状態: 有効

## 規定

`get_article` は、法令識別子、本則または原始附則の区分、条番号、任意指定の項番号、および任意指定の検索基準日を受け取り、該当部分の XML と `Citation` を返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `lawId` | string | はい | 空の値を受け付けない | 公式情報源の法令識別子 |
| `provision` | string | いいえ | `main` または `supplementary`、既定値 `main` | 本則または原始附則 |
| `article` | string | はい | `1` または `38_3_2` のような正の整数を `_` で連結した形式 | 法令 XML の `Article@Num` |
| `paragraph` | integer | いいえ | 1 以上 | 法令 XML の `Paragraph@Num` |
| `asOf` | string | いいえ | `2017-04-01` 以降の `YYYY-MM-DD` | 指定日以前で最新の本文を選ぶ基準日 |

定義していない入力項目は受け付けない。

`paragraph` を省略した場合は `Article` 全体を返し、指定した場合はその `Article` の直下にある該当 `Paragraph` だけを返す。

## 出力

| 名前 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `format` | string | はい | `xml` |
| `content` | string | はい | 該当する `Article` または `Paragraph` をシリアライズした UTF-8 XML |
| `citation` | `Citation` | はい | 法令と条文位置を確認するための情報 |

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 該当する法令または条文がない場合は `not_found` を返す。
- 情報源を利用できない場合は `source_unavailable` を返す。
- 条文位置を一意に決定できない場合は `ambiguous_location` を返す。

## 関連

- [SOT-SCN-003: 条文を取得する](../10-scenarios/03-get-article.md)
- [SOT-MODEL-004: Citation](../20-model/04-citation.md)
- [SOT-IF-006: エラー契約](06-error-contract.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-012: e-Gov 条文取得マッピング](12-egov-article-mapping.md)
