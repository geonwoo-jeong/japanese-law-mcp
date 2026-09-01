# SOT-ARCH-044: MCP 公開ツールと専門操作 registry の境界

- 状態: 有効

## 規定

MCP サーバーへ直接登録する公開ツールと、既存の専門契約を実行する in-process の読取り専用操作 registry を分離し、起動時に確定した不変な allowlist だけを発見および dispatch の対象とする。

## 起動時の組立て

各専門操作は、一つの名前、説明、入力 schema、出力 schema および handler を持つ不変な定義として一度だけ組み立てる。簡潔な公開方式と完全公開方式は同じ定義を使用し、tool 名、schema または handler を複製して別々の定義元にしない。

操作 registry は、法令コアの専門操作と、起動時に有効な拡張パックの原子的な専門操作だけで構成する。無効な pack、依存関係を満たさない pack、構成できない binding または route の操作を登録しない。構成の一部だけがそろった状態では transport を開始しない。

簡潔な公開方式では、統合照会を直接の MCP ツールとして登録し、発見と実行の二つの meta tool だけを追加する。専門操作の名前は MCP サーバーへ直接登録せず、同名の `tools/call` は SDK の未知ツールとして拒否する。専門操作は hidden MCP tool、private endpoint または第二の MCP server ではなく、同一 process 内の型付き依存と既存 handler を結ぶ registry entry とする。

完全公開方式では、同じ専門操作定義と統合照会を従来どおり直接 MCP サーバーへ登録し、発見と実行の meta tool は登録しない。

## 発見境界

発見 handler は registry の不変な snapshot だけを参照し、名前と説明の決定的な照合、名前の昇順、返却上限および省略件数を適用する。provider の状態確認、外部情報源への通信、操作の試行または動的登録を行わない。

返す schema は、同じ entry を完全公開方式で直接登録するときに使用する正確な schema とする。呼出しごとに schema を生成または変更せず、内部 handler、port、provider、route、秘密値または診断情報を公開しない。

## 実行境界

実行 handler は、検証済みの `toolName` を registry の完全一致で一度検索する。発見、実行および統合照会自身は registry に含めず、再帰 dispatch を許さない。未知の名前と無効な pack の名前は同じ公開エラーとし、実効 pack 構成を推測できる差を設けない。

入力の byte 上限、全 object の重複 key および外側の構造を検証した後、`arguments` の未変更の JSON byte 列を選択した既存 handler へ一度だけ渡す。map への復号と再直列化で数値、順序または JSON 表現を変えない。選択後の入力検証、application port の呼出し、結果の直列化および公開エラー変換は、専門操作の既存 handler を唯一の実装として再利用する。

dispatch は新しい MCP request、JSON-RPC round trip または受信 middleware の再入を作らない。root context、session 情報および取消しは外側の request から引き継ぎ、同じ request pacing、deadline、一時状態および診断境界の中で一回だけ実行する。結果の `CallToolResult` は成功、空結果、`isError`、structured content および text content を変更せず外側へ返す。

## Transport と capability

stdio と無状態の loopback Streamable HTTP は、同じ公開方式、操作 registry、schema、annotations、結果およびエラーを使用する。transport によって公開方式または pack を変えない。

サーバー capability は `tools` だけとし、MCP resource と prompt を登録しない。Streamable HTTP は既存の単一 `/mcp` endpoint と loopback 制約を維持する。

## 確認

同じ専門定義が簡潔な registry と完全公開の直接登録で一致すること、無効な操作と再帰対象が registry にないこと、簡潔な方式で専門名の直接呼出しが未知ツールとなること、dispatch が handler を一回だけ呼び結果を byte 意味上変更しないこと、および stdio と Streamable HTTP の一覧、schema、annotations、結果とエラーが一致することを検証する。

発見が外部情報源を呼ばないこと、未知名と無効名の差を公開しないこと、resources と prompts の capability および一覧が存在しないこと、並行 request 間で registry または入力を変更しないことも確認する。

## 関連

- [SOT-ARCH-005: リクエスト情報の一時性](05-ephemeral-request-lifecycle.md)
- [SOT-ARCH-006: MCP ツール境界](06-mcp-tool-boundary.md)
- [SOT-ARCH-008: 一時的な診断](08-ephemeral-diagnostics.md)
- [SOT-ARCH-019: 拡張パックの有効化境界](19-extension-pack-activation-boundary.md)
- [SOT-IF-051: MCP `query_legal_information`](../40-interfaces/51-mcp-query-legal-information.md)
- [SOT-IF-077: MCP ツール公開方式と拡張パック有効化](../40-interfaces/77-mcp-tool-exposure-and-extension-packs.md)
- [SOT-DEL-001: stdio](../60-delivery/01-stdio.md)
- [SOT-DEL-013: ローカル Streamable HTTP](../60-delivery/13-local-streamable-http.md)
