# SOT-MODEL-014: SourcePage

- 状態: 有効

## 規定

`SourcePage` は、能力別の一覧または検索結果について、今回返した件数と同じ条件で次を取得するための情報を表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `returnedCount` | integer | はい | 今回返した項目数 |
| `nextToken` | string | いいえ | 同じ条件の次の取得位置を表す不透明な継続トークン |
| `totalCount` | integer | いいえ | 情報源が示した該当件数 |
| `totalRelation` | string | 条件付き | `totalCount` が示す関係で、`exact` または `lower_bound` |

## 制約

情報源が総数を返さない場合は `totalCount` と `totalRelation` を省略し、推測した総数を返さない。

次の結果がない場合は `nextToken` を省略する。該当結果がない場合は `returnedCount` を `0` とする。

継続トークンは、利用者が内部の offset、ページ番号、分割番号または情報源固有の next key を解釈する契約にしない。

## 関連

- [SOT-IF-016: 情報源の継続取得](../40-interfaces/16-source-continuation-contract.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
