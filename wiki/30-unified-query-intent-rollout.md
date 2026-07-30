# 統合照会の意図判定導入順

この文書は、有効な SOT と現在の実装との差分、着手順および進捗を追跡する
Wiki である。公開動作または採用済み契約の定義元にはしない。

## 現在地

現行の公開既定は `corpus-v9`、`default-1` および現在の固定 profile set である。
`CueTaskRelation` の不変 model、cue schema version 3 と共通 loader は実装済みだが、
production の共通前処理は relation を生成せず、profile も relation 依存の signal、
候補保持または decision を使用していない。

したがって、relation 対応の意味判定、`corpus-v10`、`default-2` および対応する
検索例カタログは、まだ現行標準ではない。

## 推奨順序

| 段階 | 状態 | 目的 | 主な定義元 |
|---:|---|---|---|
| 1 | 完了 | relation の不変 model、cue schema version 3、共通 loader および固定 profile set の成果物整合を準備する | `SOT-MODEL-029`、`SOT-ENG-030` |
| 2 | 未着手 | 共通前処理で relation を生成し、各 profile 内で意図根拠レイヤと対象外候補 scope を適用できるようにする | `SOT-MODEL-025`、`SOT-MODEL-026`、`SOT-MODEL-029`、`SOT-ARCH-031`、`SOT-ENG-028`、`SOT-ENG-031` |
| 3 | 未着手 | 共有末尾 cue の閉じた列挙判定と evidence cluster 単位の限定分岐保持を profile 内で完成させる | `SOT-ARCH-025`、`SOT-ARCH-032` |
| 4 | 未着手 | relation 対応 profile set、corpus、baseline、標準 command、品質ゲートおよび検索例カタログを一変更で公開既定へ切り替える | `SOT-ARCH-033`、`SOT-ENG-024`、`SOT-ENG-026`、`SOT-ENG-029` |
| 5 | 未着手 | e-Gov parser のエラー分類と法令検索の canonical target 優先を、意図判定変更とは別に移行する | `SOT-IF-011`、`SOT-IF-052`、`SOT-IF-053`、`SOT-IF-054`、`SOT-ARCH-030` |
| 6 | 未着手 | 非実行案内、scenario 契約、検索例および実装状況を現行標準と同期する | `SOT-SCN-010`、`SOT-MODEL-024`、`SOT-IF-051`、`SOT-ENG-029` |

## 段階の境界

第二段階と第三段階では、次版の内部実装と test fixture を準備できる。ただし
`SOT-ARCH-033` に従い、CLI、設定、MCP または transport から選択可能にせず、
現行の公開 decision、外部呼出し境界、標準 corpus および baseline を変更しない。

第四段階だけが、relation 依存の意味判定を production composition root へ採用する
段階である。profile 実装だけ、corpus だけ、baseline だけ、検索例だけを先に
現行標準へ切り替えない。

第五段階の provider parser と mapping は、統合照会の意味解釈とは独立した変更単位に
する。第六段階の文書は、将来状態を先に「実装済み」または「現行確認済み」と
記載せず、第四段階または第五段階の完了後に同期する。

## 進行条件

各段階は一度に一つずつ進め、次の段階へ移る前に次を満たす。

- 対象 SOT へ結び付く必要最小限の検証が成功する
- 独立 reviewer の文書レビュー記録で総合評価が 8.0 / 10 以上で、blocker がない
- review 指摘を反映した後に同じ境界を再確認する
- 一段階を一つの独立した commit として確定してから次へ進む

resource 制約のあるローカル環境では `SOT-ENG-027` の段階別検証を使い、
各小変更で全評価 corpus や全 test suite を反復しない。公開既定を切り替える
第四段階では、`SOT-ENG-020` と `SOT-ENG-024` の中央品質ゲートを省略しない。

## 別管理する残課題

次は意図判定 profile set の第二段階から第四段階へ混在させない。

- `SOT-IF-004` の再試行開始間隔、共通資源予算および同時実行枠の conformance
- e-Gov 応答 parser の `invalid_source_response` と
  `source_contract_changed` の分類
- application 層 law-target resolver と検索結果内の canonical target 優先
- 非実行状態ごとの固定 notice と再照会 scenario
