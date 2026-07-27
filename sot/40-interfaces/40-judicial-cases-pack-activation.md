# SOT-IF-040: `judicial-cases` 拡張パックの有効化

- 状態: 有効

## 規定

`judicial-cases` 拡張パックは、設定ファイルの `extensionPacks.judicial-cases.enabled` が `true` の場合に限り公開する。

## 構造

```yaml
extensionPacks:
  judicial-cases:
    enabled: true
```

`extensionPacks` は pack ID を key、pack ごとの object を value とする object である。初期版で受理する pack ID は `judicial-cases` だけとし、その object が持てる項目は boolean の `enabled` だけとする。未知の pack ID、未知の pack 設定項目、型の不一致および `null` を受理しない。

## 公開ツール集合

法令コアの公開ツール集合は、`search_laws`、`get_law`、`get_article`、`search_law_content`、`list_law_updates` および `query_legal_information` の六つとする。

`judicial-cases` が有効な場合は、上記へ `search_judicial_cases` と `get_judicial_case` を加えた八つとする。それ以外の tool を pack、provider または route の登録だけで暗黙に追加しない。

## 既定値と rollback

`extensionPacks` または `judicial-cases` を省略した場合、および `enabled: false` の場合は無効とする。無設定起動では上記の法令コア六ツールを公開する。

有効化を取り消す場合は `judicial-cases` を削除するか `enabled: false` としてプロセスを再起動する。取り消し後は二つの裁判例専門ツール、裁判例 query profile の実行 contribution、裁判例 result variant、条件付き provider および二つの route を実効構成から除き、法令コアの公開面と route は変更しない。`query_legal_information` 自体は登録したままとする。

## 有効化する集合

有効な場合に限り、次を一つの集合として構成する。

- 利用シナリオ: `SOT-SCN-006`、`SOT-SCN-007`
- capability: `judicial-decision.search@1`、`judicial-decision.read@1`
- MCP ツール: `search_judicial_cases`、`get_judicial_case`
- 統合照会: 裁判例固有の実行 profile、能力別 request materializer、および `judicial_decision_search`、`judicial_decision` result variant
- provider と route: `SOT-IF-046` の条件付き組込み値

必要な binding、provider、primary route または request materializer を起動時に構成できない場合は transport を開始せず設定エラーとする。片方のツールだけを公開せず、別 provider への runtime fallback を行わない。

## 入力元

`extensionPacks` は `SOT-IF-039` が定義する設定ファイルだけから読み込む。環境変数または個別のコマンドラインフラグを定義しない。

## 確認

省略、明示した `false`、`true`、未知の pack、未知の項目および型不一致を設定テストで確認する。無効時は六ツールと法令 route、有効時は八ツール、裁判例 profile contribution と二つの裁判例 route、設定不足時は transport 開始前の失敗を composition root のテストで確認する。無効時の裁判例照会は `capability_unavailable` となり、法令または裁判例 provider を呼び出さないことを確認する。

## 関連

- [SOT-PROD-010: 裁判例拡張パック](../00-product/10-judicial-cases-extension-pack.md)
- [SOT-IF-039: 設定ソースと優先順位 v2](39-configuration-sources-and-precedence-v2.md)
- [SOT-IF-046: 裁判所「裁判例検索」の組込み採用](46-courts-hanrei-built-in-adoption.md)
- [SOT-ARCH-019: 拡張パックの有効化境界](../30-architecture/19-extension-pack-activation-boundary.md)
- [SOT-IF-051: MCP `query_legal_information`](51-mcp-query-legal-information.md)
