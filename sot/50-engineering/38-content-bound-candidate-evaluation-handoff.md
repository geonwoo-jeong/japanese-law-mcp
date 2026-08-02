# SOT-ENG-038: 統合照会の内容固定済み候補 holdout 評価 handoff

- 状態: 有効

## 規定

次版の統合照会 profile set を固定 holdout で採用判定する処理は、repository 内の
一件の閉じた評価 request、CI 専用 command、および不変な result/report の
handoff に限定し、同じ候補と holdout の組を調整ループへ再利用しない。

## 定義元の分離

評価指標と受入値は `SOT-ENG-024`、corpus、holdout digest および
`leakageGroupId` は `SOT-ENG-026`、report byte の構造と metric 計算は
`SOT-ENG-036`、採用 tuple と baseline の採用後の対応は `SOT-ENG-033`、
実行順は `SOT-ENG-039` を定義元とする。

本規定は、採用前候補を一意に指定する request、権威ある CI 評価入口、
成功・失敗 report の保存先、論理的な一回利用、再現、privacy および
採用 manifest への接続だけを定義する。製品の query profile 選択、公開 MCP、
標準評価 command または provider の振る舞いを定義しない。

## 配置

追跡する成果物は次の固定配置とする。

```text
testdata/legalquery/candidate-evaluations/
├── schema-v2.json
├── current.json
├── content-manifests/
│   └── {candidateContentId}.json
├── review-attestations/
│   └── {attestationId}.json
├── requests/
│   └── {evaluationId}.json
├── results/
│   └── {evaluationId}.json
└── failed-reports/
    └── {evaluationId}.json
```

`schema-v2.json` は JSON Schema Draft 2020-12 の閉じた schema とし、同じ schema
内の fragment 以外の `$ref` を解決しない。root は pointer、request および
result に加えて candidate content manifest と review attestation の五 variant を
`artifactKind` で閉じて区別する。新しい pointer、request および result は
schema version 2 だけを使用する。

`schema-v2.json`、`content-manifests/`、`review-attestations/`、`requests/`
および `results/` の file は一度追加した byte を変更、移動、削除または
再生成しない。schema version 2 の field、型、意味または canonical projection を
変える場合は、本規定を置き換える新しい SOT ID と新しい schema version を採用し、
既存 schema と成果物を保持する。
`failed-reports/` の file も同様に不変とする。`current.json` は、次に評価する
一件の既存 request へ進めるときだけ置き換え、任意 path を持たない。

構造上有効な report が完成する前の失敗により、候補 source、対象 SOT または
review を変更しなければ同じ request を再試行できない場合は、旧成果物を削除せず、
新しい content manifest、二件の review attestation、request および未使用の
`baselineVersion` を追加して `current.json` だけを新 request へ進める。pointer から
外れ、result を持たない旧 request は「置換済み準備」として保持する。置換済み準備は
report と result を生成せず、holdout の消費履歴には数えないが、その
`baselineVersion` は後続 request で再利用しない。

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
| `schemaVersion` | integer | `2` |
| `evaluationId` | string | 一件の request ID |

loader は正規形を検証した `evaluationId` から
`requests/{evaluationId}.json` だけを導出する。絶対 path、separator、`..`、
percent-encoding、environment、CLI override または fallback を解釈しない。

## candidate content manifest

candidate content manifest は、holdout を読む前に評価対象の意味判定内容を固定する。
次の項目だけをこの順で持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `artifactKind` | string | はい | `legal_query_candidate_content` |
| `schemaVersion` | integer | はい | `2` |
| `candidateContentId` | string | はい | 後述する内容 digest |
| `profileSet` | object | はい | 候補 profile set の identity |
| `profileArtifacts` | object[] | はい | 固定順の profile metadata と cue |
| `lexiconArtifacts` | object[] | はい | metadata が参照する辞書成果物 |
| `composition` | object | はい | 候補 composition descriptor |
| `semanticSourceSet` | object | はい | 候補の意味判定 source closure |

`profileSet` は `profileSetId`、`profileSetVersion` および `rankingVersion` だけを
この順で持つ。

`profileArtifacts` は候補 composition root の固定順で一件以上十六件以下とし、
各要素は次の項目だけをこの順で持つ。

| 項目 | 意味 |
|---|---|
| `profileId` | profile の固定 ID |
| `profileVersion` | metadata が宣言する意味版 |
| `metadataSchemaVersion` | metadata の schema version |
| `metadataCanonicalSha256` | version 別 typed metadata の canonical byte digest |
| `cueSetVersion` | cue artifact の版 |
| `cueArtifactSha256` | cue artifact 原 byte の digest |

`metadataCanonicalSha256` の入力は `SOT-ENG-035` の version 別 field 順、
型および省略可能 field の存在状態を保持した canonical projection とする。
省略した field と、存在する零、false または空配列を同一視しない。
`cueArtifactSha256` は loader が検証した一件の cue artifact 原 byte に対する
SHA-256 とする。

`lexiconArtifacts` は candidate の全 profile metadata が参照する辞書を
`lexiconId` の byte 順に一件以上十六件以下で一件ずつ持ち、各要素は `lexiconId`、
`lexiconVersion`、`files` および `aggregateSha256` だけをこの順で持つ。
`files` は一件以上千二十四件以下とし、各要素は repository-relative POSIX
`path` と原 byte の `rawSha256` だけを持つ。path の byte 昇順とし、重複、
絶対 path、separator の正規化が必要な path、`..` および symlink を拒否する。
`aggregateSha256` は各 file の `path`、ASCII space、`rawSha256`、LF を順に
連結した byte の SHA-256 とする。metadata が参照しない辞書を加えず、参照した
辞書を省略しない。

schema version 2 の `lexiconId` と file は、metadata の `lexicons` field から
loader が選ぶ次の閉じた対応だけとする。manifest の path から辞書種別または
root を推測しない。

| metadata field | `lexiconId` | loader 所有 root | 許可 file |
|---|---|---|---|
| `lawNames` | `lawNames` | `internal/lawnamelexicon/data/` | `egov-current.json`、`supplemental.json` |
| `legalConcepts` | `legalConcepts` | `internal/legalconceptlexicon/data/` | `current.json` |

各 `files` は対応する許可 file を過不足なく一件ずつ持つ。
`testdata/legalquery/` の corpus、baseline、candidate evaluation、adoption、
`docs/`、`wiki/`、別 package の `data/` または任意の追加 path を辞書として
開かない。許可 root または file set を変える場合は、新しい schema version と
本規定の後継 SOT を先に採用する。

`composition` は `descriptorSchemaVersion`、`profileSetId`、
`profileSetVersion`、`rankingVersion`、`compositionVersion`、`components`
および `descriptorSha256` だけをこの順で持つ。`profileSetId`、
`profileSetVersion` および `rankingVersion` は `profileSet` と完全一致させる。
`components` の各要素は `role`、`componentId`、`semanticVersion` および
repository-relative POSIX の `packageRoot` だけをこの順で持つ。順序は
`preprocessor`、`profileArtifacts` と同じ順の各 profile、`composer`、
`selector` とする。同じ package が複数の role を所有する場合も、role ごとの
要素を省略せず、source closure では同じ file を一件へ縮約する。

schema version 2 の role、component および `packageRoot` は次の閉じた対応とする。

| `role` | `componentId` | `packageRoot` |
|---|---|---|
| `preprocessor` | `query-preprocessor` | `internal/querypreprocess` |
| `profile` | `core` | `internal/queryprofile/core` |
| `profile` | `judicial-cases` | `internal/queryprofile/judicialcases` |
| `composer` | `candidate-composer` | `internal/application/legalquery` |
| `selector` | `legal-query-selector` | `internal/application/legalquery` |

`profile` 要素は `profileArtifacts[].profileId` と同じ固定順・同じ集合とする。
任意 root、test 専用 package、production wiring または別 `componentId` への
fallback を許さない。component を追加または移動する場合は、同じ mapping を
更新する後継 SOT と新しい schema version を先に採用する。
`descriptorSha256` は自身を除く composition field を後述の canonical encoding
へ投影した byte の SHA-256 とする。

`semanticSourceSet` は `mainModulePath`、`goLanguageVersion`、
`goToolchainVersion`、`goDebugSettings`、`goos`、`goarch`、`goamd64`、
`goexperiment`、`cgoEnabled`、`buildTags`、`files`、`moduleDependencies`
および `sourceSetSha256` だけをこの順で持つ。
build context は候補 CI command の固定値と完全一致させ、`buildTags` は空配列、
`goamd64` は `v1`、`goexperiment` は空 string、`cgoEnabled` は integer の
`0` とする。bool は candidate content の canonical 入力に許可しない。
`goDebugSettings` は固定 toolchain が root `go.mod` の有効な `godebug`
directive から解決した明示設定を、`name` と `value` だけを持つ object として
`name` の byte 順に一件ずつ保持する。directive がなければ空配列とし、重複 name、
未知構文または toolchain が受理しない値を拒否する。toolchain と
`goLanguageVersion` が決める暗黙 default は両 field で固定し、同じ値を配列へ
複製しない。process の `GODEBUG` 環境変数は allowlist へ入れない。
`files` は composition descriptor が列挙した
component package root の local module 内 transitive dependency closure から、
その context で `go list -deps -json` の `GoFiles`、`CgoFiles`、`CFiles`、
`CXXFiles`、`MFiles`、`HFiles`、`FFiles`、`SFiles`、`SwigFiles`、
`SwigCXXFiles`、`SysoFiles` および `EmbedFiles` に選択される全 file を収集する。
生成済み source も選択される field に従って含め、`TestGoFiles` と
`XTestGoFiles` は含めない。各要素は
repository-relative POSIX `path` と原 byte の `rawSha256` だけを持ち、path byte
順に一意に並べる。

`moduleDependencies` は同じ package closure が実際に import する外部 module
だけを `modulePath` と `version` の byte 順に一件ずつ持つ。各要素は
`modulePath`、`version`、`moduleZipSum`、`moduleZipRawSha256`、
`moduleZipByteLength`、`moduleZipEntryCount`、`moduleExpandedByteLength`、
`moduleGoModSum` および `moduleGoModRawSha256` だけをこの順で持つ。
固定 toolchain と build context の `go list -deps -mod=readonly` が返す module
identity、module cache で検証した `h1:` checksum、取得した zip と `go.mod` の
原 byte digest、および一回の検証済み展開で数えた byte と entry 数に完全一致させる。
標準 library は `goToolchainVersion` で固定し、配列へ重複して列挙しない。
root `go.mod` または `go.sum` の全 byte は identity に含めず、候補 component
closure に実際に寄与する module だけを選ぶ。これにより、未使用 dependency の
追加または checksum 行だけでは candidate content を変えない。

`sourceSetSha256` は `sourceSetSha256` 自身を除く `semanticSourceSet` の
全 field を前述の順で `SOT-ENG-033` の canonical encoding へ投影した byte の
SHA-256 とする。
`mainModulePath` と `goLanguageVersion` は root module の有効値を保持し、
workspace の影響は `GOWORK=off` で拒否する。

source closure は各 `packageRoot` を package として解決して始める。
存在しない root、package を含まない root、root 自身または途中の symlink、
local module 外への到達、build constraint の解釈差、同じ import path の複数解決、
または descriptor にない追加起点を拒否する。manifest が列挙した `files` から
closure を逆算せず、固定 toolchain の package loader が同じ起点と build context
から再計算した集合との完全一致を要求する。

`semanticSourceSet` は `_test.go`、corpus、baseline、評価 result、provider
adapter、transport、文書、Wiki、候補評価 command 自身、evaluator 実装、
production activation wiring および test 専用 constructor を含めない。
後二者は semantic component から分離した package または source file に置く。
選択済み source file の一部の関数だけを除外せず、wiring または constructor が
同じ file に共存する場合は閉じた source set を作れないものとして manifest を
拒否する。
evaluator の意味は `evaluatorVersion`、production への組立ては
`composition` と第 5 段階の採用検証を定義元とする。外部 module は
選択済み `moduleDependencies` で固定し、その closure に影響する `replace`、
symlink、repository 外 source、環境変数で選ぶ source、動的 path または
未追跡生成物を拒否する。

`candidateContentId` は `candidate-content-sha256-` と小文字十六進六十四桁を
連結する。suffix は `candidateContentId` 自身を除く manifest field を上記の
完全順で `SOT-ENG-033` の canonical scalar、array および object 符号化へ
投影した byte の SHA-256 とする。manifest 原 byte は同 SOT の canonical JSON
byte 規則に従う。

## review attestation

review attestation は candidate content manifest と別 file にし、review の追加で
`candidateContentId` が循環して変わらないようにする。各 attestation は次の項目
だけをこの順で持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `artifactKind` | string | はい | `legal_query_candidate_review_attestation` |
| `schemaVersion` | integer | はい | `2` |
| `attestationId` | string | はい | 後述する review digest |
| `candidateContentId` | string | はい | review した exact candidate content |
| `candidateContentManifestSha256` | string | はい | manifest 原 byte の SHA-256 |
| `reviewScope` | string | はい | `architecture` または `testability` |
| `rubricVersion` | string | はい | review rubric の固定版 |
| `rubricSha256` | string | はい | rubric canonical projection の SHA-256 |
| `reviewerAuthorityId` | string | はい | review orchestration が割り当てた不透明 authority ID |
| `reviewedSOTs` | object[] | はい | review 対象 SOT の ID と原 byte digest |
| `reviewedSOTSetSha256` | string | はい | `reviewedSOTs` の canonical digest |
| `criterionScores` | object[] | はい | scope ごとの五基準と `score20` |
| `score100` | integer | はい | 80 以上 100 以下 |
| `blockerCount` | integer | はい | `0` |
| `majorCount` | integer | はい | 0 以上 |
| `minorCount` | integer | はい | 0 以上 |
| `decision` | string | はい | `approved` |

`reviewedSOTs` の各要素は `sotId` と `sotDocumentSha256` だけをこの順で持つ。
`sotDocumentSha256` は review した repository 内 SOT file の原 byte に対する
SHA-256 とし、改行、Unicode または Markdown を正規化しない。配列は `sotId` の
byte 順に重複なく並べ、対応 request の `requiredReviewSOTs` と object 単位で
完全一致させる。一件の追加、欠落、順序差、同じ ID の別 digest または廃止 ID を
新しい評価準備では許さない。`reviewedSOTSetSha256` は `reviewedSOTs` を
`SOT-ENG-033` の canonical encoding へ投影した byte の SHA-256 とする。

`rubricVersion` は最初の採用で `sot-review-rubric-v1` とする。この版の rubric
canonical projection は、次の field と値だけを表の順で持つ object とする。

| field | 値 |
|---|---|
| `rubricVersion` | `sot-review-rubric-v1` |
| `minimumScore100` | `80` |
| `maximumScore100` | `100` |
| `minimumApprovedCriterionScore20` | `16` |
| `blockerMaximum` | `0` |
| `requiredDecision` | `approved` |
| `allowedCriterionScores` | `0`、`10`、`16`、`20` の固定順配列 |
| `scoreAnchors` | 後述する四件の `score20` と `meaning` を持つ固定順配列 |
| `architectureCriteria` | 後述する五件の `criterionId` と `question` を持つ固定順配列 |
| `testabilityCriteria` | 後述する五件の `criterionId` と `question` を持つ固定順配列 |

`scoreAnchors` は次の値だけを持つ。

| `score20` | `meaning` |
|---:|---|
| `0` | `missing-or-blocking-contradiction` |
| `10` | `intent-present-but-major-choice-open` |
| `16` | `deterministic-and-testable-with-nonblocking-gaps` |
| `20` | `closed-ownership-boundary-failure-and-verification` |

`architectureCriteria` は次の値だけを持つ。

| `criterionId` | `question` |
|---|---|
| `single-sot-ownership` | `is-each-fact-owned-once` |
| `dependency-direction` | `are-active-dependencies-current-and-acyclic` |
| `lifecycle-and-successor` | `are-replacements-versioned-and-predecessors-preserved` |
| `provider-independence` | `is-semantic-planning-free-of-provider-runtime-state` |
| `rollout-boundary` | `are-preparation-evaluation-adoption-and-publication-separated` |

`testabilityCriteria` は次の値だけを持つ。

| `criterionId` | `question` |
|---|---|
| `deterministic-input` | `are-all-semantic-inputs-content-bound-and-ordered` |
| `closed-failure-unit` | `does-each-invalid-or-ambiguous-input-have-one-failure-unit` |
| `resource-bounds` | `are-count-byte-expansion-path-and-cancellation-bounds-closed` |
| `replay-and-identity` | `can-current-validation-and-historical-replay-be-distinguished` |
| `fixed-verification` | `do-fixed-tests-cover-normal-maximum-plus-one-and-conflict-cases` |

各 attestation の `criterionScores` は自身の `reviewScope` に対応する五
`criterionId` を上表の固定順で一件ずつ持ち、各要素は `criterionId` と
`score20` だけをこの順で持つ。`score20` は四 anchor のいずれかとし、
`score100` は五件の整数和と完全一致させる。CI は文章から点数や finding severity
を推測せず、reviewer が anchor に照らして assertion した各値、和、範囲および
digest だけを検証する。全五件が `score20 >= 16`、`score100 >= 80` かつ
`blockerCount = 0` の場合だけ `decision=approved` を許す。一件でも `0` または
`10` なら、ほかの criterion の `20` で補って承認しない。

`rubricSha256` はこの object を `SOT-ENG-033` の canonical encoding へ投影した
byte の SHA-256 とし、request と二件の attestation で完全一致させる。
`score100 >= 80` は `SOT-ENG-039` の 8/10 以上という進行条件を百分率整数へ
変換した値であり、本規定が別の受入点を定義するものではない。
architecture と testability の二件は異なる `reviewerAuthorityId` を使用し、
いずれも候補 content の作成者ではない reviewer または agent が同じ manifest
byte を確認したという repository review 手続の assertion とする。

`attestationId` は `review-sha256-` と小文字十六進六十四桁を連結する。
suffix は `attestationId` 自身を除く field を上記の完全順で
`SOT-ENG-033` の canonical encoding へ投影した byte の SHA-256 とする。
attestation 原 byte も同 SOT の canonical JSON byte 規則に従う。

loader と CI は manifest への exact content binding、二 scope、異なる authority
ID、rubric の内容 digest、対象 SOT の内容 digest、score、blocker および
decision を機械検証する。
`reviewerAuthorityId` だけから人または agent の実在や非作成者性を暗号学的に
証明したとは扱わない。reviewer の独立性は `SOT-ENG-039` の進行条件と repository
review 記録を定義元とし、CI はその governance assertion を identity 証明へ
読み替えない。機械的な署名または pull request approval を将来必須にする場合は、
別の採用規定で authority と検証方式を定義する。

## 評価 request

request は次の項目だけをこの順で持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `artifactKind` | string | はい | `legal_query_candidate_evaluation_request` |
| `schemaVersion` | integer | はい | `2` |
| `evaluationId` | string | はい | 後述の候補 tuple digest |
| `evaluatorVersion` | string | はい | evaluator の固定意味版 |
| `corpusVersion` | string | はい | 評価する候補 corpus |
| `corpusManifestSha256` | string | はい | corpus manifest 原 byte の SHA-256 |
| `holdoutDigest` | string | はい | corpus manifest の holdout digest |
| `holdoutLeakageGroupDigests` | `string[]` | はい | corpus manifest が固定する holdout leakage group の digest 集合 |
| `candidateContentId` | string | はい | 評価する candidate content |
| `candidateContentManifestSha256` | string | はい | content manifest 原 byteの SHA-256 |
| `reviewRubricVersion` | string | はい | 二件の review が使う rubric 版 |
| `reviewRubricSha256` | string | はい | rubric canonical projection の SHA-256 |
| `requiredReviewSOTs` | object[] | はい | 二件の review がともに確認する閉じた SOT 内容集合 |
| `requiredReviewSOTSetSha256` | string | はい | `requiredReviewSOTs` の canonical digest |
| `reviewAttestations` | object[] | はい | content-bound な二件の review |
| `baselineVersion` | string | はい | 成功時に使用する予約 baseline 版 |

`reviewRubricVersion` と `reviewRubricSha256` は前述の
`sot-review-rubric-v1` とその canonical digest に完全一致させる。
`requiredReviewSOTs` の各要素は `sotId` と `sotDocumentSha256` だけをこの順で
持ち、`sotId` の byte 順に一意に並べる。初回の core と `judicial-cases`
候補では、正確に次の有効 SOT ID と、評価準備 commit にある各 file の原 byte
digest だけを持つ。

```text
SOT-ARCH-018
SOT-ARCH-021
SOT-ARCH-023
SOT-ARCH-025
SOT-ARCH-027
SOT-ARCH-028
SOT-ARCH-031
SOT-ARCH-033
SOT-ARCH-036
SOT-ARCH-037
SOT-ARCH-038
SOT-ARCH-039
SOT-ENG-007
SOT-ENG-008
SOT-ENG-009
SOT-ENG-020
SOT-ENG-022
SOT-ENG-023
SOT-ENG-024
SOT-ENG-025
SOT-ENG-026
SOT-ENG-027
SOT-ENG-028
SOT-ENG-030
SOT-ENG-031
SOT-ENG-032
SOT-ENG-033
SOT-ENG-035
SOT-ENG-036
SOT-ENG-038
SOT-ENG-039
SOT-IF-022
SOT-IF-023
SOT-IF-024
SOT-IF-025
SOT-IF-034
SOT-IF-040
SOT-IF-041
SOT-IF-042
SOT-IF-051
SOT-MODEL-013
SOT-MODEL-016
SOT-MODEL-018
SOT-MODEL-022
SOT-MODEL-023
SOT-MODEL-024
SOT-MODEL-025
SOT-MODEL-026
SOT-MODEL-027
SOT-MODEL-028
SOT-MODEL-030
SOT-MODEL-031
```

この閉じた集合は、schema version 2 の初回候補評価で review する固定契約集合である。
五 component、`judicial-cases` pack、七 capability、辞書、cue、候補の共有 field 型、
候補と結果の model、profile 内分岐、profile 横断合成、評価成果物、採用境界、
公開 MCP への原子的接続および導入順序を含む。第 6 段階の独立した provider parser
および resolver、第 7 段階の案内文書同期、Wiki、実装 file または commit は、
候補の holdout 評価可否を決める契約集合へ含めない。loader は Markdown link を
実行時に再帰走査して集合を増減させず、上記の exact list とだけ照合する。
廃止 SOT を有効な定義元の代わりにしない。

候補へ新しい profile、model、capability、composition 規則、評価境界または直接の
定義元 SOT を加える場合、若しくは上記の一件を後継へ移行する場合は、schema
version 2 の配列を変更しない。本規定を置き換える後継 SOT と新しい schema version
で新しい exact list を定義し、新しい二件の review と request を作る。既存 request
と historical replay は version 2 の集合を保持する。
`requiredReviewSOTSetSha256` は `requiredReviewSOTs` を `SOT-ENG-033` の
canonical encoding へ投影した byte の SHA-256 とする。

`requiredReviewSOTs` は一件以上百二十八件以下とする。新しい評価の準備では、
repository root から `sot/00-index.md` と、そこに固定された
`00-product`、`10-scenarios`、`20-model`、`30-architecture`、`40-interfaces`、
`50-engineering` および `60-delivery` の各 `00-index.md` だけを root-scoped API
で開き、各 domain index が一度だけ参照する番号付き Markdown から `sotId` を
一意に解決する。index 外 file、絶対 path、`..`、symlink、外部 URL、
同じ ID の複数 file または heading の ID と一致しない file を拒否する。
この解決は `SOT-ENG-007` から `SOT-ENG-009` の ID、状態および index 規則と
同じ検証を用い、manifest または attestation が任意の SOT path を指定できる
field を設けない。

`reviewAttestations` は `architecture`、`testability` の固定順で正確に二件を持つ。
各要素は `reviewScope`、`attestationId` および `attestationSha256` だけをこの順で
持ち、対応する不変 attestation の scope、ID および原 byte digest と完全一致させる。
二件は request と同じ `candidateContentId` と `candidateContentManifestSha256` を
参照し、各 attestation の rubric 版・digest、`reviewedSOTs` および
`reviewedSOTSetSha256` は request の対応 field と完全一致しなければならない。

result がまだない新しい評価の準備と第 5 段階への進行では、loader と CI は
repository の SOT index から各 `sotId` を一意に解決し、file の ID、`状態: 有効`
および原 byte digest を request と二件の attestation に照合する。同じ ID の本文が
review 後に一 byteでも変わった場合は、新しい二件の review と request を必要とする。

追跡済み result の歴史的 replay では、後日に SOT が廃止または明確化されても
過去の request を無効化しない。現在の SOT 状態または現在 file の byte との一致を
要求せず、不変な request と二件の attestation が同じ historical
`{sotId, sotDocumentSha256}` 集合、集合 digest および rubric digest を内部保持する
ことだけを検証する。過去値を現在の SOT へ置換または補正しない。

`SOT-ENG-036` report との一致では、request から
content manifest の `profileSet` と固定順の
`profileArtifacts[].{profileId, profileVersion}` を射影し、report の閉じた
`profileSet` と完全一致させる。report が持たない metadata、cue、辞書、
composition および semantic source の identity は比較から黙って捨てず、
content manifest、test 専用 composition root および後続 production 採用内容との
完全一致で別に検証する。

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

新しい candidate request は `SOT-ENG-026` の schema version 2 以上かつ
`corpus-v13` 以降で、同 SOT の派生観測母集団の評価準備検証を通過する corpus
だけを参照する。request constructor は `corpus-v1` から `corpus-v12` を新しい
request の ID を生成する前に拒否する。既存の不変 request、result および履歴が
`corpus-v1` から `corpus-v12` を参照する場合は、その loader と replay を拒否せず、
この新規作成境界を遡及的な schema 無効化に使用しない。

`corpusManifestSha256`、`holdoutDigest` および
`holdoutLeakageGroupDigests` を corpus manifest と完全一致させ、manifest の
一部だけを別 corpus へ差し替えない。digest 配列は同 SOT の形式、byte 順、
一意性および一件以上四百件以下を満たし、raw `leakageGroupId` を request へ
複製しない。request が参照する content manifest の exact component tuple は
test 専用 constructor が直接構成し、production の active profile set または
標準 evaluator の current tuple から推測しない。candidate command は corpus、
holdout fixture または evaluator を読む前に、content manifest、全参照 artifact、
semantic source closure、二件の review attestation および request digest を
再計算して完全一致を確認する。

`baselineVersion` は、bootstrap で request を持たずに導入する現行版を除き、
全 request、成功 baseline および失敗履歴を通じて一意に予約する。失敗した
request の予約版を後続候補へ再利用しない。

## 評価 result と report

result は次の項目だけをこの順で持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `artifactKind` | string | はい | `legal_query_candidate_evaluation_result` |
| `schemaVersion` | integer | はい | `2` |
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
GOWORK=off GOENV=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS='-mod=readonly -buildvcs=false' GOOS=linux GOARCH=amd64 GOAMD64=v1 GOEXPERIMENT= CGO_ENABLED=0 GOMAXPROCS=1 go run ./cmd/legal-query-candidate-eval --repository=. --output-directory=./.artifacts/legal-query-candidate-evaluation
```

command は上記二引数と値だけを受理する。`GOWORK=off`、
`GOENV=off`、`GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`、
`GOFLAGS=-mod=readonly -buildvcs=false`、`GOOS=linux`、`GOARCH=amd64`、
`GOAMD64=v1`、空の `GOEXPERIMENT`、`CGO_ENABLED=0` および空の build tag は
`semanticSourceSet` の固定 module・build context とし、CI workflow は manifest の
exact `goToolchainVersion` を setup する。
`GOMAXPROCS=1` は省資源実行のための固定値であり、評価意味を選ぶ
設定ではない。profile、corpus、baseline、`evaluationId`、holdout、evaluator
または output file を上書きする引数、ほかの環境変数、設定、build tag、
hidden mode または fallback を設けない。固定 `current.json` から request を
一件読み、content manifest が固定した候補 composition root を test 専用
constructor で直接構成する。

CI workflow は command の前に、content manifest と root module file だけを読み、
manifest の `goToolchainVersion` と `moduleDependencies` に完全一致する toolchain
および raw module zip・`go.mod` を evaluation ごとの新規で空の archive staging
root へ準備して、原 byte checksum と圧縮 byte 上限を検証する。この段階では
module zip を展開しない。host の ambient module cache、前回評価の extracted tree
または部分 download を再利用しない。
この準備段階は corpus manifest、holdout fixture、評価期待値、baseline または
evaluator を開かず、列挙されていない module を download しない。network を必要と
する取得はこの準備段階の終了までに限定し、その後の candidate command では cache を
read-only とし、`GOPROXY=off` と job の network 不使用条件を維持する。

command の child process へ渡す環境は、上記の固定 `GO*` 値と、setup 済み
`PATH`、`GOROOT`、read-only `GOMODCACHE`、隔離した `GOCACHE`、`TMPDIR` の
infrastructure path だけの allowlist とする。これら path の絶対値を manifest、
report、result または log へ入れず、source 選択または評価 byte に利用しない。
継承された `GOFLAGS`、`GOEXPERIMENT`、`GOAMD64`、`GODEBUG`、`GOPRIVATE`、
`GONOPROXY`、`GONOSUMDB`、`GOWORK`、`GOENV` その他の `GO*` 値は、固定値へ
置き換えるか allowlist 外として除去する。

`cmd/legal-query-candidate-eval` は、標準 library だけを import する bootstrap
package とし、candidate component、profile、composer、selector、evaluator または
test 専用 constructor を直接 import しない。bootstrap は corpus、holdout または
evaluator を開く前に、固定 pointer、request、content manifest、root module file
および exact `evaluatorVersion` の閉じた worker source registry から、次の二つを
`TMPDIR` 内の evaluation ID 専用新規 directory へ exclusive create する。

- root `go.mod` と `go.sum`、`semanticSourceSet.files` の検証済み byte、および
  exact evaluator version の worker/evaluator source closure だけを同じ
  repository-relative path に置く verified source tree
- 前節の検証済み raw archive を一回だけ展開した `moduleDependencies` だけを持つ
  fresh module cache

worker/evaluator source closure は `evaluatorVersion` の不変 registry が固定順の
path と raw digest で列挙し、candidate semantic source と混同しない。同 registry
は新しい evaluator version でだけ変更でき、追跡済み result の replay に必要な
旧 closure を保持する。source file は `8192` 件、一件 `8 MiB`、合計 `128 MiB`
を上限とし、候補 component と重複する path は同じ raw digest の一件へ縮約する。

各 file は前述の root-scoped descriptor から検証した同じ byte を一回だけ書き、
途中 directory と file の symlink、既存 entry、hardlink および path の
再解決を許さない。全 path、size と raw digest を materialize 後に再確認して
source tree と module cache を read-only に seal する。その後の component closure
用 `go list -deps -json`、worker closure 用 `go list` および worker build はすべて、
この verified source tree を working directory、この fresh module cache を
`GOMODCACHE` として実行する。元 checkout、ambient cache または閉じた descriptor
から直接 `go list` 若しくは build を行わない。

verified tree 内の各 `packageRoot` から再計算した component package、全 build input、
external module および `goDebugSettings` は content manifest と完全一致させる。
worker build 後に一件の binary を実行し、その binary が verified component を
直接構成して評価する。実行後に source tree と module cache の digest および
read-only 状態を再確認し、scratch tree は workflow artifact または評価 identity へ
含めない。候補 command の output directory は後述の report/result 二 file だけを
持ち、scratch path、absolute path または materialized source を出力しない。

この command は製品 binary、標準 evaluator、中央品質ゲート、MCP、transport
または利用者向け CLI へ登録しない。`SOT-ARCH-033` の準備状態で許可する
CI handoff 専用入口であり、利用者の query を受け取らず、候補 profile set を
production request の処理へ渡さない。

clean checkout の candidate command は外部 network、時刻、乱数、provider
response または実時間待機を使わず、一つの evaluator run だけを行う。
未追跡の固定 output root は、
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

第 4 段階 4 では、content manifest、二件の content-bound review attestation、
candidate evaluator、request、pointer および予約 baseline version を準備するが、
holdout を実行しない。

第 4 段階 5 の開始時に、同じ準備 commit の専用 CI job
`candidate-evaluation` を一回だけ手動起動する。workflow artifact から得た
result と report を、profile、corpus、evaluator、受入基準または request を
変更しない handoff commit へ追加する。成功 report は予約 baseline path、
失敗 report は failed-report path へ、その byte のまま置く。

この job は手動 dispatch だけを起動条件とし、pull request、push、schedule、
release、fork またはほかの workflow から自動起動しない。content manifest、
二件の attestation、request、pointer および workflow definition を含む不変な
準備 Git object ID を checkout し、job 開始時の対象 object ID と `HEAD` を同じ
値として検証する。branch、tag または moving ref を評価 identity にしない。
checkout 後の worktree が dirty、submodule が未知、content digest または
attestation binding が不一致の場合は command を起動しない。job の repository
permission は `contents: read` に限定する。runner が checkout と workflow
artifact upload の制御面に必要とする read token はその action だけが使用し、
candidate command の process、cache 準備 process、環境、引数、標準入力、
作業 file または log へ渡さない。release permission、write token および
environment secret は job に与えない。

Git object ID、branch 名または attestation file の存在だけで「独立 review 済み」
と推測しない。job が機械検証する権威は exact checkout、candidate content binding、
attestation の二 scope、異なる authority ID、score、blocker、decision および
request identity までとする。reviewer の非作成者性は同じ candidate content に
対する repository review 記録と `SOT-ENG-039` の進行条件で確認する。

構造上有効な `SOT-ENG-036` report byte が最初に完成した時点で、その
`holdoutDigest` を採用判定に一回使用したとみなす。passed と failed の両方が
消費に当たる。同じ `evaluationId`、不変 request および同じ report byte の CI
retry、artifact upload の再試行または後続 replay は再現確認であり、新しい利用と
数えない。有効な report ができる前の infrastructure 失敗だけは同じ ID で
再試行できる。

有効な report ができる前でも、修正後の semantic source set、対象 SOT または
review binding が旧 request と一致しなくなった場合は、同じ ID の内容を変えない。
前述の置換済み準備として旧 request と全参照成果物を保持し、別の
`evaluationId` と未使用の `baselineVersion` を持つ新 request へ pointer を進める。
新 request の source と SOT の外部参照検証は current request と、それが直接参照する
manifest と attestation だけを対象にする。置換済み準備は file 名、canonical byte、
内部 ID、digest、request から manifest・attestation への binding および資源上限を
引き続き検証するが、現在の source tree または現在の SOT byte との一致を要求しない。

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

同じ passed request が参照する `candidateContentId` と manifest 原 byte digest は、
採用変更の profile metadata、cue artifact、辞書、composition descriptor および
semantic source set から再計算した値と完全一致させる。production activation
wiring と test 専用 constructor は semantic source set に含めないため、採用変更
では別に、production composition descriptor の component tuple が同じ manifest の
`composition` と完全一致し、同じ component が active 経路から構成されることを
確認する。version string だけが同じで content digest が異なる候補、一部 component
だけを差し替えた候補または別 review attestation の候補を採用しない。

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
拒否する。root は `schema-v2.json`、`current.json`、`content-manifests/`、
`review-attestations/`、`requests/`、`results/` および `failed-reports/` だけを
持つ。各 directory は対応する正規形 ID の `.json` 通常 file だけを持ち、
subdirectory、未知 entry、未登録 file、device、FIFO および socket を拒否する。

result がない準備 tree は、一件の current request に加えて零件以上の置換済み準備を
持てる。全 request の `baselineVersion` は一意とし、各 manifest と attestation は
いずれか一件以上の request から参照され、各 request は正確な二件の attestation と
一件の manifest へ閉じて binding されなければならない。current 以外の result なし
request を current として実行せず、その reservation、件数および byte を履歴上限から
除外しない。current pointer が存在しない request を指す場合、同じ baseline の重複、
孤立成果物、置換済み準備に対応する baseline file 若しくは failed report、または
result 完成後の同一 holdout 再評価は fail-closed とする。

content manifest が参照する lexicon と local source、root module file、
evaluator version registry が列挙する worker source、および新しい評価の
`requiredReviewSOTs` が参照する index と SOT document は、candidate evaluation
subtree の path 解決器へ渡さず、repository root の開いた directory handle から
前述の loader 所有 root、許可 file、`packageRoot`、固定 worker registry および
固定 SOT index 解決結果だけを root-scoped API で開く。各 path component を
follow しないで検証し、
最後の file は一つの descriptor を開いたまま `fstat`、上限付き読取り、SHA-256
および読取り後の `fstat` を行う。
読取り前後で file type、size または利用可能な file identity が変わった場合、
short read、追記、置換または path の再解決が必要になった場合は候補全体を拒否する。
一度閉じた path を hash のために再度開かず、検証済み descriptor の同じ byte だけを
decode または package loader へ渡す。

lexicon、local source、root module、worker source、SOT index と document の
一意な file 集合、および module cache の選択済み dependency は、holdout、
corpus fixture または evaluator を開く前に次の上限を満たす。重複 path は一件に
縮約して同じ digest であることを確認し、異なる digest なら拒否する。

| 参照対象 | 件数 | 一件の原 byte | 原 byte 合計 |
|---|---:|---:|---:|
| `lexiconArtifacts` | `16` | — | — |
| 全 lexicon file | `4096` | `8 MiB` | `128 MiB` |
| `semanticSourceSet.files` | `8192` | `8 MiB` | `128 MiB` |
| evaluator/worker source file | `8192` | `8 MiB` | `128 MiB` |
| root module file | `go.mod` 一件、`go.sum` 一件 | `go.mod` `1 MiB`、`go.sum` `8 MiB` | `9 MiB` |
| `moduleDependencies` | `1024` | module zip `64 MiB`、`go.mod` `1 MiB` | module zip `512 MiB`、`go.mod` `16 MiB` |
| module zip entry | 一 module `16384`、全体 `131072` | 展開後 regular file `16 MiB` | 一 module `128 MiB`、全体 `512 MiB` |
| SOT index | `8` | `1 MiB` | `8 MiB` |
| `requiredReviewSOTs` の document | `128` | `1 MiB` | `16 MiB` |

各参照 file は宣言上限に一 byte を加えた bounded reader で一回だけ読み、
file 間および長い読取りの chunk 間で呼出し元 context の cancellation を確認する。
件数または合計上限は hash 計算中も加算し、超過を検出した時点で残りを開かない。
外部 module は CI の準備段階で manifest の exact
`modulePath`、`version`、`moduleZipSum`、`moduleZipRawSha256`、
`moduleZipByteLength`、`moduleZipEntryCount`、`moduleExpandedByteLength`、
`moduleGoModSum` および `moduleGoModRawSha256` だけを新規の空 archive staging
root へ準備する。candidate bootstrap は、圧縮 byte を bounded reader で
再検証した zip を一回だけ新規 module-cache staging directory へ展開する。
各 entry の header を処理する前に件数、正規化不要な module root prefix、
UTF-8 path の `512` byte 上限、`..`、絶対 path、separator、重複および file type を
検証し、regular file
と directory 以外、symlink、hardlink、device、FIFO および socket を拒否する。
各 regular file は宣言上限に一 byte を加えた reader で展開し、entry、module および
全体の展開 byte を同時に加算する。

展開完了後は staging tree を root-scoped API で一回列挙し、zip の全 regular file
path、entry 数、各 file size と digest、module 合計 byte、`moduleZipSum` および
manifest の count・digest と完全一致させる。欠落、追加、case collision または
展開中の置換を拒否し、一致した tree だけを read-only module cache へ原子的に
seal する。候補 worker の `go list`、build および実行後にも同じ tree digest と
read-only 状態を再確認する。候補 command はこの fresh cache root だけを
root-scoped API で開き、未列挙 module、sum 不一致、上限超過または ambient cache
fallback を holdout の読取り前に拒否する。

資源上限は次とする。

| 対象 | 上限 |
|---|---:|
| schema | `1 MiB` |
| pointer | `64 KiB` |
| candidate content manifest 一件 | `4 MiB` |
| review attestation 一件 | `256 KiB` |
| request 一件 | `256 KiB` |
| result 一件 | `256 KiB` |
| report 一件 | `4 MiB` |
| candidate content manifest 件数・原 byte 合計 | `4096`・`64 MiB` |
| review attestation 件数・原 byte 合計 | `8192`・`64 MiB` |
| request 件数・原 byte 合計 | `4096`・`32 MiB` |
| result 件数・原 byte 合計 | `4096`・`32 MiB` |
| failed report 件数・原 byte 合計 | `4096`・`256 MiB` |
| JSON depth | `16` |
| pointer・request・result・attestation 一 document の value 数 | `8192` |
| candidate content manifest 一件の value 数 | `65536` |
| report 一件の value 数 | `SOT-ENG-036` の `65536` |

各 file は open 前後に size と通常 file であることを確認し、宣言上限に一 byte を
加えた bounded reader で一回だけ読む。directory 列挙中も件数と合計 byte の上限を
超えた時点で失敗し、呼出し元 context の cancellation を各 file 間で確認する。

typed decode より前に不正 UTF-8、BOM、重複 key、`null`、未知 field、二個目の
JSON 値、後方 token、上限超過および canonical byte 不一致を拒否する。
一つの不正 artifact を無視して部分 history を返さない。

各 request は同じ ID と raw digest の一件の content manifest、および異なる
scope を持つ二件の attestation を参照する。各 attestation は同じ content ID と
manifest digest を参照し、file 名、内部 ID、raw digest、canonical digest および
review 条件を一致させる。不正な manifest または attestation を無視して request
だけを返さない。どの request からも参照されない manifest または attestation は
孤立成果物として拒否し、参照済み成果物だけの部分 history を返さない。

各 result は同じ ID の一件の request を参照し、`requestSha256` を一致させる。
一 request に result を二件作らない。`outcome=passed` では予約した baseline file が
`reportSha256` と一致し、同じ ID の failed report が存在しないことを確認する。
`outcome=failed` では同じ ID の failed report が `reportSha256` と一致し、予約した
baseline file が存在しないことを確認する。result がない request には、追跡済みの
baseline file または failed report が存在してはならない。孤立した result、report
または同名予約の衝突を無視しない。

manifest、attestation、request、result、report、output および log は、query、
fixture 本文、辞書 entry、review comment、reviewer 名、外部 response、
credential、絶対 path、時刻、host、user、commit message または個人情報を
持たない。manifest の source と lexicon file path は repository-relative path
だけを許す。候補 command と worker 自身は失敗時の stdout と stderr を空に保つ。
CI log は `candidateContentId`、`evaluationId`、artifact digest および `outcome` に
加えて、process 実行器が機械的に付加する ASCII の `exit status {code}` だけを
出せる。`code` は候補 command が返せる閉じた非零終了 code に限る。本規定は
その値と段階の意味を定義せず、段階名、内部原因本文、child の前行または file 内容を
追加しない。

## 確認

外部 network を使わない schema、loader、command plan、evaluator および adoption
契約 test で、少なくとも次の固定 test ID を確認する。

- `candidate-evaluation-closed-artifacts`
- `candidate-evaluation-request-identity`
- `candidate-evaluation-candidate-content-identity`
- `candidate-evaluation-referenced-file-bounds`
- `candidate-evaluation-review-attestation`
- `candidate-evaluation-review-content-binding`
- `candidate-evaluation-evaluator-version-match`
- `candidate-evaluation-build-context-isolation`
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

`candidate-evaluation-candidate-content-identity` は、metadata の optional field
存在状態、cue 原 byte、複数 file 辞書の path 順、composition component 順、
role ごとの `packageRoot`、semantic source closure、固定 module・build context、
`goDebugSettings`、`HFiles`、`SysoFiles`、`go:embed`、選択済み外部 module の
content checksum、raw zip digest、entry 数、展開 byte および除外境界を一件ずつ
変えた場合に `candidateContentId` が変わることを確認する。未使用 dependency
または無関係な `go.sum` 行だけの変更では同じ identity を保つ。version string が
同じでも digest が異なる場合は holdout を読む前に拒否し、許可外 lexicon path、
未知 component root、selected closure に影響する replacement、symlink、
repository 外 path および source closure 欠落を fail-closed とする。

`candidate-evaluation-referenced-file-bounds` は、現行法令名 file を含む許可三 file
を受理し、lexicon/source/root module/module/SOT index/SOT document の件数、
一件の byte および合計 byte の各正常最大を受理して一件または一 byte の超過を
拒否する。module zip は entry、entry path、展開一 file、一 module 展開合計および
全 module 展開合計の正常最大を受理し、一件または一 byte の超過、zip bomb、
重複 entry、symlink および extracted tree の欠落・追加・digest 差を拒否する。
root 外 path、index 外 SOT、途中または最後の
symlink、非 regular file、読取り中の size・identity 変更、short read、重複 path の
digest 不一致および cancellation を corpus と evaluator の読取り前に拒否する。

`candidate-evaluation-review-attestation` は architecture と testability の正確な
二 scope、異なる authority ID、同じ candidate content binding、request の
必須集合と完全一致する対象 SOT 内容、
全 criterion の `score20 >= 16`、`score100 >= 80`、`blockerCount = 0` および
`decision=approved` を確認する。
attestation の欠落、重複 scope、同一 authority、別 content、別 raw manifest
digest、必須 SOT の欠落・追加・順序差・別 document digest、集合 digest 不一致、
未知 rubric 版、rubric digest 不一致、criterion ID の欠落・追加・順序差、
anchor 外 `score20`、承認時の `score20 < 16`、五件の和と異なる `score100`、
80 未満、blocker 一件以上または非 approved では command を起動しない。
CI の機械検証が reviewer 独立性の暗号学的証明ではないことも governance test の
assertion とする。

`candidate-evaluation-review-content-binding` は result のない新しい評価では全 SOT
file が有効で原 byte digest と一致すること、一 byte の本文変更または廃止で新しい
review を要求することを確認する。追跡済み result の replay では現在の SOT 状態や
byte を参照せず、request と二件の attestation が保持する historical SOT 集合・
digest と rubric digest の内部一致だけで同じ評価 byte を再現することを確認する。

`candidate-evaluation-corpus-admissibility` は、新しい request constructor が
`corpus-v1` から `corpus-v12` を拒否し、`corpus-v13` 以降について schema version、
manifest digest、holdout digest および leakage digest の完全一致を要求することを
確認する。同じ検証で、既存 request と結果履歴の loader および replay は
`corpus-v1` から `corpus-v12` の参照を従来どおり受理することを確認する。

`candidate-evaluation-build-context-isolation` は exact toolchain と選択済み module
だけを準備した後、候補 command が `GOPROXY=off`、`GOENV=off`、
`GOTOOLCHAIN=local`、`GOAMD64=v1`、空の `GOEXPERIMENT` および閉じた環境 allowlist
で動くことを確認する。root `go.mod` の `godebug` 有無、header、`.syso` および
全選択 build input が identity に反映されること、検証 byte だけを exclusive
materialize した read-only source tree と fresh module cache から全 `go list`、
worker build および実行を行うこと、実行後も tree digest が一致することを確認する。
未準備 dependency、toolchain fallback、network access、元 checkout からの build、
ambient cache、継承した `GO*` 値、build tag または read-write tree/cache を拒否する。

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
- [SOT-ENG-039: 内容固定済み候補による統合照会の導入段階と変更順序](39-content-bound-unified-query-rollout-stages.md)
- [SOT-ENG-040: 候補 holdout 評価の閉じた失敗段階診断](40-candidate-evaluation-failure-diagnostics.md)
- [SOT-ENG-035: 統合照会 profile metadata 成果物契約](35-unified-query-profile-metadata-artifact-contract.md)
- [SOT-ENG-036: 統合照会の評価 baseline 成果物契約](36-unified-query-evaluation-baseline-artifact-contract.md)
