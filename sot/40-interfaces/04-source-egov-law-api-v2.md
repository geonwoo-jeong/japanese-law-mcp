# SOT-IF-004: e-Gov 法令 API Version 2

- 状態: 有効

## 規定

e-Gov 法令 API Version 2 は、Japanese Law MCP が日本の法令を検索および取得するための一次情報源とし、運用時の接続先と外部フィールドの定義はデジタル庁が公開する公式 OpenAPI 仕様に固定する。

## 公式仕様

- 提供主体: デジタル庁
- 仕様: [e-Gov 法令 API Version 2](https://laws.e-gov.go.jp/api/2/redoc/)
- 確認した仕様バージョン: `2.1.139`
- 確認日: `2026-08-23`
- API ベース URL: `https://laws.e-gov.go.jp/api/2`
- 情報源 ID: `e-gov-law-api-v2`
- 位置付け: 公式情報

## ProviderDescriptor

e-Gov 法令 API Version 2 アダプターの `ProviderDescriptor` は、次の値に固定する。

| 項目 | 値 |
|---|---|
| `providerId` | `e-gov-law-api-v2` |
| `source.id` | `e-gov-law-api-v2` |
| `source.name` | `e-Gov 法令 API Version 2` |
| `source.publisher` | `デジタル庁` |
| `source.authority` | `official` |
| `source.serviceUrl` | `https://laws.e-gov.go.jp/api/2/redoc/` |
| `adapterContractVersion` | `1.2.0` |
| `upstreamSpecVersion` | `2.1.139` |
| `verifiedAt` | `2026-08-23` |
| `interfaceType` | `api` |
| `credentialRequired` | `false` |

`capabilities` は、次の順序付き配列とする。

1. `law.article.read@1`、`level: core`、`stability: stable`
2. `law.content.search@1`、`level: core`、`stability: stable`
3. `law.document.read@1`、`level: core`、`stability: stable`
4. `law.revision.list@1`、`level: core`、`stability: stable`
5. `law.search@1`、`level: core`、`stability: stable`
6. `law.version.compare@1`、`level: core`、`stability: stable`

このアダプターに利用者が変更できる provider-specific setting と credential slot はなく、`settings` と `credentialEnvRefs` は空の object とする。

## プロバイダー設定

接続 origin は `https://laws.e-gov.go.jp`、API base path は `/api/2` に固定し、利用者による上書きを許可しない。ambient proxy と明示 proxy はともに使用せず、proxy の設定 scope は固定文字列 `n/a` とする。dataset、tenant および account の設定 scope も固定文字列 `n/a`、semantic configuration は空の object、credential slot fingerprints は空の object とする。

`SOT-IF-026` の provider configuration scope は、次の object を `SOT-IF-026` の方法で canonicalize して生成する。

```json
{
  "account": "n/a",
  "credentialSlots": {},
  "dataset": "n/a",
  "origin": "https://laws.e-gov.go.jp",
  "providerId": "e-gov-law-api-v2",
  "proxy": "n/a",
  "semanticConfig": {},
  "tenant": "n/a"
}
```

製品版、User-Agent、request timeout、diagnostics、transport および外部呼出しの現在時刻は、検索結果または取得位置の意味を変えないため、この scope に含めない。

## 利用範囲

Japanese Law MCP は、次の利用目的に必要な操作だけを使用する。

| e-Gov operation | 利用目的 | マッピング SOT |
|---|---|---|
| `GET /laws` | 法令名検索 | `SOT-IF-054` |
| `GET /keyword` | 法令本文検索 | `SOT-IF-010`、`SOT-IF-028` |
| `GET /law_data/{law_id_or_num_or_revision_id}` | 法令本文取得 | `SOT-IF-011` |
| `GET /law_data/{law_id_or_num_or_revision_id}` | 条文取得 | `SOT-IF-012` |
| `GET /law_data/{law_id_or_num_or_revision_id}` | 法令版間比較 | `SOT-IF-060` |
| `GET /law_revisions/{law_id_or_num}` | 法令改正履歴 | `SOT-IF-057` |

外部 API のリクエスト項目とレスポンス項目は、公式 OpenAPI 仕様を定義元とする。プロジェクト内のモデルへの変換だけを各マッピング SOT で定義する。

## 試行提供機能

確認した仕様では、次の機能が試行提供と明記されている。

- 法令本文取得 API が返す JSON 形式の本文
- 法令本文ファイル取得 API が返す JSON ファイル
- キーワード検索 API で名称に `law_num` を含むパラメータを指定した場合のレスポンス

公式機能として使用する場合は、対応するマッピング SOT に試行提供であることと変更検出方法を明記する。現在の法令本文取得と条文取得は XML を使用し、試行提供の JSON 本文には依存しない。

## 取得条件

確認した公式 OpenAPI 仕様には、認証方式、User-Agent または連絡先の指定、数値の呼び出し間隔、同時実行上限、日次上限および再利用条件の記載がない。これらを公式条件として推測しない。

このプロバイダーは、次の保守的な取得条件を使用する。

- 一つのプロセスで同時に実行する e-Gov 呼び出しは一件までとし、完了した呼び出しの履歴を保存しない。
- 一つの MCP リクエストが複数の e-Gov 呼び出しを順に行う場合は、呼び出しの開始間隔を一秒以上とする。
- 有効な `Retry-After` が返された場合は指数 backoff に代えてその値を使用するが、
  呼出し開始間隔の一秒を下限とする。現在のリクエスト期限を超える場合は
  再試行しない。
- `429`、`500`、`502`、`503` および `504` だけを自動再試行の候補とし、一秒から始めて最大八秒までの指数 backoff を使用し、同じ MCP リクエスト内で最大三回までとする。
- その他の `4xx`、構造不一致、形式不一致および安全上の上限超過を再試行しない。
- User-Agent には製品名と実行中の版を含め、検索語、法令識別子、認証情報または利用主体を含めない。

共通 capability の `asOf` が `2017-04-01` より前の場合は、e-Gov 法令 API Version 2 の対象期間外として、外部呼出し前に `unsupported_query` を返す。既存 MCP facade は各公開入力 SOT が同じ下限を入力制約として持つため、`invalid_argument` を返す。

各 operation の資源予算は次のとおりとする。

| `budgetKey` | operation | artifact | `responseBytes` | `decompressedBytes` | `entriesOrObjects` | `depth` | `parseTimeout` | `concurrencyGroup` | `concurrency` |
|---|---|---|---:|---:|---:|---:|---:|---|---:|
| `laws-json` | `GET /laws` | JSON | 8 MiB | 16 MiB | 200000 | 32 | 3s | `egov-http` | 1 |
| `keyword-json` | `GET /keyword` | JSON | 16 MiB | 32 MiB | 500000 | 64 | 5s | `egov-http` | 1 |
| `law-data-xml` | `GET /law_data/{law_id_or_num_or_revision_id}` | XML | 16 MiB | 32 MiB | 500000 | 128 | 5s | `egov-http` | 1 |
| `law-revisions-json` | `GET /law_revisions/{law_id_or_num}` | JSON | 8 MiB | 16 MiB | 200000 | 32 | 3s | `egov-http` | 1 |

HTTP content coding による展開がない場合も `decompressedBytes` は解析へ渡す byte 数の上限として適用する。法令本文が予算を超える場合は、部分的な本文を成功として返さず `source_response_too_large` とする。絶対 ceiling と同じ値を使用する `GET /keyword` および法令本文取得は、公式仕様が一回の検索で最大 1000 の一致位置を返し、法令本文のデータサイズが大きい場合があることを理由とする。

四つの budget row は同じ `egov-http` group を共有するため、operation の種類に関係なく e-Gov の外部呼出し、再試行および解析は process 全体で同時に一件までとする。

公式条件に新しい制限または要件が示された場合は、この SOT と契約 fixture を更新してから適用する。原文 URL、取得時点および変換方法は `Provenance` として返し、公式資料で確認できない再配布権または完全性を応答で主張しない。

## 確認

外部ネットワークを使わない fake transport と制御可能な時計を使い、
`429`、`500`、`502`、`503` および `504` だけを再試行候補とすること、
初回一回と最大三回の再試行を合わせた HTTP attempt が四回を超えないこと、
再試行開始間隔が一秒、二秒、四秒であり八秒を上限とすることを確認する。
有効な `Retry-After` は指数 backoff に代えて使用し、一秒未満なら一秒へ
引き上げることを確認する。request deadline を超える待機、その他の `4xx`、
構造不一致、形式不一致および安全上限超過では再試行しないことも確認する。

`laws-json`、`keyword-json`、`law-data-xml` および `law-revisions-json` は、各 byte、構造および
解析時間の上限値を受理し、一単位または一 byte 超過したときに部分結果を
返さないことを fixture で確認する。四 operation を同じ `egov-http`
concurrency group へ接続し、一つが外部呼出しから解析まで枠を保持している間に
別 operation を開始すると、後の処理が外部呼出しへ到達せず `source_busy` と
なることを確認する。cancel、timeout および parser failure の全経路で枠を
解放する。

これらを `SOT-ENG-017` の `outbound-request`、`error-normalization`、
`response-bytes-limit`、`decompressed-bytes-limit`、
`entries-or-objects-limit`、`depth-limit`、`parse-timeout`、
`concurrency-limit` および `cancellation` に接続し、operation ごとの
provider contract test と共通 conformance test の両方から到達可能にする。

## 関連

- [SOT-PROD-003: 法情報の採用基準](../00-product/03-legal-source-eligibility.md)
- [SOT-PROD-007: 情報源取得方式の選択](../00-product/07-source-acquisition-policy.md)
- [SOT-ARCH-004: 情報源アダプター境界](../30-architecture/04-source-adapter-boundary.md)
- [SOT-IF-015: 情報源操作の共通契約](15-source-operation-contract.md)
- [SOT-IF-026: プロバイダールーティング設定](26-provider-routing-configuration.md)
- [SOT-IF-057: e-Gov 法令改正履歴のマッピングと組込み採用](57-egov-law-revision-mapping-and-adoption.md)
- [SOT-IF-060: e-Gov 法令版間比較のマッピングと組込み採用](60-egov-law-version-comparison-mapping-and-adoption.md)
- [SOT-ENG-005: SOT と変更の整合](../50-engineering/05-sot-change-unit.md)
- [SOT-ENG-016: プロバイダー資源予算](../50-engineering/16-provider-resource-budgets.md)
- [SOT-ENG-017: プロバイダー適合性 matrix](../50-engineering/17-provider-conformance-matrix.md)
