# SOT-SCN-013: 法令の二つの版を比較する

- 状態: 有効

## 規定

利用者は、一つの法令 ID と比較前後の版指定を一つずつ与え、実際に選択された
二版と、本則及び原始附則に属する条の変更を出典付きで確認できる。

## 開始条件

利用者が次を指定している。

- 比較対象の `lawId`
- 比較前の `revisionId` 又は `asOf` のどちらか一方
- 比較後の `revisionId` 又は `asOf` のどちらか一方

## 基本フロー

1. Japanese Law MCP が `law.version.compare@1` の primary route を選ぶ。
2. provider が比較前後の版をそれぞれ正確に確定する。
3. provider が同じ法令に属する二版から、採用範囲の条を同一性で対応付ける。
4. provider が条の追加、削除、位置、文字列及び内部構造の変更を分類する。
5. システムが確定版、全件数、変更一覧及び各原文の出典を返す。

## 分岐

- 入力制約を満たさない場合は、外部情報源を呼び出さず `invalid_argument` とする。
- どちらかの版が存在しない場合は `not_found` とし、片方だけを返さない。
- 比較前後が同じ版へ解決された場合は、全条を同一として数え、成功した空の
  変更一覧を返す。
- 同じ法令として検証できない応答、重複する条同一性、危険な本文又は資源上限
  超過を、部分結果や推測した対応へ変換しない。
- 利用者が指定した前後関係をシステム側で入れ替えない。

## 完了条件

返された比較結果は一つの `lawId` に属する二版だけを対象とし、対象にした全条が
追加、削除、変更又は同一のいずれかへ一度だけ数えられ、各変更項目から存在する
側の公式原文を確認できる。

## 関連

- [SOT-PROD-013: 法令版間比較](../00-product/13-law-version-comparison.md)
- [SOT-MODEL-033: LawVersionComparison](../20-model/33-law-version-comparison.md)
- [SOT-IF-058: `law.version.compare` capability v1](../40-interfaces/58-law-version-compare-capability.md)
- [SOT-IF-059: MCP `compare_law_versions`](../40-interfaces/59-mcp-compare-law-versions.md)
