# ADR-003: 開発用根拠 assertion を補完した評価コーパス Version 3

- 状態: 採用
- 決定日: 2026-07-28

## 背景

`corpus-v2` の development と holdout に含まれる `evidenceCodes` を独立監査したところ、development の三件だけが同種の holdout fixture より弱い根拠 assertion を持っていた。

- 法令名の一意な誤記補正は、補正処理だけで検索対象を作るのではなく、補正先が出典付きの正式名称、略称または別名であることも根拠にする。
- 法令 ID と条項を組み立てる構造化読取りで、照会文に出典付き法令名がある場合は、その法令名一致も根拠にする。

`development-typo-deletion` と `development-typo-substitution` は `unique_typo_correction` を持つ一方で `official_alias` を欠き、`development-structure-article` は照会文の「商法」を法令 ID へ解決している一方で `official_alias` を欠いていた。これは profile の期待動作を変える判断ではなく、入力に既に存在する根拠の assertion 漏れである。

## 決定

`corpus-v1` と `corpus-v2` は変更せず、同じ schema version、seed、holdout および execution 集合を持つ `corpus-v3` を追加する。期待 assertion を変える三件には新しい case ID を割り当てる。

| Version 2 の case ID | Version 3 の case ID | 補完する根拠 |
|---|---|---|
| `development-structure-article` | `development-structure-article-grounded` | `official_alias` |
| `development-typo-deletion` | `development-typo-deletion-grounded` | `official_alias` |
| `development-typo-substitution` | `development-typo-substitution-grounded` | `official_alias` |

照会文、leakage group、coverage、decision、reason、meaning、logical step および selection は変更しない。holdout と execution の byte 列も変更しない。

固定値は次のとおりである。

| 項目 | Version 2 | Version 3 |
|---|---|---|
| development 件数 | `31` | `31` |
| holdout 件数 | `240` | `240` |
| execution 件数 | `7` | `7` |
| seed | `20260727` | `20260727` |
| holdout digest | `25b06db5e29ada2922e970a8c569ee9d2e73ce22fae4ffe3ce5880043d3543b8` | `25b06db5e29ada2922e970a8c569ee9d2e73ce22fae4ffe3ce5880043d3543b8` |

この決定時点では初回 profile と評価 command の既定 corpus を `corpus-v3` とした。過去二版も公開 loader の再現性試験を残す。後に [ADR-004](04-product-grounded-evaluation-corpus-v4.md) が既定 corpus の指定だけを `corpus-v4` へ置き換えた。

## 検証結果

- `corpus-v3` は、公開 loader による schema、fixture checksum、集合分離、最小件数、参照および holdout digest の検証に成功した。
- 補正した三件の `evidenceCodes` を正確な順序で固定し、全 development と holdout について日付および公式識別子の入力根拠検査を再実行した。
- holdout、profile および期待意味は変更していない。意味評価 command と baseline の導入前であるため、意味指標の変更前後比較は行わず、この変更を全評価ゲートの成功とは扱わない。初回 profile 採用時に `corpus-v3` 全件を評価する。

## 帰結

- 開発集合で調整する profile は、正式名称または出典付き別名と一意な誤記補正を別の根拠として扱える。
- holdout digest を変更せず、既存の受入母集団を維持できる。
- 過去の誤った assertion も `corpus-v2` から再現でき、成果物の履歴を失わない。
- 今後、入力または期待意味を変える場合は、[`SOT-ENG-026`](../../sot/50-engineering/26-legal-query-corpus-artifact-contract.md) に従って新しい corpus version と case ID を割り当てる。
