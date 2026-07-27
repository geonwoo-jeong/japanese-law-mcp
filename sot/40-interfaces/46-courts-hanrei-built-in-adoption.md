# SOT-IF-046: 裁判所「裁判例検索」の組込み採用

- 状態: 有効

## 規定

`SOT-IF-043`、`SOT-IF-044` および `SOT-IF-045` に従う `courts-hanrei-html` の二つの binding を、`judicial-cases` が有効なときだけ条件付き既定値となる組込みプロバイダーとして採用する。

## 条件付き組込み値

`extensionPacks.judicial-cases.enabled` が `true` の場合に限り、`SOT-IF-026` の組込み値へ次を加える。

```yaml
providers:
  courts-hanrei-html:
    enabled: true
    settings: {}
    credentialEnvRefs: {}
providerRoutes:
  judicial-decision.read@1:
    selection: primary
    defaultProviderId: courts-hanrei-html
  judicial-decision.search@1:
    selection: primary
    defaultProviderId: courts-hanrei-html
```

拡張パックが無効な場合は、この provider と二つの route を実効設定へ加えない。無効な状態で利用者が `courts-hanrei-html` または `judicial-decision.*` route を明示した場合は、利用されない設定として黙って保持せず、transport 開始前の設定エラーとする。

## 適用範囲

- `courts-hanrei-html` は provider-specific setting と credential slot を持たず、固定した公式 endpoint だけを使用する。
- composition root の compiled binding registry は、`judicial-cases` が有効な場合に限り、この provider の factory を呼び出す。拡張パックが無効な間は、利用者が provider または route を明示しても実効構成へ参加させない。
- 有効な pack の二つの route は同じ provider の型付き port へ到達しなければならない。
- この採用は既存の e-Gov Version 1 と Version 2 の descriptor、五 capability、primary route、結果、継続位置、設定 scope または公開法令ツールを変更しない。
- 他の拡張パックを有効にせず、`judicial-decision.*` 以外の capability を宣言しない。
- runtime fallback、aggregate route および認証情報の解決を追加しない。

## 公開

内部 binding の到達性だけでは公開面を追加しない。`judicial-cases` の有効化と、`SOT-IF-047` および `SOT-IF-048` に従う二つの MCP ツールがそろった場合だけ公開する。

## 確認

- descriptor が二つの capability だけを宣言し、binding inventory と一致すること
- conformance matrix の二行が `implemented` で、production descriptor、fixture、test、runtime registry および route と一致すること
- pack 無効時に factory を呼び出さず五つの法令 route とツールを維持すること
- pack 有効時に二つの条件付き route と七つのツールを構成すること
- provider または route の欠落、無効化および不一致を transport 開始前に拒否すること

## 関連

- [SOT-IF-040: `judicial-cases` 拡張パックの有効化](40-judicial-cases-pack-activation.md)
- [SOT-IF-041: `judicial-decision.search` capability v1](41-judicial-decision-search-capability.md)
- [SOT-IF-042: `judicial-decision.read` capability v1](42-judicial-decision-read-capability.md)
- [SOT-IF-043: 裁判所「裁判例検索」HTML 情報源](43-source-courts-hanrei-html.md)
- [SOT-ARCH-016: プロバイダーの段階的追加](../30-architecture/16-incremental-provider-onboarding.md)
