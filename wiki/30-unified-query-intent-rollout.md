# 統合照会の意図判定導入順

この文書は、有効な SOT と現在の実装との差分、着手順および進捗を追跡する
Wiki である。公開動作または採用済み契約の定義元にはしない。

## 現在地

現行の公開既定は `corpus-v9`、`default-1` および現在の固定 profile set である。
`SOT-MODEL-030` の `CueTaskRelation` 不変 model、`task_expression` predicate
対応、cue schema version 3、共通 loader、共通前処理の閉じた role 入力検証および
固定 profile の所有 ID 検証は実装済みである。一方、core の positive task cue に
対する `SOT-ENG-032` の完全対応は未実装である。production の共通前処理も
relation を生成せず、profile は relation 依存の signal、候補保持または decision
を使用していない。採用 manifest も未実装である。

したがって、relation 対応の意味判定、`corpus-v10`、`default-2` および対応する
検索例カタログは、まだ現行標準ではない。

## 推奨順序

段階そのものの定義、順序および進行条件の定義元は
[SOT-ENG-034](../sot/50-engineering/34-unified-query-rollout-stages.md) とする。
この章の表は、現時点の進捗と確認範囲を追跡するための運用上の写像である。

| 段階 | 状態 | 目的 | 主な定義元 |
|---:|---|---|---|
| 1 | 完了 | relation の不変 model、cue schema version 3、共通 loader および固定 profile set の構造整合を準備し、v2 の role 対応へ更新する | `SOT-MODEL-030`、`SOT-ENG-030` |
| 2 | 未着手 | positive task cue の role をそろえ、共通前処理で relation を生成し、各 profile 内で意図根拠レイヤと対象外候補 scope を適用できるようにする | `SOT-MODEL-025`、`SOT-MODEL-026`、`SOT-MODEL-030`、`SOT-ARCH-031`、`SOT-ENG-028`、`SOT-ENG-031`、`SOT-ENG-032` |
| 3 | 未着手 | 共有末尾 cue の閉じた列挙判定と evidence cluster 単位の限定分岐保持を profile 内で完成させる | `SOT-ARCH-025`、`SOT-ARCH-032` |
| 4 | 未着手 | 現行 `corpus-v9/default-1` の初回採用 manifest を作り、新規 holdout を含む `corpus-v10` と `default-2` の候補成果物を、profile を変えない準備変更で独立 review し、digest を固定する | `SOT-ARCH-033`、`SOT-ENG-024`、`SOT-ENG-026`、`SOT-ENG-033` |
| 5 | 未着手 | relation 対応 profile set、準備済み corpus・baseline、採用 manifest、標準 command、品質ゲートおよび検索例カタログを一変更で公開既定へ切り替える | `SOT-ARCH-033`、`SOT-ENG-024`、`SOT-ENG-026`、`SOT-ENG-029`、`SOT-ENG-033` |
| 6 | 未着手 | e-Gov parser のエラー分類と法令検索の canonical target 優先を、意図判定変更とは別に移行する | `SOT-IF-011`、`SOT-IF-052`、`SOT-IF-053`、`SOT-IF-054`、`SOT-ARCH-030` |
| 7 | 未着手 | 前段の同一変更義務に含まれない非実行案内と scenario 契約を現行標準へ同期する | `SOT-SCN-010`、`SOT-MODEL-024`、`SOT-IF-051` |

## 段階 review 記録

| 段階 | 確認日 | 独立 review | security review | blocker | 確認範囲 |
|---:|---|---:|---:|---:|---|
| 1 | 2026-07-31 | 9.2 / 10 | 9.1 / 10 | 0 | v2 role、relation 保持、閉じた role 入力、profile 所有 ID、active 成果物の不変 |

## 段階の境界

第二段階と第三段階では、次版の内部実装と development fixture を準備できる。ただし
`SOT-ARCH-033` に従い、CLI、設定、MCP または transport から選択可能にせず、
次版の cue artifact と profile metadata は test が直接構成する別 set に置く。
現行 active metadata、公開 decision、外部呼出し境界、標準 corpus および
baseline を変更しない。

第四段階では、profile、辞書または誤記規則を変えず、新しい holdout の正解、
集合分離および coverage を独立 review して digest を固定する。候補成果物を
標準 command または中央品質ゲートの現行参照先にしない。同じ変更で、現行の
`corpus-v9/default-1` 集合から `previousAdoptionId` のない初回 history manifest と
それを指す `current.json`、現行 `default.json` と同じ
`baselines/versions/default-1.json` を作り、導入前後の観測動作が一致することを
確認する。候補 `default-2` は test が直接構成する次版 set から
`baselines/versions/default-2.json` へ生成し、第五段階で version file の byte と
digest を書き換えない。

第五段階だけが、relation 依存の意味判定を production composition root へ採用する
段階である。profile 実装だけ、corpus だけ、baseline だけ、採用 manifest だけ、
検索例だけを先に現行標準へ切り替えない。標準 command が読む
`baselines/default.json` は、準備済み `default-2` version file と同じ byte へ
同じ採用変更で切り替える。

第六段階の provider parser と mapping は、統合照会の意味解釈とは独立した変更単位に
する。第五段階または第六段階で公開観測が変わる検索例カタログと Wiki の
実装済み範囲は、その段階の同じ変更で必ず同期し、第七段階へ延期しない。
第七段階は、前段の採用義務に含まれない非実行案内と scenario 契約だけを扱い、
将来状態を先に「実装済み」または「現行確認済み」と記載しない。

## 進行条件

各段階は一度に一つずつ進め、次の段階へ移る前に次を満たす。

- 対象 SOT へ結び付く必要最小限の検証が成功する
- 独立 reviewer の文書レビュー記録で総合評価が 8.0 / 10 以上で、blocker がない
- review 指摘を反映した後に同じ境界を再確認する
- 一段階を一つの独立した commit として確定してから次へ進む

resource 制約のあるローカル環境では `SOT-ENG-027` の段階別検証を使い、
各小変更で全評価 corpus や全 test suite を反復しない。公開既定を切り替える
第五段階では、`SOT-ENG-020` と `SOT-ENG-024` の中央品質ゲートを省略しない。
第二段階と第三段階の commit では、ローカルの対象 test に加え、CI で現行標準の
decision、reason、selection、meaning、step および
`SOT-ENG-033` の外部呼出し境界投影が変わらないことを一回確認する。

## 別管理する残課題

次は意図判定 profile set の第二段階から第五段階へ混在させない。

- `SOT-IF-004` の再試行開始間隔、共通資源予算および同時実行枠の conformance
- e-Gov 応答 parser の `invalid_source_response` と
  `source_contract_changed` の分類
- application 層 law-target resolver と検索結果内の canonical target 優先
- 非実行状態ごとの固定 notice と再照会 scenario
