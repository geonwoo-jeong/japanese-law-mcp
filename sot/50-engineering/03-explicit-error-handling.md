# SOT-ENG-003: 明示的なエラー処理

- 状態: 有効

## 規定

処理を継続できない状態は Go の `error` として呼び出し元へ返し、原因を保持したまま境界で SOT に定義された公開エラーへ変換する。

## 適用

エラーを無視しない。失敗をゼロ値、空文字列、空の検索結果または成功応答として返さない。

原因を追加するときは `%w` によるラップを使用し、判定には `errors.Is` または `errors.As` を使用する。

利用者向けの `ErrorResult` への変換は MCP ツール境界で行い、内部エラー文字列をそのまま公開しない。

## 確認

各失敗経路が `SOT-IF-006` のいずれかの結果または MCP のプロトコルエラーへ到達し、成功結果へ変換されないことをテストする。

## 関連

- [SOT-IF-006: エラー契約](../40-interfaces/06-error-contract.md)
- [SOT-IF-007: MCP ツール結果](../40-interfaces/07-mcp-tool-result.md)
- [SOT-MODEL-005: ErrorResult](../20-model/05-error-result.md)
