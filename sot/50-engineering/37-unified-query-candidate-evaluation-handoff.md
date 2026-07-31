# SOT-ENG-037: 統合照会の候補 holdout 評価 handoff

- 状態: 有効

## 規定

次版の統合照会 profile set を固定 holdout で採用判定する処理は、repository 内の
一件の閉じた評価 request、CI 専用 command、および不変な result/report の
handoff に限定し、同じ候補と holdout の組を調整ループへ再利用しない。

## 定義元の分離

評価指標と受入値は `SOT-ENG-024`、corpus、holdout digest および
`leakageGroupId` は `SOT-ENG-026`、report byte の構造と metric 計算は
`SOT-ENG-036`、採用 tuple と baseline の採用後の対応は `SOT-ENG-033`、
実行順は `SOT-ENG-034` を定義元とする。

本規定は、採用前候補を一意に指定する request、権威ある CI 評価入口、
成功・失敗 report の保存先、論理的な一回利用、再現、privacy および
採用 manifest への接続だけを定義する。製品の query profile 選択、公開 MCP、
標準評価 command または provider の振る舞いを定義しない。

## 配置

追跡する成果物は次の固定配置とする。

```text
testdata/legalquery/candidate-evaluations/
├── schema-v1.json
├── current.json
├── requests/
│   └── {evaluationId}.json
├── results/
│   └── {evaluationId}.json
└── failed-reports/
    └── {evaluationId}.json
```

`schema-v1.json` は JSON Schema Draft 2020-12 の閉じた schema とし、同じ schema
内の fragment 以外の `$ref` を解決しない。root は pointer、request および
result の三 variant を `artifactKind` で閉じて区別する。

`requests/` と `results/` の file は一度追加した byte を変更、移動、削除または
再生成しない。`failed-reports/` の file も同様に不変とする。`current.json` は、
次に評価する一件の既存 request へ進めるときだけ置き換え、任意 path を持たない。

成功 report は `SOT-ENG-036` の
`baselines/versions/{baselineVersion}.json` にだけ保存し、
`candidate-evaluations/` へ同じ byte を複製しない。失敗 report は
`failed-reports/{evaluationId}.json` にだけ保存し、baseline version file、
`default.json` または adoption history から参照しない。

## pointer

`current.json` は次の項目だけをこの順で持つ。

| 項目 | 型 | 値 |
|---|---|---|
| `artifactKind` | string | `legal_query_candidate_evaluation_pointer` |
| `schemaVersion` | integer | `1` |
| `evaluationId` | string | 一件の request ID |

loader は正規形を検証した `evaluationId` から
`requests/{evaluationId}.json` だけを導出する。絶対 path、separator、`..`、
percent-encoding、environment、CLI override または fallback を解釈しない。

## 評価 request

request は次の項目だけをこの順で持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `artifactKind` | string | はい | `legal_query_candidate_evaluation_request` |
| `schemaVersion` | integer | はい | `1` |
| `evaluationId` | string | はい | 後述の候補 tuple digest |
| `evaluatorVersion` | string | はい | evaluator の固定意味版 |
| `corpusVersion` | string | はい | 評価する候補 corpus |
| `corpusManifestSha256` | string | はい | corpus manifest 原 byte の SHA-256 |
| `holdoutDigest` | string | はい | corpus manifest の holdout digest |
| `holdoutLeakageGroupDigests` | `string[]` | はい | corpus manifest が固定する holdout leakage group の digest 集合 |
| `profileSet` | object | はい | test が直接構成する候補 profile set |
| `baselineVersion` | string | はい | 成功時に使用する予約 baseline 版 |

`profileSet` は、`profileSetId`、`profileSetVersion`、`rankingVersion`、
`compositionVersion` および `profiles` だけをこの順で持つ。`profiles` の各要素は
`profileId`、`profileVersion`、`cueSetVersion` だけを、候補 composition root の
固定順で持ち、一件以上十六件以下とする。この tuple は候補 profile metadata、cue artifact、
composition root と一致させる。

`SOT-ENG-036` report との一致では、request から
`profileSetId`、`profileSetVersion`、`rankingVersion` および固定順の
`profiles[].{profileId, profileVersion}` だけをこの順で射影し、report の閉じた
`profileSet` と完全一致させる。report が持たない `compositionVersion` と
`cueSetVersion` は比較から黙って捨てるのではなく、request digest、候補
composition root、profile metadata、cue artifact および後続 adoption manifest
との完全一致で別に検証する。

`evaluatorVersion` は `legal-query-evaluator-v` と先頭が零でない十進整数を連結した
64 byte 以下の ASCII 値とする。metric の計算、集合順、acceptance 判定、
report 直列化または privacy 投影の意味を変える場合は新しい版にする。source
commit、時刻、host、任意 path または実行環境を identity の代用にしない。

本規定を最初に採用する evaluator の現行版は `legal-query-evaluator-v1` とする。
evaluator package は、利用者入力や設定から変更できない current version と、
再現に必要な実装済み過去版の閉じた registry を一か所だけ持つ。新しい request は
current version と完全一致しなければならず、候補 command は corpus または
holdout を読む前に不一致を拒否する。追跡済み result の replay では、request が
固定した exact version の実装だけを registry から選び、未知版、削除済み版、
近似版または current 版への fallback を拒否する。版を増やす変更では旧 result の
byte 再現に必要な旧実装を保持する。

`evaluationId` は `evaluation-sha256-` と小文字十六進六十四桁を連結する。
suffix は `evaluationId` を除く request field を上記の完全順で
`SOT-ENG-033` の canonical scalar、array および object 符号化へ投影した byte の
SHA-256 とする。request 原 byte、pointer および result は、同 SOT の
canonical JSON byte 規則に従う。

新しい candidate request は `SOT-ENG-026` の schema version 2 以上の corpus
だけを参照する。`corpusManifestSha256`、`holdoutDigest` および
`holdoutLeakageGroupDigests` を corpus manifest と完全一致させ、manifest の
一部だけを別 corpus へ差し替えない。digest 配列は同 SOT の形式、byte 順、
一意性および一件以上四百件以下を満たし、raw `leakageGroupId` を request へ
複製しない。request の `profileSet` は test 専用 constructor が直接構成し、
production の active profile set または標準 evaluator の current tuple から
推測しない。

`baselineVersion` は、bootstrap で request を持たずに導入する現行版を除き、
全 request、成功 baseline および失敗履歴を通じて一意に予約する。失敗した
request の予約版を後続候補へ再利用しない。

## 評価 result と report

result は次の項目だけをこの順で持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `artifactKind` | string | はい | `legal_query_candidate_evaluation_result` |
| `schemaVersion` | integer | はい | `1` |
| `evaluationId` | string | はい | 対応 request |
| `requestSha256` | string | はい | request 原 byte の SHA-256 |
| `outcome` | string | はい | `passed` または `failed` |
| `reportSha256` | string | はい | report 原 byteの SHA-256 |

checkout、build、loader、tool、schema または CI infrastructure の失敗では、
構造上有効な `SOT-ENG-036` report が完成していないため result を作らない。
`outcome` は、report の全 metric と `SOT-ENG-024` の受入基準から決定し、
手動で上書きしない。

`outcome=passed` の場合だけ、同じ report byte を request が予約した
`baselines/versions/{baselineVersion}.json` へ exclusive create する。既存 file、
別 byte、symlink または同名の失敗予約を上書きしない。

`outcome=failed` では baseline version file を作らず、同じ report byte を
`failed-reports/{evaluationId}.json` へ exclusive create する。失敗 report も
`SOT-ENG-036` の構造、上限、決定的順序および privacy 境界を満たす。

成功・失敗のどちらも、production composition root、標準 command、中央品質
ゲート、current adoption tuple、`baselines/default.json` または検索例カタログを
変更しない。`outcome=failed` の result が有効でも、第 5 段階へ進めない。

## CI 専用 command

候補 holdout 評価を起動できる repository verification command は、次の一つに
固定する。

```text
GOMAXPROCS=1 go run ./cmd/legal-query-candidate-eval --repository=. --output-directory=./.artifacts/legal-query-candidate-evaluation
```

command は上記二引数と値だけを受理する。`GOMAXPROCS=1` は省資源実行のための
固定値であり、評価意味を選ぶ設定ではない。profile、corpus、baseline、
`evaluationId`、holdout、evaluator または output file を上書きする引数、
ほかの環境変数、設定、build tag、hidden mode または fallback を設けない。
固定 `current.json` から request を一件読み、候補 composition root を test 専用
constructor で直接構成する。

この command は製品 binary、標準 evaluator、中央品質ゲート、MCP、transport
または利用者向け CLI へ登録しない。`SOT-ARCH-033` の準備状態で許可する
CI handoff 専用入口であり、利用者の query を受け取らず、候補 profile set を
production request の処理へ渡さない。

clean checkout で外部 network、時刻、乱数、provider response または実時間待機を
使わず、一つの evaluator run だけを行う。未追跡の固定 output root は、
`{evaluationId}/report.json` と `{evaluationId}/result.json` だけを持ち、
通常 file、symlink 不使用および repository 内の閉じた path を検証する。
command は追跡済み source file を直接変更しない。

まだ result がない request では、提案する report/result を output root へ出し、
CI workflow artifact として handoff する。追跡済み result がある場合は同じ評価を
再現し、result と、成功 baseline または失敗 report の原 byte を完全一致で
検証する。再現 byte が異なる場合は nondeterminism の blocker とし、どちらかを
選ばない。

構造上有効な report と result を決定的に生成または再現できた場合は、
`outcome=passed` と `outcome=failed` のどちらでも command を終了 code `0` とする。
ここでの成功は評価処理と handoff byte が有効であることだけを表し、候補の採用適格を
表さない。schema、identity、version、入力、metric 計算、privacy、再現または
infrastructure の失敗では非零終了し、有効な result を捏造しない。

CI job は command が終了 code `0` で、閉じた output root に report と result の
二 file だけがある場合に限り、その二 file を一つの workflow artifact として
upload する。`outcome=failed` の artifact と job 自体は handoff のため成功できるが、
第 5 段階の gate は result の `outcome=passed` を別に要求する。tracked failed result
の replay も byte が一致すれば job を成功させ、採用適格へ読み替えない。非零終了、
欠落 file、未知 file または部分 output は handoff しない。

## handoff と一回利用

第 4 段階 4 では、review 済みの candidate evaluator、request、pointer および
予約 baseline version を準備するが、holdout を実行しない。

第 4 段階 5 の開始時に、同じ準備 commit の専用 CI job
`candidate-evaluation` を一回だけ手動起動する。workflow artifact から得た
result と report を、profile、corpus、evaluator、受入基準または request を
変更しない handoff commit へ追加する。成功 report は予約 baseline path、
失敗 report は failed-report path へ、その byte のまま置く。

この job は手動 dispatch だけを起動条件とし、pull request、push、schedule、
release、fork またはほかの workflow から自動起動しない。review 済み準備 commit の
不変な Git object ID を checkout し、job 開始時の対象 object ID、checkout の
`HEAD`、request を追加した commit および workflow definition を同じ値として
検証する。branch、tag または moving ref を評価 identity にしない。checkout 後の
worktree が dirty、submodule が未知、または対象 commit が review 済みでない場合は
command を起動しない。job の repository permission は `contents: read` に限定し、
credential、release permission、write token または environment secret を渡さない。

構造上有効な `SOT-ENG-036` report byte が最初に完成した時点で、その
`holdoutDigest` を採用判定に一回使用したとみなす。passed と failed の両方が
消費に当たる。同じ `evaluationId`、不変 request および同じ report byte の CI
retry、artifact upload の再試行または後続 replay は再現確認であり、新しい利用と
数えない。有効な report ができる前の infrastructure 失敗だけは同じ ID で
再試行できる。

一つの `holdoutDigest` に、構造上有効な result を持つ異なる
`evaluationId` を二件以上作らない。result 完成後に profile、evaluator、corpus、
辞書、期待値または受入規則を変えた次候補は、新しい corpus version、新しい
holdout digest、新しい evaluation ID および新しい baseline reservation を
必要とする。

pointer が指す current request について、command と loader は corpus、holdout
fixture または evaluator を読む前に、同じ `holdoutDigest` を持つ別
`evaluationId` の過去 result が一件でも存在しないことを確認する。存在する場合は
pending request、未追跡 output または artifact upload の有無にかかわらず即時に
拒否し、同じ digest を再実行しない。同じ `evaluationId` と不変 request の replay
だけを前述の再現確認として許可する。

同じ preflight で、passed と failed を含む全 result が参照する過去 request の
`holdoutLeakageGroupDigests` を読み、current request の同配列との積集合が
空であることを検証する。過去の corpus manifest、fixture、成功 baseline または
failed report は、この累積検査の入力として再度開かない。過去 request が保持する
digest 配列は、作成時に corpus manifest と fixture から検証済みの不変な
compact index として使う。

preflight は後述する request と result の各 `4096` 件、各 `32 MiB` の全履歴上限内で
一回だけ走査し、上限超過、孤立 result、欠落 request、digest 配列の不正、
同じ group digest または同じ holdout digest を一件でも検出した時点で、
current corpus、holdout fixture または evaluator を開かず拒否する。
preflight 後に current corpus を読み、その manifest と fixture から再計算した
`holdoutLeakageGroupDigests` が current request と完全一致することも検証する。
過去 request または result を削除、変更若しくは無視してこの検査を回避しない。

handoff commit の同じ command による byte 再現、独立 review および
`SOT-ENG-020` の権威 CI が成功してから、第 4 段階 5 を完了とする。

## 採用と rollback への接続

bootstrap の初回 baseline を除き、新しい `SOT-ENG-033` adoption manifest の
`baselineVersion` と `baselineSha256` は、正確に一件の `outcome=passed`
result、その request の予約 `baselineVersion` および `reportSha256` と一致させる。
対応する passed result がない baseline、failed result の予約版、または別 request の
report を採用しない。

同じ adoption manifest の `evaluatorVersion` は、その passed result が参照する
request の exact `evaluatorVersion` と一致させる。標準 command と production の
採用検査は current manifest が固定した版だけを閉じた registry から選び、
current evaluator、未知版または近似版へ fallback しない。

rollback 先の adoption manifest も、採用当時に対応した passed result と
version file および exact evaluator version の接続を維持する。rollback のために
result、request、report digest、evaluator version または予約 baseline version を
書き換えない。

## loader と privacy

loader は repository root 内の固定 subtree だけを OS の root API から開き、
全 path component と open 後の file descriptor で symlink と非 regular file を
拒否する。root は `schema-v1.json`、`current.json`、`requests/`、`results/` および
`failed-reports/` だけを持ち、各 directory は正規形の
`{evaluationId}.json` 通常 file だけを持つ。subdirectory、未知 entry、未登録 file、
device、FIFO および socket を拒否する。

資源上限は次とする。

| 対象 | 上限 |
|---|---:|
| schema | `1 MiB` |
| pointer | `64 KiB` |
| request 一件 | `256 KiB` |
| result 一件 | `256 KiB` |
| report 一件 | `4 MiB` |
| request 件数・原 byte 合計 | `4096`・`32 MiB` |
| result 件数・原 byte 合計 | `4096`・`32 MiB` |
| failed report 件数・原 byte 合計 | `4096`・`256 MiB` |
| JSON depth | `12` |
| pointer・request・result 一 document の value 数 | `8192` |
| report 一件の value 数 | `SOT-ENG-036` の `65536` |

各 file は open 前後に size と通常 file であることを確認し、宣言上限に一 byte を
加えた bounded reader で一回だけ読む。directory 列挙中も件数と合計 byte の上限を
超えた時点で失敗し、呼出し元 context の cancellation を各 file 間で確認する。

typed decode より前に不正 UTF-8、BOM、重複 key、`null`、未知 field、二個目の
JSON 値、後方 token、上限超過および canonical byte 不一致を拒否する。
一つの不正 artifact を無視して部分 history を返さない。

各 result は同じ ID の一件の request を参照し、`requestSha256` を一致させる。
一 request に result を二件作らない。`outcome=passed` では予約した baseline file が
`reportSha256` と一致し、同じ ID の failed report が存在しないことを確認する。
`outcome=failed` では同じ ID の failed report が `reportSha256` と一致し、予約した
baseline file が存在しないことを確認する。result がない request には、追跡済みの
baseline file または failed report が存在してはならない。孤立した result、report
または同名予約の衝突を無視しない。

request、result、report、output および log は、query、fixture 本文、辞書 entry、
外部 response、credential、絶対 path、時刻、host、user、commit message または
個人情報を持たない。CI log は `evaluationId`、request/report digest および
`outcome` だけを出せる。

## 確認

外部 network を使わない schema、loader、command plan、evaluator および adoption
契約 test で、少なくとも次の固定 test ID を確認する。

- `candidate-evaluation-closed-artifacts`
- `candidate-evaluation-request-identity`
- `candidate-evaluation-evaluator-version-match`
- `candidate-evaluation-current-single-target`
- `candidate-evaluation-production-unreachable`
- `candidate-evaluation-ci-authority`
- `candidate-evaluation-consumed-digest-preflight`
- `candidate-evaluation-deterministic-replay`
- `candidate-evaluation-outcome-exit-semantics`
- `candidate-evaluation-success-handoff`
- `candidate-evaluation-failure-history`
- `candidate-evaluation-single-holdout-use`
- `candidate-evaluation-leakage-exclusion`
- `candidate-evaluation-leakage-index-bounds`
- `candidate-evaluation-output-privacy`
- `candidate-evaluation-adoption-link`
- `candidate-evaluation-immutable-version`

`candidate-evaluation-leakage-index-bounds` は、current request の
`holdoutLeakageGroupDigests` が四百件の正常最大を受理して四百一件を拒否すること、
request と result の各 history が `4096` 件かつ `32 MiB` の正常最大を受理し、
`4097` 件または合計上限に一 byte を加えた状態を corpus と evaluator の読取り前に
拒否することを確認する。同じ検証で、過去 corpus fixture を開かず、passed と
failed の両方を走査し、一件の digest 衝突でも fail-closed となることを確認する。

## 関連

- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](../30-architecture/33-unified-query-profile-set-adoption-boundary.md)
- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-026: 統合照会の評価コーパス成果物契約](26-legal-query-corpus-artifact-contract.md)
- [SOT-ENG-027: 省資源の段階的検証](27-resource-aware-verification-stages.md)
- [SOT-ENG-033: 統合照会 profile set 採用 manifest](33-unified-query-profile-set-adoption-manifest.md)
- [SOT-ENG-034: 統合照会の意味判定変更における導入段階と変更順序](34-unified-query-rollout-stages.md)
- [SOT-ENG-036: 統合照会の評価 baseline 成果物契約](36-unified-query-evaluation-baseline-artifact-contract.md)
