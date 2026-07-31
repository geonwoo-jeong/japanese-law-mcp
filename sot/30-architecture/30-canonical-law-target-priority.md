# SOT-ARCH-030: 解決済み法令対象の検索結果優先順位

- 状態: 有効

## 規定

法令名、出典付き別名または一意な軽微誤記を一つの法令へ安全に解決できた
検索では、解決済み法令の公式 ID を provider 非依存の対象として保持し、
情報源が返した同じ対象を現在の page の先頭へ安定的に移動する。

## 解決済み対象

`ResolvedLawTarget` は application 層の law-target resolver が一回の
法令検索中だけ作る不変な内部値とし、次を持つ。

| 項目 | 型 | 意味 |
|---|---|---|
| `lawId` | string | 法令名辞書で確認した公式法令 ID |
| `officialTitle` | string | 同じ辞書 entry の正式名称 |
| `matchKind` | string | `exact`、`comparison_normalized`、`registered_term` または `unique_typo_correction` |

自然文では、共通前処理が確定した法令名候補 span が一つだけであり、その
span の解決先 law ID が一つの場合だけ作る。同等の span-aware matcher を
使う場合も、法令名候補の開始位置と終了位置を先に確定し、その span にだけ
誤記判定を適用する。検証済み query 全体への fuzzy match から
`ResolvedLawTarget` を作らない。

検索語全体が一つの法令名表現である場合は、その全体を一つの候補 span と
みなせる。一つの span が複数 law ID へ対応する場合、複数の法令名 span が
ある場合、または誤記補正の最小候補が複数ある場合は
`ResolvedLawTarget` を作らない。

`lawId` と `officialTitle` の定義元は `SOT-ENG-022` の同じ固定スナップショット
とする。provider の検索順位、検索件数、本文、URL または edit distance
だけから対象を作らない。

法令名候補 span は次の順で照合し、最初に一つの law ID だけが残った場合に
解決を完了する。

1. 正式名称または出典を持つ別名との完全一致
2. Unicode NFKC、かな、空白および句読点を比較用に正規化した完全一致
3. Kagome の user dictionary が抽出した正式名称または別名との完全一致
4. rune 単位の Damerau-Levenshtein 距離が閉じた閾値内にあり、
   最小距離の law ID が一つだけである誤記候補

3 rune 未満の span には誤記補正を行わない。3 rune 以上 9 rune 以下では
距離 1、10 rune 以上 15 rune 以下では距離 2、16 rune 以上では距離 3 以下
かつ長い方の 20% 以下を上限とする。同じ最小距離で複数の law ID が残る場合、
自然文から複数の法令名候補 span が見つかる場合または一つの別名が複数の
law ID に対応する場合は解決しない。

比較用の正規化値、読み、token または編集途中の文字列を検索語にしない。
意味が似ているだけの語は、出典を持つ別名として辞書にない限り対象にしない。

`ResolvedLawTarget` は `LegalQueryPreprocessResult`、
`QueryProfileContribution`、`LegalQueryCandidate` または `LegalQueryPlan`
へ項目として追加しない。共通前処理は位置付きの事実を返すだけとし、
`search_laws` service または統合照会の法令検索 application facade が、
自身の一回の呼出しに必要なときだけ同じ resolver から対象を作る。

## 共通化する境界

`search_laws` service と `query_legal_information` の法令検索 facade は、
同じ法令名辞書、比較用正規化および一意な誤記判定を持つ law-target resolver
を共有する。

`search_laws` service は、一つの入力に対する共通前処理で得た Kagome token
と法令名候補 span を resolver へ渡す。統合照会の法令検索 facade は、
query profile がすでに一つへ分離した `LawSearchIntentV1.query` だけを
resolver へ渡し、元の自然文を再解析せず、Kagome を二回目に実行しない。
この logical input が正式名称、登録済み別名または一意な誤記として解決
できない場合は、解決済み対象なしで既存検索を続ける。

共有するのは対象 identity の確定までとする。`search_laws` の原検索優先と
確認検索、統合照会の意味 score、候補選択、明確化および複数 step の規則を
相互に流用しない。本文検索、条文読取り、裁判例検索および provider adapter は
この sidecar を解釈しない。

`ResolvedLawTarget`、補正前後の表記、辞書 entry、span および match kind は
公開 MCP 結果へ追加しない。公開結果に現れるのは、公式情報源が実際に返した
既存の `LawSummary` だけとする。

## page 内の安定優先

法令検索の正常な一 page を得た後、`ResolvedLawTarget` がある場合だけ、
`LawSummary.lawId == ResolvedLawTarget.lawId` の item を page の先頭へ移す。
対象 item が複数ある場合は相互の provider 順を保持し、その後に対象外 item を
元の provider 順で並べる安定 partition とする。

この投影で item を追加、削除、重複排除または別 page から取得しない。
`totalCount`、`nextOffset`、continuation、snapshot、出典、各 item の版および
対象外 item の相対順を変更しない。現在の page に対象 law ID がなければ、
page を変更せず返す。対象を探すための追加 page 取得または provider fallback を
行わない。

安定優先は provider adapter ではなく、正規化済み `LawSummary` と
`ResolvedLawTarget` の両方を持つアプリケーション層で行う。provider ごとの
title 表記、検索 score または DTO に基づく特例を設けない。

## ユースケースごとの扱い

`search_laws` は、検証済み原文による最初の検索を維持する。最初の検索が
非空であっても対象 item が同じ page にあれば安定優先を適用できる。
最初の検索が正常な空結果の場合だけ、解決済みの `officialTitle` で確認検索し、
その page に同じ安定優先を適用する。

最初の検索が非空で対象 item を含まない場合は、その結果を別の検索語で
置き換えない。情報源エラーの場合も確認検索を行わない。確認検索を含む公開契約は
`SOT-IF-053` に従う。

統合照会は、query profile が選択した既存の検証済み検索語を変更せず、
法令検索 facade がその logical input から解決済み対象を作れる場合だけ、
返却された law search page に安定優先を適用する。統合照会で
`ResolvedLawTarget.officialTitle` による確認検索または検索語の書換えを
行わない。正式名称による確認検索は、`search_laws` の原検索が正常な
空結果だった場合だけに限定する。

対象を一意に作れない法令名衝突は `SOT-ARCH-028` に従い、検索結果の順位で
一件へ縮約しない。

## 確認

少なくとも次の固定 test ID を resolver、application facade および統合 contract
test で確認する。

- `law-target-resolution-parity`: `search_laws` と統合照会が同じ位置付き入力から
  同じ解決済み law ID または同じ非解決を得る
- `law-target-page-stable-partition`: 対象 item だけを先頭へ移し、対象内・対象外の
  provider 順と page metadata を保持する
- `law-target-no-extra-fetch`: 統合照会では確認検索を行わず、専門 tool でも
  本規定の正常空結果条件以外に追加 page または fallback を取得しない
- `law-target-ambiguous-no-reorder`: 複数 span、複数 law ID、短い語または同率誤記で
  page を並べ替えない
- `law-target-unified-no-reparse`: 統合照会は分離済み logical input だけを使い、
  元の自然文の再解析または二回目の Kagome を行わない

正式名称、公式略称、補足別名、自然文内の一法令名および一意な挿入・削除・
置換・転置について、`search_laws` と統合照会が同じ law ID を解決済み対象と
することを確認する。

対象 item が page の三件目にある場合は先頭へ移し、ほかの item の相対順、
件数、次位置および出典を保持することを確認する。対象が page にない場合、
複数 law ID、複数法令を含む自然文、短い語および同率誤記候補では
page を変更しないことを確認する。

`著作券法` および同じ誤記を一つだけ含む自然文は、辞書上の解決先が一意で
情報源 page に `著作権法` の law ID がある場合、その item を先頭にする。
この fixture でも追加の page 取得、item の捏造または provider package から
辞書 package への依存がないことを確認する。

## 関連

- [SOT-SCN-011: 解決済み法令を検索結果で優先する](../10-scenarios/11-prioritize-resolved-law-search-result.md)
- [SOT-MODEL-001: LawSummary](../20-model/01-law-summary.md)
- [SOT-ARCH-010: プロバイダーの分離](10-provider-isolation.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](21-provider-independent-query-preprocessing.md)
- [SOT-ARCH-028: 法令別名衝突の基本法優先順位](28-law-alias-collision-ranking.md)
- [SOT-IF-053: MCP `search_laws` v3](../40-interfaces/53-mcp-search-laws-v3.md)
- [SOT-ENG-022: 法令名検索辞書](../50-engineering/22-law-name-search-lexicon.md)
