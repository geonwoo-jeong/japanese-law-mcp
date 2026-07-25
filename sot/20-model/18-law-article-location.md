# SOT-MODEL-018: LawArticleLocation

- 状態: 有効

## 規定

`LawArticleLocation` は、日本の法令本文に公式に付された本則または原始附則の条番号と項番号を、XML、HTML または text の表現方法から独立して指定する。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `provision` | string | はい | `main` または `supplementary` |
| `articleNumber` | string | はい | 条番号と「の」による枝番号を `_` で連結した正規形 |
| `paragraphNumber` | integer | いいえ | 条の中で公式に付された項番号 |

## 制約

`provision: main` は本則、`provision: supplementary` は選択した法令に当初から付された原始附則を表す。改正法令ごとの附則を指定する意味には使用しない。

`articleNumber` は、`1` または `38_3_2` のように、一以上の正の十進整数を `_` で連結する。先頭の零、空の segment、全角数字、漢数字および区切り文字の別表記を許可しない。この正規形の `38_3_2` は、公式表示上の「第三十八条の三の二」を表す。

`paragraphNumber` は一以上の整数とする。第一項の番号が公式表示で省略される表現でも、法令構造上の第一項を指定する値は `1` とする。配列上の位置、DOM の子要素番号または検索結果の順位として解釈しない。

provider mapping は、公式資料が示す条と項の指定をこの正規形へ決定的に対応させる。公式資料から本則と原始附則、条番号または項番号を一意に確認できない provider は、推測した位置で `law.article.read@1` へ binding しない。

## 同一性

一つの `SourceResourceRef` 内での条文 fragment の同一性は、`ref` と正規化済み `LawArticleLocation` の組とする。`ref` だけで異なる条または項を重複排除しない。

## 確認

XML の属性、HTML の見出しおよび text の公式表示を使う test provider について、同じ公式条番号が同じ `LawArticleLocation` となることを確認する。枝番号、第一項、原始附則、改正法令附則との混同および曖昧な位置を拒否することを契約試験で確認する。

## 関連

- [SOT-MODEL-015: LawArticleFragment](15-law-article-fragment.md)
- [SOT-MODEL-016: SourceResourceRef](16-source-resource-ref.md)
- [SOT-IF-025: law.article.read capability v1](../40-interfaces/25-law-article-read-capability.md)
