# SOT-ENG-032: 統合照会の positive cue role 対応

- 状態: 有効

## 規定

現行の固定 profile set が採用する positive cue は、
`profileId/category/value` ごとに一つの `syntaxRole` へ完全対応させ、
同じ task の role を実装、transport または照会ごとに切り替えない。

## 適用範囲

positive cue の所有 profile と閉じた `category/value` は `SOT-ENG-031`、
`syntaxRole` と `CueTaskRelation` の意味は `SOT-MODEL-030` を定義元とする。
本規定は、その許可範囲から現行の固定 profile set が採用する完全対応だけを
定義する。

対象外意図 cue の role は `SOT-ENG-028` を定義元とし、本規定へ混在させない。
法令名、条項、事件番号、日付、法概念または一般検索語を positive cue の
`task_object` として登録しない。

## 完全対応

現行の対応は次のとおりとする。

| `profileId` | `category` | `value` | `syntaxRole` |
|---|---|---|---|
| `core` | `task` | `search` | `task_expression` |
| `core` | `task` | `read` | `task_expression` |
| `core` | `task` | `list_updates` | `task_expression` |
| `core` | `syntax` | `task_predicate` | `task_predicate` |
| `judicial-cases` | `task` | `search` | `task_expression` |
| `judicial-cases` | `task` | `read` | `task_expression` |

上表にない positive cue の `syntaxRole` は `none` とする。
一つの `profileId/category/value` に複数の role を許可せず、
term ごとに role を変えない。

## task と可変対象の分離

`task_expression` は、その登録表現だけで `search`、`read` または
`list_updates` のいずれかを特定できる positive task cue に使用する。
その出現は `SOT-MODEL-030` の終端条件と除外条件を満たす場合だけ
`direct_task` relation となる。裸の `CueMention` だけから task を作らない。

例えば、`民法第709条を見せてください。` の `見せてください` は
core の `task/read/task_expression` として `direct_task` を作り、民法と
第709条は別の位置付き target anchor として profile が束縛する。
法令名や条項を有限の cue へ登録したり、可変対象との近接だけで
`object_predicate` を作ったりしない。

一方、`SOT-ENG-028` が閉じた対象外 `task_object` として登録した表現が、
positive `task_expression` と `SOT-MODEL-030` の格助詞・文末条件で直接接続する
場合は、同じ positive cue 出現について `direct_task` と
`object_predicate` の両 relation を保持する。`EDINETを検索してください。` の
positive search を残すことを理由に対象外 relation を削除せず、対象外 relation を
作ることを理由に positive cue の role を `task_predicate` へ変えない。

`core/syntax/task_predicate` は、`SOT-ENG-028` が所有する
`task_object` と、positive task に対応しない構文述語を直接接続するための補助で
ある。単独の mention または relation の predicate 側だけを、positive task や
`explicit_task` evidence として使用しない。

したがって現行の core と `judicial-cases` の positive task は、いずれも
`direct_task` の subject として成立したときだけ `explicit_task` evidence を
作る。`judicial-cases` に positive `task_predicate` を導入したり、predicate 側
だけで search/read を確定したりしない。

## 変更

現行表へ task、profile または role の対応を追加、削除または変更する場合は、
その対応を採用する新しい有効な SOT または本規定の後継 SOT を先に採用する。

cue 成果物、profile および profile set の version 連動は `SOT-ENG-030` に従う。
公開既定の meaning、decision、step、reason または外部呼出し境界が変わる場合は、
`SOT-ARCH-033` と `SOT-ENG-024` に従い、profile set、corpus、baseline、
標準評価および検索例カタログを原子的に採用する。

## 確認

外部ネットワークを使わない loader test、共通前処理 test および profile test で、
少なくとも次の固定 test ID を確認する。

- `positive-cue-value-role-mapping`: 全 positive cue が上表または
  `none` の完全対応と一致し、task の `task_predicate` と positive cue の
  `task_object` を拒否する
- `cue-relation-positive-direct-task`: `民法第709条を見せてください。` の
  positive read cue が `direct_task` relation となり、条項を cue にしない
- `cue-relation-positive-dual-role`: `EDINETを検索してください。` の同じ
  positive search cue から `direct_task` と、unsupported object を subject とする
  `object_predicate` を作り、外部呼出しを行わない
- `positive-cue-bare-mention-rejected`: 文末条件を満たさない mention、
  引用句内の mention および閉じた言及表現から positive task を作らない
- `positive-cue-syntax-predicate-isolation`: `core/syntax/task_predicate` の
  mention または predicate だけで positive task、candidate または
  evidence code を増やさない
- `positive-cue-profile-isolation`: 同じ term を持つ別 profile の
  relation、candidate または evidence code を借用しない

## 関連

- [SOT-MODEL-025: LegalQueryPreprocessResult](../20-model/25-legal-query-preprocess-result.md)
- [SOT-MODEL-030: CueTaskRelation v2](../20-model/30-cue-task-relation-v2.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](../30-architecture/31-unified-query-intent-evidence-layer.md)
- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](../30-architecture/33-unified-query-profile-set-adoption-boundary.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-028: 統合照会の対象外意図 cue セット](28-unified-query-unsupported-intent-cues.md)
- [SOT-ENG-030: 統合照会の cue 成果物契約](30-unified-query-cue-artifact-contract.md)
- [SOT-ENG-031: 統合照会の採用済み意図 cue セット](31-unified-query-adopted-intent-cues.md)
