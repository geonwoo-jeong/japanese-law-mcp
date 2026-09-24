# SOT-ENG-045: 候補評価 handoff schema version 4 の世代分離

- 状態: 有効

## 規定

`SOT-ENG-043` により隔離した候補の後続準備は、既存の schema version 2 と 3 の
不変成果物を保ち、現行の有効 SOT 集合と readiness 境界を固定する schema version 4
の handoff として別世代に準備する。

## 適用範囲

内容固定、canonical encoding、五種類の成果物の field、型、順序、ID 導出、資源上限、
review rubric、一回利用、privacy および採用接続は `SOT-ENG-038`、入力別処理失敗の
評価写像は `SOT-ENG-041`、導入順序は `SOT-ENG-039` を引き続き定義元とする。
本規定は version 4 の exact SOT 集合、三世代同居 root および版選択だけを定義する。
version 2 と 3 の artifact、exact SOT 集合と historical replay はそれぞれ
`SOT-ENG-038` と `SOT-ENG-042` に保持する。

三世代同居 root の集合と loader は本規定を後継の定義元とし、`SOT-ENG-042` の
二世代専用 root 集合を再適用しない。version 4 にも `SOT-ENG-043` の完全性と
readiness の分離、閉じた理由、error 優先、strict 拒否と非到達境界を適用する。
同規定の二世代 schema 検査と routing に関する参照だけは本規定へ接続する。

製品 MCP、公開 CLI、設定、tool、candidate の意味判定、profile や component の意味版、
evaluator の意味、corpus schema、report schema、metric、受入値、production tuple、
標準 corpus と baseline は変更しない。

## schema の配置と版選択

候補評価 root に `schema-v4.json` を追加し、固定済み schema とともに次の正確な
九 entry だけを許可する。

```text
content-manifests/
current.json
failed-reports/
requests/
results/
review-attestations/
schema-v2.json
schema-v3.json
schema-v4.json
```

`failed-reports/` と `results/` の不在は Git が空 directory を保持しない場合だけ
論理的な空とする。それ以外は必須とし、未知 entry、symlink および型違いを拒否する。
root の schema byte 予算は `SOT-ENG-038` の schema 一件上限の三件分とする。
三つの schema 原 byte を各宣言版の固定 byte と照合し、差異を拒否する。
version 4 の schema も Draft 2020-12 の閉じた document とし、内部 fragment 以外を
参照せず、追加後は変更、移動、削除または再生成しない。

version 4 の五 artifact は `schemaVersion` を integer の `4` とする。
version 3 との差分はこの版と後述する exact SOT 集合だけとし、新しい field を加えない。
loader と decoder は制限付き root object の `artifactKind` と `schemaVersion` から
一 artifact ごとに固定の schema と validator を選ぶ。未知版、版省略、非整数、重複 key、
`null`、不正 JSON、不正 UTF-8、近似版、current 又は最新版への fallback を拒否する。

pointer、request、manifest、二件の attestation と result の参照系列は同一版とし、
一部だけの昇格および cross-generation 参照を拒否する。bootstrap の result reader も
標準 library だけで `2`、`3`、`4` を閉じて判別する。report の schema version は
`SOT-ENG-036` のままとし、handoff 版へ連動させない。

## evaluator と exact review 集合

version 4 の request は `legal-query-evaluator-v3` に exact binding する。
`SOT-ENG-041` の評価写像を変えないため evaluator の新しい意味版を作らず、
新規 request の current evaluator がこの exact 値でない場合は構築を拒否する。
historical registry の alias、range、未知版または current への fallback を許可しない。

version 4 の `requiredReviewSOTs` は、`SOT-ENG-042` が固定した version 3 の集合から
`SOT-IF-040` 一件だけを除き、次の三件を加えて `sotId` の byte 昇順へ一意に並べた
57 件とする。これを version 4 の唯一の exact 集合とする。

```text
SOT-ENG-043
SOT-ENG-045
SOT-IF-077
```

`SOT-IF-077` は廃止された pack 有効化契約の後継、`SOT-ENG-043` は readiness、
本規定は version 4 の世代境界を review 内容へ固定するために含める。
元の集合のほかの有効 SOT は引き続き適用する。link 再帰走査、実装からの推測、
部分一致または旧集合の直接書換えで増減しない。二件の新しい review は exact 集合の
有効な SOT 原 byte digest と同じ新しい candidate content に結合する。

## constructor、readiness と履歴

constructor は schema 世代を明示して manifest の canonical identity を作り、
version 3 の構築結果の版を後から `4` へ上書きしない。既存の version 3 constructor
経路を保持し、version 4 は明示した別経路で構築する。review、request と result は
参照する親 artifact の版を継承し、参照系列の版一致を検証する。

未評価 current の readiness は宣言された version 3 又は 4 の経路で source と manifest、
exact SOT 集合と evaluator を再構築する。version 3 の current を version 4 の集合や
manifest と比較せず、廃止 ID を含む既存 current は `SOT-ENG-043` の stale 拒否を保つ。
version 2 の新規準備は再開せず、固定済み成果物の integrity と historical replay を保つ。
未知版又は再構築不能を stale として受理しない。

result のある version 2、3、4 の replay は、各固定 schema、request の historical digest、
request/result/report binding と exact evaluator registry だけを使う。現在の source、
SOT、constructor 又は current evaluator へ再結合しない。
三世代の全 request を `SOT-ENG-042` の予約履歴へ含め、置換済み準備も除外しない。
同規定の version 2 二件だけの遡及除外を拡大せず、世代変更で予約値を未使用へ戻さない。

## 基盤と後続準備の境界

本規定と schema、loader、constructor、reference validator および合成検証の追加は、
新しい予約成果物を作らない独立した基盤変更とする。既存 `current.json` の byte、
schema version 2 と 3、manifest、review、request、result、report、履歴と予約値を保つ。
この基盤だけでは `SOT-ENG-039` の第 4 段階 4 は完了せず、評価を開始できない。

後続の corpus 作成、第 4 段階 3 の development による校正、第 4 段階 4 の
新しい content manifest・独立 review 二件・request・pointer の原子的準備、
第 4 段階 5 の一回評価および production 採用を別作業とする。
version 4 の第 4 段階 4 と 5 は本規定の世代分離に接続し、各番号の独立 commit と
その commit の権威 CI 成功を待つ順序は `SOT-ENG-039` のままとする。
予約済みの corpus、baseline、holdout と leakage group の再利用禁止は
`SOT-ENG-043` に従う。

## 確認

holdout fixture と外部 network を使わず、必要最小限の合成契約で次を確認する。

- `candidate-evaluation-schema-v4-version-isolation`: 五 artifact、exact SOT 集合と evaluator、未知版と版省略の拒否
- `candidate-evaluation-schema-v4-mixed-root`: 三 schema の byte 固定、未知 entry と cross-generation 参照の拒否
- `candidate-evaluation-schema-v4-historical-isolation`: version 2 と 3 の replay byte、現在鮮度の非参照、置換済み履歴の予約保全
- `candidate-evaluation-schema-v4-ready-route`: 明示 constructor と reference validator、合成 ready の strict load 成功
- `candidate-evaluation-schema-v4-bootstrap-result`: version 2、3、4 の result 読取りと未知版拒否

実 repository は `SOT-ENG-043` の既存 current integrity と stale 拒否契約を通し、
holdout reader 又は評価 worker へ到達しない入口で確認する。
適用するローカル検証と権威 CI は `SOT-ENG-020` と `SOT-ENG-027` に従う。

## 関連

- [SOT-ENG-038](38-content-bound-candidate-evaluation-handoff.md)
- [SOT-ENG-039](39-content-bound-unified-query-rollout-stages.md)
- [SOT-ENG-041](41-candidate-evaluation-case-failure-mapping.md)
- [SOT-ENG-042](42-candidate-evaluation-handoff-schema-v3.md)
- [SOT-ENG-043](43-candidate-evaluation-readiness-separation.md)
- [SOT-IF-077](../40-interfaces/77-mcp-tool-exposure-and-extension-packs.md)
