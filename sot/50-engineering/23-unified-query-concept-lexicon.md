# SOT-ENG-023: 統合法情報照会の法概念辞書

- 状態: 有効

## 規定

統合法情報照会で使う法概念辞書は、法令名辞書と分離した、出典付き、版付きかつ不変のスナップショットとして管理する。

## 役割

法令名辞書は、正式名称、読み、公式略称および出典付き別名を法令 ID へ解決する。法概念辞書は、法令名ではない利用者語から、採用済み task/resource と型付き検索語の候補を作るための根拠を提供する。

例えば、一般に使われる「永住権」という語を、公式資料で確認できる「永住許可」に関する法情報検索候補へ結び付けることはできるが、特定の法令、条文、要件または結論へ一意に縮約してはならない。

一つの概念が複数候補へ対応する事実を保持し、planner へ曖昧性として渡す。法概念辞書は、法令名辞書の alias、provider 固有 synonym または法律判断の定義元にならない。

複数候補の自動実行禁止は、照会文が候補 resource を特定しない場合の既定と
する。利用者が `条文` または `裁判例` のように一つの候補 resource を
具体的に明示した場合は、その resource の候補だけを一意な取得意図として
扱える。複数の候補 resource をそれぞれ明示した場合は、各候補を代替解釈
ではなく必須意図として `SOT-ARCH-027` の合成対象にできる。`法情報` の
ように複数 resource を包含する語だけでは、この曖昧性を解消しない。
resource を明示した場合でも、公開結果へ投影する `conceptSources` は
辞書の active version に含まれる完全な source tuple
`{conceptId,title,url,confirmedOn}` と一致しなければならない。

法概念一致を実行根拠に使用した場合は、利用者が解釈根拠を確認できるよう、`SOT-MODEL-022` の `LegalConceptSource` だけを公開結果へ投影する。内部 weight、候補 template、衝突表、入力断片または辞書全体は公開しない。

## データ

スナップショットは schema version、辞書 version、生成日時および entry 配列を持つ。各 entry は少なくとも次を持つ。

- 安定した `conceptId`
- 日本語の概念表記と比較用表記
- 根拠となる政府、裁判所、国会または採用済み公的情報源の URL
- 資料名、確認日および資料上で確認した対応の説明
- 候補にできる task/resource
- capability request へ渡せる確認済みの正式語
- 複数候補、優先関係または自動実行禁止を表す衝突情報

候補の `officialTerm` は既定の公式検索語とする。同じ概念でも、登録済みの
表記によって公的資料上の公式検索語が異なる場合に限り、候補はその表記と
公式検索語の組を `termOfficialOverrides` として持つことができる。
override の表記は同じ entry の `terms` に存在しなければならず、比較用正規化で
一致した表記にだけ適用する。誤記、未知の表記または実行時の推測から override
を作らず、一致しない場合は候補の `officialTerm` を使用する。

entry が生成できる候補は `SOT-MODEL-022` の七つの step variant に限定する。provider ID、外部 query parameter、HTML selector、任意の JSON request または法的結論を格納しない。

## 採用しない根拠

次だけを根拠に概念 entry を採用しない。

- 一般ウェブ検索の件数、検索順位または私的解説
- 埋込みベクトル、編集距離、Kagome token または単語の共起
- 出典不明の俗称、利用者の過去入力または実行時に観察した選択
- 一つの情報源の空結果若しくは多い結果

誤記補正で得た文字列を alias または概念として辞書へ自動登録しない。誤記候補は、既存の確認済み表記へ一意に到達するリクエスト内の判定に限る。

## 読込みと不変性

起動時に schema、version、必須文字列、URL、確認日、task/resource、重複、正規化衝突および未採用 variant を検証する。検証に失敗した場合は統合照会を部分的に起動せず、構成エラーとして起動を失敗させる。

読込み後の entry、索引、候補 template および利用側へ返す配列は変更できない形で扱う。実行中の照会文、候補、選択または結果を保存せず、辞書を runtime に学習または更新しない。

法令名辞書と法概念辞書は別ファイル、別 loader、別衝突検査および別 version を持つ。両方の一致が競合した場合は、正式な法令名若しくは出典付き別名の完全一致を優先し、法概念一致で上書きしない。

## 更新

entry の追加、削除、対応先変更または根拠 URL 変更では辞書 version を更新し、`SOT-ENG-024` の既存カテゴリ回帰と対象 entry の fixture を同じ変更で追加する。

## 確認

代表的な法概念、複数候補、法令名との衝突、出典不明語、未採用 resource、不正 URL、重複 entry、誤記候補および壊れた schema を fixture にする。

法令名の完全一致が法概念より優先されること、概念が法律判断へ縮約されないこと、使用した法概念の公的資料だけを公開投影できること、provider package が辞書を import しないこと、および並行照会で辞書に変更がないことを確認する。

## 関連

- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-024: LegalQueryResult](../20-model/24-legal-query-result.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](../30-architecture/27-unified-query-cross-profile-composition.md)
- [SOT-ENG-022: 法令名検索辞書](22-law-name-search-lexicon.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
