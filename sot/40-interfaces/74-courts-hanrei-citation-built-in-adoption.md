# SOT-IF-074: 判例引用追跡の組込み採用

- 状態: 有効

## 規定

`SOT-IF-070` と `SOT-IF-072` に従う citation 用 binding を、`judicial-cases` と `judicial-citations` が同時に有効なときだけ条件付き既定値となる組込み provider として採用する。

## 条件付き組込み値

両 pack が有効な場合に限り、`SOT-IF-026` の組込み値へ次を加える。

```yaml
providers:
  courts-hanrei-pdf:
    enabled: true
    settings: {}
    credentialEnvRefs: {}
providerRoutes:
  judicial-decision.case-citation.extract@1:
    selection: primary
    defaultProviderId: courts-hanrei-pdf
  judicial-decision.citing-candidate.search@1:
    selection: primary
    defaultProviderId: courts-hanrei-html
```

拡張パックが不足している場合は、この provider と二つの route を実効設定へ加えない。無効状態で利用者が `courts-hanrei-pdf` または citation route を明示した場合は、transport 開始前の設定エラーとする。

## 適用範囲

- `courts-hanrei-html` の既存 search/read binding は `judicial-cases` 単独でも維持する。
- `courts-hanrei-pdf` は `judicial-citations` が有効なときだけ factory を呼ぶ。
- citation route は runtime fallback、aggregate route または別 provider への切替えを追加しない。
- 既存の法令コア、`judicial-cases`、`legislative-history` および公開ツール契約を変更しない。

## 公開

内部 binding の到達性だけでは公開面を追加しない。`judicial-citations` の有効化と `SOT-IF-075` に従う MCP ツールがそろった場合だけ公開する。

## 確認

descriptor と route の一致、両 pack 無効時と `judicial-cases` 単独時の citation route 非登録、両 pack 有効時の原子的構成、provider 欠落時の起動失敗、および公開 tool 数の一致を確認する。

## 関連

- [SOT-IF-067: `judicial-citations` 拡張パックの有効化](67-judicial-citations-pack-activation.md)
- [SOT-IF-070: 裁判所「裁判例検索」PDF 情報源](70-source-courts-hanrei-pdf.md)
- [SOT-IF-072: 裁判所「裁判例検索」HTML 情報源 v2](72-source-courts-hanrei-html-v2.md)
