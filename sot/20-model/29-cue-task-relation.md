# SOT-MODEL-029: CueTaskRelation

- 状態: 有効

## 規定

`CueTaskRelation` は、一回の共通前処理で得た構文 cue の出現が、同じ節で
利用者の task として使われたことを表す、provider 非依存で不変な内部
sidecar モデルとする。

`CueMention` は登録語が原文のどこに現れたかだけを表し、
`CueTaskRelation` はその出現が task 表現、task の目的語または task 述語として
結び付いたことを別の事実として表す。relation がない cue 出現を、query profile
が位置または近接だけから task とみなしてはならない。

## cue の構文 role

共通前処理へ注入する各 cue entry は、次の `syntaxRole` を一つだけ持つ。

| `syntaxRole` | 意味 |
|---|---|
| `none` | task relation の構成要素にしない |
| `task_expression` | 登録表現自体が完結した task 表現である |
| `task_object` | task の対象を表し、述語との関係または単独 task が確認された場合だけ task として扱える |
| `task_predicate` | 先行する `task_object` と結び付ける述語であり、単独では task signal の根拠にしない |

`syntaxRole` は構文上の役割だけを表す。採用済みか対象外か、task または
resource の意味、signal、intent group、score、capability、pack および
provider を共通前処理へ与えない。

一つの cue entry の登録表現に複数の `syntaxRole` を混在させない。役割が異なる
表現は cue ID を分ける。

## 構造

`CueTaskRelation` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `subject` | `CueTaskRelationRef` | はい | task として解釈する cue 出現 |
| `predicate` | `CueTaskRelationRef` | はい | task 述語となる cue 出現 |
| `clauseSpan` | `QuerySpan` | はい | relation を作った一つの節 |
| `kind` | string | はい | `direct_task`、`object_predicate` または `standalone_task` |

`CueTaskRelationRef` は `profileId`、`cueId` および `span` を持ち、同じ
`LegalQueryPreprocessResult.cueMentions` に存在する一件を完全に参照する。
`surface` と `matchKind` を重複して保持しない。

`subject` と `predicate` は同じ `profileId` を持つ。共通前処理は異なる
query profile の cue を関係付けず、profile 横断の意味合成を行わない。

relation の constructor は、subject と predicate の `CueMention`、それぞれを
生成した `SOT-ENG-030` の検証済み cue entry の `syntaxRole`、`clauseSpan`
および `kind` を受ける。role は構築時の検証にだけ使い、relation へ重複して
保存しない。`LegalQueryPreprocessResult` は、構築済み relation の cue 参照、
span、kind および順序を再検証し、cue artifact を読み直さない。

## relation kind

### `direct_task`

`subject` と `predicate` は同じ cue 出現を参照し、その cue の
`syntaxRole` は `task_expression` とする。cue span より後の `clauseSpan` が
Unicode White_Space だけである場合に限り作る。依頼の活用形、敬語および
文末表現は登録表現の一部とし、共通前処理が未登録の語を補って task 表現を
拡張しない。

### `object_predicate`

`subject.syntaxRole` は `task_object`、`predicate.syntaxRole` は
`task_predicate` とする。

`subject` は `predicate` より前にあり、両者は同じ `clauseSpan` に含まれ、
次の token 列で直接接続する。

1. subject span
2. 零個以上の Unicode White_Space
3. surface が `を`、品詞が助詞、品詞細分類一が格助詞である一つの Kagome token
4. 零個以上の Unicode White_Space
5. predicate span

subject と predicate の間に上記以外の byte または token がある場合は relation を
作らない。predicate span より後の `clauseSpan` も Unicode White_Space だけで
なければならない。したがって、述語の後に名詞、別の目的語または別の task が
続く連体修飾を relation にしない。

### `standalone_task`

`subject` と `predicate` は同じ cue 出現を参照し、その cue の
`syntaxRole` は `task_object` とする。`clauseSpan` のうち subject の外側が
Unicode White_Space だけである場合に限り、短縮された単独 task として作る。
ほかの名詞、述語または検索対象と同じ節にある裸の
`task_object` を、この kind へ拡張しない。

## 節と非 task の境界

節境界は `。`、`！`、`？`、`!`、`?`、`;`、`；`、CR または LF とする。
共通前処理は原文 byte 列を左から一回走査して境界文字を `clauseSpan` に
含めず、空または Unicode White_Space だけの節を作らない。Kagome token は
その byte span を完全に含む一つの節へ所属させる。読点だけでは節を分けないが、
`object_predicate` の直接接続条件を緩和しない。

次の cue 出現から relation を作らない。

- `queryTermMentions.kind=quoted_phrase` の内側に完全に含まれる出現
- `という語`、`という言葉`、`という表現`、`という用語`、
  `という文字列` または `という文言`のいずれかへ、Unicode White_Space 以外の
  byte を挟まず直接接続する出現
- `に関する`、`に関して`、`について`または`に係る`のいずれかへ、
  Unicode White_Space 以外の byte を挟まず直接接続する `task_object`
- 別の節にある subject と predicate

この除外判定を relation kind の判定より先に行う。上記の閉じた条件に一致しない
「説明語」または「検索対象」を、共通前処理が意味から推測して追加しない。
`direct_task` と `object_predicate` の文末条件、および `standalone_task` の
節全体条件を満たさない cue は relation を持たない。

例えば、`影響グラフを作成してください。` は、登録済みの subject と predicate
が直接接続する場合に `object_predicate` を作れる。
`影響グラフという語を含む条文を検索してください。`、
`比較という言葉の定義を検索してください。`、
`「影響グラフ」を含む条文を検索してください。` および
`翻訳に関する規定を検索してください。` は上記の除外条件により relation を
作らない。`差分を説明する規定を検索してください。` は predicate が節末に
ないため relation を作らない。

## 順序、上限および不変性

relation は `clauseSpan.startByte`、`subject.span.startByte`、
`predicate.span.startByte`、`profileId`、`subject.cueId`、
`predicate.cueId` および `kind` の昇順とする。同じ値の重複を除き、
重なる複数の関係を近接だけで一件へ統合しない。

一つの前処理結果が持つ relation は百二十八件以下とする。上限を超えた場合は
切り捨てず、外部情報源を呼ぶ前に前処理エラーとする。

relation の constructor は、元の照会文に対する UTF-8 rune 境界、各 span の
包含、subject と predicate の cue 参照、profile ID、cue ID、渡された syntax
role および relation kind の対応を検証する。前処理結果の constructor は同じ
結果に存在する cue 参照と決定的順序を検証する。getter は relation と入れ子の
参照を変更できない複製として返す。

relation は score、signal、intent group、task、resource、capability、
required pack、候補、provider ID、route、外部 DTO または検索結果を持たない。
照会間で relation を保持しない。

## 確認

ネットワークを使わない model test と共通前処理 test で、少なくとも次を
確認する。

- `task_expression` の直接 task、`task_object` と述語の直接接続、および
  `task_object` だけの短縮 task を各 kind にする
- 引用句、閉じた言及表現、`に関する`などの閉じた topic 表現、別の節、
  文末でない述語および未接続の裸語から relation を作らない
- 存在しない cue、異なる profile、role と kind の不一致、範囲外 span、
  UTF-8 の途中、逆順、重複および百二十九件を拒否する
- 同じ入力と cue 語彙から同じ順序の relation を返し、getter の変更と
  並行呼出しで共有状態を変更しない

## 関連

- [SOT-MODEL-025: LegalQueryPreprocessResult](25-legal-query-preprocess-result.md)
- [SOT-MODEL-026: QueryProfileContribution](26-query-profile-contribution.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-ENG-028: 統合照会の対象外意図 cue セット](../50-engineering/28-unified-query-unsupported-intent-cues.md)
- [SOT-ENG-030: 統合照会の cue 成果物契約](../50-engineering/30-unified-query-cue-artifact-contract.md)
