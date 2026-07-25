# SOT-MODEL-002: LawDocument

- 状態: 有効

## 規定

`LawDocument` は、一つの法令リビジョンの本文を、検索基準日、本文形式、本文内容および出典によって表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `law` | `LawSummary` | はい | 対象法令と選択されたリビジョン |
| `asOf` | date | いいえ | 利用者が指定した検索基準日 |
| `format` | string | はい | `xml` |
| `content` | string | はい | UTF-8 の法令 XML |
| `citation` | `Citation` | はい | 原文を確認するための出典 |

## 制約

`asOf` はリビジョンの施行日ではない。指定日の以前で最新となるリビジョンは、`law.revisionId` と `law.revisionEffectiveDate` で示す。

`content` は情報源から取得した法令 XML の `Law` 要素を内容の変更なくシリアライズする。読みやすさのための整形結果はこの項目へ格納しない。

## 関連

- [SOT-MODEL-001: LawSummary](01-law-summary.md)
- [SOT-MODEL-004: Citation](04-citation.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
