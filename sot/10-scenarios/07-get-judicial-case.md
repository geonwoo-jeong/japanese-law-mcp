# SOT-SCN-007: 公表裁判例の詳細を取得する

- 状態: 有効

## 規定

利用者は、裁判例検索で得た情報源参照を指定し、同じプロバイダーの公式詳細ページに掲載された裁判例の詳細と公式文書リンクを取得できる。

## 開始条件

利用者が `judicial-decision` を指す有効な `SourceResourceRef` を指定し、`judicial-cases` 拡張パックが有効である。

## 基本フロー

1. MCP クライアントが検索結果から受け取った `SourceResourceRef` を変更せず送信する。
2. Japanese Law MCP が参照の provider に対する `judicial-decision.read@1` binding を選ぶ。
3. 公式詳細ページの共通項目、掲載された判示事項、裁判要旨、参照法条および公式文書リンクを `JudicialDecisionDetails` へ対応させる。
4. 同じ参照、取得経路、詳細情報および収録範囲の注意を返す。

## 分岐

- 参照の provider、source、resource type または resource ID が不正な場合は、外部情報源を呼び出さず入力エラーとする。
- 登録済みでも無効な provider は設定不足とし、別の provider へ fallback しない。
- 正確な公式詳細ページが存在しない場合は `not_found` とする。
- 詳細ページに存在しない省略可能な項目は推測せず省略する。
- PDF だけで提供される本文または要旨は URL として返し、初期版では本文へ変換しない。

## 完了条件

検索から渡した provider、source および資源参照が変わらず、詳細項目と文書 URL を公式ページで確認できる。

## 関連

- [SOT-PROD-010: 裁判例拡張パック](../00-product/10-judicial-cases-extension-pack.md)
- [SOT-MODEL-021: JudicialDecisionDetails](../20-model/21-judicial-decision-details.md)
- [SOT-IF-042: `judicial-decision.read` capability v1](../40-interfaces/42-judicial-decision-read-capability.md)
- [SOT-IF-048: MCP `get_judicial_case`](../40-interfaces/48-mcp-get-judicial-case.md)
