# SOT-MODEL-028: QueryCandidateCompositionMember

- 状態: 有効

## 規定

`QueryCandidateCompositionMember` は、一つの query profile が生成した
`LegalQueryCandidate` を、別の profile が生成した明示意図と同じ意味候補へ
結合できること、および各 logical step の原文上の基準位置を表す内部の
sidecar モデルとする。

このモデルは候補の意味、score または実行可否を単独で変更しない。
`SOT-ARCH-027` の composer だけが、検証済みの複数 member から新しい候補を
作るために使用する。

## 構造

`QueryCandidateCompositionMember` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `candidateId` | string | はい | 同じ profile contribution 内の構成元候補 |
| `role` | string | はい | 固定値 `required_member` |
| `stepOrigins` | `QueryCandidateStepOrigin[]` | はい | 構成元候補の全 step に対応する原文位置 |

`QueryCandidateStepOrigin` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `stepId` | string | はい | 構成元候補の step ID |
| `sourceStartByte` | integer | はい | 元の照会文 UTF-8 byte 列上の開始位置 |

`stepOrigins` は構成元候補の `steps` と同じ件数および同じ順序で、一つの
`stepId` を一度だけ参照する。`sourceStartByte` は零以上、元の照会文の
byte 長未満であり、UTF-8 rune の開始境界でなければならない。

法令 ID、日付、法令名、法概念、事件番号または検索語から作る step は、
その位置付き前処理事実の開始位置を使う。入力 `ref` から作る read step は、
その read を単独で根拠付けられる task cue、resource cue または read 対象 cue
のうち、最小の `sourceStartByte` を使う。同じ最小位置に複数 cue があっても、
その最小値だけを保存する。profile が位置を一意に決められない step を
composition member にしてはならない。

## `required_member`

`required_member` は、候補が利用者の同じ照会文にある明示取得意図へ
位置付きで対応することを表す。候補が別候補の代替解釈ではないことや、
profile 横断合成へ実際に採用できることを member 単体では保証しない。

profile は、少なくとも次をすべて満たす候補だけを member にできる。

- 候補が採用済み task と resource、または検証済み入力 `ref` によって
  根拠付けられている
- 候補の全 step に原文位置を割り当てられる
- 候補が照会文中の一つの明示取得意図へ対応する。ほかの候補との代替関係は
  member 単体で確定せず、contribution 全体を composer が判定する
- contribution の `selectionMode` が `automatic` である

同じ profile 内で互いに代替となる複数候補または hedge 候補でも、各候補が
明示 task/resource と全 step の原文位置を持つ場合は、composer が合成不適格を
判定できるよう member sidecar を保持できる。この sidecar は各候補の位置を
表すだけで、代替候補を同時取得すべきという意味にはしない。
弱い一般語だけの候補、resource を特定しない法概念の複数候補、
自動実行しない代替候補、および検索結果に依存する後続 read は member に
しない。

member 候補と `CandidateHedgePair` の重複は sidecar の参照整合性を壊さない
ため構造として受理する。ただし hedge は代替解釈であり、composer は当該
contribution を合成対象から除外して `composition_ineligible` を立てる。
profile は hedge を合成可能と判断したことにはならない。

一 contribution の member が複数であること、または member 以外の候補が
併存することは、各 member の構造違反にはしない。ただし、それらは
profile が一意な必須候補を確定できない状態であるため、
`SOT-ARCH-027` の composer は当該 contribution を合成に使用しない。
同じ profile で一意に必須と確定した複数 step は、可能な場合は一つの
候補と一つの member にまとめる。

法概念辞書が複数 resource 候補を持つ場合でも、照会文が一つの候補の
resource を具体的に明示したときは、その resource に対応する profile が
当該候補を一意な取得意図として扱える。`法情報` のように複数 resource を
包含する語だけでは一意化したとみなさない。複数 resource をそれぞれ
明示して取得する照会では、各 profile が対応する候補を
`required_member` として出力できる。

## 不変性と公開境界

member、step origin および getter が返す配列は不変として扱い、照会間で
保存しない。`sourceStartByte`、構成元 ID、role および未合成 member は、
`SOT-ARCH-024` に従い公開 MCP 結果へ出さない。

member は pack の有効状態、provider ID、route、外部件数、応答速度または
materialized request を持たない。

## 確認

候補と step の一対一対応、未知または重複する ID、負数、照会文の終端以上、
UTF-8 の途中、位置を持たない `ref` read、deep copy、clarification 候補、
および hedge 候補の構造的な保持を model test で確認する。
hedge と複数代替候補を合成せず合成不適格にすることは composer test で
確認する。

明示した法令検索と裁判例検索、明示した条文検索と裁判例検索、および
検証済み裁判例 `ref` の read と後続検索について、各 step の原文位置を
保持できることを profile test で確認する。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](22-legal-query-candidate.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](25-legal-query-preprocess-result.md)
- [SOT-MODEL-026: QueryProfileContribution](26-query-profile-contribution.md)
- [SOT-ARCH-024: 統合照会の内部境界と公開境界](../30-architecture/24-unified-query-internal-public-boundary.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](../30-architecture/27-unified-query-cross-profile-composition.md)
