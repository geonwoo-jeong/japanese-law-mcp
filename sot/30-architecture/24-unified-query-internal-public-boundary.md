# SOT-ARCH-024: 統合照会の内部境界と公開境界

- 状態: 廃止
- 廃止理由: 専門操作 registry と `compact`、`full` の二つの公開方式を導入し、hidden tool を禁止する従来の公開境界では現在の適用範囲を表せなくなったため
- 後継: [SOT-ARCH-044: MCP 公開ツールと専門操作 registry の境界](44-mcp-tool-operation-registry-boundary.md)

## 規定

統合照会の内部結合面は MCP ではなく型付きの in-process アプリケーション interface とし、MCP には安定した利用者向けツールだけを公開する。

## 内部境界

内部では、共通前処理、query profile、planner、selector、executor、能力別ユースケースおよび result assembler を Go の型付き interface で接続する。内部引数と戻り値には、`LegalQueryCandidate`、`LegalQueryPlan`、既存 capability の request/result および共通モデルを使用する。

内部結合のために MCP JSON、tool schema、tool 名、JSON-RPC、provider DTO または外部 URL parameter を再利用しない。内部 planner を `ProviderCapability` として登録せず、private MCP endpoint、hidden tool または第二の MCP server を設けない。

planner の可観測性は、固定 fixture を使う単体テストと評価 command で確保する。候補 score や trace を見るための `resolve_legal_query_plan` のような内部 MCP tool は採用しない。

## 公開 MCP 境界

公開 MCP 面は次で構成する。

- 決定的な入力を持つ既存専門ツール
- 日本語自然文の追加入口 `query_legal_information`

stdio とローカル Streamable HTTP は同じ公開 tool 契約を提供し、transport によって planner、pack、provider route または結果型を変えない。Streamable HTTP の待受けは `SOT-IF-029` の loopback 制約に従い、本 SOT は外部ネットワークで運用する公開ホストを追加しない。

情報源の HTTP API または HTML ページは、MCP endpoint ではなく provider adapter の外向き境界である。外部 MCP client から provider URL、認証値、selector または route を指定できる入口を設けない。

## 公開しない情報

公開 MCP 結果には、内部の数値 score、重み、閾値、token、文字位置、未選択候補の全列挙、辞書 entry 全体、provider route の決定理由、外部 raw response または stack trace を含めない。

情報源名、公式 URL、`SourceResourceRef`、`Citation` および `Provenance` は、取得結果を検証するための公開情報として保持できる。法概念を候補根拠に使用した場合は、`SOT-ENG-023` が定める `LegalConceptSource` だけを公開できる。provider や概念出典の存在を隠すのではなく、内部の選択手順と実装詳細を隠す。

`SOT-ARCH-008` の一時診断は tool 名と公開 error code だけに限定し、照会文、候補、score、辞書一致、結果または処理 trace を追加しない。

## 拡張パックとの関係

拡張パックの profile contribution、capability、route および公開 tool 集合は `SOT-ARCH-019` と `SOT-IF-067` を定義元とし、本 SOT では重複して定義しない。どの pack 構成でも、内部 contribution は in-process interface として注入し、private MCP endpoint または hidden tool へ変換しない。

## 確認

planner と executor を MCP transport なしで単体テストできること、公開 tool 一覧に内部 planner tool がないこと、stdio と Streamable HTTP の schema が一致すること、および loopback 以外で待ち受けないことを確認する。

公開結果と診断に禁止情報が現れないこと、pack 無効時に対応 provider を呼ばないこと、および既存専門ツールが planner なしで動作することを契約テストで確認する。

## 関連

- [SOT-ARCH-006: MCP ツール境界](06-mcp-tool-boundary.md)
- [SOT-ARCH-008: 一時的な診断](08-ephemeral-diagnostics.md)
- [SOT-ARCH-019: 拡張パックの有効化境界](19-extension-pack-activation-boundary.md)
- [SOT-IF-029: ローカル実行設定](../40-interfaces/29-local-runtime-configuration.md)
- [SOT-IF-051: MCP `query_legal_information`](../40-interfaces/51-mcp-query-legal-information.md)
- [SOT-DEL-001: stdio](../60-delivery/01-stdio.md)
- [SOT-DEL-013: ローカル Streamable HTTP](../60-delivery/13-local-streamable-http.md)
