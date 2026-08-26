# SOT-IF-043: 裁判所「裁判例検索」HTML 情報源

- 状態: 廃止
- 廃止理由: 被引用候補検索 capability を同じ公式 HTML provider へ追加した後の descriptor を表せないため
- 後継: [SOT-IF-072: 裁判所「裁判例検索」HTML 情報源 v2](72-source-courts-hanrei-html-v2.md)

## 規定

本規定を `SOT-IF-072` に置き換えた。以下は廃止時点の履歴であり、現行の provider 契約には適用しない。

`courts-hanrei-html` は、最高裁判所が `https://www.courts.go.jp/hanrei/` 配下で公開する HTML を安全に読み取り、`judicial-decision.search@1` と `judicial-decision.read@1` を提供する組込み provider とする。

## 識別

| 項目 | 値 |
|---|---|
| `providerId` | `courts-hanrei-html` |
| `source.id` | `courts-hanrei` |
| `source.name` | `裁判所 裁判例検索` |
| `source.publisher` | `最高裁判所` |
| `source.authority` | `official` |
| `source.serviceUrl` | `https://www.courts.go.jp/hanrei/search1/index.html` |
| `adapterContractVersion` | `1.1.0` |
| `upstreamSpecVersion` | 省略 |
| `verifiedAt` | `2026-07-26` |
| `interfaceType` | `html` |
| `credentialRequired` | `false` |

descriptor は `judicial-decision.read@1` と `judicial-decision.search@1` の二つを `extended`、`stable` として能力 ID 順に宣言する。

## 公式公開面

- 統合検索: `https://www.courts.go.jp/hanrei/search1/index.html`
- 詳細: `https://www.courts.go.jp/hanrei/{id}/detail{2..8}/index.html`
- 検索方法の案内: `https://www.courts.go.jp/hanrei/tukaikata/index.html`
- サイト利用条件: `https://www.courts.go.jp/outline/index.html`

公式に文書化された機械 API は採用せず、検索結果と詳細の HTML だけを取得する。HTML が直接示す PDF は URL と metadata だけを返し、この provider は PDF を取得、解析または再配布しない。

## 接続境界

- origin は `https://www.courts.go.jp` に固定し、設定で変更しない。
- 認証、cookie、proxy、任意 URL、background crawling、恒久 cache、稼働監視および自動再試行を追加しない。
- redirect は同じ HTTPS origin だけを許可する。
- HTML parser は DOM として解析し、正規表現だけで要素、属性または入れ子を解析しない。
- script、stylesheet、画像、PDF その他の埋込み resource を取得または実行しない。

## 構成 scope

継続トークンに使用する `SOT-IF-026` の provider configuration scope は、次の固定値とする。

| key | 値 |
|---|---|
| `providerId` | `courts-hanrei-html` |
| `origin` | `https://www.courts.go.jp` |
| `dataset` | `hanrei` |
| `tenant` | `n/a` |
| `account` | `n/a` |
| `proxy` | `n/a` |
| `semanticConfig` | 空の object |
| `credentialSlots` | 空の object |

## 資源予算

| operation | artifact | `budgetKey` | `responseBytes` | `decompressedBytes` | `entriesOrObjects` | `depth` | `parseTimeout` | `concurrencyGroup` | `concurrency` |
|---|---|---|---:|---:|---:|---:|---:|---|---:|
| search | HTML | `judicial-search-html` | 2 MiB | 4 MiB | 100000 | 64 | 2s | `courts-hanrei-html` | 1 |
| read | HTML | `judicial-read-html` | 1 MiB | 2 MiB | 50000 | 64 | 2s | `courts-hanrei-html` | 1 |

各 operation は HTTP content coding を復号した HTML を DOM parser へ渡す。byte、HTML node、深さ、解析時間および共有同時実行枠の計測と超過時の error は `SOT-ENG-016` に従う。

## エラー

HTTP 404 は正確な参照による read だけ `not_found` とする。HTTP 429、timeout、一時的な接続失敗、5xx、HTML 契約の不一致、値の不正および資源予算超過は `SOT-IF-017` に従って分類する。エラー、診断またはログへ検索語、HTML 本文、URL query または取得内容を含めない。

## 確認

descriptor の固定値、二能力の inventory、固定 origin、同一 origin redirect、cookie と追加 resource の禁止、operation ごとの全資源予算、同時実行枠の共有、公式 HTML fixture および収録範囲の注意を契約テストで確認する。

## 関連

- [SOT-PROD-010: 裁判例拡張パック](../00-product/10-judicial-cases-extension-pack.md)
- [SOT-IF-014: ProviderDescriptor](14-provider-descriptor.md)
- [SOT-IF-017: 情報源エラーの正規化](17-source-error-normalization.md)
- [SOT-IF-044: 裁判所の裁判例検索マッピング](44-courts-hanrei-search-mapping.md)
- [SOT-IF-045: 裁判所の裁判例詳細マッピング](45-courts-hanrei-read-mapping.md)
- [SOT-ENG-016: プロバイダー資源予算](../50-engineering/16-provider-resource-budgets.md)
