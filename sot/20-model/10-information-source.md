# SOT-MODEL-010: InformationSource

- 状態: 有効

## 規定

`InformationSource` は、法令に限らない一つの情報提供サービスと、その提供主体としての位置付けを表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `id` | string | はい | プロジェクト内で不変の情報源識別子 |
| `name` | string | はい | 情報提供サービスの名称 |
| `publisher` | string | はい | 情報を公開する機関または組織 |
| `authority` | string | はい | `official` または `supplementary` |
| `serviceUrl` | string | はい | 情報提供サービスを確認できる公式 HTTPS URL |

## 制約

`authority` は提供主体の位置付けだけを表す。文書の法的効力、拘束力、確定性、網羅性または内容の正しさを表さない。

同じ提供主体が異なる仕様、更新周期または利用条件のサービスを提供する場合は、サービスごとに異なる `id` を使用する。

`InformationSource` をプロバイダー基盤における情報源メタデータの定義元とする。法令向けの既存 `LegalSource` は法令公開インターフェースの互換モデルとして維持し、非法令情報へ意味を拡張しない。法令プロバイダーは `InformationSource` から `LegalSource` を導出し、同じ情報源の値を別々に定義しない。

## 関連

- [SOT-MODEL-003: LegalSource](03-legal-source.md)
- [SOT-MODEL-011: SourceResourceKey](11-source-resource-key.md)
- [SOT-PROD-003: 法情報の採用基準](../00-product/03-legal-source-eligibility.md)
