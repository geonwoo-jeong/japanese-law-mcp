# SOT-IF-070: 裁判所「裁判例検索」PDF 情報源

- 状態: 有効

## 規定

`courts-hanrei-pdf` は、裁判所「裁判例検索」の詳細 HTML が直接示す `full_text` PDF を安全に読み取り、`judicial-decision.case-citation.extract@1` だけを提供する組込み provider とする。

## 識別

| 項目 | 値 |
|---|---|
| `providerId` | `courts-hanrei-pdf` |
| `source.id` | `courts-hanrei` |
| `source.name` | `裁判所 裁判例検索` |
| `source.publisher` | `最高裁判所` |
| `source.authority` | `official` |
| `source.serviceUrl` | `https://www.courts.go.jp/hanrei/search1/index.html` |
| `adapterContractVersion` | `1.0.0` |
| `upstreamSpecVersion` | 省略 |
| `verifiedAt` | `2026-08-26` |
| `interfaceType` | `pdf` |
| `credentialRequired` | `false` |

descriptor は `judicial-decision.case-citation.extract@1` だけを `extended`、`stable` として宣言する。

## 接続境界

- origin は `https://www.courts.go.jp` に固定し、設定で変更しない。
- 許可する URL は `https://www.courts.go.jp/assets/hanrei/` 配下の `application/pdf` だけとする。
- 入力で PDF URL を自由に受け取らず、同じ request で検証済みの `JudicialDocumentLink` だけを受理する。
- OCR、外部 font・resource 取得、埋込みファイル展開、自動再試行、background crawling、恒久 cache および稼働監視を追加しない。
- parser は同一 binary の隔離 worker で実行し、timeout または panic が親 MCP process を終了させない。

## 資源予算

| operation | artifact | `budgetKey` | `responseBytes` | `decompressedBytes` | `entriesOrObjects` | `depth` | `parseTimeout` | `concurrencyGroup` | `concurrency` |
|---|---|---|---:|---:|---:|---:|---:|---|---:|
| extract | PDF | `judicial-citation-pdf` | 16 MiB | 24 MiB | 50000 | 32 | 4s | `courts-hanrei-pdf` | 1 |

追加の上限として、ページ数 300、抽出 text 2 MiB、citation occurrence 256 件を超えない。

## エラー

HTTP 404 は `not_found` とする。timeout、一時的接続失敗、5xx、MIME 不一致、PDF magic 不一致、暗号化、過大 object、過大ページ数、worker timeout、解析 panic、資源上限超過および text layer 不在は、`SOT-IF-017` に従って正規化する。text layer 不在は provider エラーではなく capability の成功出力へ反映できる。

## 確認

descriptor 固定値、固定 origin、`full_text` URL 制限、同時実行 1、worker 隔離、MIME/magic 検証、ページ数・object・展開量・抽出 text・timeout 上限、暗号化 PDF と image-only PDF の取扱い、および原文非保存を契約テストで確認する。

## 関連

- [SOT-ARCH-014: 外部原文の一時処理](../30-architecture/14-ephemeral-source-artifacts.md)
- [SOT-ARCH-043: 判例引用追跡のオンデマンド一時組立て](../30-architecture/43-on-demand-judicial-citation-assembly.md)
- [SOT-IF-068: `judicial-decision.case-citation.extract` capability v1](68-judicial-case-citation-extract-capability.md)
