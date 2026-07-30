# SOT-ENG-034: 統合照会の意味判定変更における導入段階と変更順序

- 状態: 有効

## 規定

統合照会の採用済み意味判定 profile set を変更する導入作業は、relation 生成、
profile 内適用、限定分岐、評価成果物準備、原子的採用、独立後続変更および
案内同期の七段階を、この順序で一段階ずつ進め、前段階を飛ばして公開既定動作を
切り替えない。

## 適用範囲

本規定は、`query_legal_information` の採用済み意味判定 profile set、関連
corpus/baseline、検索例カタログおよび同じ観測動作を持つ公開既定値を変更する
導入作業の順序を定める。

とくに、relation 生成と relation 依存の meaning 変更を段階導入する作業を
対象とする。provider parser、HTTP transport、MCP schema 互換性または意味判定と
独立した文書整備まで、この七段階へ機械的に当てはめない。

日常の軽微な文言修正、誤字修正または公開観測を変えない説明追加には適用しない。
一方で、cue role、relation 依存の signal、候補保持、複数主題分離、明確化条件、
selection、標準 corpus、baseline、採用 manifest または公開検索例の観測結果を
変える変更には適用する。

## 段階

### 第 1 段階: 構造準備

不変 model、cue schema、共通 loader、固定 profile 所有境界および active 成果物の不変条件を整える。ここでは、公開既定の meaning、decision、step、reason または外部呼出し境界を変えない。

### 第 2 段階: profile 内の relation 適用

positive task cue の role をそろえ、共通前処理で relation を生成し、各 profile が意図根拠レイヤと対象外候補 scope を内部で適用できるようにする。ただし、この段階では未採用の next profile set を production から選択可能にしない。

### 第 3 段階: 限定分岐と列挙境界

`SOT-ARCH-025` と `SOT-ARCH-032` が定める、共有末尾 cue による主題分離と
evidence cluster 単位の限定分岐保持を、profile 内の規則として完成させる。
この段階は意味規則そのものを再定義せず、それらの採用順序上の変更単位だけを
固定する。

### 第 4 段階: 評価成果物の準備

現行採用済み profile set を変えずに、現行採用 manifest の初回固定、
次版 corpus、baseline 候補および必要な holdout を独立成果物として準備し、
現行標準と混在させない。

### 第 5 段階: 原子的採用

production composition root、標準評価 command、corpus、baseline、中央品質ゲート、
検索例カタログおよび採用 manifest を、同じ採用変更で新しい profile set へ切り替える。profile だけ、corpus だけ、baseline だけ、または文書だけを先行切替しない。

### 第 6 段階: 意味判定から独立した後続変更

e-Gov parser の error 分類、canonical target 優先、provider mapping 調整など、意味判定 profile set と独立に進めるべき変更を別変更単位として扱う。第 5 段階へ混在させない。

### 第 7 段階: 案内と scenario 同期

前段の原子的採用義務に含まれない非実行案内、利用シナリオおよび説明文書を、現行標準の観測結果へ同期する。将来状態を先回りして「実装済み」と書かない。

## 段階ごとの進行条件

各段階は、次を満たした場合だけ次段階へ進める。

1. 対象 SOT に直接ひも付く必要最小限の検証が成功している
2. 独立 reviewer の評価が 8.0 / 10 以上で、blocker がない
3. review 指摘を反映した後に、同じ段階境界を再確認している
4. 段階の変更単位が、前後の段階と混ざらない

公開既定動作を切り替えない段階では、`SOT-ARCH-033` に従い、active artifact、
production composition root、標準 corpus、baseline および検索例カタログを現行のまま保つ。

必要最小限の検証内容は `SOT-ENG-027` を定義元とし、公開既定動作を切り替える
第 5 段階では `SOT-ENG-020` と `SOT-ENG-024` の中央品質ゲートを省略しない。

## 段階間の禁止事項

- 第 2 段階または第 3 段階で、未採用の next profile set を CLI、設定、環境変数、MCP または transport から選択可能にすること
- 第 4 段階で、次版 corpus や baseline 候補を標準 command や中央品質ゲートの現行参照先へ切り替えること
- 第 5 段階で、profile set・corpus・baseline・採用 manifest・検索例カタログの一部だけを先行採用すること
- 第 6 段階の provider parser 変更を、第 5 段階の meaning 変更に便乗させること
- 第 7 段階より前に、将来の案内文や scenario を現行確認済みのように記載すること

## Wiki との関係

実装差分、進捗、review 点数、確認日および段階の現在地は Wiki で追跡できる。ただし、段階そのものの定義、順序および進行条件の定義元は本 SOT とする。Wiki が本規定と異なる場合は、本規定を優先する。

## 確認

少なくとも次を確認する。

- 段階 1 から 7 の定義が、`query_legal_information` の意味判定変更単位を過不足なく分離している
- 現行標準を変えない段階で、production composition root、standard corpus および baseline が不変である
- 原子的採用段階で、profile set・corpus・baseline・manifest・検索例カタログの同期が要求される
- provider parser などの独立変更が、meaning 変更段階から切り離されている
- review 8.0 / 10 以上、blocker なし、および段階ごとの必要最小限検証が、次段階へ進む前提として明示されている

## 関連

- [SOT-ARCH-025: 統合照会の複数主題分離](../30-architecture/25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](../30-architecture/31-unified-query-intent-evidence-layer.md)
- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](../30-architecture/33-unified-query-profile-set-adoption-boundary.md)
- [SOT-ARCH-032: 統合照会の限定分岐保持](../30-architecture/32-unified-query-bounded-branch-retention.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-026: 統合照会の評価コーパス成果物契約](26-legal-query-corpus-artifact-contract.md)
- [SOT-ENG-027: 資源制約を踏まえた検証段階](27-resource-aware-verification-stages.md)
- [SOT-ENG-029: 統合照会の検索例カタログ](29-unified-query-example-catalog.md)
- [SOT-ENG-033: 統合照会 profile set 採用 manifest](33-unified-query-profile-set-adoption-manifest.md)
