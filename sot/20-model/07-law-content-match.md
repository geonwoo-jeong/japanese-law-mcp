# SOT-MODEL-007: LawContentMatch

- 状態: 有効

## 規定

`LawContentMatch` は、法令本文検索で一致した一つの箇所を、法令、位置、本文および出典によって表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `law` | `LawSummary` | はい | 一致箇所を含む法令リビジョン |
| `location` | string | はい | 情報源が返した法令内の位置 |
| `text` | string | はい | 強調表示を除いた一致箇所の本文 |
| `citation` | `Citation` | はい | 一致箇所の出典 |

## 制約

`text` は検索結果に含まれる範囲だけを表し、前後の本文を推測して補わない。

`location` を条項番号へ変換できない場合も、情報源が返した位置を失わない。

## 関連

- [SOT-MODEL-001: LawSummary](01-law-summary.md)
- [SOT-MODEL-004: Citation](04-citation.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
