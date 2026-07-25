# SOT-IF-014: ProviderDescriptor

- 状態: 有効

## 規定

`ProviderDescriptor` は、一つのプロバイダーアダプターの識別子、情報源、互換性、確認した外部仕様および実装する能力を、起動後に変更しない記述子として表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `providerId` | string | はい | プロジェクト内で不変のプロバイダー識別子 |
| `source` | `InformationSource` | はい | このアダプターが取得する情報源 |
| `adapterContractVersion` | string | はい | アダプター境界の互換性を表す版 |
| `upstreamSpecVersion` | string | いいえ | 公式仕様が版を示す場合の確認済みの版 |
| `verifiedAt` | date | はい | 公式仕様または公式掲載ページを最後に確認した日 |
| `interfaceType` | string | はい | `api`、`html`、`download` または `hybrid` |
| `credentialRequired` | boolean | はい | 外部情報源の利用に認証情報が必要か |
| `capabilities` | `ProviderCapability[]` | はい | 実装を宣言する能力 |

## 制約

`providerId` は小文字の ASCII 英数字とハイフンで構成し、提供主体、サービスおよび互換性を持たない版を区別できる値とする。

互換性を持つ外部仕様の minor または patch 更新、fixture の更新およびアダプターの修正では `providerId` を変更しない。互換性を持たない外部 API の系列を並行して登録する必要がある場合に、新しい版を区別する `providerId` を発行する。

`upstreamSpecVersion` は確認した外部仕様の版だけを表す。`adapterContractVersion` はプロバイダー固有の設定、parser contract、記述子および mapping の互換性を SemVer で表す。能力の型付き入出力の互換性は `ProviderCapability.majorVersion` だけを定義元とし、これら三つの版を同じ値として扱わない。

`capabilities` は能力 ID とメジャーバージョンで並べ、重複を許さない。記述子が能力を宣言することと、その能力の入出力スキーマを定義することを分け、スキーマは能力別 SOT を定義元とする。

認証方式、秘密値、外部 URL のパス、外部フィールド名および CSS selector を記述子へ含めない。

版を示さない HTML では `upstreamSpecVersion` を省略し、`verifiedAt` とプロバイダー固有の parser contract を検証の基準にする。

## 関連

- [SOT-MODEL-010: InformationSource](../20-model/10-information-source.md)
- [SOT-MODEL-013: ProviderCapability](../20-model/13-provider-capability.md)
- [SOT-ARCH-012: プロバイダーの登録](../30-architecture/12-provider-registry.md)
