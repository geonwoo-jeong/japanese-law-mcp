# SOT-ENG-028: 統合照会の対象外意図 cue セット

- 状態: 有効

## 規定

統合照会は、製品範囲外の task、resource、法的助言および翻訳を、版付きの
日本語 cue セットから決定的に検出し、採用済みの取得意図だけへ縮約して
実行しない。

## 定義元と適用範囲

採用範囲の定義元は `SOT-PROD-011` とし、本規定はその範囲を拡張または
縮小しない。本規定は、対象外意図を `SOT-MODEL-026` の signal へ変換する
語彙、照合境界、版管理および回帰確認だけを定義する。

法令名、法概念、provider 固有の検索語および一般検索語を、この cue セットへ
混在させない。対象外意図の cue は法的な同義語辞書ではなく、外部呼出しを
安全に止めるための閉じた task 語彙とする。

## 必須の意図群

対象外 cue の `intentGroup`、`value` および `signal` は次の閉じた対応とする。

| `intentGroup` | `value` | `QueryProfileSignal` | 最低限認識する表現 |
|---|---|---|---|
| `legal_advice` | `legal_advice` | `unsupported_legal_advice` | `どうすればよいですか`、`違法ですか`、`適法か判断`、`勝てますか`、`どちらが有利` |
| `translation` | `translation` | `unsupported_translation` | `翻訳`、`英訳`、`和訳`、`英語に翻訳` |
| `version_comparison` | `task_or_resource` | `unsupported_task_or_resource` | `比較して`、`比較してください`、`差分`、`二時点を比較`、`改正前後を比較`、`変更履歴を追う` |
| `relationship_analysis` | `task_or_resource` | `unsupported_task_or_resource` | `影響グラフ`、`影響マップ`、`引用関係図`、`引用関係を可視化`、`引用する裁判例のグラフ` |
| `unadopted_information_or_extension` | `task_or_resource` | `unsupported_task_or_resource` | `立法理由`、`国会審議`、`行政規則`、`税務相談`、`労務相談` |
| `external_information_source` | `task_or_resource` | `unsupported_task_or_resource` | `EDINET`、`法律事務所ブログ`、`民間判例データベース`、`未公開内部文書`、`自治体条例` |
| `explicit_out_of_scope_task` | `task_or_resource` | `unsupported_task_or_resource` | 上表の個別群に属さない、採用範囲外であることを明示した task 表現 |

表の表現は必須の回帰境界であり、同じ意図群の活用形、送り仮名、全角・半角、
句読点および敬語差を、比較用正規化または Kagome の語境界によって追加できる。
単なる部分文字列の一致で、より長い別語の内部を cue にしない。

未採用だった task または resource を後に採用する場合は、製品範囲と対応する
profile を先に採用し、対象外 cue から同じ意味群を除いた新しい
`cueSetVersion` と profile version を割り当てる。pack の有効・無効だけで
cue セットを変えない。

## 構文 role と照合境界

- cue は登録表現との完全一致、比較用正規化一致または Kagome が確認した
  登録語の span だけから作る。対象外 task を誤記補正で新しく作らない。
- 各 cue entry は `SOT-MODEL-030` の `syntaxRole` を一つ持つ。対象外 cue は
  `task_expression` または `task_object` とし、role が異なる登録表現を同じ
  cue ID に混在させない。対象外の目的語へ接続する採用済み task その他の
  構文述語は、`SOT-MODEL-030` に従い、同じ profile の positive
  `task_expression` または別の `task_predicate` cue を使用できる。
- 対象外 cue の `CueMention` が存在するだけでは signal を作らない。その cue を
  `subject` とする検証済み `CueTaskRelation` がある場合だけ、subject cue の
  `intentGroup` と上表から一つの signal を作る。
- `direct_task` は `task_expression`、`object_predicate` は
  `task_object` と同じ節の述語、`standalone_task` は節全体に単独で現れた
  `task_object` だけを根拠にする。
- `SOT-MODEL-030` の非 task 境界により relation を持たない cue 出現は、
  対象外 signal にしない。本規定の確認例はその境界を変更せず、対象外語彙に
  対する回帰例を追加するだけとする。
- 複数の profile が同じ対象外表現を認識しても、profile set は
  `SOT-MODEL-026` の固定順で同じ signal を一件にする。

`intentGroup` は cue の監査、版管理および回帰分類に使用する。単独で候補を
保持、削除、加点または順位変更する実行時分岐に使用しない。候補の保持は
signal、`CueTaskRelation`、候補ごとの根拠 span および `SOT-MODEL-026` だけから
判定する。

対象外 signal は、selector が実行対象を選ぶ前に適用できる入力として渡す。
signal から `unsupported` plan、`mixed_unsupported_intent` または
`unsupported_task_or_resource` への対応、内部候補の保持および外部呼出しの
禁止は `SOT-MODEL-023` を定義元とする。公開する固定 notice は
`SOT-MODEL-024` を定義元とし、本規定では plan reason または notice の対応を
重ねて定義しない。

## 成果物と版

cue データの配置、閉じた JSON 構造、schema version、順序、正規化語の衝突、
profile 間の語彙再利用、`profile.json` との整合および version 連動は
`SOT-ENG-030` に従う。

本規定が追加する成果物上の制約は、`category=unsupported` の entry が、上表の
`intentGroup`、`value`、signal および許可された `syntaxRole` の対応を完全に
満たすこととする。対象外以外の entry は、本規定を根拠に `intentGroup` または
signal を持たせない。同じ signal を複数 profile が作った場合の集約は
`SOT-MODEL-026` の固定順に従う。

## 確認

少なくとも次を、外部ネットワークを使わない model test、profile fixture
および loader test で確認する。将来、同じ例を統合評価 corpus へ追加する場合は、
`SOT-ENG-024` と `SOT-ENG-026` の版変更規則に従い、存在しない corpus または
baseline を現行の標準 command として先に宣言しない。

次の対象外意図固有の固定 test ID を検証結果から追跡できるようにする。
cue 成果物 loader の共通 test ID は `SOT-ENG-030` を定義元とし、本規定へ
重複して定義しない。relation 対応 profile set を変更する中央品質ゲートでは
`SOT-ENG-020` の全 Go test と統合評価の両方を成功させる。

| test ID | 固定する境界 |
|---|---|
| `cue-relation-task-and-mention` | 実 task、引用、`という語`および topic 表現 |
| `cue-relation-clause-scope` | 同じ節と別の節 |
| `cue-relation-candidate-scope` | 候補・step ごとの根拠と別候補への非共有 |

- `民法第103条を引用する裁判例の影響グラフを作成してください。` は、
  法令または条文の候補を内部に保持できても `mixed_unsupported_intent` で
  外部呼出しを零回とする。
- `2020年1月1日と2025年11月1日の個人情報保護法を比較してください。` は、
  一方の日付だけの読取りへ縮約せず `mixed_unsupported_intent` とする。
- `賃金が支払われません。どうすればよいですか。` は法的助言として
  `unsupported` とし、曖昧な取得要求の `needs_clarification` にしない。
- `「比較」を含む条文を検索してください。` は引用句内の `比較` を
  対象外 task にせず、通常の本文検索候補を作る。
- `影響グラフという語を含む条文を検索してください。`、
  `翻訳に関する規定を検索してください。`、
  `差分を説明する規定を検索してください。` および
  `英語に翻訳してくださいという文言を含む条文を検索してください。` は、
  対象外 signal を作らず、明示された採用済み検索だけを候補にする。
- `影響グラフを作成してください。` は
  `unsupported_task_or_resource`、`英語に翻訳してください。` は
  `unsupported_translation` だけを作る。
- `EDINETを検索してください。` は positive search の `direct_task` と
  `EDINET` を subject とする `object_predicate` を両方持つが、
  `unsupported_task_or_resource` により外部呼出しを零回とする。
- `影響グラフを作成してください。民法。` は別の節に裸で現れた法令名候補を
  保持しない。`民法を検索してください。影響グラフを作成してください。` は
  明示された民法検索候補だけを内部に保持する。
- `民法第103条の影響グラフを作成してください。` は同じ節で独立に
  根拠付けられた法令・条文候補を内部に保持できるが、選択または実行しない。
- 対象外 entry の未知 signal、intent group、value、syntax role、および上表の
  閉じた対応との不一致を起動時に拒否する。共通の schema、語彙衝突、
  profile 間再利用および版不整合は `SOT-ENG-030` の固定 test で確認する。
- relation 対応 corpus の coverage ID、最小件数および safety pair は
  `SOT-ENG-026` だけを定義元とし、本規定の確認例と対応する fixture を含める。

## 関連

- [SOT-PROD-011: 統合法情報照会の製品範囲](../00-product/11-unified-legal-query-scope.md)
- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-MODEL-024: LegalQueryResult](../20-model/24-legal-query-result.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](../20-model/25-legal-query-preprocess-result.md)
- [SOT-MODEL-026: QueryProfileContribution](../20-model/26-query-profile-contribution.md)
- [SOT-MODEL-030: CueTaskRelation v2](../20-model/30-cue-task-relation-v2.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-025: 統合照会のパッケージ構成](25-unified-query-package-layout.md)
- [SOT-ENG-026: 統合照会の評価コーパス成果物契約](26-legal-query-corpus-artifact-contract.md)
- [SOT-ENG-030: 統合照会の cue 成果物契約](30-unified-query-cue-artifact-contract.md)
