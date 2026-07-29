# Japanese Law MCP

Japanese Law MCP は、日本の公式法情報を AI エージェントや LLM から検索・取得するための、Go 製 MCP サーバーです。利用者のローカル環境で動作し、法情報の取得先へ直接接続します。

法令コアはデジタル庁の e-Gov 法令 API Version 2 と Version 1 を使用します。選択型の裁判例拡張パックは、最高裁判所の「裁判例検索」が公表する掲載情報を使用します。

> [!IMPORTANT]
> このツールは法的判断や法的助言を行いません。結果には公式情報源を確認するための情報を含めますが、法令の適用や裁判例の先例性は利用者が原文と現在の状況を確認してください。

## 提供する機能

無設定で起動した場合は、次の六つの法令ツールを公開します。

| MCP ツール | 用途 | 主な情報源 |
|---|---|---|
| `search_laws` | 法令名、略称、表記揺れ、自然文または軽微な誤記から法令を検索する | e-Gov 法令 API Version 2 |
| `get_law` | 法令 ID と任意の基準日から XML 法令本文を取得する | e-Gov 法令 API Version 2 |
| `get_article` | 本則または原始附則の条・項を XML と出典付きで取得する | e-Gov 法令 API Version 2 |
| `search_law_content` | e-Gov の本文検索式で一致位置を検索する | e-Gov 法令 API Version 2 |
| `list_law_updates` | 指定日に更新一覧へ掲載された法令を取得する | e-Gov 法令 API Version 1 |
| `query_legal_information` | 日本語の照会文から利用可能な法情報の取得方法を選び、解釈ごとの型付き結果を返す | e-Gov 法令 API Version 2・Version 1、最高裁判所「裁判例検索」（有効時） |

`judicial-cases` を明示的に有効化すると、次の二つを同時に追加します。

| MCP ツール | 用途 | 主な情報源 |
|---|---|---|
| `search_judicial_cases` | 裁判所が公表する裁判例を検索する | 最高裁判所「裁判例検索」 |
| `get_judicial_case` | 検索結果の `ref` から同じ掲載裁判例の詳細と公式文書リンクを取得する | 最高裁判所「裁判例検索」 |

裁判例検索はすべての判決等を収録したものではなく、PDF 本文の抽出、上訴関係、判例変更または先例性の判定は行いません。

## 法令名検索

`search_laws` は利用者の検索語を変更せず最初に e-Gov へ送ります。正常な空結果だった場合だけ、組込み辞書と Kagome による共通前処理で一つの安全な候補を選び、正式名称で一度だけ再検索します。

対応する入力の例:

- 正式名称と公式略称
- 出典を持つ補足略称
- 全角・半角、かな、空白および句読点の表記揺れ
- 法令名を一つ含む自然文
- 一意に決定できる範囲の挿入、削除、置換または隣接文字の転置

意味が似ているだけの語、複数の法令に対応する略称、同率の誤記候補または複数の法令名を含む文は自動的に一つへ決めません。情報源エラーを別の検索で隠すこともありません。

## 統合法情報照会

`query_legal_information` は、一つの日本語照会文から法令名検索、法令本文検索、法令本文取得、条文取得または更新一覧取得を選びます。`judicial-cases` が有効な場合は、裁判例検索と、入力した検証済み `ref` による裁判例詳細の取得も選択できます。判断できない場合は推測で取得せず `needs_clarification`、法的助言や翻訳などの対象外要求は `unsupported` として返します。

`それぞれ`、`個別に`、`一つずつ`、`各々`、または複数の主題をまとめて修飾する `について` がある場合は、二つ以上四つ以下の主題を原文順の独立した検索 step にできます。単に語を列挙しただけの入力は、個別検索を明示したものとは扱いません。

法令と裁判例のように異なる情報種別でも、task と対象をそれぞれ明示した照会は
`含む` がなくても原文順の一つの計画へ合成します。必要な step が五件以上に
なる場合は切り捨てや部分実行をせず、照会を分けるよう
`needs_clarification` で返します。無効な拡張パックを必要とする計画も、
法令部分だけを実行せず計画全体を `capability_unavailable` とします。

`judicial-cases` が無効でも裁判例の意図は法令検索へ置き換えずに認識し、外部情報源を呼ばない `capability_unavailable` として返します。`令和4年（ネ）第10039号の裁判例を検索してください` のように、完全な事件番号と検索 task・裁判例 resource がそろう照会は、全角数字・括弧や構造上の空白を決定的な検索語へ整えて裁判例検索として扱います。事件番号だけ、または事件番号を句読点で列挙しただけの入力は外部情報源を呼ばない `unsupported` とし、専門ツールの使用を案内します。事件番号、題名または URL だけから裁判例詳細の `ref` を推測することはありません。入力を決定的に指定した検索や取得、ページ継続には、引き続き `search_judicial_cases` と `get_judicial_case` を使用できます。

## インストール

公式リリースでは、次の四つの archive と SHA-256 checksum を提供します。

| OS | アーキテクチャ | 形式 |
|---|---|---|
| macOS | Apple Silicon (`arm64`) | `.tar.gz` |
| macOS | Intel (`amd64`) | `.tar.gz` |
| Windows | `amd64` | `.zip` |
| Windows | `arm64` | `.zip` |

[GitHub Releases](https://github.com/geonwoo-jeong/japanese-law-mcp/releases) から対象 archive と `japanese-law-mcp_<version>_checksums.txt` を取得し、checksum を照合してから展開してください。実行には Go や別の言語ランタイムを必要としません。

ソースから現在の開発版を確認する場合は Go `1.25.0` 以上を使用します。

```sh
go build -trimpath -o ./bin/japanese-law-mcp ./cmd/japanese-law-mcp
./bin/japanese-law-mcp version
```

## MCP クライアントから起動する

既定の transport は stdio です。MCP クライアントには、展開した実行ファイルの絶対パスを指定します。

```json
{
  "mcpServers": {
    "japanese-law": {
      "command": "/absolute/path/to/japanese-law-mcp"
    }
  }
}
```

引数なしで起動すると標準入力で MCP request を待ちます。

```sh
/absolute/path/to/japanese-law-mcp
```

## 裁判例拡張パックを有効にする

裁判例ツールは既定で無効です。次の設定ファイルを作成し、`--config` で明示します。

```yaml
requestTimeout: 30s
diagnostics: false
extensionPacks:
  judicial-cases:
    enabled: true
```

```sh
japanese-law-mcp --config=/absolute/path/to/config.yaml
```

`extensionPacks.judicial-cases.enabled` は設定ファイルからだけ変更できます。省略または `false` に戻して再起動すると、二つの裁判例ツールとその provider route をまとめて無効化します。

## ローカル Streamable HTTP

同じ host の複数クライアントから利用する場合は、loopback 限定の Streamable HTTP を使用できます。

```sh
japanese-law-mcp \
  --transport=streamable-http \
  --listen-address=127.0.0.1:8080
```

MCP endpoint は `http://127.0.0.1:8080/mcp` です。待受先には `127.0.0.0/8` または `::1` の IP literal だけを指定できます。外部 address、hostname、`0.0.0.0` および `::` は起動前に拒否します。

browser から接続する場合は、許可する HTTPS Origin を設定します。

```sh
japanese-law-mcp \
  --transport=streamable-http \
  --listen-address=127.0.0.1:8080 \
  --allowed-origin=https://client.example
```

この HTTP 方式は同じ OS 利用者の process を分離する認可境界ではありません。health endpoint、session 再開、稼働監視および外部公開は提供しません。

## 実行設定

主な設定は次のとおりです。

| 名前 | 既定値 | 説明 |
|---|---|---|
| `transport` | `stdio` | `stdio` または `streamable-http` |
| `requestTimeout` | `30s` | 外部 request の timeout。1 秒以上 120 秒以下 |
| `listenAddress` | `127.0.0.1:8080` | loopback HTTP の待受先 |
| `allowedOrigins` | 空 | browser request に許可する HTTPS Origin |
| `diagnostics` | `false` | request 中だけ出力する一時診断 |

設定の優先順位は、コマンドラインフラグ、環境変数、選択した設定ファイル、既定値の順です。`--config` を省略した場合は OS の利用者設定ディレクトリにある `japanese-law-mcp/config.yaml` だけを自動検索し、現在の作業ディレクトリは検索しません。

対応する設定形式は YAML、JSON および TOML です。未知の項目、重複 key、型の不一致、無効な provider route などは transport の開始前に拒否します。

## 開発と検証

最初に Git フックを導入します。

```sh
.githooks/manage install
.githooks/manage check
```

通常の開発時は、新しく追加または修正した回帰テストを一つだけ、CPU 並列度を
一に制限して確認します。

```sh
GOMAXPROCS=1 go test -p=1 ./path/to/changed/package -run '^TestTarget$'
go run ./cmd/japanese-law-mcp --help
```

package 全体への拡張は単一テストでは相互作用を確認できない場合に限ります。
provider onboarding fitness、全 package の test、coverage、lint および
脆弱性検査はローカルで重複実行せず、GitHub Actions の clean checkout に
集約します。次の権威ある品質ゲートは CI が実行するコマンドであり、通常の
ローカル開発では実行する必要がありません。

```sh
go run ./cmd/quality-gate --profile=ci --repository=. --git-repository=.
```

基準となる開発原則は [docs/development-principles.md](docs/development-principles.md)、採用済み仕様は [sot/00-index.md](sot/00-index.md)、現在の実装状況は [wiki/10-implementation-status.md](wiki/10-implementation-status.md) を参照してください。

## リリース

`main` への Conventional Commit は Release Please が Release PR へまとめます。Release PR を merge すると tag と draft GitHub release を作成し、同じ workflow が次を実行します。

1. リリース契約と生成元 commit の照合
2. 権威ある品質ゲート
3. macOS・Windows の `amd64`・`arm64` 向け archive と checksum の作成
4. 四つの対象環境での実行、stdio 初期化および MCP ツール smoke test
5. 変更履歴と SOT 差分を結合した release 情報の公開

一つでも失敗した場合は draft のままとし、同じ `main` commit の workflow を再実行して再開できます。公開済みまたは別 commit の release は再利用しません。

リポジトリの初期設定では、GitHub の Actions 設定で GitHub Actions による Pull Request の作成を許可する必要があります。追加の Personal Access Token には依存せず、Release PR の merge 後に同じ workflow 内で品質ゲートを実行します。詳細は [SOT-DEL-014](sot/60-delivery/14-release-please-automation.md) を参照してください。

## リポジトリ構成

| パス | 内容 |
|---|---|
| `cmd/` | MCP サーバーと検証 command の入口 |
| `internal/application/` | 利用目的ごとのサービスと provider route |
| `internal/source/` | e-Gov、裁判所など情報源別の独立 adapter |
| `internal/mcp/` | 公開 MCP ツール |
| `sot/` | 採用済み仕様の定義元 |
| `wiki/` | 実装状況、評価結果および ADR |
| `docs/` | 変更しない開発原則 |
