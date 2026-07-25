# SOT-SCN-001: 法令名から法令を検索する

- 状態: 有効

## 規定

利用者は、法令名または略称の一部を指定し、該当する法令の識別情報と出典を取得できる。

## 開始条件

利用者が空白以外の検索語を指定している。

## 基本フロー

1. MCP クライアントが検索条件を送信する。
2. Japanese Law MCP が公式情報源の法令一覧を法令名条件で検索する。
3. 取得した結果を `LawSummary` に変換する。
4. 総件数、次の取得位置および出典を含む `LawSearchResult` を返す。

## 分岐

- 該当する法令がない場合は、成功した空の結果として返す。
- 入力を解釈できない場合は、入力エラーとして返す。
- 情報源を利用できない場合は、情報源エラーとして返す。
- 確認できない法令を推測して結果へ加えない。

## 完了条件

返された各結果から、利用者が公式情報源上の法令を識別できる。

## 関連

- [SOT-IF-001: MCP `search_laws`](../40-interfaces/01-mcp-search-laws.md)
- [SOT-MODEL-006: LawSearchResult](../20-model/06-law-search-result.md)
- [SOT-ARCH-005: リクエスト情報の一時性](../30-architecture/05-ephemeral-request-lifecycle.md)
