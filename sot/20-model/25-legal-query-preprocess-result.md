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
| `identifierMentions` | span、surface、kind、law ID、任意の revision ID または法令番号 | 辞書スナップショットで対応を確認できる公式識別子または法令番号 |
| `dateMentions` | span、surface、`Date` | 原文に完全な暦日として明記された日付 |
| `articleMentions` | span、surface、provision、article number | 条番号または枝番号を含む条番号 |
| `paragraphMentions` | span、surface、paragraph number | 項番号 |

入力 `ref` は原文上の出現ではないため span を作らず、構造を検証した複製と有無を別に保持する。前処理結果は `ref` の provider、source、pack または選択した read capability との対応を決めない。

## span と順序

span は原文 UTF-8 の byte offset で、`startByte` を含み `endByte` を含まない。零以上で `startByte < endByte <= len(query)` とし、両端は UTF-8 rune 境界でなければならない。各出現の `surface` は `query[startByte:endByte]` と完全に一致する。

各出現配列は `startByte` の昇順、同じ開始位置では `endByte` の降順、同じ span では安定した識別子の昇順に保持する。同じ種別、span および識別対象の重複を除く。異なる法令または概念へ対応する同じ表記は縮約せず、同じ span の複数出現として保持する。

同じ種別、span および識別対象に対して複数の照合方法が成立した場合は、一件へまとめた上で、`exact`、`comparison_normalized`、`registered_term`、`unique_typo_correction` の順に強い `matchKind` を保持する。

法令名と条項、複数の項、日付と task の関連付けは前処理結果で確定しない。query profile は span と原文の順序から候補を組み立てる。前処理後に原文を別の規則で再解析して、失われた位置を補ってはならない。

## match kind

法令名と法概念の `matchKind` は次のいずれかとする。

1. `exact`: 原文の出現 `surface` 自体が、検証済みの辞書表記または cue 表記と完全一致した
2. `comparison_normalized`: `surface` と検証済み表記が Unicode 比較用正規化後に一致した
3. `registered_term`: Kagome が自然文から user dictionary 登録語として、その span の表記を抽出した
4. `unique_typo_correction`: 最小編集距離となる既存の検証済み表記が一つだけであり、その補正先へ一意に到達した

cue は最初の三種類だけを使用し、誤記から task または resource の明示 cue を作らない。

誤記候補の一意性は補正先の比較用辞書表記について判定する。一つの補正先表記が複数の法令または概念に対応する場合は、その対象をすべて同じ span の出現として保持し、前処理で一件へ選ばない。最小距離の補正先表記が複数ある場合は、どの誤記出現も作らない。

法令名の完全一致と比較用正規化一致は、同じ span の法概念一致より優先する。優先とは法概念で法令名を上書きしないことであり、別の span に根拠がある法概念を削除することではない。

前処理は出現の存在と位置だけを確定し、同じ法令名 span に後続する条番号や、二つの法令名の間にある条番号をどの法令へ対応付けるかは決めない。対応付け規則は query profile が所有する。

## 構造化参照

`identifierMentions.kind` は `law_id`、`law_revision_id` または `law_number` に限定する。

- `law_id` は対応する law ID を持つ。
- `law_revision_id` は対応する law ID と revision ID を持つ。
- `law_number` は対応する law ID と法令番号を持つ。

識別子らしい未知の文字列を、形式だけから公式識別子として採用しない。法令名辞書の同じ固定スナップショットで対応を確認できた値だけを出現にする。法令番号が複数法令へ対応する場合は、同じ span に対象ごとの出現を保持する。

日付は西暦の `YYYY年M月D日` または `YYYY-MM-DD` として完全に明記され、実在する暦日だけを `YYYY-MM-DD` の `Date` へ変換する。年若しくは月だけの値、相対日、現在日または識別子内部の八桁数字から日付を補わない。

条番号は正の十進整数を `_` で連結した `LawArticleLocation` と同じ正規形へ変換する。例えば `第398条の2` は `398_2` とする。項番号は一以上の整数とする。原始附則を原文で明示した条だけを `supplementary` とし、それ以外を `main` とする。

## 上限と不変性

原文は `LegalQueryRequest` の上限に従う。比較用正規化値は 4096 byte 以下とする。各出現配列は六十四件以下、cue 出現は百二十八件以下、全出現の合計は二百五十六件以下とする。上限を超えた場合は黙って切り捨てず、外部情報源を呼ぶ前に前処理エラーとする。

constructor は原文、`ref` および全配列を複製して検証する。getter は内部配列を変更できない複製として返す。辞書、索引、Kagome tokenizer および cue 語彙は起動後に変更せず、前処理結果を照会間で保持しない。

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
- 件数上限、UTF-8 境界、入力不変性、getter の深い複製、決定的順序、context cancellation および共有前処理器の race 非発生を確認する

## 関連

- [SOT-MODEL-016: SourceResourceRef](16-source-resource-ref.md)
- [SOT-MODEL-018: LawArticleLocation](18-law-article-location.md)
- [SOT-MODEL-022: LegalQueryCandidate](22-legal-query-candidate.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](../30-architecture/22-unified-query-planning-pipeline.md)
- [SOT-ENG-022: 法令名検索辞書](../50-engineering/22-law-name-search-lexicon.md)
- [SOT-ENG-023: 統合法情報照会の法概念辞書](../50-engineering/23-unified-query-concept-lexicon.md)
