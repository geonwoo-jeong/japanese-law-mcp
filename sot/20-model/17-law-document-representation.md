# SOT-MODEL-017: LawDocumentRepresentation

- 状態: 有効

## 規定

`LawDocumentRepresentation` は、内部の共通 capability が一つの法令リビジョンの本文を、情報源が公式に提供する安全な文字列表現、検索基準日および出典によって表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `law` | `LawSummary` | はい | 対象法令と選択されたリビジョン |
| `asOf` | date | いいえ | 利用者が指定した検索基準日 |
| `format` | string | はい | `xml`、`html` または `text` |
| `content` | string | はい | 選択した形式で表す UTF-8 の法令本文 |
| `citation` | `Citation` | はい | 原文を確認するための出典 |

## 表現の制約

`asOf` はリビジョンの施行日ではない。指定日の以前で最新となるリビジョンは、`law.revisionId` と `law.revisionEffectiveDate` で示す。

`format: xml` は、情報源が法令本文として提供した一つの XML 要素を内容の意味を変えずにシリアライズした値とする。より大きい応答から要素を選択した場合の `Provenance.transformation` は `extracted` とする。

`format: html` は、情報源が法令本文として提供した一つの HTML 文書または fragment から、script、style、event handler、外部 resource の読込みおよび実行可能内容を除いた静的な HTML とする。選択と安全化の方法は provider mapping SOT で定義し、`Provenance.transformation` を `unchanged` にしない。

`format: text` は、情報源が法令本文として明示した文字列、または公式 HTML 等から決定的に抽出した可視文字列とする。抽出した場合は見出し、条番号および改行の規則を provider mapping SOT で定義し、`Provenance.transformation` を `extracted` または `normalized` とする。

PDF、画像またはその他の binary artifact を base64 にして `content` へ格納しない。それらを共通化する場合は、文字列本文とは別の artifact capability と情報モデルを先に採用する。

各 provider は、対応する mapping SOT で返却する `format` を固定し、conformance matrix の fixture でその形式を検証する。`ProviderDescriptor` の capability 宣言だけから形式を推測せず、呼出し側が provider の mapping SOT にない形式を要求できる汎用変換契約は設けない。

## 既存公開モデルとの境界

`LawDocumentRepresentation` は内部 capability のモデルであり、既存 MCP ツールの `LawDocument` を置き換えない。

既存の `LawDocument` へ投影できるのは、`format` が `xml` であり、`content` が `SOT-MODEL-002` の XML 制約を満たす場合だけとする。`html` または `text` を XML に見せかけて投影せず、公開する必要がある場合は別の公開契約を先に採用する。

## 確認

provider が宣言した各形式について、決定的な変換、出典、unsafe content の拒否および binary artifact の拒否を契約試験で確認する。既存 `LawDocument` への投影試験では、XML 以外を拒否し、XML の内容と出典を変更しないことを確認する。

## 関連

- [SOT-MODEL-001: LawSummary](01-law-summary.md)
- [SOT-MODEL-002: LawDocument](02-law-document.md)
- [SOT-MODEL-004: Citation](04-citation.md)
- [SOT-MODEL-012: Provenance](12-provenance.md)
- [SOT-IF-024: law.document.read capability v1](../40-interfaces/24-law-document-read-capability.md)
