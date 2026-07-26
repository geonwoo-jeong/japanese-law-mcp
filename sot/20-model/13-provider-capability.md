# SOT-MODEL-013: ProviderCapability

- 状態: 有効

## 規定

`ProviderCapability` は、一つのプロバイダーが実装すると宣言する能力別契約と、その互換性および安定性を表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `id` | string | はい | 小文字の dot-separated segments で構成する能力識別子 |
| `majorVersion` | integer | はい | 入出力の意味と必須項目の互換性境界 |
| `level` | string | はい | `core`、`extended` または `provider_specific` |
| `stability` | string | はい | `stable` または `experimental` |

## 制約

`id` は、各 segment を小文字の ASCII 英数字と内部のハイフンで構成し、二つ以上の segment を `.` で連結する。`law.search`、`law.content.search` および `parliament.speech.search` はこの形式の例である。

共通能力の `id` はプロジェクトが所有し、プロバイダーは対応する入出力スキーマを変更しない。

プロバイダー固有能力の `id` は `provider.{providerId}.{operation}` の名前空間を使用する。固有能力の項目を、同じ意味が確認されていない共通能力へ流用しない。

必須項目、型または意味を変更する場合は `majorVersion` を増やす。省略可能な項目の追加は、既存の意味を変えない場合に限り同じメジャーバージョンで扱える。

試行提供の外部機能は `experimental` とし、安定した既定経路として選択しない。

能力 ID とメジャーバージョンを宣言する前に、能力別 SOT が利用目的、型付き入力、型付き出力、欠落時の扱い、継続取得、エラーおよび検証方法を定義していなければならない。能力群の一覧または名称だけを根拠に、プロバイダーが入出力を独自に定義しない。

## 関連

- [SOT-ARCH-017: 採用可能な能力群](../30-architecture/17-approved-capability-families.md)
- [SOT-IF-014: ProviderDescriptor](../40-interfaces/14-provider-descriptor.md)
