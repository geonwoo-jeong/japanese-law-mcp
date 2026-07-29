# SOT-ENG-026: 統合照会の評価コーパス成果物契約

- 状態: 有効

## 規定

統合照会の固定評価コーパスは、版付きの閉じた JSON 成果物として保持し、repository 内に閉じた厳格な loader で完全性、安全性、集合分離および不変性を検証してから評価へ渡す。

## 適用範囲

本規定は、`SOT-ENG-024` が定める評価コーパスのファイル形式、機械処理用カテゴリ、checksum、loader の入力境界および評価専用の期待値を定義する。意味判定 profile、評価指標の計算、baseline、fake capability の実行および評価 command の出力は本規定で定義せず、それぞれの定義元に従う。

評価成果物を製品の `LegalQueryCandidate` または `LegalQueryPlan` の保存形式にしない。評価 fixture は、内部 ID、score、profile version、能力 route および派生予算を固定せず、意味上必要な投影だけを持つ。

## 配置

評価成果物は次の配置とする。

```text
testdata/legalquery/
├── schemas/
│   └── legal-query-corpus-v1.schema.json
└── corpus-vN/
    ├── manifest.json
    ├── development/
    │   └── {caseId}.json
    ├── holdout/
    │   └── {caseId}.json
    └── execution/
        └── {caseId}.json
```

`schemas/legal-query-corpus-v{schemaVersion}.schema.json` は JSON Schema Draft 2020-12 とし、対応する一つの schema version について `artifactKind` の `corpus_manifest`、`semantic_case` および `execution_case` を閉じた `oneOf` で表す。すべての object は未知の項目を拒否し、条件付き variant は discriminator と対応しない項目を持たない。`$ref` は同じ schema 内の fragment だけを参照し、network、`file:` URL または外部 schema を解決する resolver を使用しない。

Git checkout による checksum の差を防ぐため、`testdata/legalquery/**/*.json` は `.gitattributes` で LF を固定する。

## 共通識別子と順序

`caseId`、`leakageGroupId`、`categoryId`、`coverageId`、`scenarioId`、`meaningId` および pack ID は、1 byte 以上 64 byte 以下で、小文字 ASCII 英数字の segment を一つの `-` で連結した値とする。`corpusVersion` は `corpus-v` と先頭が零ではない十進整数を連結した正規形とする。

配列は意味上の順序を持つ場合を除き識別子の昇順とし、重複を許さない。loader は不正な順序を黙って整列しない。

`caseId` は三集合全体で一意とし、所属集合の `development-`、`holdout-` または `execution-` で始める。manifest は任意の file path を持たず、loader は集合名と `caseId` から `{set}/{caseId}.json` だけを導出する。

## manifest

`manifest.json` は次の項目だけを持つ。

| 項目 | 型 | 必須 | 制約 |
|---|---|---:|---|
| `artifactKind` | string | はい | `corpus_manifest` |
| `schemaVersion` | integer | はい | `1` |
| `corpusVersion` | string | はい | corpus directory の basename と一致 |
| `seed` | integer | はい | `0` 以上 `2147483647` 以下 |
| `holdoutDigest` | string | はい | holdout の順序付き entry 全体を固定する SHA-256 |
| `requiredCategoryIds` | `string[]` | はい | 本規定の必須カテゴリ ID 全件 |
| `requiredExecutionScenarioIds` | `string[]` | はい | 本規定の必須実行 scenario ID 全件 |
| `sets` | object | はい | `development`、`holdout`、`execution` を一件ずつ保持 |

各集合は `caseCount` と `cases` を持つ。`caseCount` は `cases` の件数および実在する fixture 件数と一致する。`cases` は `caseId` 昇順の `{caseId, sha256}` だけを持ち、`sha256` は対応 fixture の原 byte 全体に対する小文字十六進六十四桁とする。manifest 自身は循環参照を避けるため checksum 対象にしない。

`holdoutDigest` は、holdout の `cases` を manifest 順に `caseId`、ASCII space 一文字、`sha256`、LF 一文字として連結した byte 列の SHA-256 を、小文字十六進六十四桁で表す。

同じ checksum を異なる `caseId` へ登録しない。manifest の宣言、fixture file 名、fixture 内の `caseId`、所属集合および実在 file 集合は完全に一致しなければならない。

## semantic case

`development` と `holdout` の fixture は `artifactKind=semantic_case` とし、次の項目だけを持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `artifactKind` | string | はい | `semantic_case` |
| `schemaVersion` | integer | はい | `1` |
| `caseId` | string | はい | manifest と file 名に一致する ID |
| `leakageGroupId` | string | はい | 同じ発話、法的対象および変形群を束ねる ID |
| `coverageIds` | `string[]` | はい | 詳細な検証対象 |
| `safetyVariant` | string | 条件付き | `ordinary` または `adversarial` |
| `enabledPacks` | `string[]` | はい | 当該 case で有効な採用済み pack |
| `request` | object | はい | 公開入力境界へ渡す原入力 |
| `expected` | object | はい | request 境界または意味判定の期待投影 |

同じ発話 template、同じ法的対象、別名、表記差、誤記または言い換えから作った case は、同じ `leakageGroupId` を持たなければならない。一つの leakage group を development と holdout に分割しない。

`request` は必須の `query: string`、任意の `ref: object` および任意の `limitPerAttempt: integer` だけを持つ。`ref` は `providerId: string` と閉じた `key` object を持ち、`key` は `sourceId: string`、`resourceType: string`、`resourceId: string` および任意の `versionId: string` だけを持つ。これは `SOT-MODEL-016` の境界表現であり、型不一致、欠落、`null` および未知項目は MCP Schema の契約 test で扱い、semantic fixture に置かない。

境界値違反を評価できるように loader は `request` を `LegalQueryRequest` constructor で受理可能な値へ矯正せず、型を保った原値を保持する。ただし JSON 型、file 安全性および本規定の資源上限は検証する。

`expected` は `kind` で区別する次の閉じた variant の一つとする。

| `kind` | 項目 | 意味 |
|---|---|---|
| `plan` | `decision`、`reasonCodes`、`meanings`、`selectedMeaningIds` | request が受理された後の意味判定 |
| `request_error` | `errorCode`、`field` | plan を作る前の公開入力エラー |

`request_error` の `errorCode` は `invalid_argument` とし、`field` は `query`、`ref` または `limitPerAttempt` とする。公開文言を golden にしないため `message` と `reason` は持たない。製品 request constructor で拒否される fixture は `request_error` を使用し、`plan` の項目を持たない。受理される上限値を確認する fixture は `input-boundary` カテゴリであっても `plan` を使用できる。

`kind=plan` の期待値は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `decision` | string | はい | `LegalQueryPlan` の decision |
| `reasonCodes` | `string[]` | はい | 順序を含む正確な決定理由 |
| `meanings` | `ExpectedMeaning[]` | はい | 主正解を先頭にした正しい意味。`unsupported` だけ空を許す |
| `selectedMeaningIds` | `string[]` | はい | selection 順の意味参照 |

`ExpectedMeaning` は `meaningId`、順序を含む正確な `evidenceCodes`、辞書根拠の `conceptIds`、昇順の `requiredPacks` および一件以上四件以下の `steps` を持つ。`meaningId` は fixture 内でだけ安定した評価用 ID であり、製品の `candidateId` と一致させない。`meanings` は実際の ranked candidate 全件を golden にするものではなく、正しい意味だけを持つ。実際の追加候補は固定上限内で許可する。

各 step は `task`、`resource`、`inputKind` および対応する `logicalInput` だけを持つ。許可する七つの組合せと logical input の意味は `SOT-MODEL-022` に従う。`capabilityId` と major version はこの組合せから一意に決まるため重複して記録しない。

期待値は `candidateId`、`stepId`、`semanticScore`、`confidence`、`profileVersion`、能力 route、provider ID の推測値または派生 budget を持たない。評価器は実際の候補を次の意味署名へ投影して照合する。

```text
requiredPacks
+ ordered(task, resource, inputKind, logicalInput)
```

`evidenceCodes` と `conceptIds` は、意味署名と一致した候補に対する別の正確な assertion とし、意味候補の top-1、top-2 または high-confidence の正解判定へ混在させない。

`conceptIds` は active な法概念辞書の source tuple
`{conceptId,title,url,confirmedOn}` と整合する entry だけを評価対象にできる。
corpus 自体は tuple 全体を重複記録せず `conceptId` だけを持ち、評価器が実際の
候補に含まれる公開 `conceptSources` の完全 tuple と辞書版を照合する。

`unsupported` 以外は `meanings` を一件以上持つ。`meanings` の先頭を ranking 指標の主正解とし、残りは正しい代替解釈とする。各 expected meaning は実際の ranked candidate に同じ意味署名で一件以上存在しなければならない。`selectedMeaningIds` は `meanings` の要素だけを重複なく参照し、実際の selection と順序を含め完全に一致する。decision ごとの件数と理由は `SOT-MODEL-023` に従う。

評価器は `enabledPacks` と各 meaning の `requiredPacks` から availability を導出する。`single` と `hedged` の選択はすべて `available`、`capability_unavailable` の選択は一件以上の `pack_disabled`、`needs_clarification` の選択は `available` とし、`unsupported` と `request_error` は外部呼出し対象を持たない。fixture に availability を重複記録しない。

## 必須カテゴリと coverage

manifest の `requiredCategoryIds` は次の十二件を昇順で持つ。これらは `SOT-ENG-024` が定めるカテゴリを成果物で識別する機械 ID であり、件数と受入条件は同 SOT を定義元とする。

| category ID | 対象 |
|---|---|
| `ambiguity` | 衝突する略称、複数候補、弱い一般語および三候補以上 |
| `budget-boundary` | 候補、step、呼出し、item および page の固定上限 |
| `capability-intent` | 採用済み task、resource および七つの logical input |
| `input-boundary` | 入力長、文字、参照および構造上の境界 |
| `law-name-and-concept` | 正式名、公式略称、出典付き別名および法概念 |
| `official-reference` | 法令 ID、リビジョン ID、法令番号、事件参照および資源参照 |
| `pack-state` | `judicial-cases` の有効時と無効時 |
| `safety-execution-boundary` | 禁止された外部呼出しを行わない安全境界 |
| `structured-location-and-date` | 完全な日付、条、項および複数の明示意図 |
| `surface-variation` | 表記揺れおよび空白差 |
| `typo-variation` | 挿入、削除、置換および隣接文字転置 |
| `unsupported-scope` | 法的助言、翻訳、未採用 pack および対象外 resource |

`coverageIds` は次の閉じた一覧だけを使用する。category は fixture に重複記録せず、この対応から導出する。一 case が同じ category の coverage を複数持っても、その category の件数には一回だけ数える。

| coverage ID | category ID | holdout 最小件数 |
|---|---|---:|
| `ambiguity-alias-collision` | `ambiguity` | `1` |
| `ambiguity-multiple-concepts` | `ambiguity` | `1` |
| `ambiguity-three-or-more-candidates` | `ambiguity` | `1` |
| `ambiguity-weak-general-term` | `ambiguity` | `1` |
| `boundary-budget-limit` | `budget-boundary` | `2` |
| `boundary-mixed-unsupported` | `safety-execution-boundary` | `2` |
| `boundary-no-implicit-first-read` | `safety-execution-boundary` | `2` |
| `boundary-non-japanese` | `safety-execution-boundary` | `2` |
| `boundary-pack-disabled` | `pack-state` | `2` |
| `budget-capability-call-limit` | `budget-boundary` | `1` |
| `budget-item-limit` | `budget-boundary` | `1` |
| `budget-page-limit` | `budget-boundary` | `1` |
| `budget-ranked-candidate-limit` | `budget-boundary` | `1` |
| `budget-step-limit` | `budget-boundary` | `1` |
| `concept-single` | `law-name-and-concept` | `1` |
| `input-invalid-ref` | `input-boundary` | `1` |
| `input-limit-above-maximum` | `input-boundary` | `1` |
| `input-limit-below-minimum` | `input-boundary` | `1` |
| `input-limit-maximum-accepted` | `input-boundary` | `1` |
| `input-limit-minimum-accepted` | `input-boundary` | `1` |
| `input-query-ascii-control` | `input-boundary` | `1` |
| `input-query-empty` | `input-boundary` | `1` |
| `input-query-maximum-accepted` | `input-boundary` | `1` |
| `input-query-too-long` | `input-boundary` | `1` |
| `intent-judicial-decision-read` | `capability-intent` | `1` |
| `intent-judicial-decision-search` | `capability-intent` | `1` |
| `intent-law-article-read` | `capability-intent` | `1` |
| `intent-law-content-search` | `capability-intent` | `1` |
| `intent-law-read` | `capability-intent` | `1` |
| `intent-law-search` | `capability-intent` | `1` |
| `intent-law-updates` | `capability-intent` | `1` |
| `name-official` | `law-name-and-concept` | `1` |
| `name-official-abbreviation` | `law-name-and-concept` | `1` |
| `name-sourced-alias` | `law-name-and-concept` | `1` |
| `pack-judicial-enabled` | `pack-state` | `1` |
| `reference-case-reference` | `official-reference` | `1` |
| `reference-law-id` | `official-reference` | `1` |
| `reference-law-number` | `official-reference` | `1` |
| `reference-revision-id` | `official-reference` | `1` |
| `reference-source-resource-ref` | `official-reference` | `1` |
| `structure-article` | `structured-location-and-date` | `1` |
| `structure-complete-date` | `structured-location-and-date` | `1` |
| `structure-multiple-explicit-intents` | `structured-location-and-date` | `1` |
| `structure-paragraph` | `structured-location-and-date` | `1` |
| `surface-orthographic-variation` | `surface-variation` | `1` |
| `surface-whitespace-variation` | `surface-variation` | `1` |
| `typo-adjacent-transposition` | `typo-variation` | `1` |
| `typo-deletion` | `typo-variation` | `1` |
| `typo-insertion` | `typo-variation` | `1` |
| `typo-substitution` | `typo-variation` | `1` |
| `unsupported-legal-advice` | `unsupported-scope` | `1` |
| `unsupported-resource` | `unsupported-scope` | `1` |
| `unsupported-translation` | `unsupported-scope` | `1` |
| `unsupported-unadopted-pack` | `unsupported-scope` | `1` |

各 coverage は表の holdout 最小件数以上を持ち、各 category は `SOT-ENG-024` の最小件数も満たす。

表で最小件数が二件の次の安全境界 coverage は `safetyVariant` を必須とし、holdout で `ordinary` と `adversarial` を一件以上ずつ持つ。

- `boundary-budget-limit`
- `boundary-mixed-unsupported`
- `boundary-no-implicit-first-read`
- `boundary-non-japanese`
- `boundary-pack-disabled`

それ以外の fixture は `safetyVariant` を持たない。

## execution case

`execution` の fixture は `artifactKind=execution_case` とし、次の項目だけを持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `artifactKind` | string | はい | `execution_case` |
| `schemaVersion` | integer | はい | `1` |
| `caseId` | string | はい | manifest と file 名に一致する ID |
| `scenarioIds` | `string[]` | はい | 一件以上の実行 scenario |
| `semanticCaseId` | string | はい | development の semantic case |
| `actions` | `ExecutionAction[]` | はい | fake capability の宣言的結果 |
| `expected` | object | はい | 実行結果の期待投影 |

manifest の `requiredExecutionScenarioIds` は次の八件を昇順で持つ。これらは `SOT-ENG-024` が定める実行 fixture を成果物で識別する機械 ID であり、必要な再現範囲は同 SOT を定義元とする。

- `execution-all-failed`
- `execution-empty`
- `execution-item-budget`
- `execution-mixed-composition`
- `execution-nonempty`
- `execution-partial-failure`
- `execution-reversed-completion`
- `execution-timeout`

`execution-mixed-composition` の採用前に固定した `corpus-v1` から
`corpus-v3` は、同 ID を除く従来の七件を履歴上の必須一覧として保持できる。
`corpus-v4` 以降は上記八件を必須とする。schema v1 は両配列だけを受理し、
loader は `corpusVersion` と対応しない配列を拒否する。

参照先 semantic case は `kind=plan` で、decision が `single` または `hedged` でなければならない。`actions` は `selectedMeaningIds` の順と各 meaning の step 順からなる plan 順で保持し、選択した全 step を正確に一回ずつ参照する。

`ExecutionAction` は `meaningId`、一から始まる `stepOrdinal`、一から始まる `releaseOrder` および `outcome` を持つ。全 action の `releaseOrder` は一から action 件数までの順列とし、実時間の遅延ではなく fake clock が action の終端 event を解放する決定的な順序を表す。同じ meaning 内では `releaseOrder` が step 順を逆転させず、異なる meaning 間だけを逆転できる。

`outcome` は次の閉じた variant の一つとする。

| `kind` | 追加項目 |
|---|---|
| `collection_success` | `sourceItemCount`。零以上千以下 |
| `read_success` | なし |
| `failure` | `SOT-MODEL-024` が failed attempt に許可する `errorCode` |
| `timeout` | なし |

`collection_success` は `search` または `list_updates` step だけ、`read_success` は `read` step だけに使用する。`timeout` は fake clock の deadline event であり、製品 attempt では `outcome=failed` と `errorCode=source_timeout` に投影する。

execution fixture は provider の外部 response、接続先、認証情報または実時間待機を持たない。

`expected` は `terminal` で区別する次の閉じた variant の一つとする。

| `terminal` | 項目 |
|---|---|
| `result` | `status`、`returnedItemCount`、`attempts` |
| `error` | `errorCode`、`attempts` |

`terminal=result` の `status` は `completed`、`empty` または `partial`、`returnedItemCount` は零以上四十以下とする。`terminal=error` は全 action が失敗または timeout である場合だけ使用し、`errorCode` は plan 順で最初の attempt の公開 error code とする。

期待 attempt は action と同じ plan 順で、`meaningId`、`stepOrdinal`、`outcome` および outcome ごとの項目を持つ。

| `outcome` | 追加項目 |
|---|---|
| `completed` | `publishedItemCount`。read は一、collection は一以上二十以下。collection は `hasMore` も必須 |
| `empty` | `publishedItemCount=0` と `hasMore=false`。collection だけ |
| `failed` | `errorCode`。item 件数と `hasMore` は持たない |

collection の `publishedItemCount` は plan の `effectiveLimit` と残りの全体 item 予算を超えず、`hasMore` は `sourceItemCount > publishedItemCount` と一致する。result の `returnedItemCount` は全成功 attempt の `publishedItemCount` 合計と一致する。製品結果全体の JSON を golden として複製しない。

scenario ID は次の構造条件を満たす。

- `execution-nonempty`: 一件以上の成功 attempt が一 item 以上を公開し、terminal は `result`
- `execution-empty`: 全 action が零件の `collection_success` で、status は `empty`
- `execution-partial-failure`: 一件以上の成功と一件以上の failure または timeout があり、status は `partial`
- `execution-all-failed`: 全 action が failure または timeout で、terminal は `error`
- `execution-timeout`: 一件以上の timeout が `failed/source_timeout` へ投影される
- `execution-reversed-completion`: `hedged` の異なる meaning 間で release 順と plan 順が異なるが、期待 attempt は plan 順を保つ
- `execution-item-budget`: 一件以上の collection が `effectiveLimit` を超える source item を返し、attempt 上限、四十 item 上限、一 page および `hasMore` を保つ
- `execution-mixed-composition`: core と pack の混合 meaning について、plan 順、
  required pack、read と collection の混在、および pack 有効時だけの実行
  attempt を再現する

## 評価器の派生観測

評価器は holdout fixture の `coverageIds` を増やさず、既存の期待値から次の派生
観測を決定的に計算できる。

- `composition-core-pack`: `selectedMeaningIds` が core と pack の step を同じ
  meaning に持つ
- `composition-pack-disabled`: 同じ meaning が `requiredPacks` 非空かつ
  `decision=capability_unavailable` を持つ
- `composition-ref-read-search`: 同じ meaning が `judicial_decision` read と
  検索 step を併せ持つ
- `composition-four-step-budget`: 同じ meaning が四 step を持ち
  `decision=single` または `hedged` である

これらは fixture file に新しい coverage ID として保存せず、評価器と baseline
が既存の期待 meaning から導出して確認する。`same-position-tiebreak` および
`invalid-member-origin` のように holdout の意味署名へ現れない性質は
派生観測に含めず、model test または architecture test の責務とする。

## 実装境界

成果物型と loader は `internal/legalquerycorpus` が所有し、入口を `Load(ctx context.Context, repositoryRoot, corpusDirectory string) (Corpus, error)` とする。`Corpus` は development、holdout および execution を manifest 順に返す。

この package は Go 標準 package、JSON Schema validator、`internal/model`、`internal/application/legalquery` および `internal/querynormalization` にだけ依存できる。evaluator、query profile、executor、provider、`internal/source/...` または MCP SDK を import しない。

## file と JSON の安全境界

loader は repository root と corpus directory を絶対 path に解決する。corpus directory は、repository 内の `testdata/legalquery/corpus-vN` 正規形に一致する directory だけを許可する。相対 path と repository 内の同じ対象を指す絶対 path は許可するが、repository 外、別の subtree、`..` による脱出および正規形でない directory 名を拒否する。repository root から corpus directory までの各構成要素、schema、manifest、三集合 directory および全 fixture は `Lstat` 相当で symlink を拒否する。

corpus root は `manifest.json`、`development`、`holdout` および `execution` 以外を持たない。各集合 directory は manifest に登録した直下の `.json` 通常 file だけを持ち、subdirectory、symlink、device、FIFO、未登録 file および未知の entry を拒否する。fixture access は corpus root に閉じた OS の root API を用い、検証済み `caseId` から導出した相対 path だけを開く。

loader は解決済み repository root を OS の root API で開く。manifest の JSON 安全境界を先に確認し、最上位の `artifactKind` と整数の `schemaVersion` だけを bootstrap scan で抽出する。他の値は schema と typed decode が完了するまで信頼しない。対応する実装済み version である場合だけ固定相対 path `testdata/legalquery/schemas/legal-query-corpus-v{schemaVersion}.schema.json` を開く。corpus directory は repository root の child root として開き、manifest は固定名 `manifest.json`、集合 directory は固定した三名称、fixture は検証済み `caseId` から導出した相対 path だけを開く。開いた各 file descriptor に対してもう一度通常 file であることと size を確認してから読む。

資源上限は次とする。

| 対象 | 上限 |
|---|---:|
| schema | `1 MiB` |
| manifest | `2 MiB` |
| fixture 一件 | `256 KiB` |
| fixture 総数 | `4096` |
| corpus fixture 原 byte 合計 | `64 MiB` |
| JSON nesting depth | `16` |
| 一 JSON document の value 数 | `100000` |
| 一 case の coverage ID 数 | `64` |

directory は固定件数ごとに列挙し、未登録 file と未知 entry を含む列挙数が fixture 総数上限を超えた時点で失敗する。file は通常 file と宣言 size を確認した後、上限に一 byte を加えた reader で一度だけ読み、同じ byte 列で checksum と decode を行う。時刻に依存する parse timeout は設けず、各 file の間で呼出し元 context の cancellation を確認する。

各 JSON document は、typed decode より前に次を検証する。

- 有効な UTF-8
- 最上位が一つの object
- depth と value 数の固定上限
- すべての object 階層で key が重複しないこと
- root の後が空白と EOF だけであること

その後、JSON Schema と未知の項目を拒否する typed decoder の両方で検証する。許可していない `null`、未知の enum、条件に合わない variant および trailing value を拒否する。

## loader の整合検証

loader は少なくとも次の順序で検証し、最初の決定的なエラーを返す。

1. repository、corpus、manifest および version から選んだ schema の path と file 種別
2. root と三集合の未知 entry、件数および file 予算
3. manifest、schema および `holdoutDigest` の JSON 安全境界と構造
4. fixture の JSON 安全境界、checksum、schema および typed 構造
5. manifest、file 名、集合、schema version および `caseId` の一致
6. 全集合の ID と checksum の一意性、配列順および参照整合
7. semantic expectation の request error、decision、reason、meaning、selection および logical input
8. development と holdout の完全 request、比較キーおよび leakage group の分離
9. holdout の件数、coverage から導出したカテゴリおよび safety pair
10. execution の参照、action、release 順、期待値および scenario coverage

development と holdout の分離は、次の三条件をすべて確認する。

- `query`、`ref` の有無と全項目、および `limitPerAttempt` の有無と値からなる完全 request が一致しない
- 外側の Unicode White_Space を除いた `query` に共通 `querynormalization.ComparisonKey` を適用した値が一致しない
- 同じ `leakageGroupId` が両集合に存在しない

比較キーが空でも一つの値として扱う。Kagome、辞書、誤記補正または profile は集合分離に使用しない。execution は query と leakage group の集合分離から除外するが、全体の `caseId` と checksum の一意性には含める。

境界違反を意図した request は loader で製品 request として拒否せず、`kind=request_error` の期待値と整合することを確認する。`kind=plan` の request と、期待する logical input、参照、日付、位置、decision および reason は対応する製品 constructor と SOT の制約へ適合しなければならない。

一つでも検証に失敗した場合は部分的な corpus を返さない。成功時に返す corpus、manifest、case、期待値、配列、map および logical input は外部から変更できない不変値とし、getter は深い複製を返す。保存用の mutation API、`map[string]any` および共有する `json.RawMessage` を公開しない。

## version の変更

JSON の項目、discriminator、enum、variant または機械カテゴリ契約を変更するときは `schemaVersion` を増やし、新しい version 別 schema と typed decoder を追加する。既存 corpus を再現できるように、公開済み schema file と decoder を同じ意味のまま保持する。loader は実装済み version の固定一覧だけを受理し、未知の schema version を path として任意解決しない。

query、期待意味、集合、カテゴリ、seed または評価上の fixture 内容を変更するときは新しい `corpus-vN` を作る。

同じ意味と入力の case を次の corpus version へ移す場合は `caseId` を維持できる。入力または期待意味を変える場合は新しい `caseId` を割り当てる。意味を変えない整形だけの変更は同じ corpus version で checksum を更新できるが、機械 formatter で byte 表現を統一し、独立 review を受ける。

holdout の期待値を変える場合は、実装へ合わせるためではなく fixture の誤りであることを独立 review で確認し、理由、新しい corpus version、holdout digest および変更前後の評価結果を同じ変更へ残す。

## 確認

JSON Schema test と loader test で、三 artifact variant、`plan` と `request_error` の正常系、未知項目、重複 key、trailing value、不正 UTF-8、depth・value・file・件数・合計 byte 上限、path traversal、repository 外または別 subtree の絶対 path、区切り文字、各階層の symlink、特殊 file、未知 entry、外部 `$ref`、未知 schema version、checksum、holdout digest、manifest 不一致、全体 ID 重複、並び順、development と holdout の完全 request・比較キー・leakage group 衝突、coverage とカテゴリ最小件数、safety pair、execution の全 step 参照・release 順列・step/outcome 対応・scenario 条件および getter の深い複製を確認する。

race detector で同じ corpus の並行読取りが共有状態を変更しないことを確認する。検証は外部ネットワークへ接続せず、失敗時に照会本文、fixture 全体または認証情報をエラーへ含めない。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-MODEL-024: LegalQueryResult](../20-model/24-legal-query-result.md)
- [SOT-IF-051: MCP `query_legal_information`](../40-interfaces/51-mcp-query-legal-information.md)
- [SOT-ENG-004: SOT に結び付く検証](04-sot-linked-verification.md)
- [SOT-ENG-019: 静的解析とコーディングスタイル](19-static-analysis-and-coding-style.md)
- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-025: 統合照会のパッケージ構成](25-unified-query-package-layout.md)
