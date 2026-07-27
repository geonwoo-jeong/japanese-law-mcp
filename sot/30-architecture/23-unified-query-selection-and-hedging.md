# SOT-ARCH-023: 統合照会の候補選択と制限付き実行

- 状態: 有効

## 規定

統合照会は、意味根拠による決定的な順位付け、低確信度での非実行、および最大二候補までの制限付き実行によって誤認識の影響を抑える。

## 候補生成

planner は、`SOT-PROD-011` が許可する七つの task/resource 組合せだけを検討する。照会文に根拠がない組合せを埋めるための候補は作らない。

根拠は `SOT-MODEL-022` の順序を基本とする。公式識別子と構造化参照を最も強く、一般名詞だけの一致を最も弱く扱う。正式名称、公式略称、法概念および一意な誤記補正は、それぞれの出典と衝突規則を満たす場合だけ根拠にする。

候補の `semanticScore`、同点規則、単独選択閾値、最低実行閾値および二候補を選ぶ margin は、版付き profile に固定する。score は確率ではなく、異なる profile version 間で数値を比較しない。

## 選択

選択は次の順序で行う。

1. 有効な候補を意味 score と固定 tie-break で順位付けする。
2. 上位候補が単独選択閾値を満たし、二位との差が単独 margin 以上なら一候補を選ぶ。
3. 上位二候補がともに最低実行閾値を満たし、差が hedge margin 以下で、独立実行しても意味が変わらない場合だけ二候補を選ぶ。
4. 上記を満たさない場合は、外部情報源を呼ばず明確化を求める。
5. 意味順位を確定した後で pack の実行可否を付与する。

上位候補または hedge 対象に必要な採用済み pack が無効なら `capability_unavailable` とし、利用可能な候補だけ、または法令コアの下位候補を代替実行しない。未採用 task/resource は `unsupported` とする。

実行可能な取得意図に、法的助言、翻訳または未採用 task/resource が明示的に混在する場合は、取得意図だけを実行せず `unsupported` とする。

法令コアと有効な pack が必要とする route、binding および request materializer は起動時に検証し、不備があれば transport を開始しない。正常起動後の route 不備を意味候補の availability または runtime fallback として扱わない。

閾値の具体的な数値は SOT へ直接埋め込まず、`SOT-ENG-024` の評価コーパスと profile version によって受け入れる。

## 実行

実行は `SOT-MODEL-023` の固定予算を守る。

- 選択候補は二件以下、論理 capability 呼出しは合計四回以下とする。
- collection step の `effectiveLimit` は、read item を先に予約した `SOT-MODEL-023` の式で全 step 分を実行前に確定する。
- 独立候補は同じ root context の下で並列に実行できる。
- 同じ候補内で依存する step は順序どおり直列に実行する。
- provider ごとの limiter がより厳しい場合は、アプリケーションが並列に schedule しても adapter 呼出しを直列化できる。
- timeout または cancellation は root context から全 step へ伝播する。
- attempt の公開順は計画順に固定し、通信の完了順や race の結果で変えない。
- 空結果、失敗または実返却件数によって、未使用 item 予算を別 step へ再配分しない。

検索結果の第一件を後続読取りへ暗黙に渡さない。後続 step の参照が計画時に一意でなければ、その read step を作らない。

## 結果の扱い

複数候補または複数 step の結果は attempt ごとに保持する。provider 横断の総件数、関連度、順位、継続位置または類似度を共通尺度へ変換せず、題名若しくは本文の類似だけで結果を統合しない。

空結果、件数差、情報源の速度または一つの attempt の失敗を理由に、計画確定後に task/resource を再分類しない。実行した step の少なくとも一つが成功し、別の実行済み step が失敗した場合だけ失敗を部分結果として保持する。pack 無効の非実行を部分失敗にせず、全 step 失敗はツールエラーとする。

## 確認

高い単独候補、近接する二候補、三つ以上の曖昧候補、最低閾値未満、pack 無効、対象外との混在、起動時 route 不備、二候補の部分失敗、全失敗、timeout、完了順の逆転および provider limiter を fixture で確認する。

無効な裁判例 pack を法令検索へ置き換えないこと、空結果後に別候補を追加しないこと、検索第一件を読まないこと、`effectiveLimit` を再配分しないこと、および候補数、呼出し数、item 数の上限を超えないことを明示的に検証する。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-MODEL-024: LegalQueryResult](../20-model/24-legal-query-result.md)
- [SOT-ARCH-013: 情報源の選択と組合せ](13-source-composition.md)
- [SOT-ARCH-019: 拡張パックの有効化境界](19-extension-pack-activation-boundary.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](../50-engineering/24-unified-query-evaluation-gate.md)
