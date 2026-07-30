# SOT-ENG-033: 統合照会 profile set 採用 manifest

- 状態: 有効

## 規定

統合照会の production、標準評価、baseline および検索例カタログが同じ
採用済み profile set を指すことを、repository 内の版付きで閉じた採用 manifest
により一つの tuple として検証する。

## 境界

採用と rollback の原子性、準備状態および production からの到達性は
`SOT-ARCH-033` を定義元とする。本規定は、その最終状態を機械検査する
成果物、tuple および固定 test ID だけを定義する。

採用 manifest は build、test、文書検査および release 前検査の入力であり、
利用者が profile set を選ぶ実行時設定ではない。production composition root は
manifest の任意 path、過去の `adoptionId` または未知の tuple を読み込まず、
一つの組込み既定だけを構成する。

## 配置

採用成果物は次へ置く。

```text
testdata/legalquery/adoptions/
├── schema-v1.json
├── current.json
└── history/
    └── {adoptionId}.json
```

baseline は次へ置く。

```text
testdata/legalquery/baselines/
├── default.json
└── versions/
    └── {baselineVersion}.json
```

`schema-v1.json` は JSON Schema Draft 2020-12 の閉じた schema とし、
network、`file:` URL または外部 schema を解決しない。

`current.json` は `artifactKind=legal_query_adoption_pointer`、
`schemaVersion=1` および `adoptionId` だけを持つ。loader は `adoptionId` から
`history/{adoptionId}.json` だけを導出し、任意 path を受け取らない。

history manifest は一度採用した byte を変更しない。訂正または新規採用は新しい
`adoptionId` の manifest を追加し、`current.json` を一回だけ切り替える。

## 採用 tuple

history manifest は次の項目だけを持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `artifactKind` | string | はい | `legal_query_adoption` |
| `schemaVersion` | integer | はい | `1` |
| `adoptionId` | string | はい | 後述の tuple digest |
| `previousAdoptionId` | string | いいえ | 直前の採用 manifest。初回だけ省略 |
| `profileSetId` | string | はい | production と標準評価の固定 set ID |
| `profileSetVersion` | string | はい | composition root が構成する不透明な set version |
| `rankingVersion` | string | はい | 全 profile が共有する校正版 |
| `compositionVersion` | string | はい | 採用する composer 規則の版 |
| `profiles` | object[] | はい | production の固定順に並べた profile tuple |
| `corpusVersion` | string | はい | 現行標準 corpus |
| `holdoutDigest` | string | はい | corpus manifest の holdout digest |
| `baselineVersion` | string | はい | 現行 baseline の版 |
| `baselineSha256` | string | はい | baseline file の原 byte SHA-256 |
| `catalogVersion` | string | はい | 現行検索例カタログの版 |
| `catalogSha256` | string | はい | 後述のカタログ digest |

各 `profiles` 要素は `profileId`、`profileVersion` および `cueSetVersion`
だけを持つ。profile ID の辞書順ではなく、production composition root が
候補 contribution を構成する固定順に並べ、重複を許さない。

`adoptionId` は `adoption-sha256-` と小文字十六進六十四桁を連結した値とする。
suffix は、`adoptionId` 自身を除く field を次の完全順序と符号化で直列化した
byte 列の SHA-256 とする。

```text
artifactKind
schemaVersion
previousAdoptionId
profileSetId
profileSetVersion
rankingVersion
compositionVersion
profiles
corpusVersion
holdoutDigest
baselineVersion
baselineSha256
catalogVersion
catalogSha256
```

- string は `s`、先頭零なしの十進 ASCII にした UTF-8 byte 長、`:`、原 UTF-8
  byte、`;` の順にする
- integer は `i`、先頭零なし十進 ASCII の byte 長、`:`、その十進 ASCII、
  `;` の順にする
- 省略した `previousAdoptionId` は専用の二 byte `m;` とし、空 string と区別する
- array は `a`、要素数の先頭零なし十進 ASCII、`[`、各要素、`]` の順にする
- object は `o`、field 数の先頭零なし十進 ASCII、`{`、schema field 順の
  field 名を string として符号化した値、field 値、`}` の順にする
- `profiles` の各 object は `profileId`、`profileVersion`、`cueSetVersion` の
  順に三 field を符号化する

bool、浮動小数、`null` または未知の型を canonical 入力へ許可しない。loader は
不正な field 順、配列順または `adoptionId` を整列若しくは補正せず拒否する。

`baselineSha256` は標準 command が読む `baselines/default.json` の原 byteに
対する小文字十六進六十四桁とする。`baselineVersion` は
`default-` と先頭が零ではない十進整数を連結した正規形とし、同じ byte の
`baselines/versions/{baselineVersion}.json` が存在しなければならない。

version file は一度追加した byte を変更しない。baseline の JSON 項目または
byte を変更する場合は新しい baseline version file を追加する。原子的な採用では、
採用する version file と byte が完全に同じ `default.json` へ切り替え、
version file 自体を移動、削除または再生成しない。rollback も previous tuple の
version file と同じ byte を `default.json` へ戻す。

`catalogSha256` は `docs/unified-query-examples/` 直下の現行 Markdown file を
相対 path の byte 昇順に並べ、各 file について
`relativePath`、ASCII space 一文字、file の SHA-256、LF 一文字を連結した
byte 列の SHA-256 とする。`history/` が存在する場合は対象から除外する。

## 完全一致

採用 manifest の tuple は次と完全一致しなければならない。

- `profileSetId` と `profileSetVersion`: production composition root と
  標準評価 command が構成した値
- `profileSetVersion`: production が作る全 `LegalQueryPlan.profileVersion` と
  baseline の `profileSet.profileSetVersion`
- `rankingVersion`: 全 profile metadata、構築済み profile set および baseline
- `compositionVersion`: production と標準評価が注入する composer
- `profiles`: 固定 profile 順と各 profile metadata の `profileVersion`、
  `cueSetVersion`。baseline の profile 一覧とは `profileId` と
  `profileVersion` が同じ順で一致する
- `corpusVersion` と `holdoutDigest`: corpus manifest、標準評価 report および
  baseline
- `baselineVersion` と `baselineSha256`: 標準 command が実際に読んだ file
- `catalogVersion`: 検索例カタログの `00-index.md`
- `catalogSha256`: 現行カタログの全 Markdown file

カタログの `00-index.md` が宣言する corpus version と baseline version も
manifest と一致し、各 `verification_artifact` はその corpus に実在しなければ
ならない。これによりカタログへ `profileSetVersion` を重複記載せず、
採用 manifest、baseline および corpus を介して同じ set へ結び付ける。

## 準備状態

次版の profile、corpus、baseline、catalog または history manifest を
repository に準備できるが、`current.json` は現行の採用 manifest を指し続ける。
準備中の history manifest は `adoptionId` の構造と参照先候補を検証できるが、
現行採用済み、rollback 先または production から選択可能な set とみなさない。

production が使用する active profile metadata、cue artifact、profile set version、
baseline file、`baselineVersion`、`baselineSha256` および `current.json` は
準備変更で書き換えない。次版の metadata と cue artifact は test が直接構成する
別の profile set に置き、新しい version を割り当てる。

次版 baseline は、test がその別 profile set を直接構成する内部評価入口から生成し、
`baselines/versions/{baselineVersion}.json` に新しい version と digest を持つ
候補成果物として追加する。標準 command の現行 `default`、CLI、環境変数、設定、
build tag または hidden profile-set option から次版 set を選択できるようにしない。

原子的な採用時は、準備済み baseline の byte と digest を書き換えず、
production へ採用した profile set の評価結果と完全一致することを確認してから
current tuple へ含める。production が使用する profile version、cue set version
または profile set version を変更する場合は、観測結果が同じでも新しい
baseline version、digest および adoption tuple を同じ採用変更で切り替える。

## 初回導入

採用 manifest を初めて導入する準備変更では、その時点で production、標準 command、
中央品質ゲートおよび検索例カタログが使用している現行集合から、最初の history
manifest を作る。この初回 manifest だけは `previousAdoptionId` を省略し、
`current.json` はその `adoptionId` を指す。

初回 manifest の導入前後で profile、corpus、baseline、標準 command、
検索例カタログおよび production の観測動作を変更しない。現行
`default.json` と同じ byte を持つ最初の version file も追加する。次版の候補
manifest を同じ準備変更へ追加できるが、`current.json` は現行集合の初回
manifest を指し続ける。

次版を初めて採用する変更では、その候補 manifest の `previousAdoptionId` を
初回 manifest の `adoptionId` とし、production と全採用要素を切り替える同じ変更で
`current.json` を候補 manifest へ切り替える。これにより、manifest 導入前から
現行であった集合も機械的な rollback 先として保持する。

## rollback

初回 manifest 以外の新しい history manifest は、直前の `current.json` が指した
`adoptionId` を `previousAdoptionId` に持つ。rollback 変更では、production
composition root と `SOT-ARCH-033` の全採用要素をその previous tuple へ戻し、
`current.json` も同じ `previousAdoptionId` へ戻す。

過去 manifest が存在することを、production から過去 set を選択できる入口に
しない。rollback 後の repository は、戻した tuple に対して本規定の完全一致検査を
再実行し、採用前から存在した準備要素を残す場合は `SOT-ARCH-033` の非使用条件も
再確認する。

## 外部呼出し境界の比較

準備変更の前後で production 動作が同じことを確認する場合、provider ID や通信
完了順ではなく、plan 順の次の投影を比較する。

```text
{capabilityId, stepOrdinal, normalizedLogicalInput, callCount}
```

`normalizedLogicalInput` は各 logical input constructor が保持する正規形だけを
field 順に投影し、外部 response、route、時刻または乱数を含めない。

## 確認

外部ネットワークを使わない契約 test と文書検査で、少なくとも次の固定 test ID を
確認する。

- `profile-set-atomic-adoption`: current tuple と production、corpus、baseline、
  catalog の全項目および digest が一致する
- `profile-set-initial-adoption-bootstrap`: manifest 導入前後の観測動作を
  変えず、現行集合の初回 manifest だけが `previousAdoptionId` を省略し、
  `current.json` がその manifest を指す
- `profile-set-production-evaluation-identity`: production plan の
  `profileVersion`、固定 profile 順および外部呼出し境界の投影が標準評価と一致する
- `profile-set-rollback-tuple`: current と previous の fixture を使い、
  一項目だけを戻す部分 rollback、存在しない previous および tuple 混在を拒否する
- `profile-set-pack-transport-invariance`: stdio、HTTP、pack 有効および無効で
  採用 tuple と意味認識 contribution が変わらず、pack 状態は実行可否だけに影響する
- `profile-set-preparation-unreachable`: 準備中の adoption ID を CLI、環境変数、
  設定、MCP 引数、transport または hidden tool から選べない
- `profile-set-candidate-baseline-isolation`: 次版 baseline を test が直接構成する
  set だけで version file へ生成し、現行標準 command から選べず、採用時と
  rollback 時に version file と `default.json` の byte と digest が一致する

## 関連

- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-MODEL-026: QueryProfileContribution](../20-model/26-query-profile-contribution.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](../30-architecture/27-unified-query-cross-profile-composition.md)
- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](../30-architecture/33-unified-query-profile-set-adoption-boundary.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-026: 統合照会の評価コーパス成果物契約](26-legal-query-corpus-artifact-contract.md)
- [SOT-ENG-029: 統合照会の検索例カタログ](29-unified-query-example-catalog.md)
- [SOT-ENG-030: 統合照会の cue 成果物契約](30-unified-query-cue-artifact-contract.md)
