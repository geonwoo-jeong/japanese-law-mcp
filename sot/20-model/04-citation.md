# SOT-MODEL-004: Citation

- 状態: 有効

## 規定

`Citation` は、返された法情報がどの情報源のどの法令および位置に由来するかを、利用者が再確認できる形で表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `source` | `LegalSource` | はい | 引用元の情報源 |
| `lawId` | string | はい | 引用元の法令識別子 |
| `revisionId` | string | はい | 引用元の法令リビジョン識別子 |
| `location` | string | いいえ | 条、項その他の法令内位置 |
| `url` | string | はい | 当該リビジョンの原文を確認できる HTTPS URL |

## 制約

`location` は確認できた粒度だけを法令 XML の番号で表し、存在しない条文位置を補完しない。

`url` は `LegalSource.serviceUrl` と使い分け、返された法令リビジョンを直接確認できる位置を示す。

## 関連

- [SOT-MODEL-003: LegalSource](03-legal-source.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
- [SOT-SCN-003: 条文を取得する](../10-scenarios/03-get-article.md)
