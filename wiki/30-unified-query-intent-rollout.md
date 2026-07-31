# 統合照会の意図判定導入順

この文書は、有効な SOT と現在の実装との差分、着手順および進捗を追跡する
Wiki である。公開動作または採用済み契約の定義元にはしない。

## 現在地

現行の公開既定は `corpus-v9`、`default-1` および現在の固定 profile set である。
`SOT-MODEL-030` の `CueTaskRelation` 不変 model、`task_expression` predicate
対応、cue schema version 3、共通 loader、共通前処理の閉じた role 入力検証、
固定 profile の所有 ID 検証および relation sidecar の生成は実装済みである。
さらに、test が直接構成する次版の core と `judicial-cases` profile では、
`SOT-ENG-032` の role 対応、relation に基づく明示 task・対象外 signal、
言及された cue の通常検索語化および対象外候補 scope を適用する。

profile metadata については、schema version 1 と 2 の閉じた共通 loader、
`branchRetentionMargin` の存在状態を保持する不変 model、profile 固有の
所有・参照成果物検証および固定 profile set の共有校正整合を実装した。
test が直接構成する core・`judicial-cases` の次版 fixture だけが
schema version 2 を使用し、active metadata は version 1 のまま維持する。

共有末尾列については、既存の法令名・法概念・一般検索語 span と
`direct_task` relation から閉じた separator、末尾接続、節および一意な最大列を
共通 constructor が一回だけ確認し、不変な `SharedTerminalSequence` として
全 profile へ渡す処理を実装した。sidecar は `CandidateGenerationInput` にだけ
保持し、原文、候補、plan または公開結果へ追加しない。active core と
`judicial-cases` はこの sidecar をまだ消費しないため、公開既定の意味判定は
変えていない。

ただし、production composition root が構成する active profile set は従来の
意味判定を維持し、生成済み relation を signal、候補保持または decision に
使用しない。次版 profile を CLI、設定、環境変数、MCP または transport から
選択する入口もなく、採用 manifest は未実装である。そのため、現行の標準評価
command はまだ `corpus-v9/default-1` を固定引数で参照しており、
`SOT-ENG-024` が採用 manifest 導入後に定める `current.json` 基準の command との
差分が残る。`SOT-ENG-036` が定める baseline schema、変更不能な version file、
重複 key と symlink を含む厳格な安全検証も未実装である。現行 baseline report は
profile を profile ID 順に整列しており、production composition root の固定順を
そのまま保持する同 SOT の目標状態へ移行していない。

したがって、relation 対応の意味判定、`corpus-v10`、`default-2` および対応する
検索例カタログは、まだ現行標準ではない。

## 推奨順序

段階そのものの定義、順序および進行条件の定義元は
[SOT-ENG-034](../sot/50-engineering/34-unified-query-rollout-stages.md) とする。
この章の表は、現時点の進捗と確認範囲を追跡するための運用上の写像である。

| 段階 | 状態 | 目的 | 主な定義元 |
|---:|---|---|---|
| 1 | 完了 | relation の不変 model、cue schema version 3、共通 loader および固定 profile set の構造整合を準備し、v2 の role 対応へ更新する | `SOT-MODEL-030`、`SOT-ENG-030` |
| 2 | 完了 | positive task cue の role をそろえ、共通前処理で relation を生成し、各 profile 内で意図根拠レイヤと対象外候補 scope を適用できるようにする | `SOT-MODEL-025`、`SOT-MODEL-026`、`SOT-MODEL-030`、`SOT-ARCH-031`、`SOT-ENG-028`、`SOT-ENG-031`、`SOT-ENG-032` |
| 3 | 進行中（3.3 完了） | profile metadata schema version 2、共有末尾 sidecar、private evidence cluster、core の sidecar 適用、裁判例の独立適用および test 専用固定 profile set を順に完成させる | `SOT-MODEL-031`、`SOT-ARCH-025`、`SOT-ARCH-031`、`SOT-ARCH-032`、`SOT-ARCH-034`、`SOT-ARCH-035`、`SOT-ENG-035` |
| 4 | 未着手 | 現行集合の baseline schema・初回採用 manifest・adoption 基準 command、新規 holdout を含む `corpus-v10`、development だけで校正した次版固定 profile set および `default-2` 候補を順に準備し、閉じた CI handoff で一回の holdout 採用判定を行う | `SOT-ARCH-033`、`SOT-ENG-024`、`SOT-ENG-026`、`SOT-ENG-033`、`SOT-ENG-036`、`SOT-ENG-037` |
| 5 | 未着手 | 全採用要素と current tuple を一変更で公開既定へ切り替え、公開 notice、questions、非実行時の外部呼出しゼロおよび MCP response parity を固定検証する | `SOT-ARCH-033`、`SOT-MODEL-024`、`SOT-IF-051`、`SOT-ENG-024`、`SOT-ENG-029`、`SOT-ENG-033` |
| 6 | 未着手 | `GET /laws`、`GET /keyword`、`GET /law_data` の parser を一 endpoint ずつ移行した後、法令検索の canonical target 優先を application 層へ接続する | `SOT-IF-011`、`SOT-IF-052`、`SOT-IF-053`、`SOT-IF-054`、`SOT-ARCH-030` |
| 7 | 未着手 | code や評価成果物を変えず、前段の同一変更義務に含まれない scenario、help および説明文書だけを現行標準へ同期する | `SOT-SCN-010`、`SOT-ENG-034` |

## 段階 review 記録

| 段階 | 確認日 | 独立 review | security review | blocker | 確認範囲 |
|---:|---|---:|---:|---:|---|
| 1 | 2026-07-31 | 9.2 / 10 | 9.1 / 10 | 0 | v2 role、relation 保持、閉じた role 入力、profile 所有 ID、active 成果物の不変 |
| 2 | 2026-07-31 | 9.5 / 10 | 9.3 / 10 | 0 | 共通 relation 生成、positive task role、引用・言及・topic 除外、profile 内の意図根拠、対象外候補 scope、next/active 分離 |
| 3.1 | 2026-07-31 | 9.7 / 10（test 8.0 / 10） | 9.0 / 10 | 0 | schema version 1・2 の閉じた loader、存在状態、固定 set の共有校正と digest、active version 1 と test 専用 version 2 の分離 |
| 3.2 | 2026-07-31 | 9.2 / 10（test 9.4 / 10） | 9.0 / 10 | 0 | 閉じた共有末尾列、bounded maximal-path 判定、実前処理の二代表例、128・256 上限、active core・裁判例の非消費 |

## 第 3 段階以降の SOT 文書 review

以前の第 3・第 4 段階だけを対象にした通過記録は、
`SOT-ARCH-034`、`SOT-ARCH-035` および `SOT-ENG-037` の追加と、第 5 から
第 7 段階の境界変更によって適用範囲が変わったため、現行文書の通過根拠として
使用しない。最新 tree に対する独立再 review の最終記録は次のとおりである。

| 確認日 | architecture review | testability review | blocker | major | minor | 確認範囲 |
|---|---:|---:|---:|---:|---:|---|
| 2026-07-31 | 9.6 / 10 | 9.8 / 10 | 0 | 0 | 0 | core・judicial の根拠対応、第 3 から第 7 段階、profile 候補と provider 候補の境界、candidate identity、holdout の一回利用と compact leakage index、report と履歴の資源上限、evaluator の採用・再現・rollback、corpus の不変性、固定検証 ID |

この review は文書設計だけを対象とし、第 3 段階以降の実装完了を表さない。
実装状態は次節のとおり 3.1 から 3.3 までが完了し、3.4 以降は `未着手` とする。

## 第 3 段階の内部進捗

内部順序の定義元は `SOT-ENG-034` とし、ここでは実装状態だけを追跡する。

| 順序 | 状態 | 変更単位 |
|---:|---|---|
| 3.1 | 完了 | schema version 2 の profile metadata model、loader、存在状態および固定 set 整合 |
| 3.2 | 完了 | production-neutral な `SharedTerminalSequence` sidecar |
| 3.3 | 完了 | profile-private な根拠対応と evidence cluster |
| 3.4 | 未着手 | core の sidecar 消費、複数主題 step および限定分岐 |
| 3.5 | 未着手 | sidecar を消費しない `judicial-cases` 固有の限定分岐 |
| 3.6 | 未着手 | 全 profile が schema version 2 と共有校正値を持つ test 専用固定 set |

## 段階の境界

第二段階と第三段階では、次版の内部実装と development fixture を準備できる。ただし
`SOT-ARCH-033` に従い、CLI、設定、MCP または transport から選択可能にせず、
次版の cue artifact と profile metadata は test が直接構成する別 set に置く。
現行 active metadata、公開 decision、外部呼出し境界、標準 corpus および
baseline を変更しない。

第四段階では、まず `SOT-ENG-036` の baseline schema と閉じた loader を導入し、
現行 `default.json` と同じ `baselines/versions/default-1.json`、
`previousAdoptionId` のない初回 history manifest およびそれを指す
`current.json` を同じ変更で作る。標準 command と中央品質ゲートの入口は
adoption 基準へ切り替えるが、切替前後の `corpus-v9/default-1`、production
profile set、`legal-query-evaluator-v1`、report byte および外部呼出し境界は
一致させる。

その後、profile、辞書または誤記規則を変えず、新しい holdout の正解、集合分離
および coverage を独立 review して digest を固定する。holdout を参照せず
development 集合だけで次版固定 set を校正する。次に `SOT-ENG-037` の閉じた
request と pointer、候補 set を直接構成する CI 専用入口、候補 writer、
`default-2` の予約名と出力先を別変更で準備する。これらを標準 command、製品 CLI、
設定、MCP、transport または中央品質ゲートの現行参照先にしない。
request は exact evaluator version と、corpus manifest が持つ
`holdoutLeakageGroupDigests` の compact index を固定する。

review 済みの同一 commit に対して専用 CI job を一回だけ起動し、合格 report は
`default-2` version file、失敗 report は変更不能な failed history へ同じ byte の
まま handoff する。passed と failed のどちらでも有効な result ができた
holdout digest は消費済みとし、後続候補へ再利用しない。全ての過去評価と同じ
leakage group digest を持つ新しい holdout も拒否する。この累積検査は件数と
合計 byte に上限を持つ過去の request/result だけを走査し、過去 corpus fixture を
再度開かない。失敗した予約 baseline version は再利用しない。採用前の候補 history
manifest は置かず、`current.json` と `default.json` は現行集合を指し続ける。
第五段階では合格 version file の byte と digest を書き換えず、新しい history
manifest の追加と current pointer の切替を全採用要素と同じ変更で行う。

第五段階だけが、relation 依存の意味判定を production composition root へ採用する
段階である。profile 実装だけ、corpus だけ、baseline だけ、採用 manifest だけ、
検索例だけを先に現行標準へ切り替えない。標準 command が読む
`baselines/default.json` は、準備済み `default-2` version file と同じ byte へ
同じ採用変更で切り替える。adoption manifest、合格 request、標準 command および
rollback 先はそれぞれ同じ exact evaluator version を指し、current evaluator への
fallback を許可しない。公開 notice と questions、非実行時の外部呼出しゼロ、
`content` と `structuredContent` の同値性および transport 間の同値性もこの段階で
固定検証し、第七段階へ延期しない。

第六段階の provider parser と mapping は、`GET /laws`、`GET /keyword`、
`GET /law_data` の順に一 endpoint ずつ、統合照会の意味解釈とは独立した
変更単位にする。その後、application 層の law-target resolver と page 内安定優先を
別変更で接続する。raw response、parser の error 分類および page item の同一性・
順序は provider/application 専用 fixture で検証し、意味評価 baseline に証明させない。
第六段階の各番号では評価投影と検索例カタログを不変に保つ。必要な変更がそれらを
変える場合は現行の七段階 rollout と `SOT-ENG-037` の対象外とする。provider、
parser、resolver または mapping の候補 identity、production 非到達性、評価
request、原子的採用 tuple、rollback、固定検証および資源境界を定義する別の
新しい有効な SOT が採用されるまで、その変更へ着手しない。profile set 専用の
candidate request へ component field を足して流用しない。Wiki の実装済み範囲は
実際に完了した段階の同じ変更で同期する。第七段階は、code、評価成果物または
公開 MCP 契約を変えない文書専用段階とし、将来状態を先に「実装済み」または
「現行確認済み」と記載しない。

## 進行条件

各段階および `SOT-ENG-034` が番号を付けた段階内変更は、一度に一つずつ進め、
次の番号へ移る前に次を満たす。

- 対象 SOT へ結び付く必要最小限の検証が成功する
- 独立 reviewer の文書レビュー記録で総合評価が 8.0 / 10 以上で、blocker がない
- review 指摘を反映した後に同じ境界を再確認する
- 一つの番号付き変更単位を一つの独立した commit として確定してから次へ進む
- その commit に対する適用可能な権威 CI の成功を確認してから次へ進む

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
- 非実行状態ごとの固定 notice、questions、zero-call および response parity は
  第五段階で実装・検証し、第七段階では `SOT-SCN-010` の意味を変えず
  scenario、help、リンクおよび進捗だけを同期する
