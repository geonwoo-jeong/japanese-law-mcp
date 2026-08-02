# SOT-ENG-041: 候補評価における入力別処理失敗の評価写像

- 状態: 有効

## 規定

内容固定済み候補が、有効な plan fixture を候補所有の前処理または profile 候補回収で処理できず、候補 semantic source closure 内の検出箇所が入力別処理失敗として明示した場合は、候補評価全体を構成失敗にせず、その fixture の plan 不在を定量的な評価失敗として集計する。

## 適用範囲

本規定は `SOT-ENG-038` の候補 semantic source closure を使う exact evaluator にだけ適用する。製品 MCP、公開 CLI、provider、production の profile set、標準採用済み evaluator、corpus schema、期待値、受入基準または report schema を変更しない。

`SOT-ENG-040` は候補 command の閉じた終了 code、report 完成境界および privacy を引き続き定義する。本規定は、有効な一 fixture に対する候補所有段階の処理失敗を、report 完成前の未分類内部 error と区別する範囲だけを定義する。

## 候補所有の入力別処理失敗

evaluator は plan fixture 一件を次の順で処理する。

1. context、semantic case および製品 request を検証する
2. `enabledPacks` から固定 pack state を構築し、corpus と evaluator の pack 境界を検証する
3. 候補 content manifest が固定した preprocessor を実行する
4. 候補 content manifest が固定した profile set から候補 contribution を回収する
5. selector で plan を構築する
6. plan、意味比較、再現性、実行評価、派生観測および report を検証する

三または四の呼出し位置で返った error chain に、候補 semantic source closure 内の検出箇所が付けた明示的な typed marker がある場合だけを、候補所有の入力別処理失敗とする。typed marker は、有効な request に対する誤記候補列挙または profile 候補列挙が固定処理予算を超えた箇所など、その入力だけを処理できないことを判定できる最小箇所で付ける。呼出し全体を一括して包んではならない。

evaluator は内部の閉じた stage 型と typed marker の両方を要求する。どちらか一方だけでは評価失敗へ変換せず、error 文言、prefix、具象 package 名、reflect 情報または任意文字列から分類を推測しない。typed marker を付ける検出箇所を追加または拡張する変更は、本規定との対応と、その error だけが入力別処理失敗であることを固定する test を必要とする。

候補所有の入力別処理失敗では、実際の plan を返さず、期待 plan の meaning を不一致、plan outcome を不一致として保持する有効な semantic case evaluation を返す。ranking の適用可否と分母は元の期待 plan から導出し、fixture を集計から除外しない。plan 再現性でも同じ fixture を分母へ残して不一致とし、execution fixture が参照する場合は外部呼出しを行わず実行評価の失敗として集計する。

この失敗は error 本文、query、fixture、case ID、path または候補内部値を新しい診断 field として report、result、標準出力若しくは標準 error へ追加しない。受入基準を満たさない有効な report は `outcome=failed` の handoff とし、候補を採用できない。候補所有の入力別処理失敗だけを理由に `SOT-ENG-040` の `evaluate_build` へ戻さない。

## hard error の保持

次は候補所有の入力別処理失敗へ変換せず、従来どおり report 完成前の hard error とする。

- nil、取消し済み若しくは期限超過の context、または処理中の取消し若しくは期限超過
- invalid な semantic case、製品 request、corpus、manifest、pack state、evaluator、profile set の初期構成または planning identity
- 前処理または profile 回収段階の generic error、typed marker のない error、validator error、結果不変条件違反、metadata 変更、invalid generation、aggregate または composer の error
- selector の error、invalid plan、case ID 不一致または二回目評価の内部 error
- semantic 比較、execution、派生観測、metric、report constructor または直列化の内部 error
- panic、未分類 error または閉じた stage 集合に属さない error

候補処理より前に pack state を検証し、無効な corpus の pack 境界が同じ入力で発生する候補処理 error に隠れないようにする。hard error の終了 code と privacy は `SOT-ENG-040` に従う。

## evaluator の版

この評価写像を最初に実装する exact version は `legal-query-evaluator-v3` とする。`legal-query-evaluator-v1` と `legal-query-evaluator-v2` は typed marker を評価失敗へ変換せず、従来どおり hard error とする。その error 本文、処理順、metric および report 再現を変更せず、閉じた registry に保持する。

`legal-query-evaluator-v3` の実装と exact routing は、次の候補 request より前の準備成果物として追加できる。実装追加だけでは evaluator の current version を切り替えず、既存の未完了 request、pointer、corpus、holdout、予約 baseline、review attestation または production adoption を変更しない。

この準備成果物は、`SOT-ENG-038` が固定した現行 candidate semantic source closure の file を変更してはならない。実際の preprocessor または profile の検出箇所へ typed marker を追加する変更は candidate content の変更であり、準備成果物へ混在させない。後継 handoff 契約の下で、新しい content manifest、内容固定 review、新しい request および未使用 baseline version と同じ変更系列に置く。

新しい request へ使用する前に、`SOT-ENG-038` の schema version 2 の固定 review SOT 集合を変更せず、その後継となる `SOT-ENG-042` の schema version 3 で、本規定を含む exact review SOT 集合を定義する。schema version 2 の不変成果物と historical replay は `SOT-ENG-038` に保持し、新しい request は `SOT-ENG-039` の段階順および version 3 に対する `SOT-ENG-042` の後継 handoff 境界に従い、新しい corpus、未使用 baseline version、内容固定 review、別 evaluation ID および pointer を準備する。

## 確認

holdout fixture と外部 network を使わない合成 test で、少なくとも次を確認する。

- `candidate-evaluator-v3-case-failure-scoring`
- `candidate-evaluator-v3-pack-state-precedence`
- `candidate-evaluator-v3-cancellation-hard-error`
- `candidate-evaluator-v3-unclassified-hard-error`
- `candidate-evaluator-v3-profile-invariant-hard-error`
- `candidate-evaluator-v3-selector-hard-error`
- `candidate-evaluator-v3-exact-version-routing`

準備成果物では、候補 source の error 型と同じ marker interface を実装する合成 error を使い、同じ typed marker を持つ前処理 error が v1 と v2 では hard error のままであり、v3 だけで plan 不在の評価失敗になることを確認する。profile 候補回収 error も同じ評価値へ写像し、無効 pack、取消し、期限超過、generic 前処理 error、invalid profile generation、selector error および未分類 error は v3 でも hard error になることを確認する。

実際の marker 検出箇所を候補 content へ追加する変更では、その検出箇所から profile set または preprocessor、v1、v2 および v3 evaluator までの実配線 test を追加する。registry と候補 worker は v1、v2 および v3 を exact に分け、alias、range、未知版または current 版への fallback を拒否する。

## 関連

- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-038: 統合照会の内容固定済み候補 holdout 評価 handoff](38-content-bound-candidate-evaluation-handoff.md)
- [SOT-ENG-040: 候補 holdout 評価の閉じた失敗段階診断](40-candidate-evaluation-failure-diagnostics.md)
- [SOT-ENG-042: 候補評価 handoff schema version 3 の世代分離](42-candidate-evaluation-handoff-schema-v3.md)
