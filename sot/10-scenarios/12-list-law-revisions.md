# SOT-SCN-012: 法令の改正履歴を取得する

- 状態: 有効

## 規定

利用者は、一つの法令を法令 ID または法令番号で特定し、その法令について公式情報源が保持する改正履歴を新しい履歴から順に確認できる。

## 開始条件

利用者が法令 ID または法令番号を一つ指定している。

## 基本フロー

1. Japanese Law MCP が指定値を `law.revision.list@1` の primary route へ渡す。
2. 情報源から一つの法令に属する完全な改正履歴を取得する。
3. 情報源固有の改正区分と状態を共通の `LawRevision` へ正規化する。
4. 正確な総件数とともに、情報源が示す新しい順を保持して返す。

## 分岐

- 対象法令が存在しない場合は `not_found` とする。
- 対象法令が存在し、履歴が空の場合は成功した空の一覧とする。
- 指定値が入力制約を満たさない場合は、情報源を呼び出さず `invalid_argument` とする。
- 情報源が示していない省略可能な値を推測して補わない。
- 予定施行日は確定した施行日として扱わず、廃止等の記録日は実際の法的効力発生日と断定しない。

## 完了条件

返された全履歴の `lawId` が出力の対象法令と一致し、`revisionId` が重複せず、公式情報源の順序と出典を保持している。

## 関連

- [SOT-MODEL-032: LawRevision](../20-model/32-law-revision.md)
- [SOT-IF-055: `law.revision.list` capability v1](../40-interfaces/55-law-revision-list-capability.md)
- [SOT-IF-056: MCP `list_law_revisions`](../40-interfaces/56-mcp-list-law-revisions.md)
