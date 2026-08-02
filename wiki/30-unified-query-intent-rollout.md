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

一方で、`SOT-ARCH-036`、`SOT-ARCH-037`、`SOT-ARCH-039` および
`SOT-ENG-039` は、core の 3.4.1 から 3.4.4 に必要な文書上の契約を先行して更新した。
このうち 3.4.1 は、五 input kind の閉じた cue 対応、同じ節・同じ主題への束縛、
`topicOrdinal`、`asOf` と updates 日付の差および span を持たない `ref` の例外を、
test 専用 request 内の対応として実装し、独立 review を通過した。
3.4.2 は、法令名本文検索語投影の三経路、同じ節の一意な content search 束縛、
read、law search および article read 競合時の fail-closed を純粋 projector として
実装し、独立 review を通過した。shared-terminal 消費は 3.4.3 で実装し、
3.4.4 で step 内根拠正規化、正規化後の候補和集合と score、
保持した法概念 fact だけに対応する `conceptSources`、一意な group 決定、
request と logical input の `ref` 完全一致、provider 非依存性、private mapping の寿命および
active production composition root からの非到達性を実装し、独立 review を通過した。

`SOT-ARCH-029` は、同じ step の弱い根拠を省略できる条件が選択的であったため
廃止し、閉じた優越表、`normalizationGroup`、`conceptSources` および
fail-closed を持つ `SOT-ARCH-036` に置き換えた。3.4.4 で現行 core の対応 test から
`SOT-ARCH-029` 参照を除き、`SOT-ARCH-036` の固定検証へ移行した。
廃止した SOT 原文と過去 corpus 補正の履歴コメントは、過去状態の記録として保持する。

旧 `SOT-ARCH-032`、`SOT-ARCH-034`、`SOT-ARCH-035`、`SOT-ENG-034` および
`SOT-ENG-037` は原文を保持して廃止し、それぞれ `SOT-ARCH-037`、
`SOT-ARCH-039`、`SOT-ARCH-038`、`SOT-ENG-039` および `SOT-ENG-038` を
現行の定義元とした。既存 code と test に残る旧 ID または旧振る舞いは実装差分で
あり、この文書変更では置換していない。

ただし、production composition root が構成する active profile set は従来の
意味判定を維持し、生成済み relation を signal、候補保持または decision に
使用しない。次版 profile を CLI、設定、環境変数、MCP または transport から
選択する入口はまだない。一方で、第 4.1 段階として `default-1` の version file、
初回 adoption history、`current.json`、baseline schema および閉じた loader は
実装済みであり、標準評価 command と中央品質ゲートの入口も
`testdata/legalquery/adoptions/current.json` 基準へ切り替えた。現行 current tuple は
引き続き `corpus-v9`、`default-1`、現行 production profile set および
`legal-query-evaluator-v1` を指し、切替前後の標準 report byte を変えない。
第 4.1 段階の権威 CI は成功した。

第 4.2 段階として、schema version 2 の閉じた schema、typed decoder、
development assertion 十一件、追加 holdout coverage 七件と十一 fixture、
holdout leakage group の digest 投影および `corpus-v10` を追加した。
`corpus-v9` と schema version 1 の全 byte を維持し、旧版と新準備版の tree digest、
schema byte、カテゴリ最小件数および safety pair を固定検証した。
独立 review は通過したが、この Wiki を含む第 4.2 段階の commit に対する
権威 CI も成功した。

第 4.3 段階として、schema と development だけを開く閉じた loader、fixture
原 byte の content digest、core と `judicial-cases` の development 校正版および
四十三件全体の決定的 fingerprint を追加した。holdout、manifest、execution、
過去の report または result を校正入力にせず、独立した二回の読込みと profile
構成が同じ観測を返すことを固定した。独立 review と権威 CI は通過した。

第 4.4 段階として、`SOT-ENG-038` の schema v2、content manifest、52 件の固定
review SOT digest、architecture/testability の二件の review attestation、
`corpus-v10` と `default-2` 予約名へ結び付く request、`current.json` pointer、
fail-closed な CI handoff 入口および候補 baseline writer を追加した。holdout、
result、failed report または候補 baseline の中身は生成せず、Git が追跡できない
空 history root は loader 側で論理的 empty として扱う。実際の
`testdata/legalquery/candidate-evaluations/` subtree は canonical byte で生成し、
閉じた loader の自己検証を通過した。後続の 4.5 だけが holdout の一回実行と
結果 handoff を担当する。

第 4.5 段階の準備実装と独立 review は完了した。passed/failed の result 契約、request/result 履歴だけを
使う一回利用・leakage preflight、候補 planning を注入する evaluator、環境を閉じる
標準 library bootstrap、候補専用 worker、二 file の exclusive handoff および tracked
result の byte replay 境界までは実装した。最初に準備した `default-2` request では
manual dispatch を三回試みたが、いずれも report の生成前に停止し、構造的に有効な
result は生成されなかった。このため `corpus-v10` の holdout digest は消費していない。

停止原因は、`HOME` を持たない worker の cache 解決、execution fixture と候補 plan の
不一致の失敗分類、および同値縮約前の raw draft を一つの evidence mapping の上限へ
入れていた順序の三点に分け、公開済みの `corpus-v4` だけを使って修正した。現行
holdout の正解や採点結果は修正入力にしていない。候補 source set が変わったため、
`default-2` は未消費の置換済み準備とし、不変な manifest、review および request を
保持したまま予約名を再利用しない。同値 raw draft を個別 mapping で正規化してから
縮約し、最終保持数を従来どおり十六件以下で fail-closed にする `default-3` request を
追加して current pointer を進めた。新しい exact candidate content に対する独立 review は
architecture 10.0 / 10、testability 8.4 / 10、blocker 0 である。準備 commit
`5df7e2f7635ecf1b127e1a7cce58f5cce79139ac` の権威 CI `30681776066` は成功し、
同じ commit に固定した manual dispatch `30682022985` も構造上有効な report と
result を生成した。

現行の candidate content は
`candidate-content-sha256-41a5c5dbd5d78492a153c80dc2e1913036097cbb44e0e4f053cd5607ba858945`、
current request は
`evaluation-sha256-398e801b2d7edd6068f36fa34fe94827d7d44891d59976fdc8630e4d5be7e89c`
である。result の `outcome` は `failed`、report digest は
`5d702ccb1b27b34e2007444dd8d25640c08c2ddc46f7aff206324fe9692377ba` である。
holdout の plan reproducibility と request error は 1.0 だった一方、plan outcome は
0.5191、meaning signature は 0.7183、top-1 は 0.7487、top-2 は 0.7744、
high-confidence precision は 0.9277、evidence assertion は 0.2956 だった。
execution は八件中五件が期待結果と一致し、resource 誤呼出し、budget 違反、
暗黙の初回 read および empty の再分類は零だった。report と result は CI artifact の
byte を変えず、failed history と result history へ handoff した。予約した
`default-3` baseline file は作成していない。

準備実装の最終 review は 8.9 / 10、security review は 9.1 / 10 で、blocker は
いずれも零である。handoff 成果物の review は実装 9.8 / 10、testability
10.0 / 10、security 10.0 / 10、blocker 0 で通過した。handoff commit
`71833e6d19b8efc2634c55f797578141dc86da22` の権威 CI `30682417467` では、
既存の production test が空の tracked result を期待したままであることを検出した。
その assertion を current failed result、request digest および report digest の照合へ
更新した commit `f60da386d821f73f76ec3db12a4c2b99744c8338` は、code 10.0 / 10、
testability 10.0 / 10、security 10.0 / 10、blocker 0 の独立 review と権威 CI
`30682743993` を通過した。

同じ commit に固定した tracked byte replay `30682972843` も成功した。artifact 名は
`candidate-evaluation-f60da386d821f73f76ec3db12a4c2b99744c8338` で、通常 file は
`result.json` と `report.json` の二件だけだった。両 file は tracked history と byte
単位で一致し、SHA-256 はそれぞれ
`313130ccdb69f43ef1b3bcbae38498e91eddd590d3ddc3407bc50cdd08a2510a` と
`5d702ccb1b27b34e2007444dd8d25640c08c2ddc46f7aff206324fe9692377ba` だった。

したがって、`corpus-v10` の holdout digest は消費済みであり、`default-3` は
採用対象から外れる。relation 対応の意味判定、`default-3` および対応する検索例
カタログは現行標準ではなく、第 5 段階へ進まない。次の採用候補には、過去の
passed/failed holdout と leakage group が重ならない新しい holdout の独立 review と、
第 3 段階以降の新しい候補準備が必要である。

次の候補 cycle の最初の単位として、repository に `corpus-v11` を準備した。development
43 件と execution 8 件は `corpus-v10` から byte を変えずに引き継ぎ、holdout 251 件は
新しい fixture として再生成した。manifest の holdout digest は
`a3574dd0271a6ec66761270e869c80144aef72910c64919a8561d90f0592ce30`、
holdout leakage group digest 数は 139 件である。`leakageGroupId` は
`lqg-law-`、`lqg-ls-`、`lqg-topic-`、`lqg-case-courts-` および
`lqg-concept-` の安定 prefix に限定し、`corpus-v10` manifest の digest 集合との交差を
拒否する targeted test を追加した。`./internal/legalquerycorpus` の最小 test と
schema 検証は通過した。独立 review は semantic 9.0 / 10、testability 9.0 / 10、
security 9.0 / 10、blocker 0 で通過した。この時点では `default-4` request、
review attestation および holdout 実行はまだ作成していない。

続く次候補の第 4.3 段階では、`corpus-v11` の development 四十三件だけを使い、
法令コアを `core-2026-07-31-38`、cue 集合を
`core-cues-2026-07-31-17`、profile set を
`profile-set-sha256-0b00c3409408684b825f3c0bdf1c874bdc99e5383564d8e6b66fe83d4e417a69`
へ校正した。校正 fingerprint は
`456d237f980e1114638064411fb49ef91d281989b0578279b1081011eba2d9b0` で、
request error は二件中二件、plan は四十一件中二十八件、meaning は四十三件中
三十八件、evidence は三十八件中十一件、concept assertion は一件中一件が一致した。
候補 source と SOT の現行検証、完了済み request の不変 replay を分離し、
repository の pending/replay 両状態を検証した。独立 review は実装 9.2 / 10、
testability 9.0 / 10、security 9.0 / 10、blocker 0 であり、commit
`1424b508d1f094ea8921c6a26ab1335ecb84cc5c` と replay 修正 commit
`b1ff1cb1ffbd32260867861c69b7ba388aad5d7e` に対する権威 CI
`30687556430` は成功した。この校正でも holdout fixture と評価結果は参照していない。

次候補の第 4.4 段階では、candidate content
`candidate-content-sha256-e8a5633b1acaf75bd9f2851dfe814ec1342178a9c3bf31ff11c03e900fda47d3`
を raw SHA-256
`28135a61617088cb14d1c100fb97e17454666fead9dfaa0a9f27df032290481f`
へ固定した。同じ exact byte と 52 件の SOT へ architecture 100 / 100、
testability 96 / 100、blocker 0 の独立 review を結び付け、`corpus-v11`、
`legal-query-evaluator-v1` および予約名 `default-4` を持つ request
`evaluation-sha256-2f8790cd9a969372660571031ed00069565443521ca840cdce9ef86fb1290c42`
を追加して `current.json` を未評価 request へ進めた。`default-3` の failed result と
failed report は変更不能な消費済み履歴として保持する。候補 command は起動せず、
`default-4` の report、result、failed report および baseline は作成していない。
production adoption は引き続き `corpus-v9`、`default-1` および現行 profile set を指す。

## 推奨順序

段階そのものの定義、順序および進行条件の定義元は
[SOT-ENG-039](../sot/50-engineering/39-content-bound-unified-query-rollout-stages.md)
とする。
この章の表は、現時点の進捗と確認範囲を追跡するための運用上の写像である。

| 段階 | 状態 | 目的 | 主な定義元 |
|---:|---|---|---|
| 1 | 完了 | relation の不変 model、cue schema version 3、共通 loader および固定 profile set の構造整合を準備し、v2 の role 対応へ更新する | `SOT-MODEL-030`、`SOT-ENG-030` |
| 2 | 完了 | positive task cue の role をそろえ、共通前処理で relation を生成し、各 profile 内で意図根拠レイヤと対象外候補 scope を適用できるようにする | `SOT-MODEL-025`、`SOT-MODEL-026`、`SOT-MODEL-030`、`SOT-ARCH-031`、`SOT-ENG-028`、`SOT-ENG-031`、`SOT-ENG-032` |
| 3 | 完了 | profile metadata schema version 2、共有末尾 sidecar、private evidence cluster、core の sidecar 適用、裁判例の独立適用および test 専用固定 profile set を順に完成させる | `SOT-MODEL-031`、`SOT-ARCH-025`、`SOT-ARCH-031`、`SOT-ARCH-036`、`SOT-ARCH-037`、`SOT-ARCH-038`、`SOT-ARCH-039`、`SOT-ENG-035` |
| 4 | 完了（判定は `failed`） | 現行集合の baseline schema・初回採用 manifest・adoption 基準 command、新規 holdout を含む `corpus-v10`、development だけで校正した次版固定 profile set および `default-3` 候補を順に準備し、閉じた CI handoff で一回の holdout 採用判定と tracked byte replay を完了した | `SOT-ARCH-033`、`SOT-ENG-024`、`SOT-ENG-026`、`SOT-ENG-033`、`SOT-ENG-036`、`SOT-ENG-038`、`SOT-ENG-039` |
| 5 | 保留（`default-3` は不合格） | passed の候補が得られた場合だけ、全採用要素と current tuple を一変更で公開既定へ切り替え、公開 notice、questions、非実行時の外部呼出しゼロおよび MCP response parity を固定検証する | `SOT-ARCH-033`、`SOT-MODEL-024`、`SOT-IF-051`、`SOT-ENG-024`、`SOT-ENG-029`、`SOT-ENG-033` |
| 6 | 未着手 | `GET /laws`、`GET /keyword`、`GET /law_data` の parser を一 endpoint ずつ移行した後、法令検索の canonical target 優先を application 層へ接続する | `SOT-IF-011`、`SOT-IF-052`、`SOT-IF-053`、`SOT-IF-054`、`SOT-ARCH-030` |
| 7 | 未着手 | code や評価成果物を変えず、前段の同一変更義務に含まれない scenario、help および説明文書だけを現行標準へ同期する | `SOT-SCN-010`、`SOT-ENG-039` |

## 段階 review 記録

| 段階 | 確認日 | 独立 review | security review | blocker | 確認範囲 |
|---:|---|---:|---:|---:|---|
| 1 | 2026-07-31 | 9.2 / 10 | 9.1 / 10 | 0 | v2 role、relation 保持、閉じた role 入力、profile 所有 ID、active 成果物の不変 |
| 2 | 2026-07-31 | 9.5 / 10 | 9.3 / 10 | 0 | 共通 relation 生成、positive task role、引用・言及・topic 除外、profile 内の意図根拠、対象外候補 scope、next/active 分離 |
| 3.1 | 2026-07-31 | 9.7 / 10（test 8.0 / 10） | 9.0 / 10 | 0 | schema version 1・2 の閉じた loader、存在状態、固定 set の共有校正と digest、active version 1 と test 専用 version 2 の分離 |
| 3.2 | 2026-07-31 | 9.2 / 10（test 9.4 / 10） | 9.0 / 10 | 0 | 閉じた共有末尾列、bounded maximal-path 判定、実前処理の二代表例、128・256 上限、active core・裁判例の非消費 |
| 3.4.1 | 2026-07-31 | 9.2 / 10（段階境界 9.4、test 8.8、実装 8.7） | 9.0 / 10 | 0 | 五 input kind の閉じた対応、同じ節・同じ主題、日付および span なし `ref` の task/resource 束縛 |
| 3.4.2 | 2026-07-31 | 9.7 / 10（実装 9.5 / 10） | 9.1 / 10 | 0 | 法令名本文検索語投影の三経路、完全一致する主題 span、同じ節の一意束縛、read・law search・article read 競合の fail-closed、同一表記の複数 identity、検証済み terminal task 例外 |
| 3.4.3 | 2026-07-31 | 9.3 / 10（敵対的 review 8.6 / 10） | 9.3 / 10 | 0 | shared-terminal の exact relation 消費、互換な複数 resource cue の根拠和集合、同一 span の別意味保持、異なる span の同値縮約、non-Cartesian 限定代替列、四 step 上限と五件目 `step_limit_exceeded`、cluster 単位の三件保持、active 非到達性 |
| 3.4.4 | 2026-07-31 | 9.2 / 10（testability 8.6 / 10） | 8.9 / 10 | 0 | step 内の閉じた根拠正規化、三 step 以上と入力順の決定性、候補和集合・score・`conceptSources`、group 曖昧性と source tuple 競合の fail-closed、`ref` 完全一致、provider 非依存性、private mapping 寿命、active composition root 非到達性 |
| 3.5 | 2026-07-31 | 8.5 / 10 | 9.0 / 10 | 0 | 裁判例の五 input kind、主題と span なし `ref` の分離と byte 完全一致、step 内根拠正規化、pack/provider 非意味化、shared terminal 非消費、non-Cartesian 限定代替列、同一節の競合 resource cue と曖昧な束縛の fail-closed |
| 3.6 | 2026-07-31 | 9.0 / 10 | 9.0 / 10 | 0 | 実 next profile の production 固定順構成、schema version 2 と共有校正値、欠落・重複・逆順・weight 差と順序差の拒否、active composition root と標準評価経路からの非到達性 |
| 4.1 | 2026-07-31 | 9.1 / 10（testability 8.8 / 10） | 8.8 / 10 | 0 | `default-1` version file と current baseline の byte 完全一致、初回 adoption history と `current.json`、baseline schema と閉じた loader、`--adoption` 固定の標準評価 command、中央品質ゲートの adoption 入口、catalog の corpus・baseline・version と verification artifact 実在検証 |
| 4.2 | 2026-07-31 | 8.8 / 10（testability 8.5 / 10） | 9.1 / 10 | 0 | schema version 2 の三成果物と typed decoder、development assertion 十一件、追加 coverage 七件と safety pair、leakage digest の再計算と原 ID 非露出、`corpus-v9`・schema version 1 および `corpus-v10` の byte 固定、`step_limit_exceeded` だけに限定した空 meaning 境界 |
| 4.3 | 2026-07-31 | 9.4 / 10（testability 9.0 / 10） | 9.3 / 10 | 0 | schema と development だけを開く loader、原 byte content digest、二回の独立構成、四十三件全体の fingerprint と scorecard、core・裁判例の専用校正 artifact、active からの非到達性、共有末尾の同値縮約後四 step 境界 |
| 4.4 | 2026-07-31 | 9.6 / 10（testability 9.2 / 10） | 8.6 / 10 | 0 | schema v2 の五成果物、52 件 SOT digest 付き review attestation、`corpus-v10`/`default-2` request、`current.json` pointer、空 history root の fail-closed 解釈、manual CI handoff 入口、候補 baseline writer、canonical subtree の自己検証 |
| 4.5 再準備 | 2026-07-31 | 10.0 / 10（testability 8.4 / 10、実装 8.9 / 10） | 9.1 / 10 | 0 | report 前に停止した `default-2` の三回の dispatch では有効 result と holdout 消費がないことを確認。`HOME` 非依存 cache、execution 不一致の失敗分類、raw draft の個別 evidence mapping と縮約後十六件上限を修正し、旧準備を不変に保持したまま `default-3` の exact content、二件の review attestation、request および current pointer を追加。準備 commit の権威 CI と一回の manual dispatch は成功し、構造上有効な `outcome=failed` を handoff |
| 4.5 handoff | 2026-08-01 | 10.0 / 10（testability 10.0 / 10） | 10.0 / 10 | 0 | failed report と result の request/report digest binding、baseline 非生成、消費済み履歴一件、current failed result および replay byte を確認。更新後 commit の権威 CI は成功し、同じ commit の tracked byte replay artifact 二件が tracked history と完全一致 |
| 次 cycle corpus-v11 | 2026-08-01 | 9.0 / 10（testability 9.0 / 10） | 9.0 / 10 | 0 | schema version 2、development 43 件、独立生成した holdout 251 件、execution 8 件、安定 leakage group 139 件、消費済み `corpus-v10` manifest との digest 非交差、alias 衝突・曖昧性・pack 有無・実事件参照・表記揺れ・誤記・v2 境界の代表意味、再現性および不変性。`default-4` と holdout 実行は対象外 |
| 次 cycle 4.3 | 2026-08-01 | 9.2 / 10（testability 9.0 / 10） | 9.0 / 10 | 0 | `corpus-v11` development だけによる `core-38` と profile set の校正、四十三件 scorecard と fingerprint、pending current と完了済み replay の外部参照検証分離。holdout 非参照 |
| 次 cycle 4.4 準備 | 2026-08-01 | architecture 10.0 / 10（testability 9.6 / 10） | — | 0 | exact candidate content、二件の content-bound review、`corpus-v11` / `default-4` request、未評価 current pointer。候補 command、report、result、failed report および baseline は未生成 |
| 再準備 cycle corpus-v12 | 2026-08-01 | semantic 9.5 / 10 | testability・security 8.3 / 10 | 0 | schema version 2、development 43 件、独立生成した holdout 251 件、execution 8 件、安定 leakage group 204 件、`corpus-v10`・`corpus-v11` との holdout・leakage・正規化意味群の非交差、v11 からの development・execution byte 継承、再現性および不変性。候補 request と holdout 実行は対象外 |
| 再準備 cycle 4.3 | 2026-08-01 | 9.4 / 10（testability 9.2 / 10） | 9.1 / 10 | 0 | `corpus-v12` development 43 件だけによる既存 profile set の再校正、同一 policy・version と scorecard、新 fingerprint、holdout・候補評価成果物の非参照、既存の候補 request と pointer の不変 |
| 後続 cycle evaluator v2 | 2026-08-02 | 8.8 / 10 | 9.0 / 10 | 0 | v1 の再現意味を保存した exact evaluator registry、期待 plan と実入力 error、期待 request error と実受理の二境界だけを semantic failure へ写像する v2、unknown version の fail-closed、development-only preflight |
| 後続 cycle corpus-v14 | 2026-08-02 | 内容 8.3 / 10 | 構造 8.9 / 10 | 0 | development 43 件と execution 8 件の byte 継承、独立 holdout 255 件、leakage digest 228 件、過去四版との五軸非交差、十二 category、安全対および四派生観測母集団 |
| 後続 cycle 4.4 準備 | 2026-08-02 | architecture 10.0 / 10（testability 9.2 / 10） | — | 0 | byte 不変の candidate content、同期済み 52 SOT、二件の新しい content-bound review、`legal-query-evaluator-v2`、`corpus-v14`、`default-7` request および未評価 current pointer。holdout、report、result、baseline は未生成 |

4.4 の最終 security review で残った非 blocker のうち、非 `GO*` 環境の継承は、
4.5 の bootstrap が `PATH`、`GOROOT`、`GOMODCACHE`、`GOCACHE`、`TMPDIR` と固定
`GO*` 以外を拒否することで閉じた。Go executable の provenance と read-only module
cache の実効性は、固定 setup-go を使う候補評価 `30682022985` と tracked replay
`30682972843` で確認した。

## 第 3 段階以降の SOT 文書 review

以前の第 3・第 4 段階だけを対象にした通過記録は、
`SOT-ARCH-034`、`SOT-ARCH-035` および `SOT-ENG-037` の追加と、第 5 から
第 7 段階の境界変更によって適用範囲が変わったため、現行文書の通過根拠として
使用しない。次の二行は commit `a5adf4fc3bae819579b681e1a04308779ddcf805`
時点の記録であり、今回の 3.4.1 から 3.4.4 の文書更新に対する通過記録ではない。
今回の最終 review は、指摘反映後の行を別に追加して追跡する。

| 確認日 | architecture review | testability review | blocker | major | minor | 確認範囲 |
|---|---:|---:|---:|---:|---:|---|
| 2026-07-31 | 9.6 / 10 | 9.8 / 10 | 0 | 0 | 0 | core・judicial の根拠対応、第 3 から第 7 段階、profile 候補と provider 候補の境界、candidate identity、holdout の一回利用と compact leakage index、report と履歴の資源上限、evaluator の採用・再現・rollback、corpus の不変性、固定検証 ID |
| 2026-07-31 | 9.1 / 10 | 9.4 / 10 | 0 | 0 | 0 | `SOT-ARCH-031` の同一 span 異種事実境界、共有 `explicit_task` 例外、`SOT-ARCH-034` の法令名本文検索語投影三経路、core・judicial の fail-closed fixture、および `SOT-ENG-035` の共有校正一致条件 |
| 2026-07-31 | 10.0 / 10 | 9.8 / 10 | 0 | 0 | 0 | 今回の後継 SOT、step 内根拠正規化、`ref` 忠実性、候補内容 identity、52 件の固定 review 契約集合、review attestation、検証済み source view、資源上限および七段階の導入順序。敵対的 review は 9.8 / 10、blocker・major・minor はすべて 0。文書設計だけを対象とし、実装完了は表さない |

この review は文書設計だけを対象とし、第 3 段階以降の実装完了を表さない。
実装状態は次節のとおり 3.1 から 3.6、4.1 および 4.2 が完了した。
4.3 と 4.4 は実装、独立 review および権威 CI を完了した。4.5 は worker と
handoff 境界の準備実装、三件の report 前停止への修正、新しい `default-3` に
対する独立 review、準備 commit の権威 CI および一回の manual dispatch を完了した。
判定は `failed` であり、failed report と result の handoff および独立 review は
完了した。更新後 commit の権威 CI と tracked byte replay も完了したため、第 4 段階は
不合格の結果を変更不能な履歴へ固定して終了した。第 5 段階へは進まない。

## 第 3 段階の内部進捗

内部順序の定義元は `SOT-ENG-039` とし、ここでは実装状態だけを追跡する。

| 順序 | 状態 | 変更単位 |
|---:|---|---|
| 3.1 | 完了 | schema version 2 の profile metadata model、loader、存在状態および固定 set 整合 |
| 3.2 | 完了 | production-neutral な `SharedTerminalSequence` sidecar |
| 3.3 | 完了 | profile-private な根拠対応と evidence cluster |
| 3.4 | 完了 | core の sidecar 消費、複数主題 step および限定分岐 |
| 3.5 | 完了 | sidecar を消費しない `judicial-cases` 固有の限定分岐 |
| 3.6 | 完了 | 全 profile が schema version 2 と共有校正値を持つ test 専用固定 set |

### 3.4 の内部順序

| 順序 | 状態 | 変更単位 |
|---:|---|---|
| 3.4.1 | 完了 | 五 input kind の閉じた cue 対応、同じ節・同じ主題束縛、`topicOrdinal`、`asOf` と updates 日付差、span を持たない `ref` |
| 3.4.2 | 完了 | 法令名の本文検索語投影三経路、同じ節の content search 束縛、read/law search 競合時の fail-closed |
| 3.4.3 | 完了 | shared-terminal の task/resource 束縛、sidecar 消費、同一 span の別意味、異なる span の同値縮約、topic-local draft の完全順序、限定代替列、五件目での `step_limit_exceeded`、cluster 単位の三件保持 |
| 3.4.4 | 完了 | step 内根拠正規化後の候補和集合、private mapping の寿命、logical input と入力 `ref` の一致、provider 非依存性、active profile 不変の最終照合 |

## 第 4 段階の内部進捗

内部順序の定義元は `SOT-ENG-039` とし、ここでは実装状態だけを追跡する。

| 順序 | 状態 | 変更単位 |
|---:|---|---|
| 4.1 | 完了 | 現行 baseline schema、`default-1` version file、初回 adoption manifest、固定 evaluator registry、adoption 基準の標準 command と中央品質ゲート |
| 4.2 | 完了 | schema version 2、新規 holdout を含む `corpus-v10`、集合分離、development assertion、coverage、leakage digest および旧版・準備版の byte 固定 |
| 4.3 | 完了 | holdout を使わない development 専用 loader、原 byte digest、次版 profile set 校正および四十三件全体の決定的 fingerprint |
| 4.4 | 完了 | 内容固定済み候補 request、review attestation、`current.json` pointer、manual CI handoff 入口および候補 baseline writer |
| 4.5 | 完了（`outcome=failed`、holdout 消費済み） | 一回の holdout 判定、変更不能な result と `default-3` の failed history handoff、権威 CI および tracked byte replay |
| 次 cycle 4.2 | 完了 | 過去の全評価と leakage group が交差しない `corpus-v11` の独立準備 |
| 次 cycle 4.3 | 完了 | `corpus-v11` development だけによる `core-38` と固定 profile set の校正 |
| 次 cycle 4.4 | 不確定終了（再実行禁止） | exact candidate content、二件の review attestation、`default-4` request および pending pointer。専用 CI は artifact なしで非零終了し、report 完成前とは証明できない |
| 次 cycle 診断契約 | 有効化完了 | `SOT-ENG-040` の report 完成境界、閉じた終了 code、privacy、unknown の fail-closed および coordinated adoption 条件 |
| 再準備 cycle 4.2 | 完了 | `corpus-v12` の独立 holdout、過去の `corpus-v10` と `corpus-v11` からの leakage 分離、再現性および不変 byte の固定 |
| 再準備 cycle 4.3 | 完了 | `corpus-v12` development 43 件だけによる既存 policy・version の再校正と決定的 fingerprint の固定。候補 request と pointer は不変 |
| 再準備 cycle 4.4 | 完了 | 失敗診断契約、同期後の SOT byte に結合した二件の review、`corpus-v12` / `default-5` request および未評価 current pointer。holdout は未実行 |
| 再準備 cycle 4.5 | report 前失敗（同一 ID の再実行禁止） | `default-5` request は二件の remote run がともに終了 code `12` で停止し、report、result および handoff を生成しなかった。三回目は実行せず、`corpus-v13` / `default-6` の新しい準備 cycle へ置き換える |
| 再準備 cycle 4.6 | report 前失敗（同一 ID の再実行禁止） | judicial-cases の raw 同値 draft 縮約修正、`corpus-v13`、二件の content-bound review および `default-6` request を固定した後、一回の remote run が終了 code `12` で停止。report、result および handoff は未生成 |
| 後続 cycle evaluator v2 | 完了 | v1 の再現意味を不変に保ち、期待 plan と実入力 error、および期待 request error と実受理だけを semantic failure へ写像する exact v2 と unknown version の fail-closed を固定 |
| 後続 cycle 4.2 | 完了 | `corpus-v14` の独立 holdout、`corpus-v10` から `corpus-v13` との五軸非交差、v13 development・execution byte 継承および四派生観測母集団を固定 |
| 後続 cycle 4.4 | 準備完了・未評価 | byte 不変の candidate content、二件の新しい review attestation、`legal-query-evaluator-v2`、`corpus-v14`、`default-7` request および current pointer を固定。manual workflow は未実行 |

### `default-4` の不確定終了と診断契約

`default-4` は準備 commit `eccc9c5866f10405c26db165ce968d44d5324c21` に対する
manual workflow `30688691089` で一回起動した。worker command は generic な
非零終了となり、report/result artifact は upload されなかった。ただし artifact の
欠落だけでは、構造上有効な report の完成前に失敗したとは証明できない。したがって
同じ `evaluationId` を再実行せず、`default-4`、`corpus-v11` の holdout digest および
全 leakage group digest を後続 request で再利用しない。

失敗位置を query、fixture、case ID または内部 error 本文なしで判別する設計は、
当初 `SOT-ENG-040` の草案に分離した。初回レビューは `4.0 / 10`、blocker 一件であり、
`SOT-ENG-038` の CI log privacy と不確定 retry 条件の衝突を指摘された。修正後の
独立レビューは `9.0 / 10` と `9.2 / 10`、いずれも blocker 零、major 零となった。

診断実装の先行 commit `b69da4f`、`cce4d0a`、`c69d0dc` および `60326c2` だけでは、
草案の採用完了とは扱わなかった。再準備 cycle 4.4 で、失敗 stdout/stderr の空化、
child 開始前の `worker_start` と開始後の `unknown` の厳密分離、report と result の
直列化失敗を完成境界の前後へ正しく分類する検証、および `SOT-ENG-038` の privacy
同期を一変更へそろえて `SOT-ENG-040` を有効化した。

再準備 cycle の第 4.2 段階として、`corpus-v12` を独立変更で準備した。
development 四十三件と execution 八件は `corpus-v11` から byte を変えずに継承し、
holdout 二百五十一件と leakage group 二百四件を新しく固定した。holdout digest は
`6cd334e801499b0fe7de55532afb4c32254af4e66cc69e4922b71f38124fbfc0` である。
`corpus-v10` と `corpus-v11` の holdout digest、leakage group digest および
正規化した request と期待意味の組との交差を拒否し、最初の review で見つかった
過去の空入力意味群の再ラベル付けを別の入力境界へ置き換えた。再 review は semantic
9.5 / 10、testability・security 8.3 / 10、blocker 0 で通過した。この変更では
profile、辞書、候補 request、pointer、report、result、baseline および manual workflow を
変更または実行していない。

再準備 cycle の第 4.3 段階では、`corpus-v12/development` の四十三件だけを
development-only の複製へ置き、二回独立に構成した同じ test 専用 profile set を
校正した。development が `corpus-v11` と byte 同一であるため、policy、cue、辞書、
候補生成規則および次の version は変更していない。

- core: `core-2026-07-31-38`
- judicial-cases: `judicial-cases-2026-07-31-12`
- ranking: `legal-query-ranking-2026-07-31-2`
- profile set: `profile-set-sha256-0b00c3409408684b825f3c0bdf1c874bdc99e5383564d8e6b66fe83d4e417a69`

四十三件の scorecard は、request error 2 / 2、plan outcome 28 / 41、meaning
signature 38 / 43、evidence assertion 11 / 38、concept assertion 1 / 1 で従来値と
一致した。corpus identity を含む新しい校正 fingerprint は
`785c9249653c8dcdf9bea047787995e111d8ad2bf3a3fddc86b203e2446320d4` である。
この校正では holdout、report、result、failed report または baseline を参照せず、
既存の候補 request と pointer、production の active profile および標準 command を
変更していない。

再準備 cycle の第 4.4 段階では、意味 source が同じため exact candidate content
`candidate-content-sha256-e8a5633b1acaf75bd9f2851dfe814ec1342178a9c3bf31ff11c03e900fda47d3`
を再利用し、同期後の `SOT-ENG-038` 原 byte を含む五十二件の固定 SOT 集合へ
architecture `92 / 100` と testability `80 / 100`、いずれも blocker 0 の独立 review を
新しく結び付けた。`corpus-v12`、`legal-query-evaluator-v1` および未使用予約名
`default-5` を持つ request は
`evaluation-sha256-c53a7d0d28ef35bd2aab081680c1112b6aee9e649f19fb789ec2f0e0e35a4a87`
であり、`current.json` はこの request だけを指す。置換済みの `default-4` request と
参照成果物は変更せず、同 request の holdout digest、leakage group digest および
baseline reservation を再利用していない。この段階では candidate workflow、holdout、
report、result、failed report または baseline を実行若しくは生成していない。
同じ差分の最終 code review は `9.1 / 10`、security/privacy review は
総合 `9.1 / 10`、security `9.2 / 10` で、いずれも blocker 0 となった。
security review で最初に検出した handoff 読戻しの blocker は、bootstrap の
標準 library 制約を保ったまま result と report の未知 field、重複 key、後方 token、
非 canonical byte、depth および value 上限を閉じて検証することで解消した。

### `default-5` の確定的な report 前失敗

`default-5` は準備 commit `fa4b95d4fb74649d68a044a0920f357a4edeeef4` に対する
外部権威記録の manual workflow
[`30728860067`](https://github.com/geonwoo-jeong/japanese-law-mcp/actions/runs/30728860067)
で起動した。同じ操作中に重複 dispatch
[`30728862530`](https://github.com/geonwoo-jeong/japanese-law-mcp/actions/runs/30728862530)
も発生した。GitHub Actions の両記録では candidate worker の終了 code `12`
（`evaluate_build`）で停止し、report、result および artifact を生成しなかった。
これは有効な holdout 判定または handoff ではない。終了位置が閉じた code で確定した
ため、同じ evaluation ID の三回目は実行しない。

holdout 本文、query、case ID および期待値を出力しない集計で構造を確認した結果、
`corpus-v12` の二百五十一件に対する派生観測の対象件数は
`composition-core-pack=0`、`composition-pack-disabled=10`、
`composition-ref-read-search=0`、`composition-four-step-budget=5` であった。
標準 report は四観測をすべて一件以上の分母で生成するため、semantic 集計が完了しても
対象が零件の二観測によって report を完成できない構造であった。ただし終了 code `12` は
semantic と派生観測を同じ段階へ縮約するため、この欠陥だけを当該 run の最初の停止位置と
断定しない。`default-5` の予約名または CI infrastructure の障害を原因とは扱わない。

固定済みの `corpus-v12`、request、review および二件の remote run は変更不能な
診断記録として保持する。`SOT-ENG-038` の新規 request 作成境界では
`corpus-v1` から `corpus-v12` を replay 専用とし、再利用を禁止する。次の cycle は、
四観測をそれぞれ一件以上持ち、過去に
消費または廃棄した holdout と分離した `corpus-v13`、未使用予約名 `default-6`、
新しい content-bound review 二件および新しい evaluation ID を準備してから、
一回だけ第 4 段階 5 を実行する。

### `corpus-v13` / `default-6` の再準備

`judicial-cases` の同値 raw draft が十七件以上ある場合に、同値縮約より前の共通上限で
誤って停止していた問題を profile 内の draft 別 private mapping で修正した。非同値の
最終候補十六件上限、cue set、ranking policy、active production profile および adoption
tuple は変更していない。candidate judicial profile は
`judicial-cases-2026-08-02-13`、candidate profile set は
`profile-set-sha256-c6499c5843e993d749550a1ec71ca217234f807057b8ed8dc4cc4a75af282dc6`
へ更新した。

固定 Linux build context から生成した candidate content は
`candidate-content-sha256-538c2c573b44c43b532b66a3ec6b8bc71ddb66d6cac91c4a4327f7bad2f4a610`
である。これと同期後の required SOT byte に対し、architecture review は九十六点、
testability review は九十六点、いずれも blocker、major および minor 零件で承認した。
二件の attestation、`corpus-v13`、未使用予約名 `default-6` を結合した未評価 request は
`evaluation-sha256-21e19fd4121131f60f21928d1ec900c3cff18003748634772504d5f8ea3afc0c`
であり、`current.json` はこの一件だけを指す。旧 `default-5` request と二件の失敗 run
は診断履歴として変更せず、同じ ID を再実行しない。

この時点では新 request の holdout 評価、report、result、baseline および production
adoption は未実行である。準備 commit の全検証と remote quality gate が成功した後に、
この evaluation ID を一回だけ第 4 段階 5 へ渡す。

### `default-6` の report 前失敗と evaluator v2 の準備

準備 commit `a0c0bbadeacdd92444a29a905c07240be028f64b` に対する manual workflow
[`30734360848`](https://github.com/geonwoo-jeong/japanese-law-mcp/actions/runs/30734360848)
は、候補 worker の終了 code `12`（`evaluate_build`）で停止した。report、result
および artifact は生成されていない。同じ evaluation ID、`default-6`、holdout digest
および leakage group digest は後続評価へ再利用せず、この run も再実行しない。

`corpus-v13` は四つの派生観測母集団をすべて非零にしたため、`corpus-v12` で確認した
零母集団だけでは今回の停止を説明できない。holdout を再実行せず evaluator の境界を
静的に照合した結果、期待 `plan` に対する実際の入力 error、および期待
`request_error` に対する実際の受理を、semantic 指標の不一致ではなく report 構成 error
へ昇格させる経路が残っていた。この境界では、候補が受入基準を満たさないだけでも有効な
`outcome=failed` report を残せず code `12` になる。

後続 cycle では、既存 `legal-query-evaluator-v1` の再現意味を変更せず、上記二境界だけを
semantic の失敗指標へ写像する `legal-query-evaluator-v2` を追加した。期待 `plan`
が入力 error になった case は reproducibility の失敗にも数えるが、期待
`request_error` が受理された case は従来どおりその母集団から除外する。
preprocessor、profile contribution、selector、report schema または binding の内部 error は
引き続き report 完成前失敗とする。v2 の development-only 構造検証、独立 review および
準備 commit の権威 CI は成功した。

### `corpus-v14` / `default-7` の再準備

`corpus-v14` は development 四十三件と execution 八件を `corpus-v13` から byte 単位で
継承し、独立 holdout 二百五十五件と leakage digest 二百二十八件を固定した。
`corpus-v10` から `corpus-v13` までの消費済み・廃棄済み集合に対し、case ID、完全 request、
ComparisonKey、leakage group および期待意味署名の五軸を再利用しない。十二 category、
安全 variant 対および四つの派生観測母集団はすべて非零である。内容 review は
`8.3 / 10`、構造 review は `8.9 / 10`、いずれも blocker 零で通過した。

候補内容は `default-6` 準備時の
`candidate-content-sha256-538c2c573b44c43b532b66a3ec6b8bc71ddb66d6cac91c4a4327f7bad2f4a610`
から変わっていないため、既存の immutable content manifest を再利用した。現行五十二 SOT
の原 byte と同じ candidate content に対し、architecture `10.0 / 10` と testability
`9.2 / 10` の独立 review を新しく固定した。どちらも approved、blocker と major は零であり、
testability の環境前提と負例固定検証に minor 二件を残した。

二件の attestation、`legal-query-evaluator-v2`、`corpus-v14` および未使用予約名
`default-7` を結合した未評価 request は
`evaluation-sha256-bf3567625d79634f6be2621e870459bd50221ac041dd146dbcfededec2676cb1`
であり、`current.json` はこの一件だけを指す。旧 `default-6` request と失敗 run は不変に
保持し、同じ ID、baseline、holdout digest または leakage digest を再利用しない。
この準備変更では holdout、report、result、baseline および production adoption を生成しない。
準備 commit の権威 CI が成功した後にだけ、同じ commit の manual workflow を一回起動する。

### `default-7` の report 前失敗と schema version 3 の採用規定

`default-7` の一回評価は候補 worker の終了 code `12`（`evaluate_build`）で停止し、
report、result および artifact を生成しなかった。同じ evaluation ID、予約 baseline、
holdout digest および leakage group digest は再利用せず、この run も再実行しない。

候補所有の前処理または profile 回収 error だけを一件の定量的失敗へ写像する
`legal-query-evaluator-v3` の exact routing は準備済みである。ただし現行候補 source には
実 marker を接続しておらず、current evaluator と schema version 2 request は変更して
いない。次の変更は `SOT-ENG-042` に従い、実 marker、schema version 3 の content
manifest、二件の新規 review、未使用 corpus と baseline を持つ別 request、および pointer
を一つの準備単位として作る。holdout 評価と production adoption はその後の別単位である。

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
development 集合だけで次版固定 set を校正する。次に `SOT-ENG-038` の閉じた
content manifest、二件の review attestation、request と pointer、候補 set を
直接構成する CI 専用入口、候補 writer、
初回の `default-2` 予約名と、再準備後の現行 `default-3` 予約名および出力先を
別変更で準備する。これらを標準 command、製品 CLI、
設定、MCP、transport または中央品質ゲートの現行参照先にしない。
request は exact evaluator version、corpus manifest が持つ
`holdoutLeakageGroupDigests` の compact index、profile metadata、cue、辞書、
composition および意味判定 source set の candidate content digest を固定する。
architecture と testability の二件の独立 review は、同じ candidate content、
対象 SOT の ID と原 byte digest、rubric の内容 digest、8.0 / 10 以上、
blocker なしおよび approved 判定を持つ不変 attestation として request へ
結び付ける。この準備変更では holdout を起動せず、report、result または候補
baseline を生成しない。

review 済みの同一 commit に対して専用 CI job を一回だけ起動し、合格 report は
その request が予約した version file、失敗 report は変更不能な failed history へ同じ byte の
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
`baselines/default.json` は、準備済み `default-3` version file と同じ byte へ
同じ採用変更で切り替える。adoption manifest、合格 request、標準 command および
rollback 先はそれぞれ同じ exact evaluator version を指し、current evaluator への
fallback を許可しない。合格 request の candidate content と production の
metadata、cue、辞書、composition および意味判定 source set を完全一致させ、
`core-evidence-production-neutral` を
`core-evidence-production-adopted` へ置き換える。公開 notice と questions、
非実行時の外部呼出しゼロ、
`content` と `structuredContent` の同値性および transport 間の同値性もこの段階で
固定検証し、第七段階へ延期しない。

第六段階の provider parser と mapping は、`GET /laws`、`GET /keyword`、
`GET /law_data` の順に一 endpoint ずつ、統合照会の意味解釈とは独立した
変更単位にする。その後、application 層の law-target resolver と page 内安定優先を
別変更で接続する。raw response、parser の error 分類および page item の同一性・
順序は provider/application 専用 fixture で検証し、意味評価 baseline に証明させない。
第六段階の各番号では評価投影と検索例カタログを不変に保つ。必要な変更がそれらを
変える場合は現行の七段階 rollout と `SOT-ENG-038` の対象外とする。provider、
parser、resolver または mapping の候補 identity、production 非到達性、評価
request、原子的採用 tuple、rollback、固定検証および資源境界を定義する別の
新しい有効な SOT が採用されるまで、その変更へ着手しない。profile set 専用の
candidate request へ component field を足して流用しない。Wiki の実装済み範囲は
実際に完了した段階の同じ変更で同期する。第七段階は、code、評価成果物または
公開 MCP 契約を変えない文書専用段階とし、将来状態を先に「実装済み」または
「現行確認済み」と記載しない。

## 進行条件

各段階および `SOT-ENG-039` が番号を付けた段階内変更は、一度に一つずつ進め、
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
