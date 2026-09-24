# SOT-ENG-049: 候補 worker の実行隔離と source registry

- 状態: 有効

## 規定

候補評価は、candidate semantic source と exact evaluator の worker source を別々に固定し、
両者の検証済み build input と module archive だけから隔離した実行環境を構成する。

## 限定的な後継範囲

本規定は `SOT-ENG-038` の candidate semantic module だけを raw archive の取得と
fresh module cache の対象にする部分、および worker source registry と再評価の証明境界を
限定的に引き継ぐ。semantic source identity、内容固定 review、予約と消費、一回利用、
privacy、report schema、metric と受入閾値を変更しない。

`SOT-ENG-048` の schema version 5 の exact review 集合に本規定を含める。
既存 schema、request、result、report および実在する固定 registry の byte と意味は保つ。
欠けていた旧 raw registry を現在の source から遡及して作成してはならない。

## worker registry と module の和集合

worker registry は exact `evaluatorVersion` ごとに、全 build-input の
repository-relative path と raw digest の固定順一覧、および worker が必要とする
module metadata を固定する。Go source 以外の選択 build input も対象とし、
source 件数、一件と合計 byte、path、通常 file と digest の境界は `SOT-ENG-038` に従う。
alias、range、未知版、最新版又は current への fallback を認めない。

worker module metadata は `semanticSourceSet.moduleDependencies` と同じ九項目の
閉じた表現、順序と各項目の検証を使用する。candidate の semantic identity に
worker 専用 module を加えない。

raw archive の取得と fresh cache の対象は semantic module 集合と worker module 集合の
和集合だけとする。同じ `modulePath` は version と全 metadata が一致する場合にだけ
一件へ縮約し、不一致を MVS、優先順位又は一方の上書きで解決しない。
和集合にも `SOT-ENG-038` の既存 module 件数、圧縮 byte、展開件数と展開 byte の
各資源上限を適用する。未知 module を download して不足を補わない。

component closure の再計算は `semanticSourceSet` 単独の固定内容と、worker closure の
再計算は worker registry 単独の固定内容と、それぞれ完全一致させる。
source path が candidate と重複する場合も同じ raw digest の一件へ縮約するだけとし、
後から candidate 又は worker を優先して差し替えない。

## 隔離環境の構成と実行

準備段階では閉じた registry metadata、content manifest と root module file だけを読み、
列挙された和集合の raw zip と `.mod` だけを新規の archive staging root へ取得する。
holdout、query、評価期待値又は evaluator の実行に到達しない。
同じ検証済み byte の照合、raw archive の一回だけの新規展開、全 entry の照合と
read-only seal は `SOT-ENG-038` の境界を維持する。ambient module cache、
過去の extracted tree 又は部分 download を再利用しない。

verified source tree は root module file、検証済み semantic source と worker source の
和集合だけから exclusive create し、fresh module cache は前節の検証済み archive
和集合だけから構成する。全 path、size と digest を照合して read-only に seal した後、
全 `go list` と worker build をその tree と fresh cache 上で行う。
元 checkout から build せず、固定された Go toolchain、build context、閉じた環境
allowlist と network 不使用を維持する。一件の binary だけを実行し、終了後にも
source tree と cache の digest と権限を再照合する。
他の安全境界、資源上限、scratch の非公開と出力形式は `SOT-ENG-038` に従う。

## 基盤と初回固定の順序

registry の基盤と隔離 runner の導入は、本番 v4 registry の初回固定から分離する。
本番 v4 registry は development 校正を完了した後、第 4 段階 4 の content・review・
request 固定より前又は同じ原子的準備変更で一回だけ固定する。
暫定の本番 registry を作り、同じ version のまま後で置換してはならない。
未固定 version で新しい holdout 評価を開始できない。
後の source closure 変更は、新しい exact evaluator version の registry を必要とする。

本規定の採用だけでは隔離 runner と registry 基盤の実装完了を意味しない。
未実装部分と実評価開始前の未完了条件は Wiki に追跡する。
合成 fixture による loader、constructor 又は readiness の成功を、未固定の本番
worker closure による実評価の許可へ読み替えない。

## 履歴の検証と再評価

既存 tracked artifact の byte replay と、旧 source closure を使う再評価を区別する。
byte replay は固定 schema、historical digest、request/result/report binding と
保存 byte の一致を検証し、holdout を開かず、現在の source 又は SOT へ再結合しない。
この履歴検証は旧 raw registry が存在しない場合も維持する。

旧 closure による再評価は、その版の原 build input と raw digest を証明できる場合だけ
実行する。証明できない場合は fail-closed とし、現在の source、近似版又は新しい
registry に置き換えない。artifact 履歴検証の成功を、旧 closure による再評価を
実行済みであるかのように表示しない。

## 確認

実装する変更では、合成 source と archive だけで次を確認する。

- semantic と worker module の和集合、重複の完全一致、metadata 衝突と未知 module の拒否
- semantic と worker の独立 closure 照合、重複 source digest の不一致拒否
- same-byte 検証、新規展開と全列挙、read-only seal、verified tree 上の build と実行後の再照合
- 未固定 version の評価拒否、固定 registry の不変性と未知版の拒否
- 旧 raw closure のない再評価拒否と、holdout 非到達の artifact byte replay の維持

schema version 5 の基盤変更では exact review 集合への結合と契約間の整合を検証する。
実行隔離の検証成功は、その実装と対象検証を完了する別変更で記録する。

## 関連

- [SOT-ENG-038](38-content-bound-candidate-evaluation-handoff.md)
- [SOT-ENG-039](39-content-bound-unified-query-rollout-stages.md)
- [SOT-ENG-047](47-candidate-evaluator-v4-source-generation.md)
- [SOT-ENG-048](48-candidate-evaluation-handoff-schema-v5.md)
