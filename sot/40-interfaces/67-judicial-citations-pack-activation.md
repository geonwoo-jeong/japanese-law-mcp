# SOT-IF-067: `judicial-citations` 拡張パックの有効化

- 状態: 有効

## 規定

`judicial-citations` 拡張パックは、設定ファイルの `extensionPacks.judicial-citations.enabled` が `true` であり、同時に `extensionPacks.judicial-cases.enabled` も `true` の場合に限り公開する。

## 構造

```yaml
extensionPacks:
  judicial-cases:
    enabled: true
  judicial-citations:
    enabled: true
```

初期版で受理する pack ID は `judicial-cases`、`judicial-citations` および `legislative-history` とし、各 object が持てる項目は boolean の `enabled` だけとする。未知の pack ID、未知の pack 設定項目、型の不一致および `null` を受理しない。

## 公開ツール集合

法令コアの公開ツール集合は、`search_laws`、`get_law`、`get_article`、`search_law_content`、`list_law_revisions`、`compare_law_versions`、`list_law_updates` および `query_legal_information` の八つとする。

`judicial-cases` が有効な場合は、上記へ `search_judicial_cases` と `get_judicial_case` を加えた十ツールとする。さらに `judicial-citations` も有効な場合に限り、`trace_judicial_citations` を加えた十一ツールとする。

`judicial-citations` だけで専門ツールを公開しない。`query_legal_information` は pack の有無にかかわらず従来の一ツールとして登録し、範囲を広げない。

## 既定値と rollback

`extensionPacks`、`judicial-citations` または `enabled` を省略した場合、及び `enabled: false` の場合は無効とする。

`judicial-citations` を有効に戻す場合は `enabled: true` として再起動する。無効へ戻す場合は `judicial-citations` を削除するか `enabled: false` として再起動し、citation の provider、route、依存関係および `trace_judicial_citations` だけを実効構成から除く。`judicial-cases` と `legislative-history` の公開面は変更しない。

## 有効化する集合

有効な場合に限り、次を一つの集合として構成する。

- 利用シナリオ: `SOT-SCN-015`
- capability: `judicial-decision.case-citation.extract@1`、`judicial-decision.citing-candidate.search@1`
- MCP ツール: `trace_judicial_citations`
- provider と route: `SOT-IF-074` の条件付き組込み値

必要な binding、provider、primary route または tool 依存関係を起動時に構成できない場合は transport を開始せず設定エラーとする。

## 入力元

`extensionPacks` は `SOT-IF-039` が定義する設定ファイルだけから読み込む。環境変数または個別コマンドラインフラグを追加しない。

## 確認

省略、明示した `false`、`true`、未知の pack、未知の項目、型不一致、および `judicial-citations=true` と `judicial-cases=false` の依存違反を設定テストで確認する。pack ごとの公開 tool 数、citation route の原子的追加、rollback、および `query_legal_information` の不変性を構成テストで確認する。

## 関連

- [SOT-ARCH-042: 判例引用追跡拡張パックの従属有効化](../30-architecture/42-judicial-citations-pack-dependency.md)
- [SOT-IF-039: 設定ソースと優先順位 v2](39-configuration-sources-and-precedence-v2.md)
- [SOT-IF-074: 判例引用追跡の組込み採用](74-courts-hanrei-citation-built-in-adoption.md)
