# SOT-MODEL-005: ErrorResult

- 状態: 有効

## 規定

`ErrorResult` は、処理が完了しなかった理由と利用者が次に取れる対応を、保存対象となる診断情報に依存せず判別できる形で表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `code` | string | はい | プロジェクトが定義するエラー識別子 |
| `message` | string | はい | 利用者が次の対応を判断するための説明 |
| `retryable` | boolean | はい | 入力を変えずに再試行する意味があるか |
| `details` | object | いいえ | エラーコードごとに定義された安全な追加情報 |

## 制約

`details` には JSON の文字列、数値、真偽値およびその配列だけを使用し、情報源の未加工レスポンスを格納しない。

認証情報、利用者の入力全文、外部レスポンス全文および内部スタック情報を結果へ含めない。

## 関連

- [SOT-IF-006: エラー契約](../40-interfaces/06-error-contract.md)
- [SOT-IF-007: MCP ツール結果](../40-interfaces/07-mcp-tool-result.md)
- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
- [開発原則 6](../../docs/development-principles.md#6-一時的な情報処理と明確な結果)
