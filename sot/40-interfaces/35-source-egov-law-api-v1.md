# SOT-IF-035: e-Gov 法令 API Version 1 更新一覧

- 状態: 有効

## 規定

e-Gov 法令 API Version 1 の更新法令一覧 API を、`law.update.list@1` の公式情報源として使用する。

## 公式仕様と記述子

- 公式仕様: [e-Gov 法令 API Version 1](https://laws.e-gov.go.jp/docs/law-data-basic/8529371-law-api-v1/)
- 確認日: `2026-07-26`
- operation: `GET https://laws.e-gov.go.jp/api/1/updatelawlists/{yyyyMMdd}`
- 認証、query parameter、pagination: なし
- 収録開始日: `2020-11-24`
- response: `text/xml;charset=UTF-8`

`ProviderDescriptor` は次に固定する。

| 項目 | 値 |
|---|---|
| `providerId`、`source.id` | `e-gov-law-api-v1` |
| `source.name` | `e-Gov 法令 API Version 1` |
| `source.publisher` | `デジタル庁` |
| `source.authority` | `official` |
| `source.serviceUrl` | 公式仕様 URL |
| `adapterContractVersion` | `1.0.0` |
| `upstreamSpecVersion` | `1` |
| `verifiedAt` | `2026-07-26` |
| `interfaceType` | `api` |
| `credentialRequired` | `false` |
| `capabilities` | `law.update.list@1`、`core`、`stable` |

接続 origin、path、HTTP method を利用者設定で変更できない。ambient proxy、provider-specific setting および credential slot は使用しない。日付は path segment へ `yyyyMMdd` で埋め込み、query を送らない。

`2020-11-24` より前、または呼出し開始時点を `Asia/Tokyo` で暦日にした日より後の対象日は、HTTP 呼出し前に `unsupported_query` とする。

## 取得条件と予算

一つの process でこのプロバイダーの外部呼出し、再試行、展開、解析および mapping を同時に一件だけ実行する。`429` と `5xx` だけを一秒、二秒、四秒の backoff で最大三回再試行する。外部の `Retry-After` が有効でリクエスト期限内なら優先する。それ以外の status、契約不一致、値不正および予算超過を再試行しない。

| `budgetKey` | artifact | `responseBytes` | `decompressedBytes` | `entriesOrObjects` | `depth` | `parseTimeout` | `concurrencyGroup` | `concurrency` |
|---|---|---:|---:|---:|---:|---:|---|---:|
| `update-law-list-xml` | XML | 2 MiB | 4 MiB | 16384 | 4 | 2s | `egov-v1-http` | 1 |

transfer body、gzip 展開結果、XML 構造単位、XML depth および解析時間をそれぞれ上限内に収める。`entriesOrObjects` は element、attribute、namespace declaration、空白だけではない text または CDATA、comment および processing instruction を一つの counter で数える。超過は `SOT-ENG-016` に従い、部分的な一覧を成功として返さない。XML directive、namespace、属性、未知または重複した構造を受理しない。

XML 構造単位とは別に、`LawNameListInfo` は一応答につき 512 件を上限とする。512 件目までは受理し、513 件目を検出した時点で `source_response_too_large` とする。項目数は各 `LawNameListInfo` を一件として数え、その子要素の有無によって増減させない。

`entriesOrObjects` の 16384 は、一項目につき `LawNameListInfo` 自身の一要素、確認済みの最大十五の子要素および各子要素の空でない text を合計した最大三十一構造単位を、512 件分と固定外側構造、XML 宣言および診断用の余裕を含めて受理できる上限である。2026 年 9 月 1 日に公式 endpoint から取得した `2026-06-24` の応答は、208 件、201970 bytes およびこの計測規則で 6007 構造単位であり、この上限内に収まる。共通 ceiling より小さく、確認済みの完全応答を拒否しない最小の二進単位として 16384 を採用する。

`parseTimeout` は HTTP response body の transfer 受信を終えた後に開始し、identity または gzip の展開開始から XML 解析完了まで同じ期限を共有する。network 接続、response header 待機および transfer 受信時間をこの処理期限へ含めない。

HTTP、通信、content type、圧縮、XML および値の失敗は `SOT-IF-017` へ正規化し、外部本文、対象日以外の入力、内部 path または秘密値をエラーへ含めない。

## 確認

公式の `2026-06-24` 応答と同じ 208 件、かつ 6007 構造単位以上を持つ合成応答が完全な一覧として成功することを確認する。XML 構造単位と `LawNameListInfo` 項目数について、それぞれ上限と等しい入力を受理し、一つ超過した入力を部分結果にせず `source_response_too_large` とすることを別々に確認する。

## 関連

- [SOT-IF-034: `law.update.list` capability v1](34-law-update-list-capability.md)
- [SOT-IF-036: e-Gov 更新法令一覧マッピング](36-egov-law-update-list-mapping.md)
- [SOT-ENG-016: プロバイダー資源予算](../50-engineering/16-provider-resource-budgets.md)
