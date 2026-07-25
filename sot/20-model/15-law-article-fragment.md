# SOT-MODEL-015: LawArticleFragment

- 状態: 有効

## 規定

`LawArticleFragment` は、一つの法令リビジョンから指定した条または項を、対象法令、文字列表現および出典によって表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `law` | `LawSummary` | はい | 対象法令と選択されたリビジョン |
| `location` | `LawArticleLocation` | はい | 選択した条または項の表現非依存の位置 |
| `format` | string | はい | `xml`、`html` または `text` |
| `content` | string | はい | 該当する条または項を選択した形式で表す UTF-8 の内容 |
| `citation` | `Citation` | はい | 法令と条文位置を確認するための情報 |

## 制約

`format` と `content` の表現、安全化および binary artifact の境界は `SOT-MODEL-017` と同じ規則に従う。読みやすさのための推測、見出しの生成または別の条項から補った文字列を含めない。

`law.revisionId`、`citation.revisionId` および `content` は、同じ情報源上の同じ法令リビジョンを指す。

`location` は要求を正規化した `LawArticleLocation` と一致させる。`citation.location` は同じ位置を原文で確認できる provider mapping 固有の文字列とし、`location` と異なる条または項を指してはならない。

## 関連

- [SOT-MODEL-001: LawSummary](01-law-summary.md)
- [SOT-MODEL-004: Citation](04-citation.md)
- [SOT-MODEL-017: LawDocumentRepresentation](17-law-document-representation.md)
- [SOT-MODEL-018: LawArticleLocation](18-law-article-location.md)
- [SOT-IF-025: law.article.read capability v1](../40-interfaces/25-law-article-read-capability.md)
