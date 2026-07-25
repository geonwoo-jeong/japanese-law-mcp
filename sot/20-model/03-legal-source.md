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

情報源ポートの内部では `InformationSource` を情報源メタデータの定義元とし、法令向けの公開境界で必要な場合に限り、次の決定的な投影で `LegalSource` を作る。

| `InformationSource` | `LegalSource` |
|---|---|
| `id` | `id` へ文字列を変更せず設定する |
| `name` | `name` へ文字列を変更せず設定する |
| `authority` | `authority` へ列挙値を変更せず設定する |
| `serviceUrl` | `serviceUrl` へ URL を変更せず設定する |
| `publisher` | 対応項目を作らず、他の項目へ連結または代入しない |

投影時に trimming、大文字小文字の変更、URL の再正規化、別名への置換または `publisher` の付加を行わない。一つの法令情報源について、投影対象の四項目を `LegalSource` 用と `InformationSource` 用に別々の事実として定義しない。

この投影は `publisher` を持たないため可逆変換ではない。`LegalSource` から `InformationSource` を復元、推測または既定化しない。

同じ `SourcedResource<T>` 内の `LawSummary.source` と `Citation.source` は、`Provenance.source` および `ProviderDescriptor.source` と同じ `InformationSource` からこの規則で投影した値とする。

## 確認

各法令 provider の descriptor fixture から投影を二回実行して同じ `LegalSource` を得ること、四項目が byte 単位で一致すること、および `publisher` が `name` その他の公開項目へ混入しないことを確認する。`LawSummary`、`Citation`、`Provenance` および descriptor の情報源を同じ fixture で照合する。

## 関連

- [SOT-MODEL-010: InformationSource](10-information-source.md)
- [SOT-MODEL-012: Provenance](12-provenance.md)
- [SOT-PROD-003: 法情報の採用基準](../00-product/03-legal-source-eligibility.md)
- [SOT-MODEL-004: Citation](04-citation.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
