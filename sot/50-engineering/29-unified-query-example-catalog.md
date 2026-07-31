# SOT-ENG-029: 統合照会の検索例カタログ

- 状態: 有効

## 規定

統合照会は、現行の標準評価成果物で確認できる代表的な日本語照会と期待結果を、
人間可読で版付きの Markdown カタログとして repository 内に維持する。

## 目的と境界

カタログは、現行の SOT、固定 profile set、評価 fixture および公開 MCP 契約が、
代表例でどう観測されるかを説明する派生成果物とする。意味判定の正解、採用範囲、
非実行境界または公開 status の定義元にはしない。

採用済みだが未実装の理想状態、次版の corpus、未採用 baseline または将来の
test ID は掲載しない。それらは有効な SOT と Wiki の実装差分で追跡し、現行の
固定 fixture が追加された変更で初めてカタログへ加える。

## 配置と catalog version

カタログは `docs/unified-query-examples/` 配下に置き、少なくとも次を持つ。

- `00-index.md`: 目的、読み方、`catalogVersion`、現行の corpus version と
  baseline version、および各章への索引
- `10-execution.md`: 実行可能な意味計画と、固定 execution fixture で確認した
  公開 status の代表例
- `20-clarification-and-unsupported.md`: 外部呼出しを行わない明確化、能力利用不可
  および対象外の代表例

同じ例を複数ファイルへ重複して記載しない。query、request context、期待 plan、
期待 status、検証 artifact または章構成を変更した場合は `catalogVersion` を
増やす。表記修正だけで観測内容を変えない場合は version を維持できる。

`catalogVersion` は `unified-query-examples-v` と先頭が零ではない十進整数を
連結した、64 byte 以下の ASCII 値とする。`00-index.md` に一件だけ記載し、
別名、任意の tag、日付または同じ番号の異なる内容を許可しない。

## 例の種類

各例は `semantic` または `execution` のどちらか一つとする。

- `semantic`: corpus の一つの semantic case と、query、request context および
  `expected.decision` を完全一致で対応させる
- `execution`: corpus の一つの execution case と、その
  `semanticCaseId` が指す semantic case の query と request context を完全一致で
  対応させ、`expected.status` も示す

coverage ID、scenario ID、SOT の例文または通常の unit test だけを、query と
request context の完全一致を証明する artifact として使わない。

## 各例の必須項目

各行は、少なくとも次の共通項目を持つ。

| 項目 | 内容 |
|---|---|
| `example_id` | カタログ全体で一意な、小文字 ASCII 英数字と `-` からなる安定 ID |
| `example_kind` | `semantic` または `execution` |
| `query` | 対応 fixture の `request.query` と完全一致する日本語照会 |
| `request_context` | `ref` の全項目、pack 状態および入力 option。省略時も明記する |
| `verification_artifact` | `corpus-vN:semantic:{caseId}` または `corpus-vN:execution:{caseId}` |
| `expected_plan_decision` | 対応 semantic case の `expected.decision` |
| `expected_summary` | interpretation、step または非実行理由の簡潔な説明 |
| `related_sots` | 挙動の定義元である有効な SOT ID |

加えて、`expected_public_status` は次の規則で持つ。

- `10-execution.md` の全行は `expected_public_status` を持つ
- 実行可能な `semantic` 例は `—` を入れ、公開実行 status をこの行だけで推測しない
- `execution` 例は対応 execution fixture の `completed`、`empty` または `partial`
  を入れる
- `20-clarification-and-unsupported.md` の全行は、非実行 plan と同名の
  `needs_clarification`、`capability_unavailable` または `unsupported` を入れる

`expected_plan_decision` は `single`、`hedged`、`needs_clarification`、
`capability_unavailable` または `unsupported` の一つとする。

`expected_public_status` は、上記以外の値を取らない。

`completed`、`empty`または`partial`を semantic case、coverage ID または
scenario ID だけから推測しない。同じ `query` は、`request_context` が異なり、
別の `example_id` と検証 artifact を持つ場合に限り再利用できる。

## 同期規則

公開既定実装または標準評価成果物について、次のいずれかを変更した場合は、
同じ変更でカタログを照合し、必要なら更新する。

- `query_legal_information` の公開 status、notice または interpretation の観測結果
- semantic case の query、request context、decision または step
- execution case の `semanticCaseId` または公開 status
- 標準 corpus version、baseline version、case ID または固定 profile set

カタログだけを先に変更して実装の観測結果を書き換えない。SOT だけで将来の
理想状態を採用する場合はカタログを変更せず、実装差分を Wiki で追跡する。

## 最低限掲載する例

現行標準に該当 fixture が存在する範囲で、少なくとも次を掲載する。

- 法令検索、法令読取り、条文読取り、法令本文検索および更新一覧
- 裁判例検索と、検証済み `ref` を使う読取り
- 一候補内の複数 step
- `needs_clarification`、`capability_unavailable` および `unsupported`
- execution fixture による `completed`、`empty` および `partial`

将来の relation、比較、関係分析または新しい pack の例は、その挙動を現行標準の
exact fixture で確認できるようになるまで最低件数へ算入しない。

## 確認

文書検査は、少なくとも次を確認する。

- `example_id` が一意で、全必須列が存在する
- `verification_artifact` の corpus と case が実在する
- 各行の query、request context、plan decision および public status が
  対応 fixture と完全一致する
- `expected_public_status=—` の行は、同じ行の `verification_artifact` だけから
  `completed`、`empty` または `partial` を主張していない
- execution case の `semanticCaseId` が、同じ行の query と request context を
  持つ semantic case を参照する
- `completed`、`empty`および`partial`に、それぞれ一件以上の exact execution
  fixture がある
- 非実行 status を実行結果として、または実行可能 semantic 例の status を
  推測値として記載していない
- 現行標準に存在しない case ID、corpus version または baseline versionを
  確認済みとして記載していない
- `catalogVersion` が正規形であり、現行採用 manifest と一致する
- 公開しない score、重み、棄却候補、provider route または trace を含まない

## 関連

- [SOT-MODEL-024: LegalQueryResult](../20-model/24-legal-query-result.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](../30-architecture/31-unified-query-intent-evidence-layer.md)
- [SOT-ARCH-037: 統合照会の正規化済み限定分岐保持](../30-architecture/37-unified-query-normalized-branch-retention.md)
- [SOT-IF-051: MCP `query_legal_information`](../40-interfaces/51-mcp-query-legal-information.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-026: 統合照会の評価コーパス成果物契約](26-legal-query-corpus-artifact-contract.md)
- [SOT-ENG-033: 統合照会 profile set 採用 manifest](33-unified-query-profile-set-adoption-manifest.md)
