# SOT-ENG-024: 統合照会の評価コーパスと受入基準

- 状態: 有効

## 規定

統合照会の根拠抽出、候補順位、曖昧性処理および実行予算は、版付きの固定日本語コーパスと定量的な受入基準で検証し、基準を満たさない profile を公開既定値へ採用しない。

## コーパス

コーパスは repository 内の人手確認済み fixture と、入力境界、意味候補または実行結果の期待値で構成し、開発用集合と holdout 集合を分ける。少なくとも次のカテゴリを含める。

- 公式の法令 ID、リビジョン ID、法令番号、事件参照および `SourceResourceRef`
- 正式法令名、公式略称、出典付き別名および法概念
- 法令本文検索、法令読取り、条文読取り、更新一覧、裁判例検索および裁判例読取り
- 完全な日付、条、項および複数の明示意図
- 表記揺れ、空白差、挿入、削除、置換および隣接文字転置
- 衝突する略称、複数候補へ対応する概念、弱い一般語および三候補以上の曖昧入力
- `judicial-cases` の有効時と無効時
- 法的助言、翻訳、未採用 pack、対象外 resource および入力上限を狙う adversarial 例
- 空結果、部分失敗、全失敗、timeout および完了順が逆転する実行 fixture

利用者の実照会、診断出力または実行履歴を収集してコーパスへ自動追加しない。必要な自然文例は、個人情報を含まない手作業の fixture として review する。

## 配置と最小規模

評価コーパスの配置、schema、manifest、fixture、checksum および loader の成果物契約は `SOT-ENG-026` に従う。baseline は `testdata/legalquery/baselines/default.json` に置く。

case ID は集合間で一意とし、同じ入力、正規化後に同じ入力または同じ発話・法的対象の変形群を development と holdout の両方へ置かない。

holdout は合計二百四十件以上とし、上記の必須カテゴリごとに二十件以上を持つ。複数カテゴリに属する case は各カテゴリ件数へ数えられるが、holdout 全体の最小件数を減らさない。安全境界カテゴリである pack 無効、対象外との混在、非日本語、検索第一件 read 禁止および予算超過は、それぞれ正常に拒否する例と境界を狙う例を含める。

`execution` は fake provider の結果、空結果、部分失敗、全失敗、timeout、順序逆転および item 予算を再現する。外部ネットワークへ接続する fixture を置かない。

profile 横断合成の受入では、既存 holdout fixture の期待 decision、
reasonCodes、selected meaning、step 順および requiredPacks から観測できる
性質だけを使って次を評価する。

- core と pack の混合意味が pack 有効時に実行候補となること
- 同じ意味が pack 無効時に `capability_unavailable` となること
- `ref` read と検索の混合が一つの meaning として観測できること
- 四 step の混合候補が budget 内で保持されること

同じ位置 tie-break または不正 member origin のように holdout 期待値へ直接
投影しない性質は、ranking 指標へ混ぜず model test または architecture test で
検証する。holdout digest を変えずに観測できる受入条件だけを baseline の対象と
する。

## 測定

holdout 集合で少なくとも次を測定する。

| 指標 | 受入基準 |
|---|---:|
| 同じ入力と profile に対する plan の再現率 | `100%` |
| 期待する decision、reason および selection の一致率 | `100%` |
| 意味と一致した候補の根拠・概念 assertion 適合率 | `100%` |
| 対象外、pack 無効および明確化例で誤った外部呼出しをしない率 | `100%` |
| `confidence=high` とした候補の precision | `95%` 以上 |
| 意味候補の top-1 accuracy | `90%` 以上 |
| 正解を上位二候補へ含める top-2 recall | `98%` 以上 |
| 誤った resource を実行する率 | `1%` 以下 |
| 候補、呼出し、item および page の固定予算遵守率 | `100%` |
| 検索第一件を暗黙に read する件数 | `0` |
| 空結果後に別 resource へ再分類する件数 | `0` |

カテゴリ別の件数と結果も記録し、全体平均だけで弱いカテゴリを隠さない。コーパスが小さく百分率の一件が大きく変動する場合でも、上限違反、安全境界違反および誤った外部呼出しは一件も許容しない。

## 測定母集団と正解判定

ranking 指標の母集団は、holdout のうち `kind=plan` で一件以上の正解 meaning を持ち、期待 decision が `single`、`hedged`、`needs_clarification` または `capability_unavailable` である case とする。`request_error` と、正解 meaning を持たない `unsupported` は ranking 指標から除外し、入力境界、安全な非実行および誤呼出しの指標で評価する。

semantic meaning の一致は `SOT-ENG-026` の意味署名だけで判定し、根拠または概念 assertion の成否を top-1、top-2 または high-confidence の正解判定へ混在させない。期待 `meanings` の先頭を主正解とし、次の式を使う。

- top-1 accuracy: 実際の順位一位が主正解と一致した case 数 / ranking 母集団の case 数
- top-2 recall: 実際の上位二候補以内に主正解を含む case 数 / ranking 母集団の case 数
- high-confidence precision: 実際の順位一位が `confidence=high` である case のうち主正解と一致した件数 / 実際の順位一位が `confidence=high` である case 数

high-confidence の分母が零件の場合は基準を満たしたと扱わず、profile 採用を失敗させる。複数の正しい解釈、hedged および明確化 case でも、fixture が主正解を一件定め、decision、理由、選択した meaning の完全一致は ranking 指標と別に検査する。期待 meaning に対する根拠コードと概念 ID の assertion も別に集計する。

全体指標は case を一件ずつ数える micro 集計とする。一 case が複数カテゴリに属する場合は各カテゴリ内で一回ずつ数えるが、全体集計では一回だけ数える。カテゴリ別に同じ分子、分母および割合を出力し、カテゴリ平均を全体の受入判定へ置き換えない。

## profile の校正

score の重み、閾値、margin、tie-break、根拠コード、辞書、誤記規則、selection mode または hedge pair の生成規則を変更する場合は、新しい profile version を割り当てる。複数 profile 間の score scale、confidence、閾値、margin または tie-break を変更する場合は、新しい ranking version も割り当てる。

重みと閾値は開発用集合で調整し、holdout 集合は採用判定にだけ使用する。公開 repository の holdout は秘密試験ではなく、固定 digest と変更履歴で過適合を監査する集合とする。profile、重み、閾値、辞書または誤記規則を変更する変更では、既存 holdout fixture の request、期待値、coverage、`leakageGroupId` または集合所属を同時に変更しない。

holdout の期待値を実装へ合わせて調整しない。fixture の誤りであることを独立 review で確認した場合だけ、理由、新しい corpus version および holdout digest を同じ変更へ残す。その変更では profile を変更せず、変更前後の corpus に対する評価結果を残す。

新しい有効な SOT が要求する未収録 coverage は、profile の採用変更より前の
corpus 準備変更で、新しい case と新しい corpus version へ追加する。この準備変更
では profile、重み、閾値、辞書および誤記規則を変更せず、既存 holdout case を
書き換えない。新規 case の正解、集合分離および coverage を独立 review し、
holdout digest を固定してから、その holdout を採用判定に一回だけ使用する。

候補 score の数値自体を品質指標または確率として扱わない。意味判定の評価と、provider fixture を使う実行予算、partial error および結果順序の評価を分ける。

## 変更の受入れ

すべての指標が基準を満たし、既存カテゴリを基準未満へ後退させない場合だけ、新 profile を既定値にする。安全境界を保つために一部の recall を意図的に下げる場合は、全基準を満たした上で理由と比較結果を同じ変更へ残す。

新しい pack、task、resource、根拠コードまたは辞書 entry を追加する場合は、新カテゴリの fixture と既存全カテゴリの回帰を同じ変更で実行する。

公開既定実装または標準評価成果物の観測挙動を変更する場合は、影響する fixture、
baseline および `SOT-ENG-029` が定める検索例カタログを同じ変更で更新し、
評価結果と説明文の乖離を残さない。将来の理想状態を SOT だけで採用する変更は、
現行確認済みカタログへ先行して掲載せず、実装差分を Wiki で追跡する。

採用済み profile set と準備状態の到達性、および production と標準評価を
同時に切り替える単位は `SOT-ARCH-033` に従う。本規定は、その採用時に必要な
評価成果物と受入基準を定義する。

この評価はローカル binary の意味判定を検証するものであり、稼働率収集、外部情報源の運用障害検知または利用者 telemetry を導入しない。

## 標準 command と成果物

repository root から実行する標準 command を導入する場合、その固定引数は次とする。

```text
go run ./cmd/legal-query-eval --corpus=./testdata/legalquery/corpus-v9 --profile-set=default --baseline=./testdata/legalquery/baselines/default.json --format=json
```

標準 command は、独立 review で期待値を訂正した現行の `corpus-v9` と
review 済みの `default-1` baseline を使用する。過去の corpus version は
再現用成果物として保持するが、標準 command の採用判定へ使用しない。
標準 corpus を更新する場合は、過去版を変更せず新しい corpus version と
holdout digest を割り当て、変更前後の結果と独立 review を同じ変更で残し、
標準 command と baseline を同時に更新する。

`SOT-MODEL-030` の cue task relation を公開既定の意味判定へ初めて有効化する
変更では、次を一つの整合した採用集合として完成させ、同じ採用変更で
production、標準 command、中央品質ゲートおよび検索例カタログの参照先を
切り替える。

- `SOT-ENG-030` に従う固定 profile set 全体の relation 対応 cue schema と
  loader test
- `SOT-ENG-028` の relation、対象外意図および候補 scope の model・profile test
- `SOT-ENG-026` の corpus schema version 2、`corpus-v10`、relation、
  共有末尾 cue、単純列挙および非制限 fan-out を対象とする追加 coverage
- `baselineVersion=default-2` の review 済み baseline
- `SOT-ENG-033` の current adoption tuple と履歴 manifest
- 標準 command、検索例カタログおよび中央品質ゲートの固定値切替
- 変更前後の評価結果と独立 review

corpus schema version 2、`corpus-v10` および `default-2` の候補成果物は、
`SOT-ARCH-033` の準備状態として採用変更より前に repository へ追加できる。
特に新しい holdout case は前項の corpus 準備変更で独立 review と digest 固定を
終える。採用変更は準備済み成果物を現行標準として選ぶ参照先を原子的に
切り替えるものであり、profile の結果へ合わせて同じ変更で holdout を
書き換えることを許可しない。

候補 `default-2` は `SOT-ENG-033` に従い、test が直接構成する次版 profile set で
生成する。現行標準 command の `default` から次版を選択できるようにせず、
採用変更では候補 baseline の byte と digest を変更しない。

`SOT-ARCH-033` の準備状態として、固定 profile set 全体の cue artifact と
loader を schema version 3 へそろえることはできる。この準備変更では relation
依存の signal、候補保持または公開 decision を有効にせず、標準 command、
`corpus-v9`、`default-1` および中央品質ゲートの corpus・baseline 固定値を
変更しない。test が直接構成する次版 profile set の metadata を変更した場合に、
その次版の不透明な profile version と profile set version を更新する義務は
妨げない。production の active version は `SOT-ARCH-033` に従い維持する。

上記の採用成果物がそろうまでは `corpus-v9` と `default-1` を現行標準として
維持する。profile または loader の限定 test だけの成功を、relation 対応
profile set の採用判定にしない。

command はネットワークを使用せず、固定 seed と repository 内の不変 profile・辞書・fake provider だけを使う。`default` profile set は法令コア、`judicial-cases` 有効時および無効時を manifest の指定どおり評価する。

標準出力は一つの JSON object とし、少なくとも corpus version、holdout digest、profile version 一覧、baseline version、集合別・カテゴリ別件数、各指標の分子・分母・割合、予算違反件数および失敗 case ID を持つ。照会本文、辞書 entry 全体、外部 response または個人情報を出力しない。

引数、schema、checksum、最小件数、baseline、受入基準または再現性のいずれかを満たさない場合は非ゼロ終了する。baseline file は、同じ command の JSON schema に従う review 済みの期待値と holdout digest を持ち、manifest と一致しなければならない。command 実行中に baseline を書き換えない。

統合照会の application、profile、辞書、planner model、公開 interface、評価 corpus、baseline または evaluator を変更した場合は、この command を `SOT-ENG-020` の中央品質ゲートから実行する。

## 確認

固定 seed、固定 profile、ネットワークを使わない provider fixture および標準 command で評価を再現できることを確認する。manifest の checksum、集合分離、最小件数、baseline、profile version、corpus version、カテゴリ別 score および失敗 case ID を追跡できる形で出力する。

評価 command 自身に、予算超過、誤呼出し、順序の非決定性および holdout 混入を検出するテストを持たせる。

## 関連

- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](../30-architecture/23-unified-query-selection-and-hedging.md)
- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](../30-architecture/33-unified-query-profile-set-adoption-boundary.md)
- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-ENG-004: SOT に結び付く検証](04-sot-linked-verification.md)
- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-023: 統合法情報照会の法概念辞書](23-unified-query-concept-lexicon.md)
- [SOT-ENG-026: 統合照会の評価コーパス成果物契約](26-legal-query-corpus-artifact-contract.md)
- [SOT-ENG-029: 統合照会の検索例カタログ](29-unified-query-example-catalog.md)
- [SOT-ENG-030: 統合照会の cue 成果物契約](30-unified-query-cue-artifact-contract.md)
- [SOT-ENG-033: 統合照会 profile set 採用 manifest](33-unified-query-profile-set-adoption-manifest.md)
