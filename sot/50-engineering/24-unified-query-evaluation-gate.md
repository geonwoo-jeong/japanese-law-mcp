# SOT-ENG-024: 統合照会の評価コーパスと受入基準

- 状態: 有効

## 規定

統合照会の根拠抽出、候補順位、曖昧性処理および実行予算は、版付きの固定日本語コーパスと定量的な受入基準で検証し、基準を満たさない profile を公開既定値へ採用しない。

## コーパス

コーパスは repository 内の人手確認済み fixture と期待 plan で構成し、開発用集合と holdout 集合を分ける。少なくとも次のカテゴリを含める。

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

評価物は次の配置を定義元とする。

```text
testdata/legalquery/
├── corpus-v1/
│   ├── manifest.json
│   ├── development/
│   ├── holdout/
│   └── execution/
└── baselines/
    └── default.json
```

`manifest.json` は corpus version、schema version、固定 seed、必須カテゴリ、各集合の case ID と件数、および fixture の SHA-256 を持つ。case ID は集合間で一意とし、同じ入力または正規化後に同じ入力を development と holdout の両方へ置かない。

holdout は合計二百四十件以上とし、上記の必須カテゴリごとに二十件以上を持つ。複数カテゴリに属する case は各カテゴリ件数へ数えられるが、holdout 全体の最小件数を減らさない。安全境界カテゴリである pack 無効、対象外との混在、非日本語、検索第一件 read 禁止および予算超過は、それぞれ正常に拒否する例と境界を狙う例を含める。

`execution` は fake provider の結果、空結果、部分失敗、全失敗、timeout、順序逆転および item 予算を再現する。外部ネットワークへ接続する fixture を置かない。

## 測定

holdout 集合で少なくとも次を測定する。

| 指標 | 受入基準 |
|---|---:|
| 同じ入力と profile に対する plan の再現率 | `100%` |
| 対象外、pack 無効および明確化例で誤った外部呼出しをしない率 | `100%` |
| `confidence=high` とした候補の precision | `95%` 以上 |
| 意味候補の top-1 accuracy | `90%` 以上 |
| 正解を上位二候補へ含める top-2 recall | `98%` 以上 |
| 誤った resource を実行する率 | `1%` 以下 |
| 候補、呼出し、item および page の固定予算遵守率 | `100%` |
| 検索第一件を暗黙に read する件数 | `0` |
| 空結果後に別 resource へ再分類する件数 | `0` |

カテゴリ別の件数と結果も記録し、全体平均だけで弱いカテゴリを隠さない。コーパスが小さく百分率の一件が大きく変動する場合でも、上限違反、安全境界違反および誤った外部呼出しは一件も許容しない。

## profile の校正

score の重み、閾値、margin、tie-break、根拠コード、辞書または誤記規則を変更する場合は、新しい profile version を割り当てる。

重みと閾値は開発用集合で調整し、holdout 集合は採用判定にだけ使用する。holdout の期待値を調整して実装へ合わせず、誤りであることを独立 review で確認した fixture だけを理由とともに変更する。

候補 score の数値自体を品質指標または確率として扱わない。意味判定の評価と、provider fixture を使う実行予算、partial error および結果順序の評価を分ける。

## 変更の受入れ

すべての指標が基準を満たし、既存カテゴリを基準未満へ後退させない場合だけ、新 profile を既定値にする。安全境界を保つために一部の recall を意図的に下げる場合は、全基準を満たした上で理由と比較結果を同じ変更へ残す。

新しい pack、task、resource、根拠コードまたは辞書 entry を追加する場合は、新カテゴリの fixture と既存全カテゴリの回帰を同じ変更で実行する。

この評価はローカル binary の意味判定を検証するものであり、稼働率収集、外部情報源の運用障害検知または利用者 telemetry を導入しない。

## 標準 command と成果物

repository root から実行する標準 command は次とする。

```text
go run ./cmd/legal-query-eval --corpus=./testdata/legalquery/corpus-v1 --profile-set=default --baseline=./testdata/legalquery/baselines/default.json --format=json
```

command はネットワークを使用せず、固定 seed と repository 内の不変 profile・辞書・fake provider だけを使う。`default` profile set は法令コア、`judicial-cases` 有効時および無効時を manifest の指定どおり評価する。

標準出力は一つの JSON object とし、少なくとも corpus version、profile version 一覧、baseline version、集合別・カテゴリ別件数、各指標の分子・分母・割合、予算違反件数および失敗 case ID を持つ。照会本文、辞書 entry 全体、外部 response または個人情報を出力しない。

引数、schema、checksum、最小件数、baseline、受入基準または再現性のいずれかを満たさない場合は非ゼロ終了する。baseline file は、同じ command の JSON schema に従う review 済みの期待値だけを持ち、command 実行中に書き換えない。

統合照会の application、profile、辞書、planner model、公開 interface、評価 corpus、baseline または evaluator を変更した場合は、この command を `SOT-ENG-020` の中央品質ゲートから実行する。

## 確認

固定 seed、固定 profile、ネットワークを使わない provider fixture および標準 command で評価を再現できることを確認する。manifest の checksum、集合分離、最小件数、baseline、profile version、corpus version、カテゴリ別 score および失敗 case ID を追跡できる形で出力する。

評価 command 自身に、予算超過、誤呼出し、順序の非決定性および holdout 混入を検出するテストを持たせる。

## 関連

- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](../30-architecture/23-unified-query-selection-and-hedging.md)
- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-ENG-004: SOT に結び付く検証](04-sot-linked-verification.md)
- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-023: 統合法情報照会の法概念辞書](23-unified-query-concept-lexicon.md)
