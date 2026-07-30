# SOT-ENG-031: 統合照会の採用済み意図 cue セット

- 状態: 有効

## 規定

統合照会の query profile が採用済みの task、resource、operator および
pack 境界を明示するために使う cue は、profile ごとの版付き成果物で閉じた
`category` と `value` を持ち、同じ語の意味を profile の外へ借用せず、
決定的に解釈する。

## 定義元と適用範囲

本規定は、採用済みの取得意図を表す positive cue の意味境界、profile ごとの
所有範囲および閉じた `category/value` を定義する。

公開される task/resource の組合せそのものは `SOT-PROD-011` を定義元とする。
本規定の `category/value` は cue 成果物の内部境界であり、公開 resource type を
増やしたり置き換えたりしない。

JSON 構造、schema version、順序、正規化衝突、loader および profile との整合は
`SOT-ENG-030` を定義元とする。対象外意図の閉じた対応は `SOT-ENG-028`、
cue の構文 role は `SOT-MODEL-030` を定義元とし、本規定で重複して定義しない。

法令名、法概念、一般検索語、provider 固有の検索語、unsupported cue、
検索結果依存の後続 read、および外部情報源の成否に応じてだけ意味を持つ語を、
採用済み意図 cue へ混在させない。

## 採用済み cue の所有範囲

現行の固定 profile set で採用する positive cue の `category` と `value` は、
次の閉じた集合に限る。

### `core`

| `category` | 許可する `value` |
|---|---|
| `task` | `search`、`read`、`list_updates` |
| `resource` | `law`、`law_provision`、`updates` |
| `operator` | `all`、`any`、`as_of`、`dual_candidate`、`exclude`、`individual`、`resource_choice`、`single_choice` |
| `reserved_pack` | `judicial-cases` |
| `safety` | `implicit_first_read` |
| `syntax` | `content_result_unit`、`related_law_scope`、`task_predicate` |

### `judicial-cases`

| `category` | 許可する `value` |
|---|---|
| `task` | `search`、`read` |
| `resource` | `judicial_decision` |
| `resource_scope` | `legal_information` |
| `operator` | `individual`、`resource_choice` |

将来の pack を追加する場合は、同じ形式で profile ごとの閉じた集合を定義する
新しい有効な SOT または本規定の後継 SOT を先に採用する。未採用の pack、
task または resource の `category/value` を、実装先行で cue 成果物へ入れない。

## category ごとの境界

- `task` は、採用済みの取得動作だけを表す。公開 task の採用範囲は
  `SOT-PROD-011` に従い、法的助言、翻訳、比較、影響分析その他の対象外 task は
  `SOT-ENG-028` の所有とし、本規定へ入れない。
- `resource` は、profile が task と組み合わせて使う cue-level の取得対象境界を
  表す。`law`、`law_provision`、`judicial_decision` は公開 resource へ対応し、
  `updates` は `list_updates` と組み合わせる内部境界であって、新しい公開
  resource type を追加しない。法令名辞書または法概念辞書の entry を代替しない。
- `resource_scope` は、特定の公開 resource を選択しない広い対象範囲を表す。
  同じ表現を明示 resource cue とみなさず、profile 内で resource の曖昧性を
  保持するための補助境界とし、新しい公開 resource type を追加しない。
- `operator` は、複数主題の分離、候補保持、明確化または検索語演算の解釈に
  使う補助語彙であり、単独で capability request を作らない。
- `reserved_pack` は、別 profile が所有する採用済み pack の resource 表現を
  core profile が識別する境界とする。core の一般候補への誤取り込みを防ぎ、
  予約済み pack request の signal を作るためにだけ使う。pack の有効状態を
  表さず、それだけで core の候補を作らない。
- `safety` は、暗黙の後続 read を禁止する安全境界だけを表し、別の resource へ
  再分類する cue にしない。
- `syntax` は、検索対象の範囲または task 述語のような構文補助に限り、
  task、resource または signal を単独で表さない。

## profile 間の分離

各 cue entry は一つの profile だけが所有し、その profile だけが明示
task/resource の根拠として使用する。別 profile は、その cue の近接、同名の
`value` または共有された表現だけを根拠に、自身の候補を強化しない。

同じ比較用正規化表現を複数 profile が持つことは、次のいずれかの場合に限る。

- `individual` や `resource_choice` のように、各 profile が同じ補助意味を独立に
  所有する operator
- `裁判例` のように、ある profile では `resource`、別の profile では
  `reserved_pack` として、明示的に異なる境界を所有する場合

同じ表現が複数 profile に現れても、前処理または profile がそれらを一つの共有 cue
へ平坦化しない。profile が独立に生成した候補 contribution の合成だけを、
`SOT-ARCH-027` の composer が行う。

## 構文 role の境界

採用済み意図 cue の `syntaxRole` は、次の閉じた境界に従う。

- `task` は `task_expression` または `task_predicate`
- `syntax` のうち `task_predicate` を表す entry は `task_predicate`
- それ以外の positive cue は `none`

上記は positive cue が取り得る role の上限である。現行の固定 profile set に
おける `profileId/category/value` ごとの完全な対応は `SOT-ENG-032` を
定義元とし、許可された二つの task role から実装が任意に一つを選ばない。

現行標準では、採用済み意図 cue に `task_object` を使用しない。`task_object` を
使う cue を採用する場合は、その cue が task として成立する条件、relation、
候補保持および評価 fixture を定義する新しい有効な SOT を先に採用する。

誤記補正だけで task または resource の明示 cue を新しく作らない規則、
引用句および topic 表現が relation を作らない規則、ならびに cue span の抽出方法は
`SOT-MODEL-025` と `SOT-MODEL-030` に従う。

## 変更の連動

positive cue の `category`、`value`、`syntaxRole`、所有 profile または
同じ profile 内の意味境界を変更した場合の cue 成果物、profile および
profile set の version 連動は `SOT-ENG-030` を定義元とする。

その変更が relation 依存の signal、target/resource の候補化、複数主題分離、
候補保持または公開 decision を変える場合は、必要な profile fixture と
architecture test を追加し、`SOT-ENG-024` が要求する corpus、baseline、
標準評価および検索例カタログを同じ採用変更で同期する。関係する model または
architecture の規定自体を変える場合は、その SOT も同じ変更で更新する。

## 確認

少なくとも次を、外部ネットワークを使わない loader test、profile test および
planner test で確認する。

- 各 profile が、自身に許可された `category/value` だけを持つ
- 未知の `category`、`value` または positive cue の `task_object` を起動時に拒否する
- `individual` や `resource_choice` の共有 operator を複数 profile が持っても、
  cue 自体を共有して別 profile の候補根拠にしない
- `裁判例` のように core の `reserved_pack` と `judicial-cases` の `resource` が
  同じ表現を持つ場合でも、各 profile が自分の境界だけを適用する
- positive cue の変更が relation 依存の signal、候補保持または公開 decision を
  変える場合は、`SOT-ENG-024` の標準評価と検索例カタログ更新を同じ変更で要求する

## 関連

- [SOT-PROD-011: 統合法情報照会の製品範囲](../00-product/11-unified-legal-query-scope.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](../20-model/25-legal-query-preprocess-result.md)
- [SOT-MODEL-030: CueTaskRelation v2](../20-model/30-cue-task-relation-v2.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](../30-architecture/25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](../30-architecture/27-unified-query-cross-profile-composition.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](../30-architecture/31-unified-query-intent-evidence-layer.md)
- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](../30-architecture/33-unified-query-profile-set-adoption-boundary.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-028: 統合照会の対象外意図 cue セット](28-unified-query-unsupported-intent-cues.md)
- [SOT-ENG-030: 統合照会の cue 成果物契約](30-unified-query-cue-artifact-contract.md)
- [SOT-ENG-032: 統合照会の positive cue role 対応](32-unified-query-positive-cue-role-mapping.md)
