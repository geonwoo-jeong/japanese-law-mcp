# SOT-ENG-040: 候補 holdout 評価の閉じた失敗段階診断

- 状態: 草案

## 規定

内容固定済み候補の holdout 評価が有効な handoff を作れず終了した場合は、
query、fixture、case ID、error 本文、path または資格情報を外部へ出さず、
閉じた終了 code 集合だけで失敗段階を示す。

## 適用範囲

本規定は `SOT-ENG-038` の CI 専用 command、その bootstrap、候補評価 worker
および同 command を実行する権威 CI にだけ適用する。製品 MCP、標準評価 command、
candidate report/result schema、採用判定、`outcome=passed`/`failed` の意味または
一般利用者向け error を変更しない。

`SOT-ENG-038` は one-time use、artifact 配置、retry 条件および handoff の意味を
定義元とする。本規定は、その非零終了時に何を log へ出せるか、どの段階までを
公開診断へ縮約するか、および report 完成境界の前後をどの code 群に写像するかだけを
定義する。

## report 完成境界

worker が report byte を構成した後、同じ byte を標準 report として復元し、
受入判定関数が `error=nil` を返せた時点を、構造上有効な report の完成境界とする。
受入基準未達は `error` にせず、有効な `outcome=failed` として継続する。

完成境界より前の失敗では result を作らない。完成境界より後の result 結合、
result 再検証、handoff 書込または bootstrap 読戻しの失敗では、有効な report が
既に完成したものとして扱う。tracked replay は既存の有効 report/result を
再現する段階であり、新しい holdout 実行ではない。

## 終了 code

候補 command の終了 code は次の閉じた集合とする。

| code | 段階 | report 完成 | 意味 |
|---:|---|---|---|
| `0` | `completed` | はい | 有効な report/result の生成、または追跡済み byte の再現 |
| `1` | `bootstrap_validation` | いいえ | child 起動前の bootstrap 境界、context または build 環境検証失敗 |
| `2` | `usage` | いいえ | 固定引数の不一致 |
| `10` | `prepared_load` | いいえ | pointer、request、manifest、review または履歴の準備失敗 |
| `11` | `request_binding` | いいえ | request identity、candidate content identity または固定 request 境界の不一致 |
| `12` | `evaluate_build` | いいえ | 候補 profile、corpus、evaluator または report 構成の失敗 |
| `13` | `report_binding` | いいえ | report identity と request/content binding の不一致 |
| `14` | `accept` | いいえ | report 復元または受入判定関数の error |
| `15` | `result_bind` | はい | result の request 結合失敗 |
| `16` | `result_encode` | はい | result 直列化または size 上限の失敗 |
| `17` | `result_decode` | はい | result 再検証失敗 |
| `18` | `handoff_write` | はい | report/result 二 file の staging または確定失敗 |
| `19` | `tracked_replay` | 既存 | 追跡済み report/result の再現または byte 照合失敗 |
| `20` | `handoff_read` | はい | worker 成功後の output 閉包、result decode または digest 読戻し失敗 |
| `21` | `worker_start` | いいえ | child process を開始できない、または strict な child code を復元できない開始系失敗 |
| `22` | `unknown` | 不明 | worker が allowlist 外の失敗 code を返した、または build/tool 層で分類不能な child 終了 |

`outcome=passed` と `outcome=failed` は、どちらも有効な handoff 完了として終了 code
`0` とする。ここでの成功は評価処理と handoff byte が有効であることだけを表し、
候補の採用適格を表さない。

## process 間の伝達

worker は失敗時の標準出力を空に保ち、標準 error には固定した汎用一行だけを書く。
worker の内部 error 本文、query、fixture、case ID、path、review comment、時刻、
host、user または資格情報を標準出力・標準 errorへ直接書かない。

bootstrap は child の標準出力を破棄し、標準 error の末尾 `4096` byte 以下だけを
一時 memory に保持する。`go run` で child を起動した場合は、末尾の最後の非空行が
ASCII の `exit status {code}` と完全一致し、`code` が worker から実際に返せる
`10` から `19`、または `22` の場合だけ同じ code を親の終了 code として伝達する。

最後の非空行が `exit status 1`、prefix 不一致、数値以外、allowlist 外の code、
または child を開始できない場合は `21` または `22` に縮約する。worker が成功して
handoff reader だけが失敗した場合は `20` を返す。bootstrap は child の error 本文、
前行、標準出力または file 内容を自身の標準出力、標準 error、artifact または環境へ
複製しない。

`SOT-ENG-038` が許す「固定した汎用一行」および実行器が付加する
ASCII `exit status {code}` は両立する。段階名や内部原因本文を log へ足さない限り、
この数値 code は privacy 制約に反しない。

## 再試行と不確定終了

`10` から `14`、および `21` は report 完成境界より前の失敗を示す。`15` から `20`
は report 完成境界より後、または既存 report/result の replay 境界である。`19` は
新しい holdout 消費へ数えない。

`22`、診断導入前の generic 非零終了、または code と権威 CI 記録が一致しない終了は
不確定終了とする。不確定終了は、それだけで同じ ID の再試行可否または holdout 消費を
確定しない。まず旧 request、workflow 実行記録、artifact の有無、review binding
および request byte を不変のまま保持し、同じ ID の exact retry が
`SOT-ENG-038` の infrastructure retry に当たることを独立 review で確認する。

独立 review が、構造上有効な report と result が一度も完成しておらず、候補
semantic source set、evaluator version、対象 SOT、二件の review binding および
request byte が変わっていないと確認した場合に限り、同じ ID を一回だけ再試行できる。
completion 境界越えの可能性を除外できない、source または review 対象が変わった、
あるいは権威 CI が別 commit を指す場合は、同じ ID を再試行しない。その場合は
旧 request と参照成果物を不変に保持し、未使用 baseline version、content-bound
review および別 request を準備して pointer だけを進める。

manual workflow は失敗 code を理由に自動再試行しない。次の dispatch は、同じ ID の
retry 条件または別 request の準備条件を独立 review し、対象 commit の権威 CI が
成功した後にだけ一回行う。

## 確認

外部 network と holdout を使わない contract test で、少なくとも次を確認する。

- `candidate-evaluation-outcome-exit-semantics`
- `candidate-evaluation-output-privacy`
- `candidate-evaluation-build-context-isolation`
- `candidate-evaluation-deterministic-replay`

各段階へ query、case ID、絶対 path および資格情報風の文字列を注入し、終了 code
以外の byte が stdout/stderr へ出ないこと、worker stderr の保持が `4096` byte を
超えないこと、worker 成功後の handoff 読戻し失敗が `handoff_read` へ写像されること、
allowlist 外または strict に復元できない child code が `worker_start` または
`unknown` に縮約されることを確認する。これらの test は corpus または holdout
fixture の本文を開かない。

## 関連

- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-027: 省資源の段階的検証](27-resource-aware-verification-stages.md)
- [SOT-ENG-036: 統合照会の評価 baseline 成果物契約](36-unified-query-evaluation-baseline-artifact-contract.md)
- [SOT-ENG-038: 統合照会の内容固定済み候補 holdout 評価 handoff](38-content-bound-candidate-evaluation-handoff.md)
- [SOT-ENG-039: 内容固定済み候補による統合照会の導入段階と変更順序](39-content-bound-unified-query-rollout-stages.md)
