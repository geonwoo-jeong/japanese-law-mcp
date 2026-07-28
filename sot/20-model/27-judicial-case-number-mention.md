# SOT-MODEL-027: JudicialCaseNumberMention

- 状態: 有効

## 規定

`JudicialCaseNumberMention` は、日本語照会文に完全な事件番号として明記された構造を、provider 非依存の位置付き事実として表す内部モデルとする。

このモデルは入力文字列の構造だけを確認する。裁判例の実在、裁判所、事件種別、公開カテゴリー、判決日、判決文または `SourceResourceRef` の存在を証明せず、それらを補完または推測しない。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `span` | byte span | はい | 原文中の位置 |
| `surface` | string | はい | `query[span.startByte:span.endByte]` と一致する原表記 |
| `era` | string | はい | `昭和`、`平成` または `令和` |
| `year` | integer | はい | 元号内の正の年。`元` は 1 |
| `caseCode` | string | はい | 空白と全角括弧を正規化した事件符号 |
| `serialNumber` | integer | はい | 正の事件番号 |
| `searchText` | string | はい | 構造上の表記を保った決定的な検索語 |

`span`、順序、重複排除、複製および全出現件数は `SOT-MODEL-025` に従う。

constructor は `surface` を本規定の一つの parser で再解析し、入力された `era`、`year`、`caseCode` および `serialNumber` が解析結果と完全に一致する場合だけ受理する。`searchText` は caller から受け取らず、同じ解析結果から constructor が導出する。`Validate` も保持した `surface` を再解析して全項目の一致を確認し、constructor を介さない不整合な値を拒否する。

同じ span にある出現の識別対象は、`era`、`year`、`caseCode` および `serialNumber` の組とする。

## 受理する表記

次の構造を、全必須構成要素がある場合だけ受理する。

```text
{元号}{年または元}[年]{事件符号}{番号}
```

角括弧の `年` は任意とする。`番号` は十進数字だけ、または対になった `第`、十進数字、`号` とする。`年` の有無と `第…号` の有無は互いに独立とする。

事件符号は、一組だけの対応する括弧と、括弧内の一文字以上の `Hiragana`、`Katakana` または `Han` の scalar value を必須とする。括弧の前後にも同じ文字種の接頭部または接尾部を置くことができ、括弧を除いた全文字数は一文字以上八文字以下とする。`caseCode` は、括弧を ASCII へ統一し、構造上の空白を除いた `（受）` から `(受)`、`特（わ）` から `特(わ)`、`（を）新` から `(を)新` のような値とする。

例えば `平成25(オ)1079`、`令和7(わ)第207号`、`令和元年（行ツ）164`、`令和4年(ネ)第10039号`、`平成26年特（わ）第914号` および `平成26特(わ)914` を受理する。ASCII または全角の括弧、ASCII または全角の十進数字、および各構造要素の境界にある Unicode White_Space を受理し、`surface` には原表記をそのまま保持する。ただし ASCII 制御文字、U+0085、U+2028 および U+2029 は空白としても受理しない。数字列または日本語文字列の途中にある空白は受理しない。

`searchText` は、検証済み構成要素から次のように決定的に導出する。

- 構造要素の境界にある空白を除く
- 数字で表した年と番号は、全角数字を ASCII へ変換し、先頭の零を除いた対応する正の整数の最短十進表記にする
- 全角括弧を ASCII 括弧へ変換する
- 年が 1 で `年` marker がある場合は入力の `元` または `1` にかかわらず `元年`、`年` marker がない場合は `1` とする
- 一以外の年では入力に `年` がある場合だけ `年` を保持し、`第…号` を使った場合はその対を保持する
- それ以外の文字、構成要素および順序を変更しない

例えば `令和　元　（行ツ）　１６４` の `searchText` は `令和1(行ツ)164`、`令和１年（受）第０１０５５号` は `令和元年(受)第1055号`、`令和０４年（ネ）第０１００３９号` は `令和4年(ネ)第10039号` とする。検索語の内部空白は裁判所の全文検索で別の複数語として扱われ得るため、検証済みの構造空白を原表記のまま外部検索へ渡さない。

年は、昭和では 1 以上 64 以下、平成では 1 以上 31 以下、令和では 1 以上 99 以下とする。番号は 1 以上 99,999,999 以下とする。

年だけ、符号だけ、番号だけ、零、符号付き数、小数、単独の `第` 若しくは `号`、括弧の不一致、複数の括弧組、九文字以上の事件符号、または上限を超える値からは出現を作らない。事件番号らしい文字列へ誤記補正を適用せず、欠けた構成要素を補わない。

## 利用境界

前処理は完全な事件番号を `caseNumberMentions` として一度だけ抽出し、その span を `morphological_phrase` の対象から除く。事件番号が引用区切りの内部にある場合は、利用者が検索対象を明示した事実として `quoted_phrase` を併存できる。

事件番号の出現だけでは task または resource を補わない。裁判例 query profile が採用した `search` task と `judicial_decision` resource の両 cue がある場合だけ、事件番号の位置から `judicial-decision.search` 候補を作り、根拠に `structured_reference` を付けることができる。選んだ出現の `searchText` から、`SOT-MODEL-022` の `JudicialDecisionSearchIntentV1` を constructor で作る。profile は capability request または provider query parameter を直接作らない。

同じ事件番号 span に `quoted_phrase` が併存しても、一つの検索意味を二候補へ重複させず、型付き事件番号の `structured_reference` を採用する。pack の有効状態は前処理または profile の意味候補を変更せず、`SOT-ARCH-023` の選択後にだけ実行可否へ反映する。

事件番号だけから `judicial-decision.read` 候補、`decisionId`、詳細 URL、provider ID または `SourceResourceRef` を作ってはならない。検索結果の第一件を後続読取りへ暗黙に使用せず、検索条件を provider 固有の事件番号 parameter へ分解しない。裁判所情報源への対応は `SOT-IF-044` だけを定義元とする。

## 外部根拠

裁判所の「裁判例検索」は、事件番号を左から元号、年、符号、番号の順で指定すると説明し、各構成要素の入力欄と符号一覧を公開している。公開符号には `(受)` のほか、`刑(わ)`、`特(わ)` および `(を)新` のように括弧の外側にも文字を持つ値がある。検索結果には `令和7(わ)第207号` と `平成26特(わ)914` のように、`年` と `第…号` の有無が一つの固定形式にならない表記がある。これらの事実は、2026 年 7 月 29 日に[裁判所「裁判例検索」](https://www.courts.go.jp/hanrei/search1/index.html)で確認した。

本モデルの表記受理規則と上限は、共通前処理で安全に構造を識別するための内部契約であり、裁判所が公開する全事件番号表記の網羅的な転記ではない。

## 確認

次をネットワークなしの単体テストで確認する。

- `年` と `第…号` の各組合せ、元年、全角括弧、全角数字および構成要素間の空白を同じ型へ変換する
- `(受)`、`特(わ)`、`(を)新` の符号構造を保持する
- 複数の事件番号を原文順に保持し、元の `surface` と byte span を失わない
- `surface` の構造空白、全角表記、数字の先頭の零および元年表記を決定的な `searchText` へ変換し、原表記を検索語として使わない
- caller が指定した構成要素と `surface` の再解析結果が異なる場合、および内部 `searchText` が導出値と異なる場合を拒否する
- 昭和、平成および令和の年上限、正の番号、事件符号の文字種と長さを検証する
- 年、符号若しくは番号の欠落、零、上限超過、単独 marker、複数若しくは不一致の括弧および隣接する不完全表記を採用しない
- ASCII 制御空白、U+0085、U+2028 および U+2029 を受理しない
- 事件番号 span を一般形態素句にせず、引用句との併存を保持する
- 裁判例検索候補が `structured_reference` を持ち、`searchText` を logical input に使う
- 事件番号だけでは候補を作らず、task と resource の両 cue があるときだけ一つの検索意味を作る
- 事件番号だけ、または事件番号を句読点で列挙しただけの照会は `standalone_structured_query` として外部呼出しなしの `unsupported` にする
- 引用事件番号を重複候補にせず、pack の有効・無効で同じ型付き候補と必須 pack を保持し、無効時は外部呼出しなしの `capability_unavailable` にする
- 事件番号だけから read 候補または `SourceResourceRef` を作らない
- 件数上限、入力と getter の不変性、決定的順序および並行前処理の race 非発生を確認する

## 関連

- [SOT-MODEL-016: SourceResourceRef](16-source-resource-ref.md)
- [SOT-MODEL-022: LegalQueryCandidate](22-legal-query-candidate.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](25-legal-query-preprocess-result.md)
- [SOT-MODEL-026: QueryProfileContribution](26-query-profile-contribution.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](../30-architecture/23-unified-query-selection-and-hedging.md)
- [SOT-IF-041: `judicial-decision.search` capability v1](../40-interfaces/41-judicial-decision-search-capability.md)
- [SOT-IF-044: 裁判所の裁判例検索マッピング](../40-interfaces/44-courts-hanrei-search-mapping.md)
