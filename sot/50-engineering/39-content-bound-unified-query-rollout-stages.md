# SOT-ENG-039: 内容固定済み候補による統合照会の導入段階と変更順序

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

不変 model、cue artifact の schema と loader、固定 profile 所有境界および
active 成果物の不変条件を整える。ここでいう共通 loader は cue entry と
syntax role を閉じた成果物から読むところまでとし、`CueTaskRelation` の生成、
五層の意味適用、step ごとの evidence mapping または cluster を実装しない。
公開既定の meaning、decision、step、reason または外部呼出し境界を変えない。

### 第 2 段階: profile 内の relation 適用

第 1 段階の loader が返す role だけから共通前処理で relation を生成し、各 profile
が `SOT-ARCH-031` の五層順と対象外候補 scope を候補 draft の境界へ適用できる
ようにする。この段階では、各 step を成立させた span、layer、evidence code または
`topicOrdinal` を private mapping として保持せず、evidence cluster と
`branchRetentionMargin` を適用しない。未採用の next profile set を production から
選択可能にしない。

### 第 3 段階: 限定分岐と列挙境界

`SOT-ARCH-025` と `SOT-ARCH-037` が定める、共有末尾 cue による主題分離と
evidence cluster 単位の限定分岐保持を、profile 内の規則として完成させる。
共有末尾 cue の構造確認は `SOT-ARCH-021` の閉じた separator 検証を使い、
`SOT-MODEL-031` の `SharedTerminalSequence` だけを profile へ渡し、
原文または token 列を公開しない。限定分岐は
`SOT-ENG-035` の独立した `branchRetentionMargin` を使い、`singleMargin`
または `hedgeMargin` を代用しない。この段階の `branchRetentionMargin` は
test 専用 set の暫定値に限り、active metadata、標準 command または baseline を
更新しない。

第 3 段階の内部変更は、次の順で一つずつ行う。

1. 次版 profile 用に、schema version 2 の metadata model、loader、
   `branchRetentionMargin` の存在状態および固定 set の整合検証を準備する
2. production-neutral な `SharedTerminalSequence` sidecar を構築し、active
   profile が意味候補へ使用しないことを確認する
3. `SOT-ARCH-031` の共通対応に従う private evidence mapping の寿命、step
   対応、`topicOrdinal` および cluster key の共通骨格を構築する。profile 固有の
   許可事実はこの番号で推測しない
4. `SOT-ARCH-036`、`SOT-ARCH-037` および `SOT-ARCH-039` に従い、core profile
   だけが sidecar を消費して共有末尾 cue の複数主題 step、input kind ごとの
   根拠対応および限定分岐を作る test 専用経路を完成させる
5. `SOT-ARCH-036`、`SOT-ARCH-037` および `SOT-ARCH-038` に従い、
   `judicial-cases` profile は sidecar を消費せず、自身の位置付き出現による
   input kind ごとの根拠対応と cluster だけで限定分岐を適用する
6. core と `judicial-cases` を production と同じ固定順で組み立て、全 profile が
   schema version 2、同じ ranking version および同じ
   `branchRetentionMargin` を持つ一つの test 専用固定 profile set を完成させる

1 から 5 は個別 model、loader、profile または integration test の準備であり、
それだけでは校正、holdout、標準 command、baseline、production または採用候補の
固定 profile set とみなさない。6 を満たした固定 set だけを第 4 段階の development
校正へ渡せる。

4 は、次の内部変更単位をこの順で一つずつ完了した場合だけ完了とする。

1. `3.4.1`: 五 input kind の閉じた task/resource cue 対応、同じ節・同じ主題への
   束縛、`topicOrdinal`、`asOf` と updates 日付の差、および span を持たない
   `ref` の閉じた resource cue 省略例外を確定する。`asOf` と updates 日付の差は
   `core-evidence-mapping-input-kinds` の fixture に含める
2. `3.4.2`: 法令名を本文検索語へ投影できる三経路、同じ節の一意な
   content search 束縛、および read/law search との競合時の fail-closed を確定する
3. `3.4.3`: 正確な terminal relation を持つ sidecar の消費、同一 span の別意味、
   異なる span の同値縮約、topic-local draft の完全順序、限定代替列、
   四 step 上限および cluster 単位の三件保持を確定する。ここでいう四 step
   上限は、五件目を黙って切り捨てず `step_limit_exceeded` とする境界を含む
4. `3.4.4`: step 内の根拠正規化後に候補全体へ和集合し、曖昧束縛、
   `SOT-MODEL-022` の request と logical input の `ref` 完全一致、
   private mapping の寿命、provider 非依存性および active profile の不変を
   最終照合する

`3.4.1` から `3.4.4` は、それぞれ前の番号の契約と検証を維持する独立した
変更単位とする。後続番号の実装または fixture を先行番号へ混在させず、
各番号の review、commit および権威 CI が完了してから次へ進む。

2 の `shared-terminal-production-neutral` は、第 5 段階より前の active profile が
sidecar を消費しないことを確認する移行時検証 ID とする。第 5 段階で法令コアの
採用済み profile を切り替えた後は、`SOT-ARCH-039` の core 消費契約を確認する
検証へ置き換え、永続する `SOT-MODEL-031` の非消費条件として残さない。

`3.4.4` の `core-evidence-production-neutral` も第 5 段階より前にだけ使う
移行時検証 ID とする。原子的採用と同じ変更で終了し、評価に合格した候補内容と
current adoption tuple から到達する production 経路を確認する
`core-evidence-production-adopted` へ置き換える。

この段階は意味規則そのものを再定義せず、それらの採用順序上の変更単位だけを
固定する。

### 第 4 段階: 評価成果物の準備

現行採用済み profile set を変えずに、次の順で成果物を準備する。

1. production、現行 corpus、profile set、baseline、検索例および観測動作を
   変えず、`SOT-ENG-036` の baseline schema と閉じた loader を導入し、現行
   `default.json` と同じ byte の最初の version file を作る。同じ変更で、
   現行採用集合を `SOT-ENG-033` の初回 history manifest と `current.json` へ
   固定し、標準 command と中央品質ゲートの入口だけを
   `SOT-ENG-024` の adoption 基準 command へ切り替える。切替前後で評価対象 tuple、
   report byte および外部呼出し境界を一致させる
2. profile、辞書および既存期待値を変更しない独立変更で、新しい corpus と
   holdout を review し、digest を固定する
3. holdout の内容または結果を参照せず、development 集合だけで第 3 段階の
   test 専用固定 profile set を校正し、profile version と ranking version を固定する
4. holdout の内容または結果を読まずに、`SOT-ENG-038` の閉じた candidate
   content manifest、二件の有効な review attestation、evaluation request、
   pointer、CI handoff 専用 command、`SOT-ENG-036` の候補 writer、次の
   `baselineVersion` および出力先だけを準備する。各成果物の field、content
   identity、review binding、再現および資源境界は `SOT-ENG-038` を定義元とする。
   標準 command、中央品質ゲート、製品 CLI、設定、MCP または transport から候補を
   選択できず、この時点では command を起動せず、holdout の測定値を持つ report、
   result または baseline を生成しない
5. 4 の content-bound review 済み commit に対して `SOT-ENG-038` の専用 CI job を
   一回だけ
   起動し、固定 holdout で `SOT-ENG-024` の全受入基準を検査する。profile、
   corpus、evaluator、受入値または request を変えない handoff 変更で、
   passed report は予約した baseline version file、failed report は同 SOT の
   failed history へ同じ byte のまま保存する。report、result、corpus、
   profile set、metric 計算、one-time use および privacy 境界を独立 review し、
   `SOT-ENG-036` の `evaluation-baseline-candidate-isolation` を満たし、
   passed の場合だけ第 5 段階へ進む

5 の一回起動、再試行および同じ `evaluationId` の再現確認の意味は
`SOT-ENG-038` を定義元とし、本規定では 4 の review 完了後に 5 へ進む順序だけを
固定する。

初回の relation 依存変更では、2 の候補名を `corpus-v10`、4 の予約名を
`default-2` とする。これらは今回の初回候補を識別する名前であり、将来の導入で
常に固定する版ではない。初回導入後の現行値は current adoption tuple を
定義元とする。

`default-2` の構造上有効な report が完成する前に、候補 source または review 対象を
修正しなければ再試行できなくなった場合は、第 4 段階 4 の再準備として扱う。
`SOT-ENG-038` に従って `default-2` の不変な準備成果物を置換済み準備として残し、
次の未使用予約名、二件の content-bound review、新しい request および pointer を
一変更で準備する。有効な result がない旧 request を holdout 消費または第 4 段階 5 の
完了へ読み替えず、新 request の権威 CI が成功する前に manual dispatch へ進まない。

holdout の結果を見て同じ候補 profile set の値、辞書、規則または期待値を
調整しない。受入基準を満たさない場合は第 5 段階へ進まず、失敗した候補を
採用対象から外して、新しい準備変更として第 3 段階以降をやり直す。
passed と failed のどちらでも、構造上有効な result が一件できた holdout digest は
消費済みとし、後続候補の採用判定へ再利用しない。
失敗 report の配置、不変性、予約 baseline version の再利用禁止および
全ての過去の passed/failed holdout と同じ leakage group digest を含まない
新しい holdout の bounded な compact-index 検査は
`SOT-ENG-038` に従う。新しい holdout を独立 review してから、次の第 4 段階を行う。

### 第 5 段階: 原子的採用

`SOT-ARCH-033` の「原子的な採用」に列挙された全要素と、
`SOT-ENG-033` の current adoption tuple を、同じ採用変更で新しい profile set へ
切り替える。本規定では完全な採用要素を重複して列挙せず、いずれか一部だけを
先行切替しないという順序上の制約だけを定める。

採用する profile metadata、cue、辞書、composition および意味判定 source set は、
第 4 段階 5 で `outcome=passed` となった request の candidate content と完全一致
させる。同じ変更で `core-evidence-production-neutral` を終了し、
`core-evidence-production-adopted` により、その exact content と current adoption
tuple を使う production composition root だけが `SOT-ARCH-039` の sidecar、
限定代替列および private evidence mapping を消費することを確認する。

同じ採用変更では、`SOT-MODEL-024` と `SOT-IF-051` に従い、公開 MCP の
`completed`、`needs_clarification`、`capability_unavailable` および
`unsupported` の各結果について、固定 notice と questions、非実行時の外部呼出し
ゼロ、`content` と `structuredContent` の同値性および transport 間の同値性を
固定検証する。これらの公開観測値を表す検索例カタログを変更する場合は、
`SOT-ENG-029` に従って同じ採用変更へ含める。正確な案内文を corpus や baseline の
期待値へ重複して保存しない。

### 第 6 段階: 意味判定から独立した後続変更

e-Gov parser の error 分類、canonical target 優先、provider mapping 調整など、
意味判定 profile set と独立に進めるべき変更を、第 5 段階へ混在させず、次の
順序の別変更単位として扱う。

1. `SOT-IF-054` に従い、`GET /laws` の facade と capability が共有する parser、
   page 不変条件および `invalid_source_response` と
   `source_contract_changed` の分類を完成させる
2. `SOT-IF-052` に従い、`GET /keyword` の facade と capability が共有する parser
   および同じ error 分類境界を完成させる
3. `SOT-IF-011` に従い、`GET /law_data` の XML parser、入力と応答の同一性、
   安全な XML 境界および同じ error 分類境界を完成させる
4. 前三項の provider parser を変更せず、`SOT-ARCH-030` の application 層
   law-target resolver と page 内の安定優先を共通化し、`SOT-IF-053` の
   `search_laws` と統合照会の法令検索 facade へそれぞれの契約どおり接続する

一つの endpoint の parser 変更を、別 endpoint、意図判定 profile または
canonical target の順位変更と同じ変更へまとめない。各番号は、後述する
`SOT-ENG-036` 投影と検索例カタログの項目を変えない変更だけを扱う。Wiki には
実装進捗だけを同じ変更で同期する。

第 6 段階の各番号は、provider または application 層の専用 fixture で検証する。
`SOT-ENG-036` の baseline は `query_legal_information` の意味判定と実行投影だけを
表し、provider の raw response、parser の個別分岐、error の原文、page item の
同一性または順序を証明する成果物として使わない。各番号の変更後には current
adoption 基準の標準 command を CI で一回実行し、意味判定と実行投影が変わらない
ことを確認し、report byte、current tuple および catalog digest を不変に保つ。

必要な provider または resolver 変更が `query_legal_information` の
`SOT-ENG-036` 投影若しくは検索例カタログの項目を変える場合は、第 6 段階の
番号付き変更として着手せず、本七段階 rollout の対象外とする。
`SOT-ENG-038` の request は profile set 候補だけを識別するため、provider、
parser、resolver または mapping の候補 field を同 request へ追加して流用しない。

この種の変更を実装する前に、少なくとも次を一つの閉じた契約として定義する
別の新しい有効な SOT を先に採用する。

- provider、parser、resolver または mapping の候補 component ID、意味版、
  artifact digest および候補全体の canonical identity
- 候補を production、標準 command、MCP、CLI、設定および transport から
  選択できない準備状態
- 候補 identity、corpus、評価器、report および result を結ぶ評価 request
- 合格候補を current へ原子的に採用する tuple、previous への rollback および
  一部だけを切り替えない境界
- 専用 fixture、固定検証 ID、資源上限、privacy および権威 CI

この別 SOT が有効になるまでは、投影または検索例を変える component 候補を
準備、評価若しくは採用せず、第 4 段階 4・5または第 5 段階へ再入場したものと
扱わない。同じ一 commit の第 6 段階変更で baseline、history manifest、
`default.json`、検索例カタログまたは current pointer を直接書き換えない。

### 第 7 段階: 案内と scenario 同期

前段の原子的採用義務に含まれない利用シナリオ、help および説明文書だけを、
有効な SOT の意味を変えない範囲で現行標準の観測結果へ同期する文書専用段階とする。
`SOT-SCN-010` は規範的な定義元であり、現行 code に合わせて notice、questions、
zero-call または再照会動作の意味を変更しない。許可するのは、同 SOT の意味を
保ったリンク、表記、help および Wiki の進捗の同期だけとする。

code、profile、corpus、baseline、adoption manifest、検索例カタログ、公開 notice、
questions、MCP response または外部呼出し境界を変更しない。SOT と実装の意味が
一致しないことを発見した場合は第 7 段階で文書を実装へ合わせず、必要な SOT 変更を
先に独立 review し、公開契約の変更を第 5 段階の原子的採用としてやり直す。
公開非実行案内と response parity の実装・固定検証は第 5 段階で完了させ、
ここへ延期しない。将来状態を先回りして「実装済み」と書かない。

`unified-query-guidance-document-sync` は、`SOT-SCN-010` の規範的意味、
code、catalog、adoption tuple および外部呼出し境界が変更前後で一致し、文書の
リンク、表記、help または進捗だけが現行確認済み状態へ同期したことを検証する。

## 段階別の固定検証 ID

番号付き変更単位は、少なくとも次の固定 ID で識別される検証を持つ。ここに挙げる
ID は変更順序から検証へ到達するための索引であり、各動作の定義元を置き換えない。

| 変更単位 | 定義元 | 固定検証 ID |
|---|---|---|
| 3.1 | `SOT-ENG-035` | `profile-metadata-schema-versions`、`profile-metadata-branch-retention-presence` |
| 3.2 | `SOT-MODEL-031`、`SOT-ENG-039` | `shared-terminal-sequence-contract`、`shared-terminal-production-neutral` |
| 3.3 | `SOT-ARCH-031`、`SOT-ARCH-037` | `profile-private-evidence-mapping-lifetime`、`profile-evidence-cluster-key` |
| 3.4.1 | `SOT-ARCH-039` | `core-evidence-mapping-input-kinds`、`core-evidence-mapping-topic-positive`、`core-evidence-mapping-ref-no-span` |
| 3.4.2 | `SOT-ARCH-039` | `core-law-name-content-projection` |
| 3.4.3 | `SOT-ARCH-037`、`SOT-ARCH-039` | `core-shared-terminal-task-resource-binding`、`core-shared-terminal-evidence-cluster`、`core-bounded-non-cartesian-alternatives` |
| 3.4.4 | `SOT-MODEL-022`、`SOT-ARCH-036`、`SOT-ARCH-031`、`SOT-ARCH-039` | `core-multi-step-evidence-step-local-normalization`、`core-evidence-mapping-private-lifetime`、`core-evidence-mapping-provider-independent`、`core-evidence-mapping-fail-closed`、`core-evidence-production-neutral` |
| 3.5 | `SOT-ARCH-036`、`SOT-ARCH-037`、`SOT-ARCH-038` | `judicial-evidence-mapping-input-kinds`、`judicial-evidence-mapping-private-lifetime`、`judicial-evidence-mapping-topic-positive`、`judicial-evidence-mapping-ref-no-span`、`judicial-shared-terminal-rejected`、`judicial-evidence-mapping-pack-provider-invariant`、`judicial-evidence-mapping-fail-closed`、`judicial-multi-step-evidence-step-local-normalization`、`judicial-bounded-non-cartesian-alternatives` |
| 3.6 | `SOT-ENG-035` | `profile-metadata-ranking-consistency`、`next-profile-set-fixed-composition` |
| 4.1 | `SOT-ENG-033`、`SOT-ENG-036` | `evaluation-baseline-initial-bootstrap`、`evaluation-baseline-resource-maximum`、`evaluation-baseline-history-bounds`、`profile-set-adoption-canonical-bytes` |
| 4.2 | `SOT-ENG-026` | `legal-query-corpus-v2-development-assertions`、`legal-query-corpus-v2-holdout-coverage`、`legal-query-corpus-v2-leakage-digests`、`legal-query-corpus-immutable-version` |
| 4.3 | `SOT-ENG-024` | `next-profile-set-development-only-calibration` |
| 4.4 | `SOT-ENG-038` | `candidate-evaluation-closed-artifacts`、`candidate-evaluation-request-identity`、`candidate-evaluation-candidate-content-identity`、`candidate-evaluation-referenced-file-bounds`、`candidate-evaluation-review-attestation`、`candidate-evaluation-review-content-binding`、`candidate-evaluation-evaluator-version-match`、`candidate-evaluation-build-context-isolation`、`candidate-evaluation-current-single-target`、`candidate-evaluation-production-unreachable`、`candidate-evaluation-ci-authority`、`candidate-evaluation-consumed-digest-preflight` |
| 4.5 | `SOT-ENG-036`、`SOT-ENG-038` | `candidate-evaluation-deterministic-replay`、`candidate-evaluation-outcome-exit-semantics`、`candidate-evaluation-success-handoff`、`candidate-evaluation-failure-history`、`candidate-evaluation-single-holdout-use`、`candidate-evaluation-leakage-exclusion`、`candidate-evaluation-leakage-index-bounds`、`candidate-evaluation-output-privacy`、`candidate-evaluation-adoption-link`、`candidate-evaluation-immutable-version`、`evaluation-baseline-candidate-isolation` |
| 5 | `SOT-ARCH-039`、`SOT-ENG-033`、`SOT-IF-051` | `profile-set-atomic-adoption`、`profile-set-evaluator-version-identity`、`core-evidence-production-adopted`、`legal-query-clarification-guidance`、`legal-query-pack-disabled-guidance`、`legal-query-unsupported-guidance`、`legal-query-nonexecution-zero-calls`、`legal-query-content-structured-content-parity`、`legal-query-guidance-transport-parity` |
| 6.1 | `SOT-IF-054` | `egov-laws-runtime-response-classification`、`egov-laws-contract-change-separation`、`egov-laws-facade-capability-parser-identity`、`egov-laws-page-invariants` |
| 6.2 | `SOT-IF-052` | `egov-keyword-runtime-response-classification`、`egov-keyword-contract-change-separation`、`egov-keyword-facade-capability-parser-identity`、`egov-keyword-page-invariants` |
| 6.3 | `SOT-IF-011` | `egov-law-data-runtime-response-classification`、`egov-law-data-contract-change-separation`、`egov-law-data-input-response-identity`、`egov-law-data-xml-safety-boundary` |
| 6.4 | `SOT-ARCH-030` | `law-target-resolution-parity`、`law-target-page-stable-partition`、`law-target-no-extra-fetch`、`law-target-ambiguous-no-reorder`、`law-target-unified-no-reparse` |
| 7 | `SOT-ENG-039` | `unified-query-guidance-document-sync` |

一つの固定検証 ID が複数の閉じた assertion を持つ場合は、その assertion 群を
同じ fixture 群で pass/fail できる一つの契約として扱う。例えば
`core-evidence-mapping-input-kinds` は五 input kind、閉じた cue 対応、
同じ節・同じ主題束縛、および `asOf` と updates 日付差を同じ契約として確認する。
`core-evidence-mapping-ref-no-span` は、入力 `ref` が
`boundary/official_identifier` へ寄与しても span、独立主題または cluster の
`evidenceSpan` を持たず、構造検証済み `ref.resourceType` が resource を確定し、
一意な read の位置付き task cue だけを cluster に利用できる契約として確認する。
明示 resource cue は必須にしないが、存在する全 cue の互換性を確認する。
`core-shared-terminal-evidence-cluster` は
正確な terminal relation、同値縮約、四 step 上限および cluster 単位の保持を
同じ契約として確認する。

`core-shared-terminal-task-resource-binding` は、`教えて` と
`教えてください` の閉じた正常系、同じ節の互換 resource cue、競合する
task/resource cue、別節 cue および同じ surface の別 relation を一つの
task/resource 束縛契約として確認する。

非 Cartesian の限定代替列は profile 共通の契約とするが、固定検証 ID は変更単位
ごとに分ける。3.4.3 の `core-bounded-non-cartesian-alternatives` は core の
共有末尾列で、3.5 の `judicial-bounded-non-cartesian-alternatives` は
`judicial-cases` の明示的な個別分離で同じ規則を確認し、一方の成功で他方を
代用しない。どちらも、各主題だけの topic-local draft を step 内正規化後の
score と通常 `tieBreak` で並べ、入力配列順または条件付き別名順を使わない。
step 内正規化も同様に、3.4.4 の
`core-multi-step-evidence-step-local-normalization` と 3.5 の
`judicial-multi-step-evidence-step-local-normalization` を別々に確認する。

`shared-terminal-production-neutral` は sidecar を構築しても active profile が
消費しないことを確認する移行時契約であり、`core-evidence-production-neutral` は
core の test 専用 evidence path 全体が active metadata と production runtime から
到達不能であることを確認する移行時契約とする。両者は同じ non-reachability を
重複検証するものではなく、どちらも第 5 段階で終了する。法令コアの後者は
`core-evidence-production-adopted` に置き換え、合格した candidate content と
current adoption tuple に一致する production 経路の到達性を確認する。

## 段階ごとの進行条件

各段階は、次を満たした場合だけ次段階へ進める。

1. 対象 SOT に直接ひも付く必要最小限の検証が成功している
2. 独立 reviewer の評価が 8.0 / 10 以上で、blocker がない
3. review 指摘を反映した後に、同じ段階境界を再確認している
4. 段階の変更単位が、前後の段階と混ざらない

第 3 段階の 1 から 6、`3.4.1` から `3.4.4`、第 4 段階の 1 から 5 および
第 6 段階の 1 から 4 は、それぞれ一つの独立した変更単位とする。各番号について
上記 1 から 4 を満たし、review 指摘を反映した後に一つの commit として確定する。
その commit に対する `SOT-ENG-020` と `SOT-ENG-027` の適用可能な権威 CI が
成功したことを確認してから、同じ段階の次の番号へ進む。CI が未完了、cancel
または失敗の状態では次の番号へ進まない。後続番号の成果物を先行番号へ混在させて、
先行番号の検証、review または CI を代用しない。

公開既定動作を切り替えない段階では、`SOT-ARCH-033` に従い、active artifact、
production composition root、標準 corpus、baseline および検索例カタログを現行のまま保つ。

必要最小限の検証内容は `SOT-ENG-027` を定義元とし、公開既定動作を切り替える
第 5 段階では `SOT-ENG-020` と `SOT-ENG-024` の中央品質ゲートを省略しない。

## 段階間の禁止事項

- 第 2 段階または第 3 段階で、未採用の next profile set を CLI、設定、環境変数、MCP または transport から選択可能にすること
- 第 4 段階で、次版 corpus や baseline 候補を標準 command や中央品質ゲートの現行参照先へ切り替えること
- 第 4 段階の初回 bootstrap で、標準 command の入口だけを切り替えながら、
  評価対象 tuple、report byte、検索例または production の外部呼出し境界も
  同時に変えること
- 第 4 段階で、holdout の内容または結果を `branchRetentionMargin`、重み、閾値、
  辞書、規則若しくは期待値の調整へ使用すること
- 第 5 段階で、`SOT-ARCH-033` の採用要素または `SOT-ENG-033` の tuple の
  一部だけを先行採用すること
- 第 6 段階の provider parser 変更を、第 5 段階の meaning 変更に便乗させること
- 第 7 段階へ、公開 notice、questions、response parity、code または評価成果物の
  変更を延期すること
- 第 7 段階より前に、将来の案内文や scenario を現行確認済みのように記載すること

## Wiki との関係

実装差分、進捗、review 点数、確認日および段階の現在地は Wiki で追跡できる。ただし、段階そのものの定義、順序および進行条件の定義元は本 SOT とする。Wiki が本規定と異なる場合は、本規定を優先する。

## 確認

少なくとも次を確認する。

- 段階 1 から 7 の定義が、`query_legal_information` の意味判定変更単位を過不足なく分離している
- 現行標準を変えない段階で、production composition root、standard corpus および baseline が不変である
- 初回 adoption bootstrap では、標準 command の入口を切り替えても評価対象 tuple、
  report byte および外部呼出し境界が不変である
- 原子的採用段階で、`SOT-ARCH-033` の全採用要素と `SOT-ENG-033` の
  current tuple の同期が要求される
- provider parser などの独立変更が、meaning 変更段階から切り離されている
- 第 3・第 4・第 6 段階の番号付き変更単位と `3.4.1` から `3.4.4` が、
  同じ段階内でも順に review される
- 各番号付き変更単位に固定検証 ID が対応し、その commit の権威 CI 成功を待って
  次の番号へ進む
- 第 4 段階 4 が候補の profile metadata、cue、辞書、composition、意味判定
  source set および二種の content-bound review attestation を固定している
- 第 4 段階 5 に `evaluation-baseline-candidate-isolation` が割り当てられ、
  candidate report が現行 baseline へ混入しない
- 第 5 段階で `core-evidence-production-neutral` が終了し、
  `core-evidence-production-adopted` に置き換わる
- 第 6 段階の parser と page 順序が専用 fixture で検証され、意味評価 baseline の
  責務と混同されない
- 第 7 段階が文書専用であり、公開 MCP 契約の実装・検証を第 5 段階から延期しない
- review 8.0 / 10 以上、blocker なし、および段階ごとの必要最小限検証が、次段階へ進む前提として明示されている

## 関連

- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](../30-architecture/25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](../30-architecture/31-unified-query-intent-evidence-layer.md)
- [SOT-ARCH-037: 統合照会の正規化済み限定分岐保持](../30-architecture/37-unified-query-normalized-branch-retention.md)
- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](../30-architecture/33-unified-query-profile-set-adoption-boundary.md)
- [SOT-ARCH-039: core query profile の ref 忠実な根拠対応と採用境界](../30-architecture/39-core-query-profile-evidence-mapping-v2.md)
- [SOT-ARCH-038: judicial-cases query profile の ref 忠実な根拠対応](../30-architecture/38-judicial-query-profile-evidence-mapping-v2.md)
- [SOT-ARCH-036: 複数 step 候補の step 内根拠正規化と保持](../30-architecture/36-multi-step-evidence-normalization.md)
- [SOT-MODEL-031: SharedTerminalSequence](../20-model/31-shared-terminal-sequence.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-025: 統合照会のパッケージ構成](25-unified-query-package-layout.md)
- [SOT-ENG-026: 統合照会の評価コーパス成果物契約](26-legal-query-corpus-artifact-contract.md)
- [SOT-ENG-027: 資源制約を踏まえた検証段階](27-resource-aware-verification-stages.md)
- [SOT-ENG-029: 統合照会の検索例カタログ](29-unified-query-example-catalog.md)
- [SOT-ENG-033: 統合照会 profile set 採用 manifest](33-unified-query-profile-set-adoption-manifest.md)
- [SOT-ENG-035: 統合照会 profile metadata 成果物契約](35-unified-query-profile-metadata-artifact-contract.md)
- [SOT-ENG-036: 統合照会の評価 baseline 成果物契約](36-unified-query-evaluation-baseline-artifact-contract.md)
- [SOT-ENG-038: 統合照会の内容固定済み候補 holdout 評価 handoff](38-content-bound-candidate-evaluation-handoff.md)
- [SOT-IF-011: e-Gov 法令本文取得マッピング](../40-interfaces/11-egov-law-document-mapping.md)
- [SOT-IF-052: e-Gov キーワード検索 JSON 応答の受理契約](../40-interfaces/52-egov-keyword-response-contract.md)
- [SOT-IF-053: MCP `search_laws` v3](../40-interfaces/53-mcp-search-laws-v3.md)
- [SOT-IF-054: e-Gov 法令名検索マッピング v3](../40-interfaces/54-egov-law-search-mapping-v3.md)
