# SOT-IF-005: 実行設定

- 状態: 有効

## 規定

実行設定は、トランスポート、外部リクエストのタイムアウト、HTTP の待受先と許可する Origin、および一時診断の有効化を指定する一つの設定インターフェースとして提供する。

## 設定項目

| 名前 | 型 | 必須 | 既定値 | 適用範囲 |
|---|---|---:|---|---|
| `transport` | `stdio` または `streamable-http` | いいえ | `stdio` | 全実行方式 |
| `requestTimeout` | duration | いいえ | `30s` | 外部リクエスト。設定可能範囲は 1 秒以上 120 秒以下 |
| `listenAddress` | host:port | HTTP の場合のみ | `127.0.0.1:8080` | Streamable HTTP |
| `allowedOrigins` | HTTPS Origin の配列 | いいえ | 空の配列 | Streamable HTTP |
| `diagnostics` | boolean | いいえ | `false` | 全実行方式 |

`allowedOrigins` が空の場合、`Origin` ヘッダーを含むリクエストを許可しない。`Origin` ヘッダーのない非ブラウザー接続は受け付ける。

未知の設定項目、無効な形式および必要条件を満たさない組合せは、起動時にエラーとして扱う。

## 関連

- [SOT-DEL-001: stdio](../60-delivery/01-stdio.md)
- [SOT-DEL-002: Streamable HTTP](../60-delivery/02-streamable-http.md)
- [SOT-ARCH-008: 一時的な診断](../30-architecture/08-ephemeral-diagnostics.md)
