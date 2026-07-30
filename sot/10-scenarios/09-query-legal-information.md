# SOT-SCN-009: 日本語の法情報を統合照会する

- 状態: 有効

## 規定

利用者は、一つの日本語照会文から、採用済みの法令コアおよび有効化済み拡張パックに対する検索、読取りまたは更新一覧取得を、安全に選択して実行できる。

## 開始条件

利用者が `query_legal_information` に有効な照会文、任意の一試行当たり取得上限、および参照で資源を読む場合の任意の `SourceResourceRef` を渡す。

## 基本フロー

1. システムは入力を検証し、外部情報源を呼び出す前に一つのリクエスト期限と資源予算を確定する。
2. システムは、Unicode の比較用正規化、Kagome、法令名辞書、法概念辞書および構造化識別子 parser を使って、照会文から根拠を抽出する。
3. システムは、`SOT-PROD-011` が許可する task と resource の組合せだけから意味候補を作る。
4. システムは、意味 score と実行可能性を別々に判定し、一候補、近接する二候補または非実行を決める。
5. 実行対象がある場合は、既存の型付きユースケースを通じて capability route を選び、候補間で独立した step だけを限定並列で実行する。
6. システムは、解釈、step、情報源の出典、収録範囲注意、空結果および部分失敗を混ぜずに返す。

## 分岐

- 一つの候補が十分に優勢なら、その候補だけを実行する。
- 有効な上位二候補が近接し、互いに独立している場合だけ、二候補まで実行する。
- task または resource を安全に選べない場合は、外部情報源を呼ばず `needs_clarification` とする。
- 最上位候補または一つの複数 step 解釈に必要な採用済み拡張パックが無効の場合は、利用できる部分だけを実行せず `capability_unavailable` とする。
- 対象外の依頼または未採用の task 若しくは resource である場合は、外部情報源を呼ばず `unsupported` とする。
- 実行可能な取得意図と、法的助言、翻訳または未採用 task/resource が混在する場合は、取得部分も実行せず `unsupported` とする。
- 選択した検索 step が正常な空結果を返した場合は `empty` とし、別 resource を後から追加しない。
- 一部の独立 step が成功し、別の step が失敗した場合は、成功結果と公開可能な失敗を分けて `partial` とする。
- 実行したすべての step が失敗した場合は、成功 envelope を作らず MCP ツールエラーとする。

## 複数意図と依存関係

一つの照会文に複数の明示意図がある場合は、一つの解釈内の複数 step として表せる。ただし、後続の法令または条文読取りに必要な法令 ID と条文位置は、照会文の公式識別子若しくは検証済み辞書から一意に確定できなければならない。裁判例読取りには、入力で受け取り、採用済み provider、source および resource type と照合できる `SourceResourceRef` を必須とする。

検索結果の第一件を暗黙に選び、法令本文、条文または裁判例詳細の読取りへ進まない。法令名若しくは事件番号が複数候補へ対応する場合、または検索結果を見なければ対象を決められない場合は、候補結果を返すか明確化を求める。裁判例の事件番号、題名または URL だけを `SourceResourceRef` へ推測変換しない。

## 完了条件

成功結果から、選択または非実行とした解釈、確信度区分、公開可能な根拠コード、必要な拡張パック、実行した能力、型付き結果、出典および注意へ到達できる。

異なる候補または情報源の総件数、関連度、並び順および継続位置を一つの共通値へ合成しない。

## 確認

法令名、法令番号、条文参照、事件番号、検証済み `ref`、法概念、日付、軽微な誤記、二つの明示意図、曖昧な一般語、非日本語、対象外との混在、無効な拡張パック、空結果、部分失敗および全失敗を fixture にし、分岐と外部呼出し回数を確認する。

検索結果の第一件を暗黙に読まないこと、空結果後に別 resource へ切り替えないこと、および無効な `judicial-cases` 要求で法令検索を実行しないことを明示的に確認する。

## 関連

- [SOT-PROD-011: 統合法情報照会の製品範囲](../00-product/11-unified-legal-query-scope.md)
- [SOT-SCN-010: 統合照会の非実行案内を使って再照会する](10-use-non-execution-guidance.md)
- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-MODEL-024: LegalQueryResult](../20-model/24-legal-query-result.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](../30-architecture/22-unified-query-planning-pipeline.md)
- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](../30-architecture/23-unified-query-selection-and-hedging.md)
- [SOT-IF-051: MCP `query_legal_information`](../40-interfaces/51-mcp-query-legal-information.md)
