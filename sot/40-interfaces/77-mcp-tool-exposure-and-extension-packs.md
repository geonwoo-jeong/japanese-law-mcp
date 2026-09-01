# SOT-IF-077: MCP ツール公開方式と拡張パック有効化

- 状態: 有効

## 規定

Japanese Law MCP は、設定ファイルだけで選ぶ `toolExposure` により、三つの MCP ツールから必要な専門操作を段階的に利用する `compact` と、既存の専門ツールを直接公開する `full` を提供し、既定値を `compact` とする。

## 設定

設定ファイルの最上位に、次の文字列を指定できる。

```yaml
toolExposure: compact
extensionPacks:
  judicial-cases:
    enabled: true
  judicial-citations:
    enabled: true
  legislative-history:
    enabled: true
```

`toolExposure` は `compact` または `full` だけを受理する。省略時は `compact` とし、`null`、空文字列、型の不一致または未知の値を受理しない。設定ファイル以外の環境変数または個別のコマンドラインフラグを設けない。

`extensionPacks` は pack ID を key、設定 object を value とする object とし、受理する ID は `judicial-cases`、`judicial-citations` および `legislative-history` だけとする。各 object が持てる項目は boolean の `enabled` だけとする。`extensionPacks`、pack object または `enabled` の省略、および `enabled: false` は当該 pack を無効とする。未知の pack ID、未知の項目、型の不一致および `null` を受理しない。

`judicial-citations.enabled: true` は `judicial-cases.enabled: true` を必要とし、依存先を自動で有効にしない。依存違反は provider factory の生成、外部呼出しおよび transport の開始より前に設定エラーとする。`extensionPacks` も設定ファイルだけから読み込み、環境変数または個別フラグを設けない。設定ファイルの入力元と優先順位は `SOT-IF-039` に従う。

## 利用可能な専門操作

法令コアでは、次の七操作を常に利用可能にする。

- `search_laws`
- `get_law`
- `get_article`
- `search_law_content`
- `list_law_revisions`
- `compare_law_versions`
- `list_law_updates`

有効な pack は、次の読取り専用操作を原子的に加える。

| pack | 追加する専門操作 | 追加する provider route 数 |
|---|---|---:|
| `judicial-cases` | `search_judicial_cases`、`get_judicial_case` | 2 |
| `judicial-citations` | `trace_judicial_citations` | 2 |
| `legislative-history` | `search_diet_speeches` | 1 |

各操作の入力、出力、情報源、エラーおよび利用条件は、その操作固有の有効な SOT を定義元とする。公開方式はこれらを変更しない。

`judicial-cases` が有効な場合は、二つの操作、`SOT-SCN-006` と `SOT-SCN-007`、二つの capability、裁判例 facade、request materializer、統合照会 result variant、HTML provider および二 route を一つの集合として構成する。裁判例の意味認識 contribution は pack の状態にかかわらず固定 profile set に含め、無効時は外部呼出しなしの `capability_unavailable` とする。

`judicial-citations` も有効な場合に限り、`SOT-SCN-015`、二つの到達可能な capability route、PDF と HTML の条件付き provider、専用 application service および `trace_judicial_citations` を別の一集合として加える。`query_legal_information` の範囲は広げない。

`legislative-history` が有効な場合は、`SOT-SCN-014`、`parliament.speech.search@1`、NDL provider、primary route および `search_diet_speeches` を一集合として加える。第一段階では国会発言固有の統合照会 contribution を追加せず、`query_legal_information` の範囲を広げない。

必要な binding、provider、primary route、facade、request materializer または操作を一つでも構成できない場合は transport を開始しない。一部の構成要素だけを到達可能にしない。

## `compact` の公開ツール

`compact` は、pack の組合せにかかわらず、次の三つだけを MCP `tools/list` に直接公開する。

1. `discover_legal_tools`
2. `execute_legal_tool`
3. `query_legal_information`

専門操作名は直接公開しない。専門操作名を `tools/call` へ直接指定した場合は、操作が registry に存在していても MCP の未知ツールとして拒否する。`query_legal_information` は従来の schema、annotations、結果およびエラーで直接呼び出し、`execute_legal_tool` の対象にはしない。

`discover_legal_tools` は `readOnlyHint: true`、`openWorldHint: false` とする。`execute_legal_tool` は、登録対象を読取り専用 allowlist に限定して `readOnlyHint: true` とし、実行時に公式の外部情報源へ接続し得るため `openWorldHint: true` とする。`query_legal_information` の既存 annotations は変更しない。

## `discover_legal_tools`

入力は次の JSON object とする。

| 名前 | 型 | 必須 | 制約 |
|---|---|---:|---|
| `query` | string | いいえ | trim 後に UTF-8 で 1 byte 以上 256 byte 以下。ASCII 制御文字を含めない |
| `limit` | integer | いいえ | 1 以上 16 以下。既定値 5 |

`arguments` の JSON byte 列全体は、前後の空白を含めて 16384 byte 以下とし、`inputSchema` に `x-maxJsonBytes: 16384` を明示する。object 以外、重複 key、未知の項目、`null`、型の不一致、不正 UTF-8、不正 surrogate、非整数および範囲外を受理せず、外部情報源を呼び出す前に `invalid_argument` とする。

`query` を省略した場合は利用可能な全専門操作を候補とする。指定した場合は、Unicode の前後空白を除き小文字化した値が、操作名または説明を同様に小文字化した値へ部分一致する操作だけを候補とする。照合後の候補を操作名（`tools[].name`）の code point 昇順で並べ、先頭から `limit` 件まで返す。発見は外部情報源を呼ばない。

成功時の `structuredContent` は次の閉じた JSON object とする。

| 名前 | 型 | 意味 |
|---|---|---|
| `totalCount` | integer | `query` 適用後に一致した利用可能な専門操作の総数 |
| `returnedCount` | integer | `tools` の配列長 |
| `omittedCount` | integer | `totalCount - returnedCount` |
| `truncated` | boolean | `omittedCount > 0` と同値 |
| `tools` | object[] | 返した専門操作 |

各 `tools[]` は `name`、`description`、`inputSchema` および `outputSchema` だけを持つ。schema は、同じ操作を `full` で直接公開するときの JSON Schema と byte 意味上同じ内容とする。該当がない場合は三件数を `0`、`truncated: false`、`tools: []` とした成功結果とする。

## `execute_legal_tool`

入力は次の JSON object とする。

| 名前 | 型 | 必須 | 制約 |
|---|---|---:|---|
| `toolName` | string | はい | trim 後に UTF-8 で 1 byte 以上 128 byte 以下。ASCII 制御文字を含めない |
| `arguments` | object | はい | 選択した専門操作の `inputSchema` に適合する JSON object |

`arguments` の外側を含む JSON byte 列全体は、前後の空白を含めて 65536 byte 以下とし、`inputSchema` に `x-maxJsonBytes: 65536` を明示する。外側 object は `toolName` と `arguments` だけを持つ。object 以外、項目の欠落、重複 key、未知の項目、`null`、型の不一致、不正 UTF-8、不正 surrogate または上限超過を受理しない。入れ子の `arguments` にある全 object は、配列内の object を含め重複 key を受理しない。

検証済みの `toolName` は trim 後の値で利用可能な専門操作へ完全一致させる。`discover_legal_tools`、`execute_legal_tool` および `query_legal_information` は指定できない。未知の名前と無効な pack の操作名は、どちらも外部情報源と専門 handler を呼び出さず、`code: invalid_argument` および `details.field: toolName` の同じ公開エラーとし、原因の差を公開しない。

操作を選択した後は、検査した `arguments` の JSON byte 列を再直列化せず既存 handler へ一度だけ渡す。選択した操作固有の schema 違反は、その handler が従来どおり外部情報源を呼ぶ前に `invalid_argument` とする。

公開する `outputSchema` は任意の JSON object を表す汎用 schema とする。実行時は、選択した専門操作の `CallToolResult` を変更せず返す。成功時の structured content と text content、空結果、`isError`、公開 error code、`retryable` および安全な details は、`full` における同じ専門操作の直接呼出しと一致させる。正確な操作固有の出力 schema は発見結果から取得する。

## `full` の公開ツール

`full` は発見と実行の meta tool を登録せず、法令コア七操作と `query_legal_information` の八ツールを直接公開する。有効な pack の専門操作を同じ名前と schema で追加し、従来の直接呼出し契約を維持する。

## 構成別の件数

有効な構成の件数は次のとおりとする。

| `judicial-cases` | `judicial-citations` | `legislative-history` | 利用可能な専門操作数 | `compact` 公開ツール数 | `full` 公開ツール数 | provider route 数 |
|---:|---:|---:|---:|---:|---:|---:|
| `false` | `false` | `false` | 7 | 3 | 8 | 7 |
| `false` | `false` | `true` | 8 | 3 | 9 | 8 |
| `true` | `false` | `false` | 9 | 3 | 10 | 9 |
| `true` | `false` | `true` | 10 | 3 | 11 | 10 |
| `true` | `true` | `false` | 10 | 3 | 11 | 11 |
| `true` | `true` | `true` | 11 | 3 | 12 | 12 |

`judicial-cases: false` と `judicial-citations: true` の二構成は、`legislative-history` と `toolExposure` の値にかかわらず無効とする。

## Rollback

従来の直接公開が必要な場合は `toolExposure: full` として再起動する。簡潔な公開面へ戻す場合は `toolExposure` を削除するか `compact` として再起動する。切替えは provider、route、pack の状態または操作契約を変更しない。

pack を無効へ戻す場合は、当該 `enabled` を削除するか `false` として再起動し、その pack の操作、provider、route および専用 application 構成だけを実効構成から除く。`compact` の三ツールは維持し、発見結果から無効な操作を除く。`full` では無効な操作の直接ツールを除く。

## MCP capability と transport

両方式ともサーバー capability は `tools` だけとし、MCP resource と prompt を登録または公開しない。stdio と無状態の loopback Streamable HTTP は、同じ設定に対して同じ公開ツール集合、schema、annotations、操作集合、結果およびエラーを提供する。HTTP は既存の単一 `/mcp` endpoint だけを使用する。

## 確認

少なくとも次を設定、MCP 契約および transport smoke test で確認する。

- `toolExposure` の省略、`compact`、`full`、未知値、`null`、型不一致、および環境変数と個別フラグが存在しないこと
- 三 pack の省略、`false`、`true`、未知項目、型不一致、`null` および依存違反
- 上表の六構成で専門操作、公開ツール、route、provider factory および binding inventory の件数と原子性が一致すること
- `compact` の一覧が常に三ツールだけで、専門操作名の直接呼出しが未知ツールとなること
- `full` の一覧が従来の八から十二ツールとなり、meta tool がないこと
- 発見の入力境界、照合、名前順、schema、空結果、上限ちょうど、上限超過および四件数の関係
- 実行の全体 byte 上限、外側と全入れ子 object の重複 key、未知項目、欠落、`null`、型不一致、再帰拒否、未知名と無効名の同一エラー、および外部呼出し前の拒否
- 全専門操作について `compact` の実行と `full` の直接呼出しの成功結果、空結果、入力エラーおよび情報源エラーが一致すること
- annotations、stdio と Streamable HTTP の parity、`resources/list` と `prompts/list` の非提供、および Streamable HTTP の単一 loopback endpoint

## 関連

- [SOT-PROD-017: 段階的に利用する簡潔な MCP 公開面](../00-product/17-compact-mcp-public-surface.md)
- [SOT-SCN-017: 専門操作を発見して実行する](../10-scenarios/17-discover-and-execute-legal-tool.md)
- [SOT-ARCH-019: 拡張パックの有効化境界](../30-architecture/19-extension-pack-activation-boundary.md)
- [SOT-ARCH-041: 拡張パックの専門公開面の段階採用](../30-architecture/41-staged-specialist-extension-surface.md)
- [SOT-ARCH-042: 判例引用追跡拡張パックの従属有効化](../30-architecture/42-judicial-citations-pack-dependency.md)
- [SOT-ARCH-044: MCP 公開ツールと専門操作 registry の境界](../30-architecture/44-mcp-tool-operation-registry-boundary.md)
- [SOT-IF-039: 設定ソースと優先順位 v2](39-configuration-sources-and-precedence-v2.md)
- [SOT-IF-051: MCP `query_legal_information`](51-mcp-query-legal-information.md)
- [SOT-IF-065: 国立国会図書館の国会発言検索の組込み採用](65-ndl-diet-speech-built-in-adoption.md)
- [SOT-IF-074: 判例引用追跡の組込み採用](74-courts-hanrei-citation-built-in-adoption.md)
