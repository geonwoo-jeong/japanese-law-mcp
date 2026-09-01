# SOT-IF-037: e-Gov 法令 API Version 1 の組込み採用

- 状態: 有効

## 規定

`SOT-IF-035` と `SOT-IF-036` に従う `e-gov-law-api-v1` の `law.update.list@1` binding を、無設定時に利用できる組込みプロバイダーとして採用する。

## 組込み既定値の差分

`SOT-IF-026` の組込み既定値へ、次のプロバイダーと primary route だけを追加する。

```yaml
providers:
  e-gov-law-api-v1:
    enabled: true
    settings: {}
    credentialEnvRefs: {}
providerRoutes:
  law.update.list@1:
    selection: primary
    defaultProviderId: e-gov-law-api-v1
```

`e-gov-law-api-v2` の有効化と、`SOT-IF-026` が定める同 provider の六つの既定 route は変更しない。起動時の compiled binding registry は両方の descriptor と binding を保持し、`law.update.list@1` は `e-gov-law-api-v1` の型付き port へ到達できなければならない。

## 適用範囲

- `e-gov-law-api-v1` は provider-specific setting と credential slot を持たない。組込み値から設定または credential を解決せず、固定された公式 endpoint だけを使用する。
- この追加は既存の四能力の provider、結果、順序、継続位置、continuation token の構成 scope および rollback の選択を変更しない。
- `law.update.list@1` は完全な一日分を返し、continuation と組込み rollback を使用しない。
- この採用が直接追加するのは内部 capability route の到達性だけであり、同じ変更の責任として公開 MCP ツールを追加しない。現在の `list_law_updates` と公開ツール構成は `SOT-IF-076` が定義する。
- 利用者指定の `providers`、`providerRoutes`、credential environment reference および rollback override の設定入力は、この採用の範囲に含めない。

## 確認

- composition root が e-Gov Version 1 と Version 2 の descriptor を登録し、五つの組込み primary route を正しい型付き port へ解決することを確認する。
- e-Gov Version 1 の conformance matrix row を `implemented` とし、この SOT を適用規定として参照する。
- 内部 route の組込み採用と、`SOT-IF-076` による公開ツールを独立して検証できることを確認する。

## 関連

- [SOT-IF-026: プロバイダールーティング設定](26-provider-routing-configuration.md)
- [SOT-IF-034: `law.update.list` capability v1](34-law-update-list-capability.md)
- [SOT-IF-035: e-Gov 法令 API Version 1 更新一覧](35-source-egov-law-api-v1.md)
- [SOT-IF-036: e-Gov 更新法令一覧マッピング](36-egov-law-update-list-mapping.md)
- [SOT-IF-076: MCP `list_law_updates` v2](76-mcp-list-law-updates-v2.md)
- [SOT-ARCH-016: プロバイダーの段階的追加](../30-architecture/16-incremental-provider-onboarding.md)
