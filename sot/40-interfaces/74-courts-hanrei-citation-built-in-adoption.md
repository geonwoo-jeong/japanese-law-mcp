# SOT-IF-074: 判例引用追跡の組込み採用

- 状態: 有効

## 規定

`SOT-IF-072` と `SOT-IF-073` に従う `courts-hanrei-html`、および `SOT-IF-070` と `SOT-IF-071` に従う `courts-hanrei-pdf` を、pack の有効状態に応じた条件付き組込み provider として採用する。

本規定は `SOT-IF-046` の後継であり、既存の裁判例 search/read の binding、route、結果および公開ツールの意味を変更しない。

## 条件付き組込み値

`extensionPacks.judicial-cases.enabled` が `true` の場合は、`SOT-IF-026` の組込み値へ既存の次を加える。

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

最終接続で候補検索 row を `implemented` にした後は、HTML descriptor と compiled binding inventory を三能力で一致させる。ただし `judicial-citations` が無効な間は候補検索 route、引用追跡 application service および専門操作を構成せず、候補検索 binding を公開処理から到達不能にする。準備中の `planned` row は production descriptor 又は compiled binding inventory に含めない。

さらに両 pack が有効な場合に限り、次を加える。

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

依存 pack が不足している場合は、PDF provider と citation 二 route を実効設定へ加えない。無効状態で利用者が `courts-hanrei-pdf` または citation route を明示した場合は、transport 開始前の設定エラーとする。

## 適用範囲

- `courts-hanrei-html` の既存 search/read binding は `judicial-cases` 単独でも維持する。
- `courts-hanrei-pdf` は `judicial-citations` が有効なときだけ factory を呼ぶ。
- citation route は runtime fallback、aggregate route または別 provider への切替えを追加しない。
- 既存の法令コア、`judicial-cases`、`legislative-history` および公開ツール契約を変更しない。
- 二つの追加 route は有効で `implemented` の binding へ到達し、PDF provider、二 route および `trace_judicial_citations` 専門操作を同じ起動条件で原子的に構成する。

準備中の二つの conformance row は `planned` とし、production route と公開操作から到達不能にする。provider package、fixture、契約テスト、compiled binding、route および専門操作がそろう最終接続変更でだけ、二 row を同時に `implemented` へ変更する。

## 公開

内部 binding の到達性だけでは利用可能な専門操作を追加しない。`judicial-citations` の有効化と `SOT-IF-075` に従う専門操作がそろった場合だけ、`SOT-IF-077` の方式で公開する。

## 確認

descriptor、compiled binding inventory、conformance matrix、fixture、資源予算および route の一致、`planned` row の到達不能、両 pack 無効時と `judicial-cases` 単独時の citation route 非登録、両 pack 有効時の原子的構成、provider 欠落時の起動失敗、PDF factory の条件付き呼出し、ならびに `SOT-IF-077` の専門操作数と公開ツール数の一致を確認する。

## 関連

- [SOT-IF-077: MCP ツール公開方式と拡張パック有効化](77-mcp-tool-exposure-and-extension-packs.md)
- [SOT-IF-068: 判決文の判例引用抽出 capability](68-judicial-case-citation-extract-capability.md)
- [SOT-IF-069: 被引用候補検索 capability](69-judicial-citing-candidate-search-capability.md)
- [SOT-IF-070: 裁判所「裁判例検索」PDF 情報源](70-source-courts-hanrei-pdf.md)
- [SOT-IF-071: 裁判所 PDF の判例引用抽出マッピング](71-courts-hanrei-pdf-extract-mapping.md)
- [SOT-IF-072: 裁判所「裁判例検索」HTML 情報源 v2](72-source-courts-hanrei-html-v2.md)
- [SOT-IF-073: 裁判所検索の被引用候補マッピング](73-courts-hanrei-citing-candidate-search-mapping.md)
