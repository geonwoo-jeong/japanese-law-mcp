# SOT-ENG-036: 統合照会の評価 baseline 成果物契約

- 状態: 有効

## 規定

統合照会の評価 baseline は、固定 corpus と固定 profile set に対する決定的な
評価結果を保持する、版付きで閉じた JSON 成果物とする。標準評価 command は
baseline を書き換えず、同じ入力から生成した report をこの契約へ投影して
完全一致と受入基準を検証する。

## 適用範囲

本規定は、baseline と評価 report が共有する JSON 構造、項目順、metric の
計算規則、loader の安全境界、version file の不変性および決定的な byte 表現を
定義する。

評価 corpus と fixture は `SOT-ENG-026`、指標の意味と受入値は `SOT-ENG-024`、
採用 tuple、digest および rollback は `SOT-ENG-033` を定義元とする。本規定は
それらの値または導入順を重複して定義しない。

本 baseline は、意味 plan、selection、根拠、実行 attempt、公開 execution status、
件数および固定予算を、本規定の field へ投影した範囲だけを証明する。provider の
raw JSON/XML をどの error code へ分類したか、公開 tool error の詳細、
`LawSummary` item の identity、page 内の完全順または parser と facade が同じ
raw byte を受理したかは投影しない。これらは対応する provider、interface または
application の固定 fixture を定義元とし、baseline が同じであることを代用証拠に
しない。

## 配置

成果物は次の配置とする。

```text
testdata/legalquery/
├── schemas/
│   └── legal-query-baseline-v1.schema.json
└── baselines/
    ├── default.json
    └── versions/
        └── {baselineVersion}.json
```

schema は JSON Schema Draft 2020-12 とし、すべての object を閉じる。
`$ref` は同じ schema 内の fragment だけを参照し、network、`file:` URL または
外部 schema を解決しない。

`default.json` は current adoption tuple が選ぶ既定 baseline である。
`versions/{baselineVersion}.json` は一度追加した byte を変更しない版付き成果物とする。
採用中の version file、`default.json` および digest の一致条件は
`SOT-ENG-033` を定義元とする。

## 初回 bootstrap と候補準備

`SOT-ENG-033` の初回 adoption manifest を導入する変更では、本規定の schema、
閉じた loader および現行 report の決定的な直列化を先に有効にする。同じ変更で、
現行 `default.json` と byte が完全に同じ
`versions/{baselineVersion}.json` を exclusive create し、現行 corpus、
production profile set、評価値および検索例を変更せず、最初の history manifest と
current tuple へ結び付ける。

この初回 bootstrap は、次版 baseline 候補の生成を許可するものではない。次版の
候補 profile set を直接構成する内部評価入口、次の `baselineVersion` および候補
writer は、`SOT-ENG-034` 第 4 段階の後続番号で別に準備する。候補 writer を
標準 command、中央品質ゲート、CLI、設定、MCP または transport から呼び出せる
ようにしない。

## top-level object

baseline と標準 report は、次の項目だけをこの順で持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `artifactKind` | string | はい | `legal_query_evaluation` |
| `schemaVersion` | integer | はい | 現行構造は `1` |
| `corpusVersion` | string | はい | 評価した corpus |
| `holdoutDigest` | string | はい | corpus manifest の holdout digest |
| `profileSet` | object | はい | 評価した固定 profile set |
| `baselineVersion` | string | はい | `default-` と正の十進整数 |
| `sets` | object | はい | `development`、`holdout`、`execution` |

`baselineVersion` は正規表現 `default-[1-9][0-9]*` に一致させる。

標準 command が読む `default.json` と、その command が生成する標準 report の
`corpusVersion`、`holdoutDigest`、`profileSet` および `baselineVersion` は、
`current.json` が指す current adoption tuple と完全一致させる。

採用済みまたは過去の各 `versions/{baselineVersion}.json` は、自身が評価した
corpus、holdout digest、固定 profile set、baseline version および原 byte digest を
持つ一つの `SOT-ENG-033` history manifest と完全一致させる。version file を現行
current tuple へ無条件に一致させず、過去の file は採用当時の history manifest と
一致し続ける。

採用前候補の version file は、まだ history manifest を持たず、自身の field と
準備済み target corpus、holdout digest、test 専用固定 profile set および予約した
baseline version を完全一致させる。原子的採用と同じ変更で初めて、同 file の
digest を持つ history manifest を追加する。候補または過去の version file を
標準 command から選択できるようにしない。

## `profileSet`

`profileSet` は次の項目だけをこの順で持つ。

| 項目 | 型 | 必須 |
|---|---|---:|
| `profileSetId` | string | はい |
| `profileSetVersion` | string | はい |
| `rankingVersion` | string | はい |
| `profiles` | object[] | はい |

`profiles` の各 object は `profileId`、`profileVersion` だけをこの順で持つ。
配列は評価対象の composition root が構成する固定 profile 順と一致させる。
配列は一件以上十六件以下とし、各 ID と version は `SOT-ENG-033` の採用 tuple の
正規形と上限を満たす。
current baseline と標準 report では production composition root、採用前候補では
test が直接構成した target composition root の順を使う。loader または評価器が
profile ID 順へ並べ替えてはならない。

`profileSetId`、`profileSetVersion`、`rankingVersion` および各 profile version は、
評価対象の composition root と profile metadata に完全一致させる。採用済みまたは
過去の version file は、さらに同 baseline version に対応する history manifest と
一致させる。current baseline と標準 report は production と current adoption tuple
にも一致させる。

## `sets`

### `development`

`development` は `caseCount` だけを持つ。値は corpus manifest の development
件数と一致する一以上の整数とする。個々の開発用結果や query は baseline に
保存しない。

### `holdout`

`holdout` は次の項目だけをこの順で持つ。

| 項目 | 型 | 必須 |
|---|---|---:|
| `caseCount` | integer | はい |
| `metrics` | metric[] | はい |
| `categories` | category[] | はい |
| `derivedObservations` | metric[] | はい |
| `failedCaseIds` | string[] | はい |

`caseCount` は corpus manifest の holdout 件数と一致する一以上の整数とする。
`metrics` は次の
metric ID をこの順で一件ずつ持つ。

1. `plan-reproducibility`
2. `plan-outcome`
3. `request-error`
4. `meaning-signature`
5. `top-1`
6. `top-2`
7. `high-confidence-precision`
8. `evidence-assertion`
9. `concept-assertion`

`categories` は corpus manifest の `requiredCategoryIds` と同じ ID を昇順で
一件ずつ持つ。各 category object は `categoryId`、`caseCount`、`metrics` だけを
この順で持ち、`metrics` は上記から `plan-reproducibility` を除いた八件を
同じ順で持つ。

`derivedObservations` は次の metric ID をこの順で一件ずつ持つ。

1. `composition-core-pack`
2. `composition-pack-disabled`
3. `composition-ref-read-search`
4. `composition-four-step-budget`

### `execution`

`execution` は次の項目だけをこの順で持つ。

| 項目 | 型 | 必須 |
|---|---|---:|
| `caseCount` | integer | はい |
| `metrics` | metric[] | はい |
| `wrongResourceCallCount` | integer | はい |
| `budgetViolationCount` | integer | はい |
| `attemptOrderViolationCount` | integer | はい |
| `implicitFirstReadCount` | integer | はい |
| `emptyReclassificationCount` | integer | はい |
| `failedCaseIds` | string[] | はい |

`caseCount` は corpus manifest の execution 件数と一致する一以上の整数とする。
五つの違反件数は零以上とする。各 execution metric の `denominator` は
`caseCount` と一致させる。`metrics` は次の ID をこの順で一件ずつ持つ。

1. `expected-execution`
2. `no-wrong-resource-call`
3. `budget-adherence`
4. `attempt-order-determinism`
5. `no-implicit-first-read`
6. `no-empty-reclassification`

## metric と失敗 ID

metric object は `metricId`、`numerator`、`denominator`、`ratio`、
`failedCaseIds` だけをこの順で持つ。

- `numerator` と `denominator` は零以上の整数とし、`numerator` は
  `denominator` 以下とする
- `denominator` が正の場合、`ratio` は `numerator / denominator` の
  IEEE 754 倍精度値と完全一致させる
- `denominator` が零の場合、`numerator` と `ratio` は零とする
- `failedCaseIds` は、その metric の母集団で失敗した case ID を
  corpus manifest の評価順に、重複なく持つ
- `failedCaseIds` の件数は `denominator - numerator` と一致させる

holdout と execution の top-level `failedCaseIds` は、各集合で一件以上の
失敗を持つ case ID の和集合を、corpus manifest の評価順に重複なく持つ。
category の metric は、その category に属する case だけを母集団とする。

`derivedObservations` の分母は一以上とする。通常の metric で分母が零になることは
許容するが、`SOT-ENG-024` が分母零を不合格とする指標の受入判定を変えない。

## JSON、loader および writer の安全境界

baseline loader は、一 byte 以上四 MiB 以下の regular file を固定 repository
path から読む。symbolic link、repository 外への path 解決、不正 UTF-8、重複 key、
`null`、未知項目、二個目の JSON 値および後方の非空白 token を拒否する。
JSON depth は `12`、一 report の value 数は `65536` を上限とする。

`baselines/` は `default.json` と `versions/` だけを持ち、`versions/` は正規形の
`{baselineVersion}.json` 通常 file だけを持つ。subdirectory、symlink、device、
FIFO、socket および未知 entry を拒否する。version file は `4096` 件以下、
原 byte 合計 `256 MiB` 以下とし、`default.json` はこの合計に含めず個別の
四 MiB 上限を適用する。directory 列挙中に件数または宣言 size の合計が上限を
超えた時点で失敗し、不正な一件を無視して部分 history を返さない。

report 上限は、`SOT-ENG-026` の holdout 四百件以下と execution 四百件以下から
次のように導出する。十二 category、holdout の九 metric、category ごとの八 metric、
四 derived observation と top-level 失敗 ID により、holdout の同一 case ID は
最大百十回現れる。execution の六 metric と top-level 失敗 ID により、
execution の同一 case ID は最大七回現れる。したがって全 case が全対象 metric で
失敗する最大投影でも、case ID の出現は
`400 * 110 + 400 * 7 = 46800` 件以下である。

case ID は最大六十四 ASCII byte であり、JSON の引用符と区切りを含め一出現
六十七 byte 以下であるため、全失敗 ID の byte は `3135600` 以下となる。
閉じた field 名、十二 category、百十五個以下の metric object、十六件以下の
profile、整数、ratio、container および固定識別子からなる非 case-ID 投影は、
`262144` byte と `4096` values を上限とする。writer と loader はこの二つの
部分上限も別に検証する。これにより、正当な最大 report は
`3135600 + 262144 = 3397744` byte 以下、`46800 + 4096 = 50896` values 以下となり、
全体の四 MiB と `65536` values の上限内に収まる。

将来 schema が category、metric、profile または集合別 fixture 上限を増やす場合は、
正当な最大 report を拒否しない新しい baseline schema version、部分上限および
全体資源上限を同じ変更で採用する。

標準 report は file から decode せず、固定 evaluator が不変な typed report として
memory 上に構築し、検証後に一回だけ JSON byte へ直列化する。標準 command は
その byte を stdout へ出力し、baseline file を作成または変更しない。

第 4 段階の候補 evaluator は、合否にかかわらず同じ typed report を本規定へ
一回だけ直列化する。保存先、result、logical one-time use および CI handoff は
`SOT-ENG-037` を定義元とする。候補 baseline writer は
`outcome=passed` の同じ report byte だけを受け取り、request が予約した
`baselines/versions/{baselineVersion}.json` へ exclusive create する。
`outcome=failed` の report を baseline path へ置かない。既存 file、symlink、
非 regular file、repository 外 path または四 MiB を超える byte を拒否し、
別の直列化や再計算で byte を変えない。

object の key 順と本規定で固定した配列順を検証し、不正な値を並べ替え、丸め、
補完または既定値へ置換しない。schema validation と意味上の相互参照検証の
どちらか一方だけで済ませない。loader、schema および evaluator は外部 network、
時刻、乱数または provider response に依存しない。

標準 report の JSON object は baseline と同じ field と順序を使う。
command は query 本文、辞書 entry、外部 response、file system の絶対 path、
時刻、処理時間、host 情報または個人情報を出力しない。

## version、byte 一致および変更

構造、field の意味、型、必須性または固定 metric 集合を変える場合は新しい
schema version と schema file を追加し、履歴 baseline の再現に必要な旧 decoder を
保持する。

評価値、corpus、profile set または profile version を変える場合は、新しい
`baselineVersion` の version file を追加する。既存 version file を上書き、
移動、削除または再生成しない。

version file、`default.json`、`baselineSha256`、current tuple および rollback の
byte 一致と切替単位は `SOT-ENG-033` に従う。本規定の loader は、その検査に必要な
閉じた構造と決定的な byte 表現を保証する。

## 確認

外部 network を使わない schema、loader、evaluator および採用 manifest の
契約 test で、少なくとも次の固定 test ID を確認する。

- `evaluation-baseline-closed-json`: 空入力、上限超過、symlink、不正 UTF-8、
  重複 key、`null`、未知項目、二個目の値および後方 token を拒否する
- `evaluation-baseline-resource-maximum`: holdout 四百件と execution 四百件が
  全対象 metric で失敗する正当な最大 report を受理し、case-ID 部分、
  非 case-ID 部分、四 MiB、`65536` values、version file 件数および
  history 合計 byte の各上限を一単位超える成果物を拒否する
- `evaluation-baseline-history-bounds`: `versions/` の未知 entry、subdirectory、
  symlink、特殊 file、`4096` 件超過および `256 MiB` 超過を拒否し、
  不正な一件を除いた部分 history を返さない
- `evaluation-baseline-initial-bootstrap`: 現行 `default.json` と最初の version
  file の byte が一致し、初回 history manifest、current tuple、標準 report および
  adoption 基準 command が同じ corpus と production profile set を指す。次版候補
  writer または候補 profile set は標準経路から到達できない
- `evaluation-baseline-candidate-isolation`: 候補 version file は準備済み target
  corpus、profile set、`SOT-ENG-037` の passed result および予約版に一致し、
  採用前に history manifest、current tuple、`default.json` または標準 command
  から到達できない。failed report は baseline path に存在しない
- `evaluation-baseline-structure`: top-level、profile set、三集合、category、
  metric および違反件数の項目と固定順を検証する
- `evaluation-baseline-metric-order`: holdout、category、derived observation
  および execution の metric ID が閉じた完全順と一致する
- `evaluation-baseline-metric-arithmetic`: 分子、分母、割合、失敗 ID 件数、
  分母零および derived observation の分母を検証する
- `evaluation-baseline-corpus-identity`: corpus version、holdout digest、
  集合件数、category ID および case ID が corpus manifest と一致する
- `evaluation-baseline-profile-set-identity`: production 固定順、
  profile set version、ranking version および各 profile version が一致する
- `evaluation-baseline-version-identity`: baseline version が正規形であり、
  current tuple が指す version file と `default.json` の byte および採用 digest が
  一致する。候補と過去版には `default.json` との一致を要求しない
- `evaluation-baseline-output-privacy`: query、辞書 entry、外部 response、
  絶対 path、時刻、host 情報および個人情報を出力しない

## 関連

- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](../30-architecture/33-unified-query-profile-set-adoption-boundary.md)
- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-026: 統合照会の評価コーパス成果物契約](26-legal-query-corpus-artifact-contract.md)
- [SOT-ENG-029: 統合照会の検索例カタログ](29-unified-query-example-catalog.md)
- [SOT-ENG-033: 統合照会 profile set 採用 manifest](33-unified-query-profile-set-adoption-manifest.md)
- [SOT-ENG-034: 統合照会の意味判定変更における導入段階と変更順序](34-unified-query-rollout-stages.md)
- [SOT-ENG-037: 統合照会の候補 holdout 評価 handoff](37-unified-query-candidate-evaluation-handoff.md)
