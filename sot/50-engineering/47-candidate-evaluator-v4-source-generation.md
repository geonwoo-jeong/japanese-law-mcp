# SOT-ENG-047: 候補 evaluator v4 の source 世代

- 状態: 有効

## 規定

corpus schema version 3 を読む候補 worker の source 世代は、
`legal-query-evaluator-v4` の exact version で区別する。評価の計算意味は
`SOT-ENG-041` の v3 を継承し、source closure の変更を既存の exact version へ
遡及適用しない。

## 評価の意味と版選択

本規定の版追加は、worker が import する corpus loader の source が
`SOT-ENG-046` により変わることを、`SOT-ENG-038` の version 別 source closure
境界で区別するために行う。前処理と profile 回収の typed な入力別処理失敗、
hard error、request 境界不一致、意味比較、再現性、実行評価、派生観測、metric、
分母および受入閾値は v3 と同一とする。report schema は version 1 のままとする。

registry、constructor と候補 worker は v4 を exact に選択する。
v1、v2、v3 の分岐と既存の評価写像を保持し、alias、range、未知版、current 又は
最新版への fallback を許可しない。製品の標準 evaluator と採用済み production
tuple は切り替えない。候補準備用 current evaluator の切替と handoff 版の
構築境界は `SOT-ENG-048` に従う。

新しい worker 経路は handoff schema version 5、corpus schema version 3 と
exact evaluator v4 を同時に要求する。旧 handoff schema version 2、3、4 は
corpus schema version 1、2 と既存の evaluator の組合せに限り、corpus schema
version 3 又は evaluator v4 を旧世代へ混在させない。旧世代の固定成果物と
result/report の historical replay は従来の identity と評価写像を保持する。

## 実評価前の source 固定

この基盤追加は v4 の実行可能な constructor と version routing を準備する範囲であり、
`SOT-ENG-038` と `SOT-ENG-049` の verified source tree、fresh module cache、immutable な
raw-digest worker registry の完了を意味しない。これらの実行隔離基盤と v4 の
source registry freeze は、後続の隔離実装で満たす実評価前の必須条件とする。
v4 registry の初回 freeze は development による校正を完了した後、新しい
content manifest、review と request の原子的準備より前又は同じ変更で行う。
暫定 registry を先に固定し、同じ v4 のまま後で差し替えてはならない。
元 checkout での `go run` を実評価の適合実行とみなさない。

同じ evaluator version の固定済み closure を変更してよいという例外を設けない。
実在する旧版の固定 closure を保持し、新しい版への
追加でだけ source registry を変更する。旧 raw registry のない場合の履歴検証と
再評価の区別、worker module の分離および初回固定は `SOT-ENG-049` に従う。この基盤では新しい corpus、baseline、
content manifest、review、request、pointer 又は評価 result を作らない。

## 確認

実際の holdout 内容を開かず、合成入力で次を確認する。

- `candidate-evaluator-v4-scoring-equivalence`: v3 と v4 の plan、評価値、入力別
  処理失敗および hard error が同じになる。
- `candidate-evaluator-v4-exact-version-routing`: v1、v2、v3、v4 を exact に
  構築し、旧版の評価写像を保持して未知版と alias を拒否する。
- `candidate-evaluator-v4-corpus-generation-binding`: 新旧の handoff、corpus と
  evaluator の組合せを分離し、混在と未知版を拒否する。

## 関連

- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-038: 統合照会の内容固定済み候補 holdout 評価 handoff](38-content-bound-candidate-evaluation-handoff.md)
- [SOT-ENG-041: 候補評価における入力別処理失敗の評価写像](41-candidate-evaluation-case-failure-mapping.md)
- [SOT-ENG-046: 空入力境界と fresh 意味 holdout の分離](46-fresh-semantic-holdout-corpus-v3.md)
- [SOT-ENG-048: 候補評価 handoff schema version 5 の世代分離](48-candidate-evaluation-handoff-schema-v5.md)
- [SOT-ENG-049: 候補 worker の実行隔離と source registry](49-candidate-worker-execution-isolation.md)
