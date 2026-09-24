# SOT-ENG-048: 候補評価 handoff schema version 5 の世代分離

- 状態: 有効

## 規定

fresh 意味 holdout を使う後続候補は、`SOT-ENG-046` の corpus schema version 3 と
`SOT-ENG-047` の exact evaluator v4 を、schema version 5 の handoff へ一意に結合する。

## 限定的な後継範囲

本規定は `SOT-ENG-045` の同居 root、版選択、新規 request と readiness の経路を
後継とする。五成果物の field、型、canonical encoding、ID 導出、review rubric、
資源上限、一回利用、privacy と採用接続は `SOT-ENG-038`、導入順序は
`SOT-ENG-039`、入力別処理失敗の計算意味は `SOT-ENG-041` に従う。
完全性と readiness の分離、閉じた stale 理由、error 優先と strict 拒否は
`SOT-ENG-043` を維持し、その世代参照だけを本規定へ接続する。

旧 schema version 2、3、4 の schema、exact SOT 集合、成果物と historical replay の
意味を変更しない。それぞれ `SOT-ENG-038`、`SOT-ENG-042`、`SOT-ENG-045` を
定義元として保つ。report schema と計算方法、全 metric と受入閾値、製品 MCP、
公開 CLI、設定、provider、production tuple、標準 corpus と baseline を変更しない。

## schema と exact binding

候補評価 root は次の正確な十 entry だけを許可する。

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
schema-v5.json
```

`results/` と `failed-reports/` の不在だけを論理的な空として扱い、他は必須とする。
未知 entry、symlink と型違いを拒否する。schema 原 byte 予算は `SOT-ENG-038` の
一件上限の四件分とし、各 schema をその版の固定 byte と照合する。
新 schema は Draft 2020-12 の閉じた document とし、内部 fragment 以外を参照しない。
追加後は旧版と同様に byte を変更、移動、削除または再生成しない。

五 artifact の `schemaVersion` は integer の `5` とし、新しい field を加えない。
request の `evaluatorVersion` は正確に `legal-query-evaluator-v4`、参照する corpus の
schema は正確に `3` とする。corpus version の新規作成下限と派生観測母集団は
`SOT-ENG-038` と `SOT-ENG-026` に従う。constructor、reference validator と worker は
この組合せを検証し、旧 request に corpus schema 3 又は evaluator v4 を結び直さない。
旧世代の replay はその宣言版と既存 exact evaluator を保つ。

loader は各 artifact の制限付き root object から版を判別し、固定された schema、
decoder と validator の組を選ぶ。未知版、版省略、非整数、重複 key、null、
不正 JSON、不正 UTF-8、alias、range、current 又は最新版への fallback を拒否する。
pointer、request、manifest、二件の review と result の参照系列は同一版とし、
一部昇格と cross-generation 参照を拒否する。bootstrap result reader は標準 library
だけで `2`、`3`、`4`、`5` を閉じて判別し、report の版を連動させない。

version 5 の `requiredReviewSOTs` は version 4 の exact 集合に次の四件だけを加え、
`sotId` の byte 昇順に一意に並べた集合とする。

```text
SOT-ENG-046
SOT-ENG-047
SOT-ENG-048
SOT-ENG-049
```

旧集合は書き換えず、link 再帰走査、部分一致又は実装からの推測を使用しない。
新しい二件の独立 review は、この exact 集合の有効な SOT 原 byte digest と同じ
candidate content に結合する。

## 基盤と予約準備の分離

本基盤では、候補の新規 request 用の単一 current evaluator を v4 に進めることだけを、
予約成果物の原子的準備から分離して許可する。これは `SOT-ENG-042` と
`SOT-ENG-045` の current evaluator 切替時点だけの限定的な後継であり、schema 別の
current や fallback を設ける根拠にしない。schema 3 と 4 の新規 request 構築は
固定 evaluator v3 と current v4 の不一致により拒否する。内容 constructor と readiness
の旧版経路は残し、既存 current は自身の版で再構築して stale を保つ。

この current 切替は production と標準 command の evaluator を切り替えない。
result のある旧世代の replay は固定 schema、historical digest と
request/result/report binding だけを使い、現在の source、SOT、constructor 又は
current evaluator へ再結合しない。全世代の全 request を `SOT-ENG-042` の予約履歴へ
含める。置換済み準備も除外せず、同規定の二件の遡及除外を拡大しない。

基盤追加では既存 pointer、corpus、content、review、request、result、report、baseline、
adoption と予約値を変更せず、新しい予約成果物も作成しない。
後続の corpus 作成、development 校正、content・review・request・pointer の原子的準備、
一回評価と production 採用は別作業とする。第 4.4 と第 4.5 の版参照だけを本規定へ
接続し、各段階の独立 commit と権威 CI を待つ順序は `SOT-ENG-039` のままとする。
実評価前の隔離実装と v4 source registry 固定条件は `SOT-ENG-047` と `SOT-ENG-049` に従う。
合成 ready の成功だけではこれらの実行準備を完了したことにならない。

## 確認

実 holdout を開かず、合成 fixture と公開 manifest metadata だけで次を確認する。

- `candidate-evaluation-schema-v5-version-isolation`: 五成果物、exact SOT 集合、evaluator と corpus の組合せ、未知版と cross-generation 拒否
- `candidate-evaluation-schema-v5-history`: 四 schema の固定 byte、旧版 replay と全予約の衝突拒否
- `candidate-evaluation-schema-v5-ready-route`: constructor、reference validator、合成 ready と strict 成功、実 current の stale と strict 拒否
- `candidate-evaluation-schema-v5-bootstrap-result`: 新旧 result の読取り、未知版拒否と report schema 不変

同 commit の入力境界 quality gate は `SOT-ENG-046` を満たし、適用可能なローカル検証と
権威 CI は `SOT-ENG-020` と `SOT-ENG-027` に従う。

## 関連

- [SOT-ENG-026](26-legal-query-corpus-artifact-contract.md)
- [SOT-ENG-038](38-content-bound-candidate-evaluation-handoff.md)
- [SOT-ENG-039](39-content-bound-unified-query-rollout-stages.md)
- [SOT-ENG-041](41-candidate-evaluation-case-failure-mapping.md)
- [SOT-ENG-042](42-candidate-evaluation-handoff-schema-v3.md)
- [SOT-ENG-043](43-candidate-evaluation-readiness-separation.md)
- [SOT-ENG-045](45-candidate-evaluation-handoff-schema-v4.md)
- [SOT-ENG-046](46-fresh-semantic-holdout-corpus-v3.md)
- [SOT-ENG-047](47-candidate-evaluator-v4-source-generation.md)
- [SOT-ENG-049](49-candidate-worker-execution-isolation.md)
