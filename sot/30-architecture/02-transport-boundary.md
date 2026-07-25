# SOT-ARCH-002: MCP トランスポート境界

- 状態: 有効

## 規定

MCP トランスポート境界は、接続、MCP メッセージの送受信およびプロトコル上の状態だけを扱い、法令検索、法令取得または外部 API 変換の判断を持たない。

## 意味

stdio と Streamable HTTP は同じ MCP ツールとユースケースを共有する。トランスポートの違いによって、法情報の意味や結果形式を変えない。

## 確認

トランスポートを切り替えても、同じツール入力に対する構造化結果が同じインターフェース契約を満たすことを確認する。

## 関連

- [SOT-DEL-001: stdio](../60-delivery/01-stdio.md)
- [SOT-DEL-013: ローカル Streamable HTTP](../60-delivery/13-local-streamable-http.md)
