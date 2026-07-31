# SOT-ARCH-036: 複数 step 候補の step 内根拠正規化と保持

- 状態: 有効

## 規定

一つの query profile が、照会文で独立に明示された一件以上の取得意図を
`LegalQueryCandidate` へ変換する場合は、各 logical step の同値経路を閉じた
優越規則で先に正規化し、その後に全 step の根拠を候補へ和集合する。

## 正規化の入力

正規化の入力は、profile の private evidence mapping が同じ request 内で検証した
位置付き事実と、次の値だけとする。

- step の型付き logical input の完全な意味署名である
  `stepMeaningSignature`
- `topicOrdinal`
- 同じ typed target と同値な生成経路にだけ profile が付与する request-local の
  不透明な `normalizationGroup`
- `SOT-ARCH-031` の layer と evidence code
- 同じ位置付き事実を識別する request-local ID と span

原文を再解析せず、距離、配列順、候補全体の score、provider、route、外部結果、
別 request の状態または未検証の文字列一致から同値性を補わない。

## step 内の同値単位

まず、同じ step、同じ request-local 事実、同じ span、同じ layer および同じ
evidence code の重複を一件へ縮約する。

その後の弱い根拠の除去は、次の key が完全一致する同値単位の中だけで行う。

1. `stepMeaningSignature`
2. `topicOrdinal`
3. `normalizationGroup`

`normalizationGroup` は、二件以上の位置付き事実が、profile 固有 SOT の
input kind 対応と投影規則だけから、同じ logical input の同じ typed target
field と evidence purpose をそれぞれ単独で完全一致の値へ生成できる場合にだけ
付与する。ID は request 内だけで比較する不透明値とし、typed target field の
canonical projection、purpose、`stepMeaningSignature` および `topicOrdinal` の
組から決定的に作る。原文またはこの組を ID から復元可能にする必要はない。

異なる step、`topicOrdinal`、検索語、article、paragraph、演算子項目、取得対象
または evidence purpose を同じ group にしない。group を持たない根拠は、前段の同一
`{request-local fact ID, layer, evidence code}` の重複除去だけを適用し、
優越規則を適用しない。

同じ事実が複数の group に属する場合、または採用済み profile 規則だけでは group
を一意に決められない場合は、配列順、span 距離または score で補わず候補 draft
全体を拒否する。同じ surface、同じ span、近い位置または同じ evidence code
だけでは、異なる位置付き事実または異なる意味を同値にしない。

## 閉じた優越規則

同じ同値単位に次の上位 code と下位 code がともにある場合は、下位 code を
必ず除去する。表にない組では、強い順だけを理由に一方を除去しない。

| 上位 code | 同じ同値単位で除去する下位 code |
|---|---|
| `official_identifier` | `official_alias`、`legal_concept`、`morphological_context`、`unique_typo_correction`、`general_term` |
| `official_alias` | `morphological_context`、`general_term` |
| `legal_concept` | `morphological_context`、`general_term` |
| `morphological_context` | `general_term` |

`structured_reference`、`explicit_task` および `explicit_resource` は、この表で
他の code を除去せず、他の code によっても除去しない。同じ target を表す
`official_identifier` と `structured_reference`、または同じ step の target と
task/resource は、意味上の役割が異なるためともに保持する。

別 step、別 `topicOrdinal`、別 logical input、別取得対象または別 evidence purpose
にある下位 code は、候補の別 step に上位 code があることを理由に除去しない。
例えば、法令 ID による法令読取りと正式名称による法令検索を一候補へまとめる
場合は、`official_identifier` と `official_alias` をともに保持する。
表にない `official_alias` と `legal_concept`、またはこれらと
`unique_typo_correction` の組も、同じ group にあることだけを理由に除去しない。

## 候補への統合と score

各 step で前節までの正規化を完了した後、全 step の evidence code を和集合し、
重複 code を一件へ縮約して `SOT-MODEL-022` の強い順に並べる。候補の
`semanticScore` は、この最終集合だけを版付き profile metadata の weight へ
渡して計算する。

候補全体の未正規化集合から score を計算した後で `evidenceCodes` だけを変えず、
step 間の和集合へ優越表をもう一度適用しない。

`conceptSources` は、最終集合に `legal_concept` がある場合だけ、実際に保持した
法概念事実の source を持つ。除去した `legal_concept` だけに属する source は
除去し、別 step または別 group に同じ法概念の保持済み事実がある場合はその
source を保持する。保持する source は `conceptId` の byte 順に統合し、同じ
`conceptId` に異なる `title`、`url` または `confirmedOn` がある場合は候補 draft
全体を拒否する。最終集合に `legal_concept` がなければ空配列とする。

最終候補へ private evidence mapping、request-local ID、同値単位または step ごとの
内部根拠列を公開しない。profile 横断の候補合成は `SOT-ARCH-027` に従い、
本規定の step 内正規化を別 profile の contribution 間で再実行しない。

## 変更

同値単位、優越表、正規化順序または score へ渡す集合を変える場合は、本規定を
置き換える新しい SOT ID を先に採用する。本規定に基づく実装採用を完了する変更では、
影響する全 profile の新しい `profileVersion` も同じ変更で採用する。
profile 間で共有する score 尺度または候補順位が変わる場合は
`rankingVersion` も更新し、`SOT-ENG-024` の development 校正と holdout 採用判定を
やり直す。

## 確認

複数 step を構築する各 profile は、profile 固有の固定検証 ID で少なくとも次を
確認する。

- 同じ step・target・purpose の各優越組が表どおり一意に正規化される
- 表にない code 組、異なる purpose、異なる `normalizationGroup` および
  group を持たない根拠は保持される
- 別 step の強い code によって、独立 step の `official_alias`、
  `legal_concept`、`morphological_context` または `general_term` が失われない
- 三 step 以上でも step 内正規化、候補和集合、`SOT-MODEL-022` の順序および
  score の入力集合が決定的である
- 位置付き事実の入力配列順を変えても、同じ `evidenceCodes`、score および
  `conceptSources` になる
- 同じ事実の group 重複、group の一意決定不能および同じ `conceptId` の
  source tuple 衝突では、部分候補を残さず fail-closed となる
- 除去した `legal_concept` だけの source は除去され、別 step または別 group の
  保持済み `legal_concept` の source は残る

core と `judicial-cases` は異なる固定検証 ID を持ち、一方の成功で他方を
代用しない。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](../20-model/25-legal-query-preprocess-result.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](27-unified-query-cross-profile-composition.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](31-unified-query-intent-evidence-layer.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](../50-engineering/24-unified-query-evaluation-gate.md)
