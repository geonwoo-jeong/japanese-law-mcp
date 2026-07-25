# SOT-MODEL-012: Provenance

- 状態: 有効

## 規定

`Provenance` は、返された情報の取得元、原文の参照先、取得時点および変換の種類を、再確認できる形で表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `source` | `InformationSource` | はい | 情報を取得した情報源 |
| `resourceKey` | `SourceResourceKey` | はい | 情報源上の資源と版 |
| `url` | string | はい | 原文または公式掲載情報を確認できる HTTPS URL |
| `retrievedAt` | date-time | はい | このリクエストで取得した時刻 |
| `sourceUpdatedAt` | date-time または date | いいえ | 情報源が明示した更新時点 |
| `mediaType` | string | はい | 取得した表現の MIME type |
| `location` | string | いいえ | 原文内で確認できる位置 |
| `transformation` | string | はい | `unchanged`、`extracted`、`normalized` または `derived` |
| `methodId` | string | いいえ | 変換方法を定義する SOT または契約の識別子 |
| `inputKeys` | `SourceResourceKey[]` | いいえ | 加工に使用した入力資源 |
| `contentDigest` | string | いいえ | 取得したバイト列の `sha256:` 形式のダイジェスト |

## 制約

`transformation` が `unchanged` 以外の場合は `methodId` を必須とする。`derived` の場合は `inputKeys` も必須とする。

`retrievedAt` は情報の発生日、施行日、裁決日、観測期間、提出日または有効判定日を表さない。これらの意味を持つ日付は能力別モデルに保持する。

HTML から文字列を取り出した結果は `extracted`、文字コードや構造を意味を変えずに共通モデルへ対応させた結果は `normalized`、計算、比較または推論を加えた結果は `derived` とする。

## 関連

- [SOT-PROD-005: 加工情報の区別](../00-product/05-derived-information.md)
- [SOT-MODEL-011: SourceResourceKey](11-source-resource-key.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
