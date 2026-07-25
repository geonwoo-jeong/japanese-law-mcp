# SOT-IF-013: MCP プロトコル基準

- 状態: 有効

## 規定

Japanese Law MCP は MCP 仕様 `2025-11-25` を実装基準とし、初期化時にこのプロトコルバージョンと `tools` capability を提示する。

## 適用

- MCP の初期化、ライフサイクル、メッセージおよびエラーは `2025-11-25` の仕様に従う。
- stdio と Streamable HTTP は同じプロトコルバージョンを提示する。
- 対応していないバージョンを対応済みとして通知しない。
- MCP Tasks、Resources、Prompts および Sampling の capability は、それぞれを定義する SOT が追加されるまで提示しない。

## 確認

初期化レスポンスのプロトコルバージョンと capability、および各ツール結果が当該バージョンのスキーマに適合することを契約テストで確認する。

## 関連

- [MCP Specification `2025-11-25`](https://modelcontextprotocol.io/specification/2025-11-25)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-DEL-001: stdio](../60-delivery/01-stdio.md)
- [SOT-DEL-013: ローカル Streamable HTTP](../60-delivery/13-local-streamable-http.md)
