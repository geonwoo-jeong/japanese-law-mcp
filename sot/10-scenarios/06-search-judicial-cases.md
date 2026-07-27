# SOT-SCN-006: 公表裁判例を検索する

- 状態: 有効

## 規定

利用者は、裁判例に関するキーワードを指定し、最高裁判所の「裁判例検索」に掲載された該当裁判例を、出典と収録範囲の注意とともに検索できる。

## 開始条件

利用者が一つの有効な検索語を指定し、`judicial-cases` 拡張パックが有効である。

## 基本フロー

1. MCP クライアントが検索語と任意の取得上限を送信する。
2. Japanese Law MCP が `judicial-decision.search@1` の primary route を通じて公式の統合検索を呼び出す。
3. 公式結果の各掲載を `JudicialDecisionSummary` へ対応させる。
4. 収録範囲の注意、掲載情報および情報源が明示する場合の該当件数を返す。

## 分岐

- 該当する掲載がない場合は成功した空の一覧として返す。
- 検索語または取得上限が制約を満たさない場合は、外部情報源を呼び出さず入力エラーとする。
- 同じ裁判例が複数カテゴリーへ掲載されていても、異なる公式詳細ページを自動で統合しない。
- 情報源が示していない事件名、裁判種別、結果または文書 URL を推測しない。
- 情報源を利用できない場合または HTML が採用した契約を満たさない場合は、原因を保持した情報源エラーとして返す。

## 完了条件

各結果が同じプロバイダーの公式詳細ページ URL を保持し、日付が `YYYY-MM-DD` で表され、公式原文と収録範囲の注意を確認できる。

## 関連

- [SOT-PROD-010: 裁判例拡張パック](../00-product/10-judicial-cases-extension-pack.md)
- [SOT-MODEL-020: JudicialDecisionSummary](../20-model/20-judicial-decision-summary.md)
- [SOT-IF-041: `judicial-decision.search` capability v1](../40-interfaces/41-judicial-decision-search-capability.md)
- [SOT-IF-047: MCP `search_judicial_cases`](../40-interfaces/47-mcp-search-judicial-cases.md)
