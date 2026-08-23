# SOT-IF-063: 国立国会図書館の国会発言検索 API 情報源

- 状態: 有効

## 規定

`ndl-diet-speech-api` は、国立国会図書館が公式に文書化した国会会議録検索システムの発言単位 JSON API を安全に読み取り、`parliament.speech.search@1` を提供する provider とする。

## 識別

| 項目 | 値 |
|---|---|
| `providerId` | `ndl-diet-speech-api` |
| `source.id` | `ndl-diet-records` |
| `source.name` | `国会会議録検索システム` |
| `source.publisher` | `国立国会図書館` |
| `source.authority` | `official` |
| `source.serviceUrl` | `https://kokkai.ndl.go.jp/` |
| `adapterContractVersion` | `1.0.0` |
| `upstreamSpecVersion` | 省略 |
| `verifiedAt` | `2026-08-23` |
| `interfaceType` | `json-api` |
| `credentialRequired` | `false` |

descriptor は `parliament.speech.search@1` だけを `extended`、`stable` として宣言する。

## 公式公開面と利用条件

- 発言単位 API: `https://kokkai.ndl.go.jp/api/speech`
- API 仕様: `https://kokkai.ndl.go.jp/api.html`
- サービス: `https://kokkai.ndl.go.jp/`
- 利用規約: `https://www.ndl.go.jp/sitepolicy/terms`

API の利用に手続きは不要である。User-Agent、連絡先、日次上限、`Retry-After` および固定した backoff 値は、確認済みの公式仕様に必須値が示されていないため推測しない。

公式仕様は、多重リクエストを避け、データ取得完了から数秒程度を空けることを求める。
3 秒は公式仕様が示した固定値ではなく、「数秒程度」を実装可能にするため本製品が
保守的に定める値である。この provider は一つの process 内で発言検索を一件だけ
同時実行し、レスポンス本文の取得完了後も同じ gate を 3 秒保持する。取得、解析および
mapping が全て終了し、かつ本文取得完了から 3 秒が経過した後にだけ gate を解放する。
gate は operation の返却前に解放し、完了済み呼出しの時刻または回数を別の request へ
持ち越さない。

context が取得中または 3 秒の待機中に取り消された場合は直ちに gate を解放し、成功結果を返さない。自動再試行を行わず、一つの operation で外部 HTTP request を一回だけ送る。

## 接続境界

- origin と endpoint は `https://kokkai.ndl.go.jp/api/speech` に固定し、設定で変更しない。
- method は `GET`、応答形式は `recordPacking=json` に固定する。
- 認証、cookie、proxy、任意 header、任意 URL、background crawling、恒久 cache および稼働監視を追加しない。
- redirect は追跡しない。3xx 応答を同じ origin の別 path または同じ endpoint へ
  自動で再送せず、`invalid_source_response` とする。
- percent-encode 後の scheme、authority、path、`?` および query を含む要求 URL 全体を UTF-8 で 2000 byte 以下とし、超過する入力は外部呼出し前に `invalid_argument` とする。
- HTTP response body 以外の script、画像、PDF または別 URL を取得しない。

## 構成 scope

provider configuration scope は次の固定値とする。

| key | 値 |
|---|---|
| `providerId` | `ndl-diet-speech-api` |
| `origin` | `https://kokkai.ndl.go.jp` |
| `dataset` | `diet-speech` |
| `tenant` | `n/a` |
| `account` | `n/a` |
| `proxy` | `n/a` |
| `semanticConfig` | 空の object |
| `credentialSlots` | 空の object |

## 資源予算

| operation | artifact | `budgetKey` | `responseBytes` | `decompressedBytes` | `entriesOrObjects` | `depth` | `parseTimeout` | `concurrencyGroup` | `concurrency` |
|---|---|---|---:|---:|---:|---:|---:|---|---:|
| search | JSON | `diet-speech-search-json` | 8 MiB | 16 MiB | 200000 | 32 | 2s | `ndl-diet-speech-api` | 1 |

HTTP content coding を復号した JSON を parser へ渡す。byte、JSON value、深さ、解析時間および同時実行枠は `SOT-ENG-016` に従って実測する。取得済み本文の復号と解析には上表の `parseTimeout` を適用する。

外向き HTTP request は、呼出し元の context により短い期限がある場合はその期限を
優先し、それ以外でも開始から 20 秒を絶対上限とする provider-local context で実行
する。この上限は response header の受信だけでなく、上表の `responseBytes` までの
body 読取りを含む。20 秒を超えた場合は body の部分結果を破棄して
`source_timeout` とし、同じ要求を自動再試行しない。取得後の復号と解析には、親
context と独立した新しい root context を作らず、親 context に上表の
`parseTimeout` を追加した子 context を使用する。復号、JSON 解析および mapping は
この同じ子 context の期限内に完了させ、gate はその成否にかかわらず operation の
終了処理まで保持する。

## エラー

- HTTP 429 は `rate_limited`、timeout は `source_timeout`、一時的な接続失敗と 5xx は `source_unavailable` とする。
- 3xx は追跡せず `invalid_source_response` とする。
- API の混雑を示す error code `19001` は `source_unavailable` とする。公式応答に待機値がない場合は `retryAfter` を合成しない。
- 共通入力検証と要求 URL の上限を通過した後に API が返した検索条件 error は `invalid_source_response` とし、外部 message または details を公開しない。
- 成功時 media type が JSON でない場合、JSON を一つの object として解析できない場合、重複 key、trailing data、型の不一致または page 不変条件の違反は `invalid_source_response` とする。
- 保存した公式契約自体の変更を確認した場合だけ `source_contract_changed` とする。一件の不正応答を契約変更へ読み替えない。
- 応答または展開後の byte 上限、構造上の危険および解析時間上限は `SOT-ENG-016` と `SOT-IF-017` に従う。

エラー、診断、ログまたは test failure に、検索条件、URL query、発言本文、発言者名、外部 error body または取得内容を含めない。

## 確認

descriptor の固定値、一能力だけの inventory、固定 endpoint、redirect の非追跡、認証と追加 resource の禁止、2000 byte の要求 URL、20 秒の HTTP 上限、単一 HTTP request、同時実行一件、取得完了後 3 秒の gate、取消、timeout、panic、解析失敗および mapping 失敗を含む全終了経路での解放、自動再試行の禁止、全資源予算、JSON の安全な解析、公式 error および秘密・検索内容の非露出を契約テストで確認する。

## 関連

- [SOT-PROD-014: 立法過程拡張パックの国会発言検索](../00-product/14-legislative-history-extension-pack.md)
- [SOT-IF-014: ProviderDescriptor](14-provider-descriptor.md)
- [SOT-IF-017: 情報源エラーの正規化](17-source-error-normalization.md)
- [SOT-IF-062: `parliament.speech.search` capability v1](62-parliament-speech-search-capability.md)
- [SOT-IF-064: 国立国会図書館の国会発言検索マッピング](64-ndl-diet-speech-search-mapping.md)
- [SOT-ENG-016: プロバイダー資源予算](../50-engineering/16-provider-resource-budgets.md)
