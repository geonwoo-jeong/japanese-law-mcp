# SOT-MODEL-025: LegalQueryPreprocessResult

- 状態: 有効

## 規定

`LegalQueryPreprocessResult` は、一つの日本語照会文から得た provider 非依存の事実を、原文上の位置を失わない不変な値として表す内部モデルとする。

## 構造

結果は、検証済みの原文、比較用正規化値、入力で受け取った任意の `SourceResourceRef`、および次の出現配列を持つ。

| 出現 | 必須項目 | 意味 |
|---|---|---|
| `lawNameMentions` | span、surface、law ID、revision ID、法令番号、正式名称、match kind | 法令名辞書に登録した正式名称、読みまたは別名との一致 |
| `legalConceptMentions` | span、surface、concept ID、正式表記、match kind | 法概念辞書との一致 |
| `cueMentions` | span、surface、profile ID、cue ID、match kind | 起動時に query profile から渡された構文 cue との一致 |
| `identifierMentions` | span、surface、kind、law ID、任意の revision ID または法令番号 | 辞書スナップショットで法令との対応を確認できる公式識別子、構造を検証した法令履歴 ID または法令番号 |
| `dateMentions` | span、surface、`Date` | 原文に完全な暦日として明記された日付 |
| `articleMentions` | span、surface、provision、article number | 条番号または枝番号を含む条番号 |
| `paragraphMentions` | span、surface、paragraph number | 項番号 |
| `caseNumberMentions` | span、surface、era、year、case code、serial number、search text | `SOT-MODEL-027` の完全な裁判事件番号 |
| `queryTermMentions` | span、surface、kind | 原文で検索対象として区切られた引用句、または一回の形態素解析から文法的な検索対象として確認できた句 |
| `cueTaskRelations` | subject、predicate、clause span、kind | `SOT-MODEL-029` に従い、一回の形態素解析から確認した cue の task 関係 |

入力 `ref` は原文上の出現ではないため span を作らず、構造を検証した複製と有無を別に保持する。前処理結果は `ref` の provider、source、pack または選択した read capability との対応を決めない。

## span と順序

span は原文 UTF-8 の byte offset で、`startByte` を含み `endByte` を含まない。零以上で `startByte < endByte <= len(query)` とし、両端は UTF-8 rune 境界でなければならない。各出現の `surface` は `query[startByte:endByte]` と完全に一致する。

各出現配列は `startByte` の昇順、同じ開始位置では `endByte` の降順、同じ span では安定した識別子の昇順に保持する。同じ種別、span および識別対象の重複を除く。異なる法令または概念へ対応する同じ表記は縮約せず、同じ span の複数出現として保持する。

同じ種別、span および識別対象に対して複数の照合方法が成立した場合は、一件へまとめた上で、`exact`、`comparison_normalized`、`registered_term`、`unique_typo_correction` の順に強い `matchKind` を保持する。

法令名と条項、複数の項、日付と task の意味上の関連付けは前処理結果で
確定しない。共通前処理は `SOT-MODEL-029` の構文 relation だけを確定し、
query profile は span、原文順および検証済み relation から候補を組み立てる。
前処理後に原文を別の規則で再解析して、節、助詞、述語または失われた位置を
補ってはならない。

## match kind

法令名と法概念の `matchKind` は次のいずれかとする。

1. `exact`: 外側の空白を除いた照会文全体が、検証済みの辞書表記または cue 表記と完全一致した
2. `comparison_normalized`: `surface` と検証済み表記が Unicode 比較用正規化後に一致した
3. `registered_term`: Kagome が自然文の一部から user dictionary 登録語として、その span の表記を抽出した
4. `unique_typo_correction`: 最小編集距離となる既存の検証済み表記が一つだけであり、その補正先へ一意に到達した

cue は最初の三種類だけを使用し、誤記から task または resource の明示 cue を作らない。

誤記候補の一意性は補正先の比較用辞書表記について判定する。一つの補正先表記が複数の法令または概念に対応する場合は、その対象をすべて同じ span の出現として保持し、前処理で一件へ選ばない。最小距離の補正先表記が複数ある場合は、どの誤記出現も作らない。

法令名の完全一致と比較用正規化一致は、同じ span の法概念一致より優先する。優先とは法概念で法令名を上書きしないことであり、別の span に根拠がある法概念を削除することではない。

前処理は出現の存在と位置だけを確定し、同じ法令名 span に後続する条番号や、二つの法令名の間にある条番号をどの法令へ対応付けるかは決めない。対応付け規則は query profile が所有する。

## 一般検索語

`queryTermMentions.kind` は `quoted_phrase` または `morphological_phrase` に限定する。いずれも原文の候補 span を保持するだけであり、検索対象の task、resource、論理条件、外部情報源へ送る値または意味 score を確定しない。

`quoted_phrase` は `「…」`、`『…』`、`“…”` または `"…"` の対応する区切りの内部とする。区切り自体を span に含めず、内部の先頭と末尾にある Unicode White_Space は span を内側へ移して除く。内部の空白と記号は原文どおり保持する。空、空白だけ、不完全または入れ子になった区切りからは出現を作らない。同じ span に法令名、法概念、cue または構造化参照がある場合も、利用者が検索対象を明示した事実として `quoted_phrase` を保持する。

`morphological_phrase` は、同じ照会で一回だけ得た Kagome token 列から、次の条件を満たす最大の日本語名詞句として作る。

- 文字、数字または結合文字からなる一つ以上の自立した名詞 token を中心とし、単独の数詞、代名詞、非自立名詞、接頭辞または接尾辞を中心にしない
- 前後の名詞接続接頭辞と名詞接尾辞、および名詞句をつなぐ `の` は、中心となる名詞がある場合だけ同じ span に含める。`第` その他の数接続接頭辞、数詞および助数詞は、それらの後ろに自立した名詞の中心が続く場合だけ修飾部として同じ span に含める
- Unicode Script property が `Hiragana`、`Katakana` または `Han` の scalar value を一文字以上含み、比較用正規化後に二文字以上であり、部分的な n-gram を追加せず最大 span だけを保持する
- `について`、`に関する`、`に係る`、`を含む`、`を除く` またはその活用形へ接続する、同じ節の注入済み cue へ `を` 若しくは `は` で接続する、または `の` の直後に注入済み cue がある句だけを採用する。`を` または `は` と cue の間に動詞、形容詞若しくは助動詞がある場合は、後続 cue を前の名詞句の接続根拠にしない
- 上記の句に `と`、`または`、`又は`、`若しくは`、`および`、`及び`、`並びに` または読点で直接並列された句は、個別の出現として採用する
- 原文全体が一つの名詞句と句読点または空白だけからなる場合は、明示 cue がなくても一つの弱い一般語として採用できる

法令名、法概念、cue、公式識別子、日付、条、項、事件番号、`quoted_phrase` または user dictionary token と一 byte でも重なる形態素句は作らず、それらを名詞句の境界とする。不完全または入れ子になった引用区切りの内側も保護し、そこにある文字列を `morphological_phrase` として再解釈しない。引用区切りの内側にある cue は辞書出現として保持しても、引用区切りの外側にある形態素句の構文上の接続根拠にはしない。`morphological_phrase` の候補を作るために task、resource または provider 固有語を共通前処理へ埋め込まず、能力別 profile が注入した cue の位置だけを構文上の境界として利用する。

query profile は、`queryTermMentions` と cue その他の出現の位置から候補を作り、`all`、`any`、`exclude`、一つの検索か複数 step か、および採用する task/resource を決める。引用句または形態素句を外部情報源へそのまま送れるとはみなさず、選んだ logical input の constructor で能力ごとの文字数、空白および演算子制約を検証する。

## 構造化参照

`identifierMentions.kind` は `law_id`、`law_revision_id` または `law_number` に限定する。

- `law_id` は対応する law ID を持つ。
- `law_revision_id` は対応する law ID と revision ID を持つ。
- `law_number` は対応する law ID と法令番号を持つ。

`law_id` と `law_number` は、法令名辞書の同じ固定スナップショットで
値と法令の対応を確認できる場合だけ出現にする。法令番号が複数法令へ
対応する場合は、同じ span に対象ごとの出現を保持する。

`law_revision_id` は、辞書に完全一致する値に加え、次の条件をすべて
満たす同じ法令の別の法令履歴 ID を出現にできる。

- 先頭の十五文字が固定スナップショットに存在する `law_id` と完全一致する
- 全体が `{law_id}_YYYYMMDD_{amending_law_id}` の四十文字であり、
  `YYYYMMDD` が実在する暦日である
- `amending_law_id` が ASCII の大文字英字または数字十五文字である。
  制定時を表す `000000000000000` も含む
- 前後に ASCII の英数字または `_` が隣接せず、原文上の一つの識別子として
  区切られている

この構造は e-Gov 法令データドキュメンテーションの
[法令履歴 ID](https://laws.e-gov.go.jp/docs/law-data-basic/da91fe9-law-revisions/)
に従う。前処理で確認するのは既知の法令との対応、閉じた構造および暦日の
妥当性までとし、その履歴が情報源に実在するかは外部取得結果との
`revisionId` 完全一致で確認する。構造が不正な法令履歴 ID らしい文字列を、
内部に含まれる `law_id` だけの出現へ縮退させない。

これら以外の識別子らしい未知の文字列を、形式だけから公式識別子として
採用しない。

完全な裁判事件番号は `identifierMentions` ではなく、[SOT-MODEL-027](27-judicial-case-number-mention.md) の `caseNumberMentions` として保持する。これは入力の構造を確認した位置付き事実であり、公式情報源で存在または一意な資源対応を確認した識別子ではない。

日付は西暦の `YYYY年M月D日` または `YYYY-MM-DD` として完全に明記され、実在する暦日だけを `YYYY-MM-DD` の `Date` へ変換する。年若しくは月だけの値、相対日、現在日または識別子内部の八桁数字から日付を補わない。

条番号は正の十進整数を `_` で連結した `LawArticleLocation` と同じ正規形へ変換する。例えば `第398条の2` は `398_2` とする。項番号は一以上の整数とする。原始附則を原文で明示した条だけを `supplementary` とし、それ以外を `main` とする。

## 上限と不変性

原文は `LegalQueryRequest` の上限に従う。比較用正規化値は 4096 byte 以下とする。各出現配列は六十四件以下、cue 出現は百二十八件以下、全出現の合計は二百五十六件以下とする。`caseNumberMentions` と `queryTermMentions` も各配列と全出現の上限に含める。`cueTaskRelations` は出現件数とは別に百二十八件以下とする。上限を超えた場合は黙って切り捨てず、外部情報源を呼ぶ前に前処理エラーとする。

constructor は原文、原文から決定的に導出した比較用正規化値、`ref` および全配列を複製して検証する。`cueTaskRelations` は同じ結果の `cueMentions` だけを参照し、参照先 cue を欠く relation、relation だけを残した結果または `SOT-MODEL-029` と異なる順序を拒否する。relation の syntax role と kind の対応は relation 自身の constructor で検証し、前処理結果の constructor が cue artifact を再読込しない。getter は内部配列と relation の入れ子を変更できない複製として返す。辞書、索引、Kagome tokenizer、cue 語彙および syntax role は起動後に変更せず、前処理結果を照会間で保持しない。

結果は score、confidence、task、resource、capability、required pack、候補、選択、provider ID、route、外部 DTO、検索結果または法的結論を持たない。辞書内部の候補 template と weight も保持しない。

## エラー境界

nil context と cancellation を拒否またはそのまま伝播し、途中までの結果を返さない。辞書、tokenizer、cue または索引の構築に失敗した場合は起動を失敗させる。照会時の解析失敗を空の一致へ変換しない。

## 確認

次をネットワークなしの単体テストで確認する。

- 二つの法令名と二つの条番号を含む原文で span と出現順を保持する
- 同じ条の複数項、枝番号、附則、全角数字および完全な暦日を正規形へ変換する
- revision ID 内部の数字、不完全日および存在しない日を日付にしない
- 正式名称、別名、比較用正規化、Kagome 抽出、一意な誤記および曖昧な補正先を区別する
- 同じ表記が複数対象に対応する事実を保持し、一件へ選ばない
- 法令名と法概念の競合、profile cue の差替え、入力 `ref` の複製および provider 非選択を確認する
- 完全な事件番号を型付き出現として抽出し、不完全な事件番号を採用せず、事件番号だけから `ref` を作らない
- 引用区切りを除いた検索語、内部空白、同じ語の複数 span、空・不完全・入れ子の引用および引用と辞書出現の併存を確認する
- 文法的に接続した最大名詞句と並列句だけを抽出し、後続する自立名詞を持つ数詞修飾部は保持し、述語を越えた cue、制御語、単独数詞、事件番号、user dictionary token および保護された構造化 span を一般検索語にしない
- query profile が一般検索語へ論理条件と task/resource を付与し、能力別 logical input の制約を満たさない原文をそのまま外部情報源へ送らない
- cue の直接 task、目的語と述語の直接接続および単独 task を
  `cueTaskRelations` として保持し、引用句、言及表現、別の節および中間述語を
  task relation にしない
- relation の cue 参照、syntax role、順序、件数上限および getter の深い複製を
  確認し、query profile が原文から同じ関係を再解析しない
- 件数上限、UTF-8 境界、入力不変性、getter の深い複製、決定的順序、context cancellation および共有前処理器の race 非発生を確認する

## 関連

- [SOT-MODEL-016: SourceResourceRef](16-source-resource-ref.md)
- [SOT-MODEL-018: LawArticleLocation](18-law-article-location.md)
- [SOT-MODEL-022: LegalQueryCandidate](22-legal-query-candidate.md)
- [SOT-MODEL-027: JudicialCaseNumberMention](27-judicial-case-number-mention.md)
- [SOT-MODEL-029: CueTaskRelation](29-cue-task-relation.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](../30-architecture/22-unified-query-planning-pipeline.md)
- [SOT-ENG-022: 法令名検索辞書](../50-engineering/22-law-name-search-lexicon.md)
- [SOT-ENG-023: 統合法情報照会の法概念辞書](../50-engineering/23-unified-query-concept-lexicon.md)
