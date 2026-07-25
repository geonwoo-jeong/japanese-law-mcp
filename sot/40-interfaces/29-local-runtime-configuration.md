# SOT-IF-029: ローカル実行設定

- 状態: 有効

## 規定

実行設定は、利用者のローカル環境で使う transport、外部 request の timeout、loopback HTTP の待受先と許可する Origin、および一時診断の有効化を指定する一つの設定インターフェースとして提供する。

## 設定項目

| 名前 | 型 | 必須 | 既定値 | 適用範囲 |
|---|---|---:|---|---|
| `transport` | `stdio` または `streamable-http` | いいえ | `stdio` | 全実行方式 |
| `requestTimeout` | duration | いいえ | `30s` | 外部 request。設定可能範囲は 1 秒以上 120 秒以下 |
| `listenAddress` | host:port | HTTP の場合のみ | `127.0.0.1:8080` | ローカル Streamable HTTP |
| `allowedOrigins` | HTTPS Origin の配列 | いいえ | 空の配列 | ローカル Streamable HTTP |
| `diagnostics` | boolean | いいえ | `false` | 全実行方式 |

`allowedOrigins` が空の場合、`Origin` header を含む request を許可しない。`Origin` header のない非 browser 接続は受け付ける。

`listenAddress` の host は IP literal の `127.0.0.0/8` または `::1` だけを許可する。hostname、`0.0.0.0`、`::`、private address、link-local address、Unix domain socket および外部 address は、名前解決または接続を行う前に起動時の設定エラーとする。

`diagnostics` は `SOT-ARCH-008` の一時出力だけを有効にする。health endpoint、`liveness`、`readiness`、稼働率収集または運用障害検知を有効にしない。

未知の設定項目、無効な形式および必要条件を満たさない組合せは、起動時にエラーとして扱う。

## 確認

既定値、各境界値および未知項目を契約テストで確認する。すべての非 loopback 待受先が、socket の作成と外部接続より前に拒否されることを確認する。

## 関連

- [SOT-DEL-001: stdio](../60-delivery/01-stdio.md)
- [SOT-DEL-013: ローカル Streamable HTTP](../60-delivery/13-local-streamable-http.md)
- [SOT-ARCH-008: 一時的な診断](../30-architecture/08-ephemeral-diagnostics.md)
- [SOT-IF-026: プロバイダールーティング設定](26-provider-routing-configuration.md)
