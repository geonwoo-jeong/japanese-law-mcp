# SOT-IF-020: 設定ソースと優先順位

- 状態: 有効

## 規定

実行設定は、コマンドラインフラグ、環境変数、選択された一つの設定ファイル、既定値の順に優先し、項目ごとに最も優先度の高い入力を採用する。

## 優先順位

```text
コマンドラインフラグ
> 環境変数
> 明示した設定ファイルまたは利用者設定ファイル
> SOT-IF-005 の既定値
```

`--config` を指定した場合は、そのファイルだけを設定ファイルとして使用する。ファイルが存在しない、読み取れない、対応する形式として解析できない、または拡張子が `.yaml`、`.yml`、`.json`、`.toml` のいずれでもない場合は設定エラーとする。

`--config` を指定しない場合は、OS が返す利用者設定ディレクトリにある `japanese-law-mcp/config.yaml` だけを自動検索する。そのファイルが存在しない場合は設定ファイルを使用せず、既定値を適用する。現在の作業ディレクトリと、その親ディレクトリは自動検索しない。

自動検索した設定ファイルが存在していても、読み取れない場合または YAML として解析できない場合は設定エラーとする。

## 環境変数

| 環境変数 | 対応する値 |
|---|---|
| `JAPANESE_LAW_MCP_TRANSPORT` | `transport` |
| `JAPANESE_LAW_MCP_REQUEST_TIMEOUT` | `requestTimeout` |
| `JAPANESE_LAW_MCP_LISTEN_ADDRESS` | `listenAddress` |
| `JAPANESE_LAW_MCP_ALLOWED_ORIGINS` | `allowedOrigins` |
| `JAPANESE_LAW_MCP_DIAGNOSTICS` | `diagnostics` |

`JAPANESE_LAW_MCP_ALLOWED_ORIGINS` は、Origin をカンマで区切り、各値の前後にある空白を除いた配列として解釈する。空文字列を指定した場合は空の配列として扱い、設定ファイルで指定した Origin をすべて上書きする。

設定ファイルが持てる最上位項目は、`SOT-IF-005` が定義する `transport`、`requestTimeout`、`listenAddress`、`allowedOrigins` および `diagnostics` だけとする。未知の項目、型が異なる値、無効な形式および成立しない組合せは、サーバー起動前の設定エラーとする。

## 確認

同じ項目へ異なる値を各入力元から与えた契約テストで優先順位を確認する。明示ファイルの欠落、利用者設定ファイルの不在、作業ディレクトリの非検索、対応する各ファイル形式および未知の項目の拒否も確認する。

## 関連

- [SOT-IF-005: 実行設定](05-runtime-configuration.md)
- [SOT-IF-019: コマンドラインインターフェース](19-command-line-interface.md)
- [SOT-ARCH-015: 起動時設定境界](../30-architecture/15-startup-configuration-boundary.md)
