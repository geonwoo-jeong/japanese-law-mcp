# SOT-MODEL-003: LegalSource

- 状態: 有効

## 規定

`LegalSource` は、法情報を提供するサービスと、その情報源としての位置付けを表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `id` | string | はい | プロジェクト内の情報源識別子 |
| `name` | string | はい | 情報源サービスの名称 |
| `authority` | string | はい | `official` または `supplementary` |
| `serviceUrl` | string | はい | 情報源サービスの公式 HTTPS URL |

## 制約

`serviceUrl` は情報源サービスを示し、個別の法令や条文の参照先には使用しない。個別の参照先は `Citation.url` だけで表す。

`authority` は情報の正しさを独自に保証する値ではなく、提供主体の位置付けを示す。

## 共通情報源との対応

情報源ポートの内部では `InformationSource` を情報源メタデータの定義元とし、法令向けの公開境界で必要な場合に限り、`id`、`name`、`authority` および `serviceUrl` を同名項目へ変換して `LegalSource` を作る。

一つの法令情報源について、これらの値を `LegalSource` 用と `InformationSource` 用に別々の事実として定義しない。

## 関連

- [SOT-MODEL-010: InformationSource](10-information-source.md)
- [SOT-PROD-003: 法情報の採用基準](../00-product/03-legal-source-eligibility.md)
- [SOT-MODEL-004: Citation](04-citation.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
