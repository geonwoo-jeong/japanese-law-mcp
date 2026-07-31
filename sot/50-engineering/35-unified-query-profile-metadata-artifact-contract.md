# SOT-ENG-035: 統合照会 profile metadata 成果物契約

- 状態: 有効

## 規定

統合照会の各 query profile が使用する `data/profile.json` は、profile の対象、
score policy、selection policy、tie-break および参照成果物の版を保持する、
版付きで閉じた JSON 成果物とする。共通 model と profile 固有 loader は、
schema version ごとの項目有無を失わずに検証し、起動後に変更できない metadata
として query profile へ渡す。

## 適用範囲

本規定は、`profile.json` の配置、共通構造、schema version、loader の安全境界、
不変性および固定 profile set 内の整合を定義する。

次は本規定で再定義しない。

- task/resource/input kind の意味と profile 固有の対象集合
- cue entry の構造と意味
- 辞書 entry の構造、出典および採用範囲
- score、閾値、margin、tie-break および profile version の具体値
- profile set の校正、採用、rollback および導入順

これらは各 model、architecture、辞書、cue 成果物、評価および採用 SOT を
定義元とする。

## 配置と入力境界

各 profile package は、次の一つの active 成果物を所有する。

```text
internal/queryprofile/<profile>/data/profile.json
```

production は package に埋め込んだこの固定 path の成果物だけを読み、CLI、環境変数、
設定、MCP 引数または transport から別 path や未採用版を選択できるようにしない。
採用前の候補成果物は test が直接構成する別の内部 fixture とし、
`SOT-ARCH-033` の準備状態から production へ到達させない。

一成果物は一 byte 以上六万五千五百三十六 byte 以下の UTF-8 JSON object とする。
空入力、UTF-8 として不正な byte、重複 key、`null`、未知項目、二個目の JSON 値、
JSON 後方の非空白 token および対応しない schema version を拒否する。

profile metadata は package 内の埋込み設定であり、独立した JSON Schema file を
持たない。schema version ごとの閉じた typed document と、後述の意味検証を
機械契約とする。`testdata/legalquery/schemas/` または profile directory に
重複する schema file を生成せず、schema version の追加時は version 別 typed
document と固定 contract test を追加する。

## 共通 object

top-level object は次の項目だけを持つ。次の表と各 object の項目一覧は field set を
示し、JSON object の key 順を意味として固定しない。loader は key 順の違いを
拒否せず、順位、version または digest の入力にしない。意味を持つ配列順だけを
各節の規定どおり検証する。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `schemaVersion` | integer | はい | 本成果物構造の版 |
| `profileId` | string | はい | 所有する profile の固定 ID |
| `profileVersion` | string | はい | profile 固有規則と参照成果物の組合せの版 |
| `rankingVersion` | string | はい | 固定 profile set が共有する校正尺度の版 |
| `cueSetVersion` | string | はい | 対応する cue 成果物の版 |
| `targets` | object[] | はい | profile が採用する能力対象の固定順 |
| `score` | object | はい | 根拠ごとの重みと confidence 境界 |
| `selection` | object | はい | 実行可否、単独選択、hedge および限定分岐の境界 |
| `tieBreak` | string[] | はい | 通常候補の完全順序 |
| `conditionalTieBreaks` | object | profile 条件付き | profile 固有 SOT が採用した条件付き完全順序 |
| `lexicons` | object | はい | 参照する法令名辞書と法概念辞書の版 |

`profileId` は一 byte 以上六十四 byte 以下で、小文字 ASCII 英数字の segment を
一つのハイフンで連結した値とする。各 version は一 byte 以上百二十八 byte 以下の
有効な UTF-8 とし、Unicode control character または先頭と末尾の Unicode
White_Space を含めない。profile package は `profileId` が自身の所有 ID と
一致することを検証する。

### `targets`

`targets` は一件以上七件以下とし、各 object は `task`、`resource`、
`inputKind` だけを持つ。三値の組合せは `SOT-MODEL-022` の採用済み組合せで
なければならず、`inputKind` を重複させない。

配列順は profile 固有 SOT が採用する能力順と一致させる。core は法令コア五能力、
`judicial-cases` は裁判例二能力の固定順を各 package で検証する。loader は不正な
順序を並べ替えない。

### `score`

`score` は次の項目だけを持つ。

| 項目 | 型 | 必須 |
|---|---|---:|
| `minimum` | integer | はい |
| `maximum` | integer | はい |
| `evidenceWeights` | object[] | はい |
| `highConfidenceAt` | integer | はい |
| `mediumConfidenceAt` | integer | はい |

`minimum` は零以上、`maximum` は `minimum` 以上かつ百万以下とする。
`maximum` は `minimum` と全 weight の合計に一致させる。confidence 境界は
score 範囲内に置き、`highConfidenceAt` は `mediumConfidenceAt` 以上とする。
`evidenceWeights` は `SOT-MODEL-022` が許可する九つの根拠 code を、同 SOT の
強い順に一件ずつ持つ固定完全順とする。各 object は `code` と一以上百万以下の
整数 `weight` だけを持ち、code を重複させない。

### `selection`

`selection` の共通項目は次の四件とする。

- `singleThreshold`
- `minimumExecutionThreshold`
- `singleMargin`
- `hedgeMargin`

各値は零以上の整数とし、閾値は score 範囲内、margin は
`score.maximum - score.minimum` 以下とする。
`minimumExecutionThreshold` は score の minimum 以上、
`singleThreshold` は `minimumExecutionThreshold` 以上かつ score の maximum
以下、`hedgeMargin` は `singleMargin` 以下とする。採用する具体値は
`SOT-ARCH-023` および `SOT-ENG-024` の校正で検証する。

schema version 2 は、これらに `branchRetentionMargin` を必須で追加する。
この値は零以上かつ `score.maximum - score.minimum` 以下とし、
`singleMargin` または `hedgeMargin` から補完しない。

### `tieBreak` と `conditionalTieBreaks`

`tieBreak` は次の完全順とする。

1. `evidence_set`
2. `step_count`
3. `meaning_signature`
4. `source_position`

core profile は `SOT-ARCH-028` の候補上限超過時に使用する
`conditionalTieBreaks.lawAliasCollisionGroupsOverCandidateLimit` を持ち、
次の完全順とする。

1. `evidence_set`
2. `step_count`
3. `source_position`
4. `meaning_signature`

この条件付き項目を採用していない profile は `conditionalTieBreaks` 自体を
持たない。共通 loader は未知の条件名を受理せず、profile 固有 loader が
許可された条件名と完全順を検証する。

### `lexicons`

`lexicons` は `lawNames` と `legalConcepts` だけを持つ。各値は対応する
起動時辞書成果物の version と完全一致しなければならない。loader は不一致を
警告にせず profile 構成エラーとする。

## schema version

`schemaVersion=1` は `selection.branchRetentionMargin` を持たない既存成果物とし、
現行採用集合、履歴再現および rollback のために受理し続ける。

`schemaVersion=2` は `selection.branchRetentionMargin` を持つ最初の版とする。
version 1 が同項目を持つ場合、version 2 が同項目を欠く場合、および未知の version
は loader error とする。

共通 model は version 1 と version 2 を、項目の存在状態を保持する閉じた版型として
扱う。version 1 を version 2 と同じ値型へ変換するために零、既存 margin または
暫定定数を補わない。package 間で参照する accessor は次とする。

```text
BranchRetentionMargin() (value int, present bool)
```

version 1 は `(0, false)`、version 2 は検証済み値と `true` を返す。
`present=false` の零を設定値として扱わず、有効な零と欠落を区別する。

本成果物の object または field の意味、型、必須性を変える場合は新しい schema
version を追加し、履歴再現に必要な旧 decoder と検証を保持する。具体的な規則、
辞書、cue、重みまたは校正値の変更は、それぞれの定義元に従って
`profileVersion`、`rankingVersion` または `cueSetVersion` を更新する。

## 固定 profile set の整合

同じ固定 profile set の全 profile は、一つの `rankingVersion` と、同じ score
scale、confidence、selection policy および tie-break 校正を持つ。
限定分岐を検証する固定 set は、全 profile を schema version 2 にそろえ、
同じ `branchRetentionMargin` を持たなければならない。

ここでいう共有校正値の一致は、各 profile metadata の次の値が byte 表記ではなく
検証済み値として完全一致することを意味する。

- `score.minimum`、`score.maximum`、九件の `evidenceWeights` の code・weight・順、
  `highConfidenceAt` および `mediumConfidenceAt`
- `selection.singleThreshold`、`minimumExecutionThreshold`、`singleMargin`、
  `hedgeMargin` および schema version 2 の `branchRetentionMargin`
- 共通 `tieBreak` の全要素と順

`conditionalTieBreaks` は profile 固有 SOT が採用した条件だけを各 profile loader
で検証するため、異なる profile 間で object の有無を一致させない。条件付き順序を
共通 `tieBreak` の不一致を隠す代替値としても使わない。

個別 profile の loader、private evidence mapping または candidate generation を
単独 test fixture で構築することは、校正、holdout、baseline または採用候補としての
固定 profile set を構成したことにはならない。固定 set として評価する時点で初めて、
全 profile の schema version、ranking version および共有校正値の一致を検証する。
準備、校正、holdout 判定および原子的採用の順序は `SOT-ENG-034` だけを
定義元とする。

## loader と不変性

schema version ごとに別の閉じた typed decoder で項目有無を確認してから、
共通 metadata model と profile 固有規則を検証する。一部だけ有効な metadata、
旧版からの暗黙補完、既定値への fallback または警告付き継続を許可しない。

loader は少なくとも次を照合する。

- profile ID、各 version、target 組合せと固定順
- score、selection、tie-break および条件付き tie-break
- cue 成果物の `profileId` と `cueSetVersion`
- 法令名辞書と法概念辞書の version
- 固定 profile set 内の ranking version、schema version および共有校正値

構築した metadata は field を公開せず、slice、map および入れ子 object の accessor
は深い複製または不変な値を返す。request ごとに metadata を変更せず、loader は
外部 network、時刻、乱数または環境状態へ依存しない。

## 確認

外部 network を使わない loader、model、profile set および不変性 test で、
少なくとも次の固定 test ID を確認する。

- `profile-metadata-closed-json`: 空入力、上限超過、不正 UTF-8、重複 key、
  `null`、未知項目、二個目の値および後方 token を拒否する
- `profile-metadata-schema-versions`: version 1 と version 2 の項目有無を区別し、
  未知 version、version 1 への field 混入および version 2 の field 欠落を拒否する
- `profile-metadata-branch-retention-presence`: 有効な零と欠落を区別し、
  既存 margin から暗黙補完しない
- `profile-metadata-profile-ownership`: profile ID、target の固定順、
  cue set および辞書 version の不一致を拒否する
- `profile-metadata-ranking-consistency`: 固定 set 内の ranking version、共有 score、
  selection、tie-break および schema version の混在を、任意の固定 set
  constructor に共通する loader 検査として拒否する
- `next-profile-set-fixed-composition`: 次版の全 profile を production と同じ固定順で
  一回だけ構成し、schema version 2、ranking version、score scale、confidence、
  selection、共通 tie-break および `branchRetentionMargin` の完全一致を確認する。
  ranking version が同じでも共有校正値の一項目だけが異なる set と、値が同じでも
  profile 順または evidence weight 順だけが異なる set を拒否する。この検証は
  `profile-metadata-ranking-consistency` と同じ loader 検査を、次版の正確な
  composition へ適用し、profile の欠落、重複または test 用の別順も追加で拒否する
- `profile-metadata-conditional-tie-break`: core の条件付き完全順を検証し、
  未採用 profile の条件付き項目と未知条件を拒否する
- `profile-metadata-immutability`: accessor の戻り値を変更しても metadata、
  別 profile または後続 request を変更できない

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](../30-architecture/23-unified-query-selection-and-hedging.md)
- [SOT-ARCH-028: 法令別名衝突の基本法優先順位](../30-architecture/28-law-alias-collision-ranking.md)
- [SOT-ARCH-032: 統合照会の限定分岐保持](../30-architecture/32-unified-query-bounded-branch-retention.md)
- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](../30-architecture/33-unified-query-profile-set-adoption-boundary.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-025: 統合照会のパッケージ構成](25-unified-query-package-layout.md)
- [SOT-ENG-030: 統合照会の cue 成果物契約](30-unified-query-cue-artifact-contract.md)
- [SOT-ENG-033: 統合照会 profile set 採用 manifest](33-unified-query-profile-set-adoption-manifest.md)
- [SOT-ENG-034: 統合照会の意味判定変更における導入段階と変更順序](34-unified-query-rollout-stages.md)
