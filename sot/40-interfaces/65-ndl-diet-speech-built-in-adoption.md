# SOT-IF-065: 国立国会図書館の国会発言検索の組込み採用

- 状態: 有効

## 規定

`SOT-IF-063` と `SOT-IF-064` に従う `ndl-diet-speech-api` の binding を、
`legislative-history` の第一段階専門公開面に使用する条件付き組込み値として採用する。

## 組込み値

`extensionPacks.legislative-history.enabled` が `true` の場合だけ、実効構成へ次を加える。

```yaml
providers:
  ndl-diet-speech-api:
    enabled: true
    settings: {}
    credentialEnvRefs: {}
providerRoutes:
  parliament.speech.search@1:
    selection: primary
    defaultProviderId: ndl-diet-speech-api
```

provider は `parliament.speech.search@1` だけを宣言する。provider-specific setting と
credential slot を持たず、固定した公式 endpoint だけを使用する。

## 公開境界

次を同じ変更単位で満たす場合だけ、組込み採用を完了とする。

- provider descriptor、compiled binding inventory および conformance matrix の一行が同じ一能力を示す
- matrix row が `implemented` で、provider package、fixture および契約テストから到達できる
- `parliament.speech.search@1` の primary route が pack 有効時だけ構成される
- `search_diet_speeches` が同じ pack 条件で登録される
- pack が無効な場合は provider factory、binding、route および専門ツールが構成されない

準備段階の `planned` row、provider package または test 用 descriptor を production
descriptor、runtime binding、route または公開採用として扱わない。第一段階では
統合照会の意味認識・実行 contribution を組込み条件にせず、これらを暗黙に追加しない。

## 非影響と失敗

- e-Gov Version 1、e-Gov Version 2 および `courts-hanrei-html` の descriptor、binding、route、設定および公開ツールを変更しない。
- `parliament.*` 以外の capability を宣言しない。
- runtime fallback、aggregate route、認証情報、任意 endpoint または別の NDL API operation を追加しない。
- pack 有効時に binding、route または専門ツールが欠ける場合は transport 開始前の設定エラーとし、部分的に公開しない。

## 確認

descriptor、binding inventory、`implemented` row、fixture、契約テスト、条件付き primary
route および専門ツールの一致を確認する。pack 無効時は factory を呼ばず、既存 provider、
route、公開ツールおよび統合照会の profile set が変わらないことも確認する。

## 関連

- [SOT-IF-061: `legislative-history` 拡張パックの専門公開面](61-legislative-history-pack-activation.md)
- [SOT-IF-062: `parliament.speech.search` capability v1](62-parliament-speech-search-capability.md)
- [SOT-IF-063: 国立国会図書館の国会発言検索 API 情報源](63-source-ndl-diet-speech-api.md)
- [SOT-IF-064: 国立国会図書館の国会発言検索マッピング](64-ndl-diet-speech-search-mapping.md)
- [SOT-IF-066: MCP `search_diet_speeches`](66-mcp-search-diet-speeches.md)
- [SOT-ARCH-016: プロバイダーの段階的な追加](../30-architecture/16-incremental-provider-onboarding.md)
- [SOT-ARCH-041: 拡張パックの専門公開面の段階採用](../30-architecture/41-staged-specialist-extension-surface.md)
- [SOT-ENG-017: プロバイダー適合性 matrix](../50-engineering/17-provider-conformance-matrix.md)
