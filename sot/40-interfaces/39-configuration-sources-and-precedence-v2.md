# SOT-IF-039: 設定ソースと優先順位 v2

- 状態: 有効

## 規定

実行設定は、コマンドラインフラグ、環境変数、選択された一つの設定ファイル、既定値の順に優先し、項目ごとに最も優先度の高い入力を採用する。

## 優先順位

```text
コマンドラインフラグ
> 環境変数
> 明示した設定ファイルまたは利用者設定ファイル
> SOT-IF-029、SOT-IF-026、SOT-IF-067 および SOT-IF-061 の既定値
```

`--config` を指定した場合は、そのファイルだけを設定ファイルとして使用する。ファイルが存在しない、読み取れない、対応する形式として解析できない、または拡張子が `.yaml`、`.yml`、`.json`、`.toml` のいずれでもない場合は設定エラーとする。

設定ファイルは 1 MiB 以下、object と配列の入れ子は 16 段以下とする。重複 key、YAML の alias、anchor、merge key、custom tag、および JSON、YAML または TOML の仕様外拡張を拒否する。制限超過または曖昧な構造は、値の一部を採用せず設定エラーとする。

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

設定ファイルが持てる最上位項目は、`SOT-IF-029` が定義する `transport`、`requestTimeout`、`listenAddress`、`allowedOrigins` および `diagnostics`、`SOT-IF-026` が定義する `providers` および `providerRoutes`、ならびに `SOT-IF-067` と `SOT-IF-061` が定義する `extensionPacks` だけとする。未知の項目、型が異なる値、無効な形式および成立しない組合せは、サーバー起動前の設定エラーとする。

`providers` と `providerRoutes` の階層ごとの優先順位、空値および atomic な route object の扱いは `SOT-IF-026` に従う。`extensionPacks` の値、空値および既定値は、`judicial-cases` と `judicial-citations` については `SOT-IF-067`、`legislative-history` については `SOT-IF-061` に従う。秘密値そのものは、この設定ファイルまたはコマンドライン引数から受け取らない。

`providers`、`providerRoutes` および `extensionPacks` は、個別のコマンドラインフラグまたは設定値を表す環境変数を持たない。これら三つの名前空間では、選択された設定ファイルと有効な SOT が定義する組込み既定値だけを使用する。`credentialEnvRefs` が指す環境変数は秘密値の取得元であり、設定構造の入力元または優先順位の一段として扱わない。

## 確認

同じ項目へ異なる値を各入力元から与えた契約テストで優先順位を確認する。明示ファイルの欠落、利用者設定ファイルの不在、作業ディレクトリの非検索、対応する各ファイル形式、サイズと深さの境界、重複 key、YAML の alias 等、および未知の項目の拒否も確認する。

`extensionPacks` が設定ファイルだけから読み込まれ、環境変数または個別フラグで変更できないことも確認する。

## 関連

- [SOT-IF-029: ローカル実行設定](29-local-runtime-configuration.md)
- [SOT-IF-026: プロバイダールーティング設定](26-provider-routing-configuration.md)
- [SOT-IF-067: `judicial-cases` と `judicial-citations` の有効化](67-judicial-citations-pack-activation.md)
- [SOT-IF-061: `legislative-history` 拡張パックの専門公開面](61-legislative-history-pack-activation.md)
- [SOT-IF-019: コマンドラインインターフェース](19-command-line-interface.md)
- [SOT-ARCH-015: 起動時設定境界](../30-architecture/15-startup-configuration-boundary.md)
