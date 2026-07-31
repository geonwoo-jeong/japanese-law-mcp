# SOT-ARCH-034: 法令コア query profile の根拠対応

- 状態: 有効

## 規定

法令コア query profile は、共通前処理の位置付き事実を、法令コアが所有する
五つの input kind の step と private evidence mapping へ決定的に対応付け、
provider、route または外部結果に依存せず、各 step の主題と独立根拠を確定する。

## 所有境界

本規定は、法令コア profile における次だけを定義する。

- input kind ごとに利用できる前処理事実
- 事実と step の同一主題への束縛
- `topicOrdinal` と `evidenceSpan` を作るための profile 固有対応
- 追加分岐に必要な独立した正の根拠
- 共有末尾列を法令本文検索へ変換できる範囲

task、resource、capability および logical input の七組は `SOT-MODEL-022`、
位置付き事実と layer/evidence code の共通対応は `SOT-ARCH-031`、
複数主題の分離は `SOT-ARCH-025`、cluster key、同値 draft の統合、margin および
三件上限は `SOT-ARCH-032` を定義元とする。本規定は、それらを再定義しない。

法令コア profile metadata の `targets` は、次の input kind をこの順で一件ずつ
持つ。対応する task、resource、capability ID および major version は
`SOT-MODEL-022` から一意に決める。

1. `law_search`
2. `law_content_search`
3. `law_read`
4. `law_article_read`
5. `law_updates`

法令コア候補の `requiredPacks` は空配列とする。

## private evidence mapping

private mapping の項目、request 内だけの寿命、公開禁止および競合時の扱いは
`SOT-ARCH-031` を定義元とする。本規定は、法令コアの各 step へどの事実を
対応させるかだけを定義する。

同じ候補に複数 step があっても、ある step の事実を別 step へ複製しない。
候補 draft を結合する場合は、ordered logical input の意味署名と
`topicOrdinal` が一致する step 同士だけの mapping を和集合にし、配列 index
だけで別 step の根拠を借りない。同じ事実、span、layer および code の重複を
除き、layer 順、span 順、`SOT-MODEL-022` の evidence code 順に保持する。

範囲外 span または profile 固有の禁止事実を含む mapping は不正とする。
束縛が一意でない事実の除外後に根拠のない主題が残る場合、および同じ事実の
競合を検出した場合は contribution を構築せず、いずれか一方を推測して残さない。

明示 task/resource の事実を step へ対応させる場合は、次の順で束縛する。

1. 型付き logical input と、`SOT-ARCH-025` が分離した主題範囲を先に確定する
2. その step へ採用した位置付き target の全 span を含む
   `direct_task.clauseSpan` のうち、当該 input kind の task cue に対応する relation
   だけを残す
3. 同じ節の採用済み resource cue のうち、当該 input kind に対応する意味だけを
   残す
4. relation と resource の意味が一意に定まる場合だけ、その位置付き事実を
   `explicit_task_resource` へ対応させる

span を持たない `ref` だけを target とする `law_read` では、2 の代わりに、
照会全体の `direct_task` relation から、当該 logical input を成立させられる
core の read relation が正確に一件だけあることを確認する。resource は
構造検証済み `ref.resourceType=law` から一意に定まるため、別の明示 resource cue
を必須にしない。同じ節に採用済み resource cue がある場合は、適用できる全 cue が
`law` と両立しなければならない。一件の read relation ならその位置付き cue を
mapping と cluster に使用し、零件または二件以上なら read 候補を作らない。
配列の最初の read cue、`ref` に近そうな語、照会全体または零長 span を代用しない。

`ref` と位置付き article target を持つ `law_article_read` は、article/paragraph
span を含む節で 2 から 4 を適用し、同じ節の read relation と
`law_article_read` の対応を一意に束縛する。`ref.resourceType=law`、
article/paragraph target および read relation によって resource が一意な場合は、
別の明示 `law_provision` cue を必須にしない。同じ節に採用済み resource cue が
ある場合は、適用できる全 cue が `law_provision` と両立しなければならない。
`ref` 自体にはこの場合も span を補わない。

明示 relation または resource cue がないことを許す semantic candidate では、
存在しない明示根拠を補わず、上表の semantic と独立した正の根拠の条件だけを
評価する。複数の relation または cue を同じ意味へ決定的に縮約できない場合は、
配列の先頭、距離、長い span、score または先に生成した候補から一件を選ばない。
別の節、別の主題または別 step の明示根拠を借りず、共有末尾列の terminal
relation だけを後述の閉じた例外として扱う。

## input kind ごとの対応

次の表の「明示根拠」「target」「semantic」は、`SOT-ARCH-031` の共通表にある
事実のうち、当該 input kind に利用できるものを閉じて列挙する。

| input kind | 主題と `topicOrdinal` | 明示根拠 | target | semantic | 独立した正の根拠 |
|---|---|---|---|---|---|
| `law_search` | `SOT-ARCH-025` で個別分離した法令名、identifier または検索語を原文順に数える。分離しない場合は全 step を `1` とする | 同じ節で対象を束縛する core の search `direct_task` relation と law resource cue | 法令名、law ID、revision ID、法令番号、law resource へ明示的に束縛した `quoted_phrase` | law resource へ明示的に束縛した `morphological_phrase` | 公式 identifier、法令番号若しくは法令名、または search と law resource の明示根拠に束縛した検索語。形態素句だけで追加分岐にする場合は `SOT-ARCH-025` の独立主題でなければならない |
| `law_content_search` | 共有末尾列では後述の同値縮約後の topic span、個別分離では法概念、検索語または本文中の literal な法令名を原文順に数える。分離しない場合は `1` とする | 同じ節で対象を束縛する core の search `direct_task` relation、law provision resource cue、および検証済み共有末尾 relation | `quoted_phrase`、または後述の 2 若しくは 3 の条件で本文検索語へ変換した法令名。法令の identity としての法令名は利用しない | 公的 source を持つ法概念と `morphological_phrase` | 公的 source を持つ法概念、同じ節の明示検索対象、`SOT-ARCH-025` が独立主題とした一般検索語、または後述の 2 若しくは 3 で本文検索語へ変換した法令名。task/resource cue だけでは足りない |
| `law_read` | 明示 read と結び付いた一つの法令 target を一主題とする。入力 `ref` は span を作らず、同じ read の位置付き cue を原文順の根拠にする。分離しない場合は `1` とする | core の read `direct_task` relation と、位置付き target では同じ read に束縛した law resource cue。span なしの `ref` では前節の閉じた例外に従う | 法令名、law ID、revision ID または法令番号。法令 `ref` は span なしの `official_identifier` としてだけ利用する | 利用しない | 検証済み law `ref`、または一意な公式 identifier・法令番号・法令名と明示 read の組。一つの裸の法令名または一般語だけで read を作らない |
| `law_article_read` | 同じ read に束縛した法令 target と article/paragraph location の組を一主題とする。法令 target が `ref` の場合は最初の article span を位置付き target とする。個別に明示された組は原文順に数え、分離しない場合は `1` とする | core の read `direct_task` relation と、位置付き法令 target では同じ read に束縛した law provision resource cue。`ref` と位置付き article target の組では前節の閉じた例外に従う | 法令名、law ID、法令番号、article および任意の paragraph。法令 `ref` は span を持たない | 利用しない | 検証済み law `ref` または一意な法令 target、検証済み article location、および明示 read がすべて必要。article だけ、revision ID と article の組、または一般語で補わない |
| `law_updates` | 明示的に個別分離された完全な日付を原文順に数える。通常の一日付は `1` とする | core の list-updates `direct_task` relation と updates resource cue | 完全な暦日 | 利用しない | 完全な暦日と、同じ主題の list-updates/updates 明示根拠が必要。cue だけ、不完全日付または相対日は不可 |

cue の role と意味における閉じた対応は次のとおりとする。ここでいう resource cue
の意味は前処理 artifact が持つ意味であり、候補 model の `resource` 値とは
同じ列ではない。とくに `law_updates` の候補 resource は `law` だが、
明示 resource cue の意味は `updates` とする。

| input kind | 候補の task/resource | task cue の意味 | resource cue の意味 |
|---|---|---|---|
| `law_search` | `search/law` | `search` | `law` |
| `law_content_search` | `search/law_provision` | `search` | `law_provision` |
| `law_read` | `read/law` | `read` | `law` |
| `law_article_read` | `read/law_provision` | `read` | `law_provision` |
| `law_updates` | `list_updates/law` | `list_updates` | `updates` |

`law_article_read` に `law` cue、`law_read` に `law_provision` cue、または
`law_updates` に `law` cue を近い resource として代用しない。許可した cue が
ない semantic candidate は弱い候補として評価できても、別の意味の cue を
明示根拠に格上げしない。

同じ節または同じ主題への束縛を一意に決められない場合は、その事実を当該
input kind に使わない。近いという理由だけで別の節、別 task または別 resource の
事実を借りない。

`topicOrdinal` の共通計算は `SOT-ARCH-032` を定義元とする。本規定の表と
共有末尾列が示す core 固有の主題分離結果だけを同計算へ渡し、step 件数、
input kind、候補結合または生成順を別の ordinal 入力にしない。

## 日付束縛

検索または read の `asOf` に使う完全な日付 mention は、選択済み logical input
と同じ step へ `target_anchor/structured_reference` として対応できる。この対応は
`law_search`、`law_content_search`、`law_read` および `law_article_read` に限り、
その logical input が実際に同じ日付を `asOf` として保持する場合だけ作る。
`asOf` は独立した正の根拠、独立主題、追加 step、追加分岐または cluster の
`evidenceSpan` にしない。

`law_updates` の取得対象である完全な日付は、同じ主題の
`list_updates/updates` 明示根拠と組み合わせた
`target_anchor/structured_reference` とし、独立した正の根拠および private
mapping の cluster span 利用可能事実にできる。ただし、最終的な
`evidenceSpan` は `SOT-ARCH-032` の優先順で選ぶため、成立に必須の
`list_updates/updates` 明示根拠が先に選ばれる。日付を理由にこの優先順を
上書きしない。同じ日付 mention を一つの draft で `asOf` と updates target の
両方へ束縛できる場合は、どちらかを推測して使わず、その曖昧な事実を除いた後に
各 step の成立条件をもう一度確認する。

## 法令名を本文検索語へ投影する条件

法令名出現を `law_content_search` の検索語へ変換できるのは、法令の identity
として `law_search`、`law_read` または `law_article_read` へ束縛する場合ではなく、
次の順で最初に成立する一つの経路がある場合だけとする。先の経路が成立した場合は、
後の経路から検索語または根拠を重ねない。

1. 同じ span の `quoted_phrase` が、同じ節の core search task と
   law provision resource に束縛されている
2. 同じ span の `quoted_phrase` がなく、`SOT-ARCH-025` の `それぞれ`、
   `個別に`、`一つずつ`または`各々`の明示 cue による個別分離で、その法令名 span
   自体が一つの主題範囲と完全一致し、同じ主題の task/resource が
   `law_content_search` に一意に定まる
3. 同じ span の `quoted_phrase` がなく、後述する検証済み共有末尾列で、その
   法令名 span が一つの `topicSpan` と完全一致する

1 では `quoted_phrase` の原文表記を検索語とし、layer は `target_anchor`、
根拠 code は `general_term` とする。法令名出現は同じ span の候補意味を確認する
補助事実にはできるが、検索語、独立した正の根拠または cluster span を二重に
追加しない。

2 と 3 では、同じ主題範囲と完全一致する同一 span・同一 `surface` の
法令名出現を、一つの本文検索表記群として扱う。この群は profile 内の一時的な
判定単位であり、新しい前処理事実、field または公開 model として保存しない。
群の `surface` から前後の Unicode White_Space だけを除いた原文表記を、一つの
`allTerms` 要素とする。比較用正規化値、読み、law ID、revision ID、法令番号
または辞書の canonical title へ置換しない。

群に `matchKind` が `unique_typo_correction` ではない法令名出現が一件以上ある
場合は、その該当出現をすべて、`law_content_search` に限って同じ主題の
明示検索対象とし、同じ step の `target_anchor/official_alias` へ対応させる。
同一 span・同一 `surface` が複数の law identity に対応していても、どの identity
も選ばず、別 draft、別 step、追加の正の根拠または複数の検索語へ増やさない。
この identity の多重性は、本文検索語の step 束縛における曖昧性とみなさない。
候補の `official_alias` は重複のない一 code とし、群は独立した正の根拠を
一件だけ満たす。派生した `allTerms` の文字列を別の事実にしたり、
`general_term` を追加したりしない。

群の全出現が `matchKind=unique_typo_correction` の場合は、同じ span の
`quoted_phrase` が 1 の条件を独立に満たさない限り、2 または 3 だけで
本文検索語へ変換しない。この場合も変換根拠は `quoted_phrase` とし、
誤記補正した法令名を検索語へ置き換えない。

同じ法令名 span に read、law search または article read の target 束縛も成立する
場合は、節、relation または主題の境界だけで content search への束縛を一意に
分離できなければ本文検索語へ投影しない。法令名が照会文に存在すること、法令名と
law provision cue の距離が近いこと、または同じ候補に別の content search step が
あることだけでは変換しない。law ID、revision ID および法令番号を本文検索語へ
変換しない。

三経路の task、resource、主題および競合確認は、法令名 span と同じ節にある事実
だけで行う。先行節または後続節の content search cue を借りず、一つの節に
content search と read、law search または article read の束縛が競合して
一意に分けられない場合は、原文順、距離または relation 配列の先頭で解消しない。

## 共有末尾列

法令コア profile だけが、`SOT-MODEL-031` の `SharedTerminalSequence` を
`law_content_search` の複数主題へ利用できる。

1. terminal relation が core の採用済み search task と law provision resource に
   一意に対応することを確認する
2. 各 `topicSpan` と完全一致する法令名、法概念または検索語のうち、本文検索の
   logical input を構築できる意味だけを評価する
3. 異なる span から同じ logical input の意味署名が得られた場合は
   `SOT-ARCH-025` に従い根拠 span を和集合し、最初の位置へ一 step として縮約する
4. 同じ span の異なる意味は縮約せず、同じ `topicOrdinal` と
   `evidenceSpan` を持つ代替 draft として評価する
5. 同値縮約後の独立 step が五件以上なら、一部を残さず
   `step_limit_exceeded` とする

1 の照合には、sidecar が保持する正確な `terminalTaskRelation` の identity、
`clauseSpan`、subject および predicate を使い、同じ surface の別 relation、
同じ節の最初の relation または距離が近い cue へ置き換えない。
terminal subject が `SOT-ARCH-025` で閉じて採用した `教えて` または
`教えてください` の場合は、その対応自体が core の
`search/law_provision` を一意に表すため、別の明示 resource cue がないことだけを
理由に拒否しない。

それ以外の terminal subject では、同じ節の採用済み resource cue を
`law_provision` の一意味へ一意に縮約できる場合だけ sidecar を消費する。
`教えて` または `教えてください` の節に明示 resource cue がある場合も、
適用できる全 cue と relation が `law_provision` と core search の terminal
relation に両立しなければならない。`law`、`updates`、
`judicial_decision` その他の意味と競合する cue、content search 以外の採用済み
task cue relation、別節だけにある cue、または複数の意味へ残る cue を、
配列順若しくは距離で一件へ解消しない。
同じ節に、terminal relation の `search/law_provision` と両立しない採用済み
task cue relation または resource cue が一件でもある場合は sidecar を消費しない。
とくに read、list-updates または別 profile の task を terminal relation より
弱いものとして捨てず、節、relation または主題の境界で両方を一意に分離できる
場合だけ、それぞれの独立 step として評価する。

同じ `topicSpan` に複数の意味があるかは、少なくとも input kind、型付き logical
input の意味署名、元になった位置付き事実の kind、および profile 固有の意味 identity
で区別する。同じ span の法概念、`morphological_phrase`、`quoted_phrase` または
法令名投影が同じ検索文字列を作っても、異なる意味 identity である場合は一 draft
へ evidence を先に和集合しない。同じ事実 kind と同じ意味 identity を持つ重複経路
だけを同値として縮約する。法令名投影で同一 surface の複数 law identity を一群へ
扱う境界は、前節の規則を優先する。

二つ以上の topic がそれぞれ複数意味を持つ場合の draft 列は、
`SOT-ARCH-032` の限定代替列に従う。各 topic の第一候補からなる基準 draft と、
一度に一 topic だけを追加意味へ置き換えた draft を評価し、複数 topic の追加意味を
同時に組み合わせた Cartesian 分岐を作らない。core の topic-local draft は、
当該 topic の型付き logical input、位置付き正の根拠および検証済み共有末尾
relation だけを持つ。同じ span の `quoted_phrase`、法令名投影、法概念または
`morphological_phrase` の各意味は、この topic-local draft を
`SOT-ARCH-032` の score と通常 `tieBreak` で比較して第一候補と追加意味を決める。

共有 terminal cue の span は各 step の `explicit_task` 根拠として対応できるが、
各 topic は自身の topic span に、上表の独立した正の根拠を一つ以上持たなければ
ならない。同じ terminal cue だけで根拠のない topic を追加しない。

共有末尾列を `law_search`、read、article read、updates または
`judicial-cases` の候補へ変換しない。profile は sidecar から原文、separator、
token または新しい cue を復元しない。

## cluster 対応

`topicOrdinal` の値、同じ主題の共有および未分離時の `1` は
`SOT-ARCH-032` に従う。core profile は、前述の input kind ごとの主題分離と
検証済み共有末尾列の結果だけを同規定へ渡す。

各 step の `evidenceSpan` は、上表で当該 step に利用可能かつ実際にその step を
生成した事実だけから、`SOT-ARCH-032` の
`explicit_task_resource`、`target_anchor`、`semantic_expansion` の優先順で
一件を選ぶ。入力 `ref`、`asOf` だけの日付、別 step の cue または候補全体の
span を代用しない。

同じ cluster と完全な意味署名を持つ draft の統合、代表 draft、根拠と
`conceptSources` の和集合および source tuple 衝突時の失敗は
`SOT-ARCH-032` に従う。本規定で score を再計算したり、生成経路の件数を
分岐数として加えたりしない。

候補の `evidenceCodes` と score に使う code は、`SOT-ARCH-029` に従い、
各 step の同値経路を先に正規化してから候補全体へ和集合する。別 step の
`official_identifier` を理由に、独立した content search step の
`legal_concept`、`morphological_context` または `general_term` を削除しない。

## provider と実行状態からの独立

候補生成、主題、根拠、score および cluster の入力に、provider ID、source ID、
canonical URL、endpoint、query parameter、adapter DTO、検索結果、route 優先、
通信成否または稼働状態を使用しない。

入力 `SourceResourceRef` は共通の不透明参照として、法令系 input kind が許す
`resourceType` と version 構造だけを profile で確認する。provider/source の
registry 対応、capability binding および materialization は後段へ残す。
法令コア profile が候補生成時に e-Gov その他の provider route を選ばない。

## 確認

外部 network を使わない profile、cluster および integration test で、少なくとも
次の固定 test ID を確認する。

- `core-evidence-mapping-input-kinds`
- `core-evidence-mapping-private-lifetime`
- `core-evidence-mapping-topic-positive`
- `core-evidence-mapping-ref-no-span`
- `core-shared-terminal-evidence-cluster`
- `core-evidence-mapping-provider-independent`
- `core-law-name-content-projection`
- `core-evidence-mapping-fail-closed`
- `core-evidence-production-neutral`
- `core-shared-terminal-task-resource-binding`
- `core-multi-step-evidence-step-local-normalization`
- `core-bounded-non-cartesian-alternatives`

五 input kind の許可事実と禁止事実、同一 span の別意味、異なる span の同一意味、
一から四主題、五主題、`asOf` と updates 日付の差、法令 `ref`、別節 cue、
別 profile cue および provider metadata の差を fixture にする。

五 input kind では、閉じた task/resource cue 対応をそれぞれ正常系にし、
`law_article_read` への `law`、`law_read` への `law_provision` および
`law_updates` への `law` の代用を拒否する。同種または異種の複数 step では、
各 target を含む節の relation と resource cue だけが対応し、要求全体の最初の cue
を全 step へ共有しないことを確認する。`topicOrdinal` は明示的な主題分離でだけ
増え、分離しない能力別 multi-step はすべて `1` になることを確認する。

`asOf` を持つ search、content search、read および article read では、日付を
`target_anchor/structured_reference` へ対応させても独立した正の根拠または
cluster span にしない。同じ日付を updates target とする正常系では、同じ主題の
明示 cue と合わせて正の根拠および cluster span 利用可能事実にできる一方、
最終的な `evidenceSpan` は `SOT-ARCH-032` の優先順どおり明示 cue になることを
確認する。

`core-evidence-mapping-ref-no-span` では、法令 `ref` を
`boundary/official_identifier` へ対応させても span を補わず、独立主題、
追加 step または cluster の `evidenceSpan` にしないことを確認する。
span を持たない `law_read` では、一意な compatible read relation の位置付き
task cue を cluster に使用し、別の明示 resource cue を要求しない。同じ節に
resource cue がある場合は、すべて `law` と両立する場合だけ正常系とする。
read relation が一件、零件および二件以上の fixture、ならびに互換 resource cue
なし、互換 cue あり、競合 cue ありの fixture を組み合わせる。
`ref` を持つ article read では、一意に束縛した read task cue と位置付き article
target だけを `SOT-ARCH-032` の優先順へ渡し、別の明示 resource cue を要求しない。
同じ節に resource cue がある場合の両立条件も同様に確認する。

法令名と同じ span の引用句、明示的な個別主題、共有末尾 topic、裸の法令名、
一意な誤記補正、同一 surface の複数 law identity、read と content search の
競合および law ID を fixture にし、許可した三条件だけが原文表記の本文検索語に
なること、ならびに複数 identity を選択または複数検索へ展開しないことを確認する。
引用句、個別主題または共有末尾の task/resource が別節にだけある場合、および
同じ節の read または law search と一意に分離できない場合は投影しない。

共有末尾では、sidecar の正確な terminal relation を使うこと、
`教えて` と `教えてください` は別の明示 resource cue がなくても閉じた
`law_content_search` 対応として成立すること、および同じ節に競合する resource
cue または両立しない task cue がある場合は成立しないことを確認する。
同じ topic span に法概念と
`morphological_phrase` その他の別意味がある場合は evidence を一 draft へ
和集合せず、別の代替 draft とする。二 topic が同時に多義である fixture は
`core-bounded-non-cartesian-alternatives` で、基準 draft と一 topic ずつの
置換だけを作り、Cartesian 組合せを作らない。同一 topic に
`quoted_phrase`、法令名投影、法概念および `morphological_phrase` の意味を
与え、topic-local draft の正規化済み score と通常 `tieBreak` から第一候補と
全追加意味の順が一意に決まり、入力配列順を変えても同じになることを確認する。

一つの事実を同じ draft の複数 step または複数 `topicOrdinal` へ束縛できる
曖昧な節、同じ draft・step・事実へ競合する layer/code または cluster span 可否を
与えた mapping、範囲外 span および独立した正の根拠を失った主題を fixture にし、
一方を選んだ contribution または部分的な mapping を作らないことを確認する。
検証済み共有末尾 relation を各 topic step の明示 task 根拠として共有し、各 topic
span が別の正の根拠を持つ正常系は、この曖昧性として拒否しない。

複数 step 候補の evidence と score は
`core-multi-step-evidence-step-local-normalization` で、step 内の同値経路を
正規化した後で和集合する。別 step の強い根拠によって独立 step の弱い根拠が
削除されず、`SOT-ARCH-029` の最終順序になることを確認する。

`core-evidence-production-neutral` では、第 5 段階より前の active profile
metadata と production composition root から本規定の test 専用経路を構成できず、
active request が sidecar、限定代替列または private evidence mapping を
decision、候補または外部呼出しへ使用しないことを、file name の除外ではなく
loader の構成不能性と active runtime の観測で確認する。

`core-shared-terminal-task-resource-binding` では、`教えて` と
`教えてください` の正確な terminal relation が、別の明示 resource cue なしで
`search/law_provision` へ一意に対応する正常系を確認する。同じ節の互換な
`law_provision` cue、競合する resource cue、両立しない task cue、別節だけの cue、
同じ surface の別 relation および複数意味へ残る cue をそれぞれ fixture にし、
許可した正常系以外では sidecar を消費しないことを確認する。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](../20-model/25-legal-query-preprocess-result.md)
- [SOT-MODEL-031: SharedTerminalSequence](../20-model/31-shared-terminal-sequence.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-029: 複数 step 候補の根拠保持](29-multi-step-evidence-preservation.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](31-unified-query-intent-evidence-layer.md)
- [SOT-ARCH-032: 統合照会の限定分岐保持](32-unified-query-bounded-branch-retention.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](../50-engineering/24-unified-query-evaluation-gate.md)
- [SOT-ENG-035: 統合照会 profile metadata 成果物契約](../50-engineering/35-unified-query-profile-metadata-artifact-contract.md)
