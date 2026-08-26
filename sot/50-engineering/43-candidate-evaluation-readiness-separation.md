# SOT-ENG-043: 候補評価準備の鮮度と製品品質ゲートの分離

- 状態: 有効

## 規定

未消費の current 候補評価が現在の意味 source、SOT または evaluator から
乖離した場合は、固定済み成果物の完全性と候補評価を実行できる鮮度を別々に
検証し、成果物が完全なまま鮮度だけを失った候補を `stale` として隔離する。

## 適用範囲

本規定は、`SOT-ENG-038` および `SOT-ENG-042` に従って準備済みで、まだ result を
持たない current request と現在の repository 内容が乖離した場合に適用する。
schema version 2 と 3 の成果物形式、canonical identity、固定 SOT 集合、予約履歴、
historical replay および一回利用規則は変更しない。

artifact の種類判別、schema version ごとの schema 選択、mixed-generation root entry
集合、cross-version binding および historical replay の入口は、引き続き
`SOT-ENG-042` だけを定義元とする。本規定は、その integrity 検査を通過した
未評価 current request に対する readiness 分類と拒否契約だけを定義する。

current request に対応する result が既に存在する場合は、`stale` 判定の対象にしない。
その状態は不変な request/result/report の replay としてだけ扱い、現在の source、
current SOT または current evaluator へ再結合しない。

製品の `query_legal_information`、profile 選択、標準 evaluator、採用済み baseline、
公開 MCP、provider、corpus schema、評価指標または受入値は変更しない。

## 限定的な後継範囲

本規定は、repository 完全性を通過し result を持たない current について、後述する
閉じた鮮度理由を一件以上検出した場合に限り、次の既存規定を限定的に引き継ぐ。

- 同 SOT の「評価 request」と「確認」にある、result のない current について現在の
  SOT 状態と原 byte の一致を製品変更の完了条件にもする規定
- 同 SOT の「一回利用」にある、source、SOT または review drift の発生と同時に
  新 request へ pointer を進める規定
- `SOT-ENG-039` の第 4 段階にある、report 完成前の drift と同時に新しい review、
  request および pointer を再準備する規定

`current.json` と request の artifact としての意味は、引き続き「次の評価 identity」を
示す pending request であり、field、型、canonical identity または schema の意味を
変更しない。`stale` は同じ artifact byte から現在の snapshot に対して導出する
非実行可能性の overlay であり、artifact に保存しない。これは `SOT-ENG-039` 第 4 段階 4
の完了、同段階 5 への進行、評価済み、holdout 消費または新しい候補準備を意味しない。
候補評価を再開する前に、`SOT-ENG-038`、`SOT-ENG-039` および本規定に従う新世代
request へ pointer を進める義務は残る。

この限定的な overlay は、pointer が repository 完全性と readiness を通過する別 request へ
原子的に進むか、同じ current に result が結び付いて historical replay へ移る時点で終了する。
schema version 2 と 3 の既存 request、review、manifest、result、report、exact SOT 配列、
予約値および historical replay の意味には適用せず、同じ成果物を補正して再び `ready` に
する根拠にも使用しない。製品変更が統合照会の意味、公開 interface、corpus、baseline、
evaluator または別の `SOT-ENG-020` 適用対象を変更する場合は、その変更固有のゲートを
常に実行し、`stale` overlay を免除理由にしない。

## 二つの検証境界

検証は一つの clean checkout と exact commit に固定した root-scoped snapshot を所有する
次の三段階で行う。各段階は前段の不変な出力だけを受け取り、repository root、commit、
file identity または検査中の byte が変化した場合は再試行や fallback を行わず error にする。

1. `artifact integrity inspection` は候補評価 root の全成果物を読み、内部 binding を
   検証した閉じた evaluation graph を返す
2. `current readiness assessment` は同じ snapshot と graph だけを使い、現在の source、
   SOT、evaluator および corpus manifest との一致を `ready | stale | error` へ写像する
3. `evaluation load` は前二段階が成功し readiness が `ready` の場合だけ、同じ graph から
   evaluator に渡す payload を作る

中央品質ゲートは一と二を所有する。候補評価 command は一から三を同じ順で所有し、二を
単独で省略できる内部入口、CLI option、environment、test hook または fallback を持たない。
dispatch job は checkout、固定 toolchain および private cache の準備までは行えるが、二が
`ready` を返す前に evaluator、holdout reader、report writer または result writer を開始しない。

一と二は外部 network を使用しない。二の source closure 再構成は request が固定する
toolchain と検証済み local module cache を `GOPROXY=off`、`GOWORK=off` および固定 build
context で使用できる。corpus manifest は読めるが、holdout fixture、development case、
query 原文または provider を開かない。

local module cache は `SOT-ENG-038` の build context isolation と module archive 検証を
通過した process-local の `verified module-cache descriptor` として snapshot に固定する。
readiness assessment は descriptor が列挙する path、module identity、zip checksum、原 byte
digest および byte 上限だけを使用し、検査中の path 又は byte の変化を error にする。
descriptor 自体を request、manifest、report または追跡成果物へ追加しない。

### repository 完全性検査

候補評価 root loader は、`SOT-ENG-020` の中央品質ゲートと worker の共通入口として、
候補評価 root に対して少なくとも次を fail-closed に検証する。

- schema version 2 と 3 の固定 schema byte および閉じた root entry 集合
- pointer、manifest、review attestation、request、result および report の
  canonical byte、file 名と内部 ID、および schema version
- request から manifest と二件の review への完全な binding
- evaluator の閉じた historical registry、予約値の一意性、一回利用履歴、
  result と report、および採用済み baseline の binding
- current 以外を含む全追跡成果物の件数、byte、通常 file、symlink および
  orphan 制約

この検査は、現在の HEAD から再構成した candidate content、現在有効な SOT 原 byte、
または新規 request 用 current evaluator との一致を、固定済み成果物の完全性と
同一視しない。完全性を通過した後の鮮度乖離だけを、後述する readiness 検査で
`stale` として扱う。

### candidate readiness 検査

候補 subtree 外の固定参照を再構成する readiness validator は、repository 完全性検査に
成功した current request についてだけ、holdout fixture を開く前に次を現在の
repository から再構成する。

- candidate content manifest と semantic source closure
- schema version ごとの exact `requiredReviewSOTs` に対応する index、状態および原 byte
- 新規 request 用 current evaluator
- corpus manifest、holdout digest、leakage group digest および予約 baseline の制約

全項目が一致する場合だけ readiness を `ready` とし、候補 payload を worker へ渡せる。
閉じた鮮度理由だけが見つかった場合は `stale` とし、strict loader、候補評価 command、
dispatch workflow および worker は同じ current を非ゼロ終了で拒否する。
current projection の比較単位は、validator が現在の repository から再構成した
canonical JSON byte またはその digest と、固定済み manifest・request が保持する
canonical byte または digest の exact 一致とする。

`stale` では、holdout fixture、development case および query 原文を開かず、評価、
report、result、failed report、baseline version、adoption history または別の
`current.json` を生成若しくは更新しない。dispatch job は readiness 検査のための
control-plane command までは起動できるが、`ready` でない commit の evaluator、
holdout reader または出力 writer を実行しない。

## readiness 状態と理由

readiness 状態は次の二値とする。

| 状態 | 意味 |
|---|---|
| `ready` | 完全性と現在の外部参照がともに一致し、候補評価を開始できる |
| `stale` | 完全性は保たれているが、現在の外部参照との鮮度が失われ、候補評価を開始できない |

`stale` の理由は次の閉じた値だけとする。

| 理由 | 条件 |
|---|---|
| `candidate_content_drift` | 現在の source closure を正常に再構成できるが、固定済み candidate content と一致しない |
| `review_sot_lifecycle_drift` | exact SOT ID が正しい index と file に存在するが、現在は `有効` ではない |
| `review_sot_digest_drift` | exact SOT ID が現在も `有効` だが、固定済み原 byte digest と一致しない |
| `current_evaluator_drift` | request の evaluator は historical registry に存在するが、新規 request 用 current evaluator ではない |

複数の独立した鮮度乖離を検出した場合は、上表の順序で重複なく保持する。error 文言の
解析で理由を推測せず、型付きの閉じた値として伝達する。

次は `stale` ではなく repository 完全性または readiness 検査の error とし、中央品質
ゲートも失敗させる。

- schema、canonical byte、file、ID、版または内部 binding の不一致
- artifact の欠落、削除、変更、孤立、未知 entry、symlink または上限超過
- 未対応 evaluator、SOT index 又は file の欠落、SOT heading 又は ID の不一致
- current request と corpus manifest、holdout digest、leakage group digest 又は
  baseline 予約制約の不一致
- source closure、SOT index、corpus manifest または検証環境を安全に再構成できない状態
- 取消し、I/O error、未知の鮮度理由または検証器の内部 error

一つの検査で `stale` と error の両方を検出した場合は error を優先する。これにより
鮮度乖離が artifact 改変または corpus 改変を隠さない。

error が返る場合は readiness 状態値を同時に返さない。`ready` と `stale` は integrity
成功後の current readiness を表す二値であり、error は別の失敗結果として扱う。

## 中央品質ゲートとの接続

中央品質ゲートは、実 repository の候補評価 root に対して repository 完全性検査を
必ず成功させる。current readiness が `ready` なら strict loader が同じ current を
受理することを確認し、`stale` なら理由が閉じた値であることと strict loader、worker、
dispatch gate および出力生成が同じ current を拒否することを確認する。

したがって `stale` 自体は製品変更の中央品質ゲートを失敗させない。ただし候補評価
workflow の readiness gate は失敗したままとし、`stale` を warning、成功した dispatch、
空の評価結果または評価済み扱いへ変換しない。実 repository の current が `stale`
であっても、完全性検査と `stale` 拒否契約が通る限り、製品品質ゲート側では fail-open
にも fail-skip にもしない。

`SOT-ENG-020` が禁じる「失敗した検査の警告化」は、本 readiness 検査にも同じく適用する。
候補評価 root に関する中央品質ゲート内の readiness 検査は、`ready` の受理又は
`stale` の閉じた理由と拒否契約の成立を成功条件とし、それ以外の error、未知状態、
理由集合の逸脱又は reject 漏れを失敗とする。

実 repository に対する中央品質ゲートの証跡は、少なくとも current evaluation ID、
integrity 成功、current readiness の判定結果、固定順の stale 理由列、strict loader の
受理又は拒否結果、および worker・dispatch・出力生成の非到達確認へ到達できることを
要求する。free-form な文言だけに依存して判定しない。

これは `SOT-ENG-020` の失敗を warning へ緩和する例外ではない。製品側の適用可能な
ゲートは artifact integrity inspection、readiness の閉じた分類、stale 時の非到達性、
採用済み production tuple および標準 evaluator の検証であり、すべて成功しなければならない。
候補評価側の適用可能なゲートは readiness が `ready` であることまでを含むため、stale では
常に失敗する。

既存の確認項目は次の所有境界で解釈し、同じ失敗を無音で成功へ変換しない。

| 確認対象 | repository 完全性 | current readiness | stale 時の製品 gate | stale 時の候補 gate |
|---|---:|---:|---:|---:|
| schema、canonical byte、内部 ID、request・review・manifest binding | 必須 | 対象外 | 失敗 | 失敗 |
| `candidate-evaluation-current-single-target` の pointer と request の内部 binding | 必須 | 対象外 | 失敗 | 失敗 |
| `candidate-evaluation-candidate-content-identity` の固定 manifest 内部 identity | 必須 | 対象外 | 失敗 | 失敗 |
| 固定 manifest と現在の source closure の一致 | 前段通過後 | 必須 | stale を検証して成功 | 失敗 |
| `candidate-evaluation-review-content-binding` の request と二 review の historical binding | 必須 | 対象外 | 失敗 | 失敗 |
| result のない current と現在の SOT 状態・digest の一致 | 前段通過後 | 必須 | stale を検証して成功 | 失敗 |
| current evaluator の鮮度 | historical 対応版の存在を検証 | 必須 | stale を検証して成功 | 失敗 |
| corpus manifest と予約値 | 必須 | 必須 | 不一致は失敗 | 不一致は失敗 |

表の「stale を検証して成功」は不一致自体の成功ではなく、完全性通過後に閉じた理由で
隔離され、strict evaluation load へ到達しないことの成功を意味する。

tracked result を持つ historical replay は現在の source 又は SOT 鮮度を再検証せず、
`SOT-ENG-038` と `SOT-ENG-042` の固定 request、result、report、schema および exact
evaluator だけで再現する。

## 不変成果物と後続世代

`stale` の解消を理由として、次を行わない。

- schema version 2 又は 3 の exact SOT ID 配列を変更する
- `SOT-IF-040` を同じ配列内で `SOT-IF-067` へ置き換える
- 既存 manifest、review、request、result、report または schema を変更、移動、削除する
- current request の corpus、holdout、leakage group digest または baseline version を再利用する
- stale request の digest を現在の source 又は SOT から再計算して上書きする
- schema version を増やさずに `current.json` だけを別 identity へ進める

候補評価を再開する場合は、本規定と `SOT-ENG-042` を参照する別の有効な SOT で
新しい schema version、exact SOT 集合および root entry 集合を採用する。その後、
過去の全 request と衝突しない新しい corpus、holdout、leakage group digest、baseline
version、candidate content、独立 review 二件、request および pointer を一つの準備単位で
追加する。現在予約済みの `corpus-v16` と `default-8` は再利用しない。

## 確認

外部 network と holdout fixture を使わない contract test で、少なくとも次を確認する。

- `candidate-evaluation-stale-current-repository-integrity`
- `candidate-evaluation-stale-reason-closed-set`
- `candidate-evaluation-stale-strict-loader-rejection`
- `candidate-evaluation-stale-worker-unreachable`
- `candidate-evaluation-stale-holdout-unreachable`
- `candidate-evaluation-stale-output-unreachable`
- `candidate-evaluation-stale-does-not-mask-artifact-corruption`
- `candidate-evaluation-ready-current-strict-loader`
- `candidate-evaluation-historical-replay-freshness-isolation`
- `candidate-evaluation-v2-v3-artifact-immutability`

実 repository の検査は `ready` と `stale` のどちらでも完全性を省略しない。合成 fixture で
四つの理由、未知理由、複数理由の固定順、stale と改変の同時発生、strict dispatch の拒否、
fresh current の受理および tracked replay の現在鮮度非依存を確認する。

合成 fixture では、閉じた stale 理由とは別に、少なくとも次を current drift と同時又は
単独で発生させ、error が優先し、holdout・worker・dispatch・出力生成が非到達であることを
確認する。

- 取消し
- I/O error
- validator の内部 error
- source closure の再構成失敗
- SOT index 又は file の欠落、SOT ID 又は heading の不一致
- 未対応 evaluator
- corpus manifest、holdout digest、leakage group digest 又は baseline 予約の不一致

## 関連

- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-027: 省資源の段階的検証](27-resource-aware-verification-stages.md)
- [SOT-ENG-038: 統合照会の内容固定済み候補 holdout 評価 handoff](38-content-bound-candidate-evaluation-handoff.md)
- [SOT-ENG-039: 内容固定済み候補による統合照会の導入段階と変更順序](39-content-bound-unified-query-rollout-stages.md)
- [SOT-ENG-042: 候補評価 handoff schema version 3 の世代分離](42-candidate-evaluation-handoff-schema-v3.md)
