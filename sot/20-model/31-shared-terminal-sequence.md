# SOT-MODEL-031: SharedTerminalSequence

- 状態: 有効

## 規定

`SharedTerminalSequence` は、複数の位置付き主題が一つの節末 task cue を
共有できることを、前処理結果から一回だけ検証して query profile へ渡す、
provider 非依存で不変な内部 sidecar とする。

この sidecar は `CandidateGenerationInput` の一部であり、
`LegalQueryPreprocessResult`、`QueryProfileContribution`、候補、plan、
診断または公開結果へ保存しない。

## 構造

`SharedTerminalSequence` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `topicSpans` | `QuerySpan[]` | はい | 原文順に並ぶ二件以上二百五十六件以下の構造上の主題 |
| `terminalTaskRelation` | `CueTaskRelation` | はい | 主題列が共有する節末の `direct_task` relation |

`topicSpans` の各 span は、同じ前処理結果の `lawNameMentions`、
`legalConceptMentions` または `queryTermMentions` の一件以上と完全一致する。
同じ span に複数の意味出現がある場合は一つの構造上の主題として span を
重複させず、意味候補を一件へ選ばない。異なる span に同じ意味出現が反復された
場合は、共通 constructor で意味を比較または縮約せず、各位置を保持する。

`terminalTaskRelation.kind` は `direct_task` とし、その `subject` と
`predicate` は同じ cue 出現を参照する。全 `topicSpans` と relation の
`subject.span` は relation の一つの `clauseSpan` に含まれる。

sidecar は原文、surface、比較用正規化値、token、separator、tail connector、
task/resource、演算子、score、signal、候補、pack、provider または外部結果を
持たない。

## 構築

前処理結果から profile 用入力を作る共通 constructor は、
`SOT-ARCH-025` の閉じた separator、末尾接続、節および `direct_task`
relation の条件を満たす場合だけ sidecar を作る。

一つの `direct_task` relation について、前後へ別の有効な主題を追加できない
最大の `topicSpans` が構造上ただ一件に定まる場合だけ、その一件を作る。
有効な最大列の部分列を別 sidecar として作らない。同じ span の複数出現は
前節の規則で一件にし、異なる span は意味が同じでも構築時に除かない。

重なり方が異なる mention などから、互いに同一でない最大列が二件以上成立する
場合は、その relation の sidecar を作らない。共通 constructor は、件数、
span の長さ、出現種別、辞書上の優先度または配列順で一件を選ばない。
この非生成は意味候補の棄却ではなく、共有末尾 cue による fan-out を適用しない
安全側の構文判定とする。

共通 constructor は、検証に必要な原文をその場でだけ読み、sidecar を作った後に
profile 用入力へ原文、任意の部分文字列または token 列を残さない。構文検証から
`law_content_search` その他の意味候補を作らない。

## 順序、上限および不変性

sidecar は `terminalTaskRelation` の `SOT-MODEL-030` による順序、
`topicSpans[0].startByte` および topic span 列の順で保持する。同じ
`terminalTaskRelation` と同じ `topicSpans` を持つ重複を除く。

一つの `CandidateGenerationInput` が持つ sidecar は百二十八件以下とする。
`CueTaskRelation` 一件につき最大一件であるため、relation 上限を超える
sidecar を作らない。`topicSpans` は、`SOT-MODEL-025` が許す全出現の合計と
同じ二百五十六件を上限とする。五件目以降も切り捨てず、profile が意味署名の
縮約後に `step_limit_exceeded` を判定できる形で保持する。有効な前処理結果から
この上限を超える列は作れない。上限を超える入力は、外部情報源を呼ぶ前に
入力構築エラーとする。

constructor は、参照元の前処理結果、UTF-8 rune 境界、span の包含・昇順・
非重複、既存出現との完全一致、terminal relation の参照と kind、
`SOT-ARCH-025` の閉じた条件、最大列および決定的順序を検証する。

package 間で使う accessor は次に固定する。

- `CandidateGenerationInput.SharedTerminalSequences() []SharedTerminalSequence`
- `SharedTerminalSequence.TopicSpans() []QuerySpan`
- `SharedTerminalSequence.TerminalTaskRelation() CueTaskRelation`

各 accessor は slice と relation の入れ子を含む深い複製を返す。
`SharedTerminalSequence` の field を直接公開せず、空の sidecar 列は
共有末尾列がない有効な入力を表す。

## 利用境界

法令コア profile は `SOT-ARCH-025` に従い、sidecar の terminal cue を自身の
採用済み task/resource と照合して、論理演算子、意味上の主題および step を
決める。構造上の異なる span が同じ logical input の意味署名へ到達した場合の
縮約も profile の責務とする。

`judicial-cases` profile はこの sidecar を裁判例 search/read step または
候補 fan-out の生成に使用しない。裁判例固有の根拠対応と evidence cluster は
同 profile の位置付き出現から独立に作る。

その他の profile も、各 profile の新しい有効な SOT が採用を明示しない限り
sidecar を意味候補へ変換しない。profile、composer、selector または provider が
sidecar から原文を復元したり、別の separator 条件を補ったりしない。

## 確認

ネットワークを使わない model test、共通 constructor test および profile
contract test で、少なくとも次を確認する。

- `shared-terminal-sequence-contract`: 後続の構造例、不正入力、上限および
  accessor の不変性を一つの閉じた sidecar 契約として確認する
- `永住許可、帰化を教えてください` と
  `永住許可と帰化について教えてください` から、二つの topic span と一つの
  `direct_task` relation を持つ sidecar を作る
- 一つの terminal relation から最大列一件だけを作り、部分列を重複生成しない
- 非同一の最大列が二件成立する重なりでは、どちらも選ばず sidecar を作らない
- 同じ span の複数意味を一主題とし、異なる span の同じ意味は構造段階で
  失わない
- 五件以上の topic span を切り捨てず、二百五十六件まで保持する
- `judicial-cases` profile が sidecar から search/read step または fan-out を
  生成しない
- 単純列挙、separator 違反、未知の接続、別の節、文末でない cue または
  `direct_task` でない relation から sidecar を作らない
- 存在しない relation、範囲外 span、UTF-8 の途中、逆順、重複、
  二百五十七 topic および百二十九件目の sidecar を拒否する
- 三つの accessor から返した値を変更しても元の入力を変更できず、profile 用入力が
  原文、任意の部分文字列、surface、比較用正規化値または token 列を公開しない

## 関連

- [SOT-MODEL-025: LegalQueryPreprocessResult](25-legal-query-preprocess-result.md)
- [SOT-MODEL-026: QueryProfileContribution](26-query-profile-contribution.md)
- [SOT-MODEL-030: CueTaskRelation v2](30-cue-task-relation-v2.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](../30-architecture/25-unified-query-multi-topic-separation.md)
- [SOT-ENG-025: 統合照会のパッケージ構成](../50-engineering/25-unified-query-package-layout.md)
