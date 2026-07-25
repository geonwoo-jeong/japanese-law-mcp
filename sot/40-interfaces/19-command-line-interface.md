# SOT-IF-019: コマンドラインインターフェース

- 状態: 有効

## 規定

公式実行ファイルは、引数を指定しないルートコマンドでサーバーを起動し、実行設定を上書きするフラグ、ヘルプ表示およびバージョン表示を提供する。

## コマンド

| 形式 | 動作 |
|---|---|
| `japanese-law-mcp [flags]` | 検証済みの実行設定でサーバーを起動する。設定を上書きしない場合は `SOT-IF-005` の既定値に従い `stdio` を使用する |
| `japanese-law-mcp --help` | 利用方法、使用できるフラグおよび `version` コマンドを日本語で標準出力へ表示する |
| `japanese-law-mcp version` | 実行ファイルへ埋め込まれたバージョンを標準出力へ一行で出力する |

ルートコマンドは位置引数を受け付けない。`--help` と `version` はサーバーを起動しない。

## フラグ

| フラグ | 対応する値 | 指定方法 |
|---|---|---|
| `--help` | 日本語のヘルプ | boolean |
| `--config` | 読み込む設定ファイル | 一つのファイルパス |
| `--transport` | `transport` | 一つの値 |
| `--request-timeout` | `requestTimeout` | 一つの duration |
| `--listen-address` | `listenAddress` | 一つの host:port |
| `--allowed-origin` | `allowedOrigins` | Origin ごとに繰り返し指定 |
| `--diagnostics` | `diagnostics` | boolean |

各設定値の意味、制約および既定値は `SOT-IF-005` を定義元とする。利用者向けの説明とエラーは日本語で出力し、サーバーの `stdio` 実行中は `SOT-DEL-001` に従って標準出力を MCP メッセージ専用とする。

## 確認

各コマンドとフラグについて、動作、標準出力、標準エラー、サーバー起動の有無および無効な位置引数の拒否を CLI 契約テストで確認する。

## 関連

- [SOT-IF-005: 実行設定](05-runtime-configuration.md)
- [SOT-IF-020: 設定ソースと優先順位](20-configuration-sources-and-precedence.md)
- [SOT-IF-021: プロセス終了コード](21-process-exit-codes.md)
- [SOT-DEL-001: stdio](../60-delivery/01-stdio.md)
