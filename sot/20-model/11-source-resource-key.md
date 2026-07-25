# SOT-MODEL-011: SourceResourceKey

- 状態: 有効

## 規定

`SourceResourceKey` は、異なる情報源の識別子を同じ識別子空間へ混在させず、一つの情報源上の資源と版を参照するための共通キーを表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `sourceId` | string | はい | `InformationSource.id` |
| `resourceType` | string | はい | 能力別モデル SOT が定義する資源種別 |
| `resourceId` | string | はい | 情報源が使用する資源識別子 |
| `versionId` | string | いいえ | 情報源が明示する版、改訂またはリビジョンの識別子 |

## 制約

識別の単位は、`sourceId`、`resourceType`、`resourceId` および存在する場合の `versionId` の組とする。

情報源の識別子は不透明な文字列として保持し、大文字小文字、先頭のゼロ、区切り文字または文字コード上の表記を変更しない。

公式識別子がない HTML またはファイルでは、情報源ごとの SOT が安定した公式 URL のパスを識別子として採用できる。名称、題名、住所または本文の類似性から識別子を生成しない。

異なる情報源の資源を同一と判断する場合は、公式に確認できる対応表または相互参照を別の SOT で定義する。

## 関連

- [SOT-MODEL-010: InformationSource](10-information-source.md)
- [SOT-MODEL-012: Provenance](12-provenance.md)
- [SOT-ARCH-013: 情報源の選択と組合せ](../30-architecture/13-source-composition.md)
