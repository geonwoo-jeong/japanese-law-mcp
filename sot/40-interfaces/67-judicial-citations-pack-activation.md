# SOT-IF-067: `judicial-cases` と `judicial-citations` の有効化

- 状態: 有効

## 規定

`judicial-cases` は単独で有効化できる裁判例パックとし、`judicial-citations` は `judicial-cases` に依存する 1-hop の判例引用追跡パックとする。両方を設定で明示的に有効化し、依存先を自動で有効化しない。

本規定は `SOT-IF-040` の後継であり、同規定が定めた `judicial-cases` の既定値、既存二ツールおよび統合照会の意味を維持したまま、`judicial-citations` の条件を追加する。

## 構造

```yaml
extensionPacks:
  judicial-cases:
    enabled: true
  judicial-citations:
    enabled: true
```

`SOT-IF-061` と合わせて受理する pack ID は `judicial-cases`、`judicial-citations` および `legislative-history` とし、各 object が持てる項目は boolean の `enabled` だけとする。未知の pack ID、未知の pack 設定項目、型の不一致および `null` を受理しない。

`judicial-citations.enabled` が `true` で `judicial-cases.enabled` が `true` でない構成は、provider factory の生成、外部呼出しおよび transport の開始より前に設定エラーとする。

## 公開ツール集合

法令コアの公開ツール集合は、`search_laws`、`get_law`、`get_article`、`search_law_content`、`list_law_revisions`、`compare_law_versions`、`list_law_updates` および `query_legal_information` の八つとする。

`judicial-cases` は `search_judicial_cases` と `get_judicial_case` の二ツールと二 route、`judicial-citations` は `trace_judicial_citations` の一ツールと二 route、`legislative-history` は `search_diet_speeches` の一ツールと一 route を加える。有効な組合せの総数は次のとおりとする。

| `judicial-cases` | `judicial-citations` | `legislative-history` | 公開ツール数 | provider route 数 |
|---:|---:|---:|---:|---:|
| `false` | `false` | `false` | 8 | 7 |
| `false` | `false` | `true` | 9 | 8 |
| `true` | `false` | `false` | 10 | 9 |
| `true` | `false` | `true` | 11 | 10 |
| `true` | `true` | `false` | 11 | 11 |
| `true` | `true` | `true` | 12 | 12 |

`judicial-cases: false` と `judicial-citations: true` の二構成は `legislative-history` の値にかかわらず無効とする。`query_legal_information` は pack の有無にかかわらず従来の一ツールとして登録し、範囲を広げない。

## 既定値と rollback

`extensionPacks`、各 pack または `enabled` を省略した場合、及び `enabled: false` の場合は当該 pack を無効とする。

`judicial-citations` を有効に戻す場合は `enabled: true` として再起動する。無効へ戻す場合は `judicial-citations` を削除するか `enabled: false` として再起動し、citation の provider、route、依存関係および `trace_judicial_citations` だけを実効構成から除く。`judicial-cases` と `legislative-history` の公開面は変更しない。

## 有効化する集合

`judicial-cases` が有効な場合は、次を一つの集合として構成する。

- 利用シナリオ: `SOT-SCN-006`、`SOT-SCN-007`
- capability: `judicial-decision.search@1`、`judicial-decision.read@1`
- MCP ツール: `search_judicial_cases`、`get_judicial_case`
- 既存の裁判例 facade、request materializer、統合照会 result variant、HTML provider と二 route

`judicial-citations` も有効な場合に限り、次を別の一つの集合として追加する。

- 利用シナリオ: `SOT-SCN-015`
- 到達可能な capability route: `judicial-decision.case-citation.extract@1`、`judicial-decision.citing-candidate.search@1`
- MCP ツール: `trace_judicial_citations`
- provider と route: `SOT-IF-074` の条件付き組込み値
- `trace_judicial_citations` 専用の application service と内部 graph 組立て

必要な binding、provider、primary route または tool 依存関係を起動時に構成できない場合は transport を開始せず設定エラーとする。

## 入力元

`extensionPacks` は `SOT-IF-039` が定義する設定ファイルだけから読み込む。環境変数または個別コマンドラインフラグを追加しない。

## 確認

省略、明示した `false`、`true`、未知の pack、未知の項目、型不一致、および `judicial-citations=true` と `judicial-cases=false` の依存違反を設定テストで確認する。上表の六つの有効な組合せについて固定順の tool、route、provider factory および binding inventory を確認する。citation route の原子的追加、rollback、既存二裁判例ツールの JSON 契約および `query_legal_information` の不変性も確認する。

## 関連

- [SOT-ARCH-042: 判例引用追跡拡張パックの従属有効化](../30-architecture/42-judicial-citations-pack-dependency.md)
- [SOT-IF-039: 設定ソースと優先順位 v2](39-configuration-sources-and-precedence-v2.md)
- [SOT-IF-074: 判例引用追跡の組込み採用](74-courts-hanrei-citation-built-in-adoption.md)
