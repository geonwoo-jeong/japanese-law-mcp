# SOT-ARCH-040: 法令版間比較の境界

- 状態: 有効

## 規定

法令版間比較は、`law.version.compare@1` という独立した provider capability を
通して実装し、ユースケース境界が provider 固有の XML、HTML、JSON 又は
差分アルゴリズムへ依存しない構成とする。

## 責任の分離

比較対象の二版選択、採用範囲の条の切出し、比較用文字列と構造の正規化、
条の対応付け、差分分類及び出典の生成は、provider ごとの adapter が責任を持つ。
各 provider の parser、正規化器、fixture 及び外部契約は provider 固有の
package とファイルへ置き、別 provider の変更で共有ユースケース又は既存
provider の parser を変更しない。

アプリケーション層は、共通入力の検証、primary route の選択、request context、
共通結果の検証及び公開結果への投影だけを担当する。`law.document.read@1` を
二回呼び出した後に、アプリケーション層で provider 固有の表現を解釈する構成は
採用しない。

共通化するのは、二版選択の意味、条の同一性、変更種別、変更理由、件数及び
出典を持つ `LawVersionComparison` までとする。provider の表現では同じ意味を
安全に得られない項目を推測して共通化しない。

## ルーティングと公開面

一回の比較は一つの primary provider とその一つの source だけを使用し、前後版を
別 provider へ分けず、失敗時に別 provider へ fallback 又は fan-out しない。

この機能は `compare_law_versions` という専門操作として提供し、MCP からの直接公開または発見と実行による到達方法は `SOT-IF-077` に従う。この操作は
`query_legal_information` の task、profile、辞書、候補又は実行経路には
参加させない。統合法情報照会で比較を対象外とする `SOT-PROD-011` と
`SOT-PROD-012` の境界を維持する。

## 一時処理と安全

取得した二版の原文、比較用文字列、構造表現及び索引は一回の request 中だけ
保持し、成功又は失敗時に破棄する。同じ法令として検証できない場合、条同一性が
重複する場合又は比較が完了しない場合は、部分結果、途中までの差分、推測した
対応又は切り詰めた成功を返さない。

provider adapter は、前後の原文取得に使う既存上限に加え、対象条数、比較用
文字列サイズ、変更件数、処理時間及び成功結果サイズの上限を持つ。上限超過は
原因に応じて `source_response_too_large` 又は `source_processing_limit` とする。

## 確認

ユースケースのテストが外部 API 形式を扱わず `law.version.compare@1` port
だけへ依存すること、provider ごとの parser を独立して置くこと、統合法情報照会と
評価 corpus を変更しないこと、fallback しないこと、及び adapter が部分結果や
一時原文を残さないことを確認する。

## 関連

- [SOT-PROD-011: 統合法情報照会の製品範囲](../00-product/11-unified-legal-query-scope.md)
- [SOT-PROD-012: 日本法情報照会の意図分類と受理境界](../00-product/12-japanese-legal-query-intent-taxonomy.md)
- [SOT-PROD-013: 法令版間比較](../00-product/13-law-version-comparison.md)
- [SOT-ARCH-010: プロバイダーの分離](10-provider-isolation.md)
- [SOT-ARCH-014: 外部原文の一時処理](14-ephemeral-source-artifacts.md)
- [SOT-ARCH-020: 採用済みユースケース境界](20-adopted-use-case-boundary.md)
- [SOT-IF-058: `law.version.compare` capability v1](../40-interfaces/58-law-version-compare-capability.md)
- [SOT-IF-077: MCP ツール公開方式と拡張パック有効化](../40-interfaces/77-mcp-tool-exposure-and-extension-packs.md)
