# ADR-002: 入力根拠を明示した評価コーパス Version 2

- 状態: 採用
- 決定日: 2026-07-28

## 背景

`corpus-v1` の holdout 二百四十件を独立監査したところ、四件の期待値に照会入力から一意に導けない構造化値があった。

- `holdout-budget-01` と `holdout-budget-06` は、照会文に完全な日付がない一方で、期待する `law_updates` step に日付を持っていた。
- `holdout-safety-11` と `holdout-safety-12` は、照会文または `ref` に実際の法令 ID がない一方で、期待する根拠に `official_identifier` を持っていた。

前処理または profile が現在日や暗黙の検索結果から値を補うと、同じ入力に対する再現性と「検索第一件を暗黙に read しない」安全境界を損なう。

## 決定

`corpus-v1` は変更せずに保存し、同じ schema version と seed を持つ `corpus-v2` を追加する。四件は入力を変更した新しい case ID とし、期待する decision、meaning、step、根拠および coverage は変更しない。

| Version 1 の case ID | Version 2 の case ID | 入力へ追加した根拠 |
|---|---|---|
| `holdout-budget-01` | `holdout-budget-01-explicit-date` | `2026年7月1日` |
| `holdout-budget-06` | `holdout-budget-06-explicit-date` | `2026年6月29日` |
| `holdout-safety-11` | `holdout-safety-11-grounded-evidence` | `129AC0000000089` |
| `holdout-safety-12` | `holdout-safety-12-grounded-evidence` | `140AC0000000045` |

固定値は次のとおりである。

| 項目 | Version 1 | Version 2 |
|---|---|---|
| holdout 件数 | `240` | `240` |
| seed | `20260727` | `20260727` |
| holdout digest | `5b909cc6d80a5d94664b7598b8824ebfc30ed39151172084b235dc210f0ab2ac` | `25b06db5e29ada2922e970a8c569ee9d2e73ce22fae4ffe3ce5880043d3543b8` |

標準評価 command を導入するときは `corpus-v2` を使用する。`corpus-v1` も loader の再現性試験を残し、過去の digest を検証できる状態にする。

## 検証結果

- `corpus-v1` と `corpus-v2` は、公開 loader による schema、fixture checksum、集合分離、最小件数および holdout digest の検証に成功した。
- `corpus-v2` の development と holdout について、期待する完全日付が照会文に存在すること、および `official_identifier` が照会文の実 ID または同じ資源を指す `ref` に根拠を持つことを試験した。
- profile と期待値を変更していない。意味評価 command と baseline の導入前であるため、意味指標の変更前後比較は行わず、この変更を全評価ゲートの成功とは扱わない。初回 profile 採用時に `corpus-v2` 全件を評価する。

## 帰結

- planner は現在日または検索順から構造化値を推測しなくてよい。
- Version 1 の再現性を失わず、初回 profile の採用基準を入力根拠のある Version 2 に固定できる。
- 今後、入力または期待意味を変更する場合も、[`SOT-ENG-026`](../../sot/50-engineering/26-legal-query-corpus-artifact-contract.md) に従って新しい corpus version と case ID を割り当てる。
