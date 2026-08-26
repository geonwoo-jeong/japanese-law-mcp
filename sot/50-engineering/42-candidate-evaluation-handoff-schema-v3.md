# SOT-ENG-042: 候補評価 handoff schema version 3 の世代分離

- 状態: 有効

## 規定

`SOT-ENG-041` の入力別処理失敗写像を実際の候補 source へ接続する候補評価は、
既存の schema version 2 成果物を変更せず、`legal-query-evaluator-v3` と候補内容を
一意に結ぶ schema version 3 の handoff として別世代に準備する。

## 適用範囲

本規定は `SOT-ENG-038` の内容固定、review、request、result、一回利用、privacy、
採用接続および資源上限を維持したまま、schema version 3 へ移行する差分だけを定義する。
schema version 2 の不変な schema、pointer、manifest、attestation、request、result、
report および historical replay は引き続き `SOT-ENG-038` を定義元とする。
schema version 3 の導入後に version 2 と version 3 が同居する root、loader、版選択および
closed-artifact 集合は、本規定だけを後継の定義元とする。`SOT-ENG-038` の version 2 専用
root 集合を mixed-generation root へ再適用しない。

製品 MCP、公開 CLI、provider、production profile set、標準 evaluator、corpus schema、
metric、受入基準、report schema、公開 error または adoption tuple を変更しない。

## schema の配置と版選択

候補評価 root へ固定済み `schema-v2.json` と並べて `schema-v3.json` を一件追加する。
両 file は JSON Schema Draft 2020-12 の閉じた schema とし、同じ document 内の fragment
以外を参照しない。追加後はどちらも byte を変更、移動、削除または再生成しない。

schema version 3 の導入時点では、`SOT-ENG-038` の schema version 2 専用 root entry
集合を、次の正確な八 entry を許可する集合へ置き換える。

```text
content-manifests/
current.json
failed-reports/
requests/
results/
review-attestations/
schema-v2.json
schema-v3.json
```

このうち `content-manifests/`、`current.json`、`requests/`、`review-attestations/` および
二つの schema file は必須とし、`failed-reports/` と `results/` は Git が空 directory を
保持しない場合だけ不在を論理的な空として扱う。これ以外の file、directory、symlink
または型の違う entry を拒否する。version 2 の loader と replay は、固定 byte と一致する
`schema-v3.json` が同じ root に存在することだけを未知 entry として拒否せず、意味検証には
`schema-v2.json` だけを使用する。version 3 の loader と replay も同様に、固定済み
`schema-v2.json` の存在を要求するが、意味検証には `schema-v3.json` だけを使用する。

## 導入段階への接続

schema version 3 に限り、本規定は `SOT-ENG-039` の第 4 段階 4 と 5 にある
`SOT-ENG-038` への handoff 参照を置き換える。第 4 段階 4 の候補準備は本規定の
「候補内容と準備単位」、第 4 段階 5 の一回評価は本規定の世代分離と
`SOT-ENG-038` の handoff、一回利用および privacy を合わせて適用する。第 1 段階から
第 7 段階までの順序、各段階の公開境界、進行条件および第 5 段階の原子的採用は
`SOT-ENG-039` のままとし、schema version 2 の第 4 段階 4 と 5 も変更しない。

schema version 3 の pointer、candidate content manifest、review attestation、evaluation
request および evaluation result は、`schemaVersion` を除き `SOT-ENG-038` の schema
version 2 と同じ field、型、順序、canonical encoding、件数上限および ID 導出を使用する。
全 artifact の `schemaVersion` は integer の `3` とする。入力別処理失敗の policy 名、
error 型名、検出 package または任意文字列を新しい field として重複して持たない。
その意味版は `evaluatorVersion`、実装内容は candidate の `semanticSourceSet`、規範内容は
後述する `requiredReviewSOTs` へそれぞれ一度だけ結合する。

loader は pointer と各 directory 内の一 artifact ごとに、byte、深さおよび値数を制限した
root object から `artifactKind` と `schemaVersion` だけを fail-closed に判別する。その後、
version 別の固定 table から schema、decoder および validator を一組だけ選ぶ。
schema 判別前にも重複 key、`null`、不正 UTF-8、不正 JSON および整数でない version を
拒否する。未知版、版の省略、`2` と `3` の混在、近似版、最新版または current 版への
fallback を許可しない。一つの pointer が選んだ schema を directory 内の全履歴へ一括適用
せず、version 2 と version 3 の不変成果物を各 file の宣言版で検証する。constructor も
同じ version 別 table を使い、version 2 の field へ version 3 の値を後から上書きしない。

`current.json` が schema version 2 なら current binding は version 2 の request 系列だけを、
schema version 3 なら current binding は version 3 の request 系列だけを読む。pointer、
その pointer が参照する request、manifest、二件の attestation および result は同じ
schema version でなければならず、cross-version の参照または一部だけの昇格を拒否する。
同じ root にある別世代の historical request と result は無視せず、各宣言版で検証して
世代横断の一回利用検査と replay へ含める。version 2 の historical replay は version 3 の
current SOT、current evaluator または constructor を参照せず、固定済み version 2 の
schema と exact evaluator registry だけで再現する。version 3 の historical replay も、
固定済み `schema-v3.json`、request が保持する historical digest、request/result binding
および exact evaluator registry だけで再現し、将来の schema、SOT、constructor、current
evaluator、alias、range または fallback を参照しない。

## evaluator と review の固定

schema version 3 の新しい request は、`evaluatorVersion` を正確に
`legal-query-evaluator-v3` とする。version 1、version 2、未知版、range、alias または
current version への fallback を許可しない。`legal-query-evaluator-v3` が新規 request
用の current evaluator でない間は version 3 request を構築せず、既存 version 2 request
の replay だけを許可する。

schema version 3 の `requiredReviewSOTs` は、`SOT-ENG-038` が schema version 2 に
固定した exact SOT ID 集合へ、次の三件を加え、`sotId` の byte 順に一意に並べた集合と
する。

```text
SOT-ENG-040
SOT-ENG-041
SOT-ENG-042
```

version 2 の集合は変更しない。version 3 の二件の review attestation は、この exact
集合の各 SOT 原 byte digest、同じ candidate content manifest および同じ rubric digest
へ新しく結合する。旧 attestation の再利用、集合の部分一致、廃止 SOT への置換、Markdown
link の再帰走査または実装 file からの SOT 推測を許可しない。

## 候補内容と準備単位

実際の typed marker は `SOT-ENG-041` が許可する候補所有の最小検出箇所だけへ追加する。
その source file は version 3 candidate content manifest の `semanticSourceSet` に含め、
marker の追加前に作成した content manifest、review または request を再利用しない。
marker の interface、evaluator の stage 判別、error 文言または package 名を候補 content
identity の代わりにしない。

最初の version 3 評価準備は、次を一つの原子的な準備変更にする。

1. typed marker の実検出箇所と `SOT-ENG-041` の実配線 test
2. 必要な profile または component の意味版更新
3. candidate evaluation の新規 request constructor が使う current evaluator の
   `legal-query-evaluator-v3` への exact 切替
4. schema version 3 の新しい candidate content manifest
5. architecture と testability の新しい二件の review attestation
6. 未使用 corpus、holdout、leakage group digest および baseline version を結ぶ新しい request
7. その request だけを指す schema version 3 の `current.json`

三の current evaluator は新しい candidate evaluation request の構築時だけを指す。
標準 command と production adoption は adoption manifest が保持する exact evaluator を
引き続き使用し、candidate constructor の current 切替を標準または production の evaluator
切替へ流用しない。

同じ準備変更で holdout、report、result、baseline version file、production metadata、
adoption history、標準 command、MCP、CLI、設定または transport を実行若しくは切り替えない。
対象 commit の権威 CI と独立 review が成功した後にだけ、別の一回実行として候補評価を
起動する。

version 3 の marker 判定または scored failure への写像を説明するために、marker の型名、
package、error 本文、query、fixture、case ID、path、候補内部値または一件別の失敗理由を
標準出力、標準 error、log、report、result若しくは新しい診断 field へ追加しない。
report 完成前後の出力と終了 code は `SOT-ENG-040`、評価値への写像は `SOT-ENG-041` に従う。

## 一回利用と履歴

version 3 の repository preflight は、version 2 と version 3 の全ての不変 request を
件数と合計 byte の上限内で読み、current 以外の各 request が持つ evaluation ID、baseline
version、holdout digest および leakage group digest を、result や failed report の有無に
かかわらず予約済み履歴として扱う。result がある request は request/result binding と
report を、failed report がある request は request/report binding を追加で検証する。
pointer から外れ result も failed report もない置換済み準備も履歴から除外しない。

current request は schema version にかかわらず各過去 request との衝突を一件ずつ拒否する。
唯一の遡及除外は、schema version 3 の導入前に `SOT-ENG-038` の current と置換済み準備
として有効だった、次の schema version 2 evaluation ID 二件の組だけとする。

```text
evaluation-sha256-1001bab1bab4c88533769e89e5ad7a4aed78e043239344b67a6d450b41adfdbd
evaluation-sha256-398e801b2d7edd6068f36fa34fe94827d7d44891d59976fdc8630e4d5be7e89c
```

evaluation ID は request の canonical tuple 全体を固定するため、同じ corpus、evaluator または
baseline の属性だけを名乗る別 request を遡及除外しない。この二 request は既存の一回利用
規則に従って同じ holdout 予約値を保持する。repository preflight は非 current の過去 request
同士を再比較せず、これ以外の version 2 または version 3 current と全過去 request の衝突を
拒否する。

権威 CI の workflow 記録は repository artifact loader の入力へ加えず、一回実行を開始する
dispatch gate が evaluation ID と対象 commit の既存 run を確認する。workflow log、holdout
内容または任意 path を request、result若しくは report へ複製しない。repository preflight
と dispatch gate のどちらか一方だけで一回利用を満たしたとは扱わない。

過去に消費または廃棄した evaluation ID、baseline version、holdout digest および leakage
group digest を schema version の変更で未使用に戻さない。`SOT-ENG-040` の report 完成前
失敗または不確定終了に対する再試行条件も世代間で同一に適用する。

新しい version 3 request は、現在 pointer が指す未完了 version 2 request を変更せず、
別 ID と別予約値で置き換える。旧 pointer から外れた request、manifest、attestation、
失敗記録および権威 CI 記録を削除、補正または version 3 へ変換しない。

## 確認

holdout fixture と外部 network を使わない contract test で、少なくとも次を確認する。

- `candidate-evaluation-schema-v3-version-isolation`
- `candidate-evaluation-schema-v2-replay-with-schema-v3-present`
- `candidate-evaluation-schema-v3-load-with-schema-v2-present`
- `candidate-evaluation-schema-v3-cross-version-reference-rejection`
- `candidate-evaluation-schema-v3-exact-evaluator-binding`
- `candidate-evaluation-schema-v3-current-switch-production-neutral`
- `candidate-evaluation-schema-v3-review-sot-set`
- `candidate-evaluation-schema-v3-marker-content-binding`
- `candidate-evaluation-schema-v3-cross-version-consumption-history`
- `candidate-evaluation-schema-v3-superseded-request-reservation`
- `candidate-evaluation-schema-v3-dispatch-history-gate`
- `candidate-evaluation-schema-v2-historical-replay`
- `candidate-evaluation-schema-v3-future-version-replay-isolation`
- `candidate-evaluation-schema-v3-production-unreachable`

version 判別の正常系、未知版、版省略、重複 key、整数でない版、pointer と request の版差、
request と manifest、attestation または result の版差、version 2 evaluator への downgrade、
version 3 evaluator の exact routing、旧 digest と baseline の再利用拒否、および current
production identity の不変を確認する。実 marker の検出箇所から preprocessor または
profile、version 1、version 2 および version 3 evaluator までの実配線は
`SOT-ENG-041` の固定検証と同じ候補準備変更で確認する。

## 関連

- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-027: 省資源の段階的検証](27-resource-aware-verification-stages.md)
- [SOT-ENG-038: 統合照会の内容固定済み候補 holdout 評価 handoff](38-content-bound-candidate-evaluation-handoff.md)
- [SOT-ENG-039: 内容固定済み候補による統合照会の導入段階と変更順序](39-content-bound-unified-query-rollout-stages.md)
- [SOT-ENG-040: 候補 holdout 評価の閉じた失敗段階診断](40-candidate-evaluation-failure-diagnostics.md)
- [SOT-ENG-041: 候補評価における入力別処理失敗の評価写像](41-candidate-evaluation-case-failure-mapping.md)
- [SOT-ENG-043: 候補評価準備の鮮度と製品品質ゲートの分離](43-candidate-evaluation-readiness-separation.md)
