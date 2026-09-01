# SOT-MODEL-033: LawVersionComparison

- 状態: 有効

## 規定

`LawVersionComparison` は、一つの法令について確定した二つの版と、本則及び
原始附則に属する条の追加、削除又は変更を表す共通モデルとする。

## 比較結果

`LawVersionComparison` は次の項目を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `lawId` | string | はい | 比較対象の法令 ID |
| `scope` | enum | はい | 固定値 `main_and_original_supplementary_articles` |
| `before` | `LawVersionSnapshot` | はい | 比較前として扱う確定版 |
| `after` | `LawVersionSnapshot` | はい | 比較後として扱う確定版 |
| `beforeArticleCount` | integer | はい | 比較前版で対象にした条の総数 |
| `afterArticleCount` | integer | はい | 比較後版で対象にした条の総数 |
| `addedCount` | integer | はい | 追加された条数 |
| `removedCount` | integer | はい | 削除された条数 |
| `modifiedCount` | integer | はい | 変更された条数 |
| `unchangedCount` | integer | はい | 両版で同一だった条数 |
| `totalCount` | integer | はい | `items` に含む変更の正確な件数 |
| `items` | `LawVersionChange[]` | はい | 条単位の変更一覧 |

`LawVersionSnapshot` は次の項目を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `law` | `LawSummary` | はい | 実際に選択された法令版 |
| `asOf` | date | いいえ | `asOf` で選択した場合の利用者指定日 |
| `citation` | `Citation` | はい | 版全体の公式原文を確認する出典 |

## 変更項目

`LawVersionChange` は次の項目を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `changeKind` | enum | はい | `added`、`removed` 又は `modified` |
| `changeReasons` | enum[] | 条件付き | `modified` と判定した理由 |
| `before` | `LawVersionArticle` | 条件付き | 比較前版の条 |
| `after` | `LawVersionArticle` | 条件付き | 比較後版の条 |

`LawVersionArticle` は次の項目を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `location` | `LawVersionArticleLocation` | はい | 条の同一性と構造上の位置 |
| `articleTitle` | string | いいえ | 公式の条名 |
| `articleCaption` | string | いいえ | 公式の条見出し |
| `text` | string | はい | 比較用に正規化した条の可視文字列 |
| `citation` | `Citation` | はい | 当該版の条を確認する出典 |

`LawVersionArticleLocation` は次の項目を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `provision` | enum | はい | `main` 又は `supplementary` |
| `articleNumber` | string | はい | 単一条番号又は比較対象としてまとめられた条範囲の正規形 |
| `partNumber` | string | いいえ | 編番号 |
| `chapterNumber` | string | いいえ | 章番号 |
| `sectionNumber` | string | いいえ | 節番号 |
| `subsectionNumber` | string | いいえ | 款番号 |
| `divisionNumber` | string | いいえ | 目番号 |

`provision: supplementary` は原始附則だけを表し、改正法令ごとの附則を表さない。

単一条の `articleNumber` は `SOT-MODEL-018` と同じ正規形とする。公式資料が一つの
条要素で連続する削除条をまとめている場合に限り、開始条番号と終了条番号を一つの
`:` で連結した `38:84` 又は `38_2:38_4` の形を使用する。両端はそれぞれ
`SOT-MODEL-018` の条番号の正規形とし、開始条番号は終了条番号より前でなければ
ならない。単一条番号と条範囲はいずれも UTF-8 で 64 byte 以下とする。

条範囲は公式資料の一つの条要素に対応する一つの比較単位として保持し、範囲内の
個別条へ展開しない。この条範囲の表現は版間比較専用であり、個別条を指定する
`LawArticleLocation` では `:` を許可しない。

## 同一性と変更理由

一つの版における条の同一性は、`provision` と `articleNumber` の組とする。編、章、
節、款又は目の番号は位置を説明する値であり、条の同一性には含めない。このため、
同じ条が別の章等へ移動しても削除と追加には分けず、変更として扱う。
`provision` 又は `articleNumber` が変わった場合は別の同一性として削除と追加に
分け、同じ条の移動であると推測しない。条範囲の両端又はまとめ方が変わった場合も、
範囲内の個別条との対応を推測しない。

`changeReasons` は次の値だけを使用し、`location`、`text`、`structure` の順に
重複なく並べる。

| 値 | 意味 |
|---|---|
| `location` | 同じ条の編、章、節、款又は目の位置が変わった |
| `text` | 比較用の可視文字列が変わった |
| `structure` | 文字列以外の条内部の要素又は属性構造が変わった |

## 制約

- `lawId` は空文字にしない。
- `before.law.lawId`、`before.citation.lawId`、`after.law.lawId` 及び
  `after.citation.lawId` は `lawId` と一致しなければならない。
- 比較前後の `LawSummary.source` は同じ情報源とし、各 `Citation.source` も
  対応する `LawSummary.source` と一致させる。
- 各 snapshot の `citation.revisionId` は `law.revisionId` と一致させる。
- すべての件数は 0 以上とし、次を満たす。
  - `beforeArticleCount = removedCount + modifiedCount + unchangedCount`
  - `afterArticleCount = addedCount + modifiedCount + unchangedCount`
  - `totalCount = addedCount + removedCount + modifiedCount = items の件数`
- 一つの版で同じ `provision` と `articleNumber` を重複させない。
- `added` は `after` だけ、`removed` は `before` だけを必須とし、
  `changeReasons` を設定しない。
- `modified` は同じ条同一性を持つ `before` と `after`、及び一件以上の
  `changeReasons` を必須とする。
- 条が可視文字列を持たない場合も `text` は省略せず空文字として保持する。
  条の有無は `before` と `after` の有無で区別する。
- 省略可能な構造番号、条名又は条見出しを別の位置又は推測値から補わない。

## 並び順

`items` は、比較後版に存在する `added` と `modified` を比較後版の文書順で並べ、
その後に `removed` を比較前版の文書順で並べる。内部 map の反復順、文字列の
辞書順又は provider の検索順位を公開順にしない。

## 確認

前後版と出典の整合、件数の等式、同一性の重複拒否、空文字の条、同版比較、
単一条と条範囲の共存、条範囲の一単位保持、不正、同一又は逆順の範囲の拒否、
各 `changeKind` と `changeReasons` の条件、JSON 表現、決定的な並び順及び入力の
不変性を単体テストで確認する。

## 関連

- [SOT-MODEL-001: LawSummary](01-law-summary.md)
- [SOT-MODEL-004: Citation](04-citation.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
- [SOT-MODEL-018: LawArticleLocation](18-law-article-location.md)
- [SOT-PROD-005: 加工情報の区別](../00-product/05-derived-information.md)
- [SOT-IF-058: `law.version.compare` capability v1](../40-interfaces/58-law-version-compare-capability.md)
