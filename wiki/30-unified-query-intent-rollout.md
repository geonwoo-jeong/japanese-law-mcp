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
権威 CI はまだ判定前である。

したがって、`corpus-v10` は準備済みであっても、relation 対応の意味判定、
`default-2` および対応する検索例カタログとともに、まだ現行標準ではない。

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
| 4 | 着手中（4.1 完了、4.2 実装・独立 review 通過、権威 CI 判定対象） | 現行集合の baseline schema・初回採用 manifest・adoption 基準 command、新規 holdout を含む `corpus-v10`、development だけで校正した次版固定 profile set および `default-2` 候補を順に準備し、閉じた CI handoff で一回の holdout 採用判定を行う | `SOT-ARCH-033`、`SOT-ENG-024`、`SOT-ENG-026`、`SOT-ENG-033`、`SOT-ENG-036`、`SOT-ENG-038`、`SOT-ENG-039` |
| 5 | 未着手 | 全採用要素と current tuple を一変更で公開既定へ切り替え、公開 notice、questions、非実行時の外部呼出しゼロおよび MCP response parity を固定検証する | `SOT-ARCH-033`、`SOT-MODEL-024`、`SOT-IF-051`、`SOT-ENG-024`、`SOT-ENG-029`、`SOT-ENG-033` |
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
実装状態は次節のとおり 3.1 から 3.6 までと 4.1 が完了した。
4.2 は実装と独立 review を完了して権威 CI の判定対象であり、4.3 以降は
`未着手` とする。

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
| 4.2 | 実装・独立 review 通過（権威 CI 判定対象） | schema version 2、新規 holdout を含む `corpus-v10`、集合分離、development assertion、coverage、leakage digest および旧版・準備版の byte 固定 |
| 4.3 | 未着手 | holdout を使わない development 専用の次版 profile set 校正 |
| 4.4 | 未着手 | 内容固定済み候補 request、review attestation および CI handoff 入口 |
| 4.5 | 未着手 | 一回の holdout 判定、変更不能な result と `default-2` 候補 handoff |

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
`default-2` の予約名と出力先を別変更で準備する。これらを標準 command、製品 CLI、
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
