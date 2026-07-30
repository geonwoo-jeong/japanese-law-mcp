# SOT-ENG-030: 統合照会の cue 成果物契約

- 状態: 有効

## 規定

統合照会の query profile が共通前処理へ渡す cue 語彙は、profile ごとに独立した
版付きの閉じた JSON 成果物として保持し、共通 loader が構造、版、語彙衝突および
profile との対応を起動時に検証する。

## 適用範囲

本規定は cue 成果物に共通する JSON 構造、schema version、識別子、順序、
語彙衝突、version 連動および loader test を定義する。各 `category` と `value` の
意味、task/resource への対応、score、signal および採用範囲は、それを所有する
profile または領域別 SOT を定義元とし、本規定へ集約しない。

採用済み意図の positive cue の閉じた所有範囲は `SOT-ENG-031`、現行
profile set の positive cue role 完全対応は `SOT-ENG-032`、対象外意図の
閉じた対応は `SOT-ENG-028`、`syntaxRole` の意味と `CueTaskRelation` の
構築規則は `SOT-MODEL-030` を定義元とする。

## 配置と最上位構造

各 query profile の cue 成果物は、その profile package の `data/cues.json` に
置く。一つの成果物は次の項目だけを持つ。

| 項目 | 型 | 必須 | 制約 |
|---|---|---:|---|
| `schemaVersion` | integer | はい | 共通 loader が実装した閉じた version |
| `profileId` | string | はい | 所有する query profile の ID |
| `cueSetVersion` | string | はい | cue 集合の不透明な版 |
| `cues` | `CueEntry[]` | はい | 一件以上、`cueId` 昇順 |

`CueEntry` は次の項目だけを持つ。

| 項目 | 型 | 必須 | 制約 |
|---|---|---:|---|
| `cueId` | string | はい | 成果物内で一意 |
| `category` | string | はい | profile が定義する閉じた値 |
| `value` | string | はい | category に対応する閉じた値 |
| `intentGroup` | string | 条件付き | 領域別 SOT が必要とする場合だけ |
| `signal` | string | 条件付き | 領域別 SOT が必要とする場合だけ |
| `syntaxRole` | string | schema version 3 では必須 | `SOT-MODEL-030` の閉じた値 |
| `terms` | `string[]` | はい | 一件以上、比較用正規化値の昇順 |

`profileId`、`cueId` および `cueSetVersion` は、一 byte 以上百二十八 byte 以下の
ASCII 小文字英数字を、単一の `-` で連結した正規形とする。`category`、`value`、
`intentGroup` および `signal` は、それぞれの定義元が許可する ASCII の閉じた
値とする。空文字、`null`、重複 key、未知項目、未知 enum および trailing value を
拒否する。

## schema version

`schemaVersion=3` は `CueTaskRelation` に対応する最初の cue 成果物 schema とし、
全 entry に `syntaxRole` を必須とする。version 1 と version 2 は relation
非対応の履歴 schema であり、欠落した `syntaxRole` を `none` として補わない。
未知 version は起動時に拒否する。

relation を有効にする固定 profile set は、参加する全 profile の cue 成果物を
version 3 へそろえる。一部だけを version 3 にした profile set、または
version 1、version 2 と version 3 が混在する profile set を relation 対応として
起動しない。relation をまだ有効にしない準備状態と production からの到達性は
`SOT-ARCH-033`、標準評価成果物と受入基準の切替は `SOT-ENG-024` に従う。

cue 成果物の schema version は `profile.json` の schema version と独立する。
ただし、同じ profile の二成果物は `profileId` と `cueSetVersion` が一致しなければ
ならない。

## 語彙の一意性と衝突

登録表現は、有効な UTF-8 の非空文字列とする。loader は共通の比較用正規化を
各表現へ一回適用し、次を検証する。

- 一つの entry 内で、比較用正規化後に同じ表現を重複させない
- 一つの profile 内で、一つの正規化表現を複数の `cueId` に所属させない
- `category`、`value`、`intentGroup` の有無と値、`signal` の有無と値、および
  `syntaxRole` の組を意味 tuple とする
- 同じ開始位置で同じ tuple の長短表現が重なる場合は、最長 span 一件だけを
  cue mention の候補にする
- 異なる tuple の表現が同じ正規化値を持つ場合、または包含関係により同じ入力の
  同じ開始位置から異なる tuple を作り得る場合は、profile の起動を失敗させる

異なる profile は、能力別の意味を独立して持つため、同じ正規化表現または包含語を
別の tuple で再利用できる。profile set は profile 間の語彙を衝突として扱わず、
`SOT-MODEL-030` に従って異なる `profileId` の cue mention を一つの relation に
結び付けない。

## 順序と版の連動

`cues` は `cueId` の byte 昇順、各 `terms` は比較用正規化値と原文字列の順による
完全順序とする。loader は不正な順序を黙って整列しない。

登録表現、category/value の対応、`intentGroup`、`signal`、`syntaxRole` または
照合境界を変更した場合は、`cueSetVersion`、対象 profile の profile version
および固定 profile set version を変更する。schema の形を変えた場合は
`schemaVersion` も増やし、旧 version の成果物と loader を再現可能なまま保持する。
score、閾値、margin または profile 横断の尺度を変えない限り、ranking version は
変更しない。

## loader の責務

共通 loader は、外部 network を使わず、次の順序で検証する。

1. file size、UTF-8、JSON depth、value 数、重複 key および EOF
2. 実装済み schema version と閉じた typed decode
3. ID、配列順、entry と term の一意性
4. profile 内の正規化語と包含語の衝突
5. `profile.json` との `profileId` と `cueSetVersion` の一致
6. 固定 profile set の schema version と profile ID の整合
7. profile または領域別 SOT が定める category/value と条件付き項目の対応

構造検証が完了する前に cue の意味を profile へ渡さない。失敗した cue 成果物を
無視して profile を部分起動したり、未知項目を捨てて受理したりしない。

## 確認

少なくとも次の固定 test ID を、外部 network を使わない loader test と
profile-set test から追跡できるようにする。

| test ID | 固定する境界 |
|---|---|
| `cue-loader-schema-v3` | version 3 の必須項目、旧版混在および未知版の拒否 |
| `cue-loader-duplicate-term-owner` | 同じ profile の正規化語を複数 cue ID に所属させない |
| `cue-loader-longest-same-tuple` | 同じ tuple の長短語は最長 span 一件 |
| `cue-loader-tuple-conflict` | 同じ profile の異なる tuple の同一語・包含語は起動失敗 |
| `cue-loader-cross-profile-reuse` | profile 間の同語再利用を許し relation は結合しない |
| `cue-loader-profile-version-match` | profile ID と cue set version の一致 |

schema version 3 の全 entry に `syntaxRole` があること、version 1 または version 2
から role を補わないこと、profile set 内の混在を拒否すること、配列順を自動修正
しないこと、および getter の変更と並行読取りで共有状態を変更しないことも
確認する。

## 関連

- [SOT-MODEL-026: QueryProfileContribution](../20-model/26-query-profile-contribution.md)
- [SOT-MODEL-030: CueTaskRelation v2](../20-model/30-cue-task-relation-v2.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](../30-architecture/33-unified-query-profile-set-adoption-boundary.md)
- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-028: 統合照会の対象外意図 cue セット](28-unified-query-unsupported-intent-cues.md)
- [SOT-ENG-031: 統合照会の採用済み意図 cue セット](31-unified-query-adopted-intent-cues.md)
- [SOT-ENG-032: 統合照会の positive cue role 対応](32-unified-query-positive-cue-role-mapping.md)
