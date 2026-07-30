# SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界

- 状態: 有効

## 規定

統合照会の公開既定動作は、composition root が起動時に構成する一つの採用済み
意味判定 profile set だけから生成する。次の profile set を準備する実装や成果物を
段階的に追加できるが、採用条件がそろう前に、利用者入力、設定または公開 MCP から
その一部へ到達できるようにしない。

## 採用済み profile set

採用済み profile set は、次のすべてで同じ固定 profile、規則および版を使用する。

- `query_legal_information` を構成する production composition root
- `SOT-ENG-024` の標準評価 command
- 中央品質ゲート
- `SOT-ENG-029` の現行検索例カタログ

一つの binary は、起動時に採用済み profile set を一つだけ構成する。利用者が
CLI、環境変数、設定ファイル、MCP 引数または transport の違いによって、未採用の
意味判定規則、profile version、ranking version または corpus version を選ぶ入口を
設けない。

拡張パックの有効状態は、採用済み profile set の意味認識 contribution を
入れ替えない。pack の意味認識と実行 contribution の境界は `SOT-ARCH-019` に
従い、本規定は pack ごとの tool、binding または route の有効化を再定義しない。

## 準備状態

次の profile set のための実装または成果物は、次のいずれかを満たす場合だけ、
採用前の準備状態として repository に置ける。

1. production 経路でも構造を検証または sidecar を生成するが、その値を候補、
   signal、selection、plan、外部呼出しまたは公開結果の変更に使用せず、
   現行の active cue artifact、profile metadata および profile set version も
   変更しない
2. test が明示的に直接構成する別の内部実装であり、production composition root、
   標準評価 command および中央品質ゲートの既定 profile set へ登録せず、
   次版の cue artifact と profile metadata を現行成果物とは別に保持する
3. 次版の corpus または baseline の候補成果物であり、現行標準から参照しない

準備要素が存在するだけで採用済み profile set とみなさない。共通前処理が
`CueTaskRelation` を不変な sidecar として生成しても、採用済み profile が
relation 依存の signal、候補保持または decision に使用しない間は、relation
依存の意味判定を有効化したことにはならない。

準備状態から到達する test 専用の組立てを、build tag、非公開フラグ、未知の設定
key、環境変数または hidden MCP tool によって production から選択可能にしない。
準備中の実装が不完全な場合も、採用済み profile set を実行時 fallback として
混合したり、照会ごとに二つの profile set を比較実行したりしない。

test 専用の別実装は、同じ package 内の非公開 constructor、testdata または
埋込み fixture を使って次版 profile を直接組み立ててもよい。ただし、
production composition root、現行の埋込み metadata path、標準評価 command、
中央品質ゲートおよび公開 MCP の到達経路と共有してはならない。

準備変更は、現行標準の corpus version と baseline version、公開 decision、
選択した meaning、step、reason、外部呼出し境界および検索例の観測結果を
変更しない。production が使用する cue artifact、profile version または
profile set version も変更せず、`SOT-ENG-033` の current tuple と完全一致させる。

次版の cue 成果物その他の検証済み metadata を変更した場合に、次版の不透明な
profile version または profile set version を更新する義務は維持する。ただし
それらは test が直接構成する別の profile set に属し、原子的な採用まで
production の active artifact を上書きしない。

production が使用する profile version、cue set version または profile set
version を変更する場合は、意味計画の観測結果が同じであっても準備変更ではなく、
`SOT-ENG-033` の新しい tuple、baseline version と digest を含む原子的な
採用変更として扱う。

## 原子的な採用

意味判定の観測動作を変更する profile set は、次のすべてを同じ採用変更で
一致させた場合だけ公開既定値にする。

- production composition root が構成する profile、共通前処理および planner
- profile version、ranking version、composition version および profile set version
- 対象動作を検証する model、profile、planner、application および MCP 契約 test
- `SOT-ENG-024`、`SOT-ENG-026` および `SOT-ENG-036` が要求する corpus、
  baseline、標準 command および変更前後の評価
- 中央品質ゲートの固定値
- `SOT-ENG-029` の現行検索例カタログ
- `SOT-ENG-033` の current adoption tuple と履歴 manifest
- Wiki の実装済み範囲

一部だけを先に公開既定値へ切り替えず、旧 corpus で新 profile を採用したり、
新 corpus と baseline を旧 production profile の標準として宣言したりしない。
採用変更の完了時には、production 経路と標準評価経路が、同じ採用済み
profile set version で構成されなければならない。

過去の corpus、schema および再現に必要な loader は、各定義元の版管理規則に
従って保持する。履歴成果物を保持することは、その profile set を再び利用者が
選択できることを意味しない。

## rollback

採用後に rollback する場合は、「原子的な採用」に列挙した全要素と Wiki の
実装済み範囲を、相互に整合する直前の採用済み集合へ戻す。production
composition root、共通前処理、planner、profile 実装と metadata、cue artifact
と loader、標準 corpus、baseline、標準 command、中央品質ゲートおよび
検索例カタログの一部だけを戻して混在状態を作らない。

直前の採用前から準備状態にあった relation sidecar、test 専用実装または候補成果物
だけを残す場合は、本規定の「準備状態」の全条件を再び満たし、production の
decision、reason、selection、meaning、step および外部呼出し境界に使われない
ことを確認する。

採用済み集合と直前の集合を機械的に特定する tuple、固定 test ID および
準備中成果物との分離は `SOT-ENG-033` に従う。

## 確認

少なくとも次を、composition root、標準評価 command、品質ゲート plan および
文書検査で確認する。

- 準備中の内部実装を CLI、設定、MCP または transport から選べない
- 準備変更の前後で、現行標準 corpus に対する decision、reason、selection、
  meaning、step および外部呼出し境界が一致する
- production composition root と標準評価 command が、同じ採用済み
  profile set version を使用する
- 新しい意味判定の採用時に、対応 corpus、baseline、標準 command、中央品質
  ゲートおよび検索例カタログが同じ変更で切り替わる
- pack の有効状態または transport によって意味認識 profile set が変わらない

## 関連

- [SOT-ARCH-019: 拡張パックの有効化境界](19-extension-pack-activation-boundary.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](22-unified-query-planning-pipeline.md)
- [SOT-ARCH-024: 統合照会の内部境界と公開境界](24-unified-query-internal-public-boundary.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](31-unified-query-intent-evidence-layer.md)
- [SOT-ARCH-032: 統合照会の限定分岐保持](32-unified-query-bounded-branch-retention.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](../50-engineering/24-unified-query-evaluation-gate.md)
- [SOT-ENG-026: 統合照会の評価コーパス成果物契約](../50-engineering/26-legal-query-corpus-artifact-contract.md)
- [SOT-ENG-029: 統合照会の検索例カタログ](../50-engineering/29-unified-query-example-catalog.md)
- [SOT-ENG-033: 統合照会 profile set 採用 manifest](../50-engineering/33-unified-query-profile-set-adoption-manifest.md)
- [SOT-ENG-035: 統合照会 profile metadata 成果物契約](../50-engineering/35-unified-query-profile-metadata-artifact-contract.md)
- [SOT-ENG-036: 統合照会の評価 baseline 成果物契約](../50-engineering/36-unified-query-evaluation-baseline-artifact-contract.md)
