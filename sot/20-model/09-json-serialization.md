# SOT-MODEL-009: JSON シリアライズ

- 状態: 有効

## 規定

情報モデルを JSON として表すときは、各モデルに記載した項目名と型を使用し、省略可能な値が存在しない場合はその項目自体を省略する。

## 形式

- `string` は UTF-8 の JSON 文字列とする。
- `integer` は小数部を持たない JSON number とする。
- `boolean` は JSON の `true` または `false` とする。
- `date` は暦日を表す `YYYY-MM-DD` 形式の文字列とする。
- `date-time` はタイムゾーンを含む RFC 3339 形式の文字列とする。
- `object` と配列は、参照先モデルまたは項目ごとの規定に従う。

## 制約

モデルが明示的に `null` を型として許可しない限り、欠落した値を `null`、空文字列、ゼロまたは空オブジェクトへ置き換えない。

## 関連

- [情報モデル SOT 索引](00-index.md)
- [SOT-IF-007: MCP ツール結果](../40-interfaces/07-mcp-tool-result.md)
