# SOT-IF-072: 裁判所「裁判例検索」HTML 情報源 v2

- 状態: 有効

## 規定

`courts-hanrei-html` は、最高裁判所が `https://www.courts.go.jp/hanrei/` 配下で公開する HTML を安全に読み取り、`judicial-decision.search@1`、`judicial-decision.read@1` および `judicial-decision.citing-candidate.search@1` を提供する組込み provider とする。

## 識別

本規定は `SOT-IF-043` の後継であり、既存の search/read 契約を変えず、次の descriptor を現行の定義元とする。

| 項目 | 値 |
|---|---|
| `providerId` | `courts-hanrei-html` |
| `source.id` | `courts-hanrei` |
| `source.name` | `裁判所 裁判例検索` |
| `source.publisher` | `最高裁判所` |
| `source.authority` | `official` |
| `source.serviceUrl` | `https://www.courts.go.jp/hanrei/search1/index.html` |
| `adapterContractVersion` | `1.2.0` |
| `upstreamSpecVersion` | 省略 |
| `verifiedAt` | `2026-07-26` |
| `interfaceType` | `html` |
| `credentialRequired` | `false` |

descriptor は `judicial-decision.citing-candidate.search@1`、`judicial-decision.read@1`、`judicial-decision.search@1` を capability ID 順に `extended`、`stable` として宣言する。`verifiedAt` は既存 operation を含む全契約の再確認日のうち最も古い日とし、候補検索だけの確認日で上書きしない。

## 公式公開面

- 統合検索: `https://www.courts.go.jp/hanrei/search1/index.html`
- 詳細: `https://www.courts.go.jp/hanrei/{id}/detail{2..8}/index.html`
- 検索方法の案内: `https://www.courts.go.jp/hanrei/tukaikata/index.html`
- 掲載判例の説明: `https://www.courts.go.jp/hanrei/setumei/index.html`
- サイト利用条件: `https://www.courts.go.jp/outline/index.html`

公式に文書化された機械 API は採用せず、検索結果と詳細の HTML だけを取得する。HTML が直接示す PDF は URL と metadata だけを返し、この provider は PDF を取得または解析しない。

## citation 候補検索の追加境界

- 被引用候補検索は、既存検索と同じ統合検索 HTML を使用する。
- 追加 capability は、事件番号と存在する場合の判例集表記だけを検索語として使う。
- 検索語、HTML 本文、URL query または結果本文をエラー、診断またはログへ含めない。
- citation pack が無効な場合は、この capability の route を登録しない。
- origin は `https://www.courts.go.jp` に固定し、認証、cookie、proxy、任意 URL、background crawling、恒久 cache、稼働監視および自動再試行を追加しない。
- HTML parser は DOM として解析し、script、stylesheet、画像、PDFその他の埋込み resource を取得又は実行しない。
- 候補の詳細 HTML 又は PDF を連鎖取得しない。

## 資源予算

既存 search/read の資源予算を変更しない。候補検索は一 operation 内の最大二 response を合算し、次の上限を使用する。

| operation | artifact | `budgetKey` | `responseBytes` | `decompressedBytes` | `entriesOrObjects` | `depth` | `parseTimeout` | `concurrencyGroup` | `concurrency` |
|---|---|---|---:|---:|---:|---:|---:|---|---:|
| citing-candidate-search | HTML | `judicial-citing-candidate-search-html` | 4 MiB | 8 MiB | 200000 | 64 | 4s | `courts-hanrei-html` | 1 |

最初の request 前に共有同時実行枠を一回取得し、最大二回の取得、解析および mapping が全て終了するまで保持する。成功、失敗、timeout および取消の全経路で枠を返却する。

## 確認

descriptor の全固定値と三能力 inventory、既存二能力契約、pack 無効時の citation route 非登録、固定 origin、最大二 request、候補 resource 非取得、資源予算、全終了経路での同時実行枠解放、および公式 HTML fixture を契約テストで確認する。

## 関連

- [SOT-IF-069: `judicial-decision.citing-candidate.search` capability v1](69-judicial-citing-candidate-search-capability.md)
- [SOT-IF-073: 裁判所検索の被引用候補マッピング](73-courts-hanrei-citing-candidate-search-mapping.md)
