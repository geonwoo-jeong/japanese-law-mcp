# SOT-PROD-012: 日本法情報照会の意図分類と受理境界

- 状態: 有効

## 規定

`query_legal_information` が扱う日本語照会は、採用済みの法情報取得、外部呼出しを伴わない明確化、利用可能性制約付きの採用済み取得、対象外の非実行要求という四つの製品クラスへ決定的に分類し、同じ照会文を広い検索や回答生成へ読み替えない。

## 目的

本規定は、日本の法令・裁判例 MCP がどの種類の利用者要求を製品として受け、
どの種類を受けないかを、task 名、cue 語彙、公開 status または provider 名ではなく、
製品上の意図クラスとして固定する。

公開 task/resource の採用範囲自体は [SOT-PROD-011](11-unified-legal-query-scope.md) を
定義元とし、意図を検出する語彙、relation、score、候補保持、公開 status および
MCP 契約は、関連する model・architecture・interface SOT を参照する。
本規定は、どの意図クラスを製品として受理し、どの境界で止めるかだけを定義する。

ここで定義する四クラスは製品上の意図境界であり、公開結果の `status` と
一対一対応するものではない。どの意図クラスがどの公開 status へ写像されるかは
[SOT-IF-051](../40-interfaces/51-mcp-query-legal-information.md)、
[SOT-MODEL-023](../20-model/23-legal-query-plan.md) および
[SOT-MODEL-024](../20-model/24-legal-query-result.md) を定義元とする。

## 意図クラス

### 1. 採用済み取得意図

利用者が日本語で、採用済みの公式法情報を取得する意図を示し、採用済みの規則で
安全に取得要求へ解釈できる照会は、実行候補になり得る。

このクラスに属する取得は、
[SOT-PROD-011](11-unified-legal-query-scope.md) が定める採用済みの
task/resource 組合せに限る。`ref` 入力境界および pack 条件も同 SOT に従い、
本規定では取得組合せを重複して定義しない。

### 2. 非実行の明確化意図

照会文が日本語の法情報取得を意図していても、安全に一意化できない場合は、取得要求を推測実行せず、外部呼出しなしの明確化へ分類する。

このクラスに属する代表例は次のとおりとする。

- resource、対象法令、条文位置または主題分離が安全に確定しない
- 同じ略称や概念が複数の採用済み意味へ分岐する
- 一候補の step 上限を超え、切り詰め実行できない
- 同じ照会文から複数候補を内部保持しても、実行条件を満たす一候補または hedge pair へ絞れない

このクラスは「取得要求ではない」ことを意味しない。取得の意図は認めるが、安全に決められないため非実行とする境界である。

### 3. 利用可能性制約付きの採用済み取得意図

採用済みの取得意図だが、有効な拡張パックがなければ実行できない照会は、
採用済み取得の下位分類として扱い、実行候補を保持したまま availability 上の
非実行として扱う。

現行では、`judicial-cases` が必要な裁判例検索または裁判例読取りだけがこのクラスに属する。将来、別の採用済み pack が追加された場合は、その pack を要求する取得意図も同じクラスへ属する。

pack 依存の取得意図は、法令コアの下位候補へ自動で置き換えず、利用可能な一部 step だけを部分実行しない。

### 4. 対象外の非実行意図

次の要求は、法情報 MCP の取得意図へ縮約せず、外部呼出しなしの対象外とする。

| 非実行クラス | 内容 |
|---|---|
| `legal_advice` | 違法性評価、勝敗予測、個別事案への法適用、どうすべきかの助言 |
| `translation` | 条文、裁判例または説明の翻訳 |
| `comparison_or_trace` | 二時点比較、改正差分、連続追跡、比較表生成 |
| `relationship_analysis` | 影響グラフ、引用関係図、可視化、ネットワーク分析 |
| `unadopted_pack_or_source` | 統合照会では未採用の legislative-history、tax、labor、民間 DB、自治体条例などの範囲 |
| `answer_synthesis` | 取得した資料から最終的な法律回答や結論文を作る要求 |

採用済み取得意図と対象外意図が同じ照会文に混在する場合は、取得部分だけを分離実行しない。照会全体を対象外として扱う。

## 分類原則

### 日本法情報への限定

本規定の「採用済み取得意図」は、日本の法令・裁判例という採用済みの公式情報源に対する取得要求だけを指す。一般ウェブ検索、海外法、私的データベース、解説記事または自由回答生成を含めない。

### 製品クラスと公開結果の分離

本規定の四分類は製品上の受理境界であり、公開 `status`、内部 signal、
selection mode または非実行 reason code そのものではない。具体的な公開結果への
写像は、[SOT-SCN-009](../10-scenarios/09-query-legal-information.md)、
[SOT-ARCH-023](../30-architecture/23-unified-query-selection-and-hedging.md) および
[SOT-IF-051](../40-interfaces/51-mcp-query-legal-information.md) を定義元とする。

### 取得と回答生成の分離

本システムは、法情報を取得し、構造化した結果を返す。取得した結果を材料に最終的な法的結論、翻訳文、比較文または推奨文を生成することは、意図クラスの段階で対象外とする。

### 決定的分類

同じ照会文を、外部結果、LLM、利用者履歴または provider 固有の都合で別の意図クラスへ動的に移さない。分類は、日本語照会文、検証済み `ref`、採用済み pack 状態および固定 profile set から決定的に行う。

### 広い fan-out の禁止

曖昧だからという理由で、照会文全体を複数 resource・複数 provider へ一斉検索し、後段の人間または LLM に正解判定を委ねない。曖昧な取得要求は、実行より先に明確化または対象外へ分類する。

### provider 名と意図クラスの分離

provider/source 名は、意図クラスそのものではない。利用者が provider 名を述べても、
それが採用済み取得対象を一意に示さない限り、取得意図へ自動変換しない。

## 期待する利用者体験

- 法令や裁判例を調べたい利用者は、task 名を知らなくても日本語照会から採用済み取得へ到達できる
- 曖昧な照会は、黙って広い検索をせず、明確化が必要だと分かる
- 対象外の要求は、取得部分だけが勝手に実行されず、MCP の責務境界が保たれる
- pack がないため実行できない要求は、対象外ではなく availability の問題として扱われる

## 確認

少なくとも次を確認する。

- 採用済み取得、明確化、pack 依存および対象外の各クラスに属する代表照会が、公開結果で異なる非実行理由または実行可否として観測できる
- 法的助言、翻訳、比較および関係分析が、採用済み取得意図へ縮約されない
- 曖昧な照会が、複数 provider への広い fan-out ではなく明確化になる
- pack 依存の裁判例要求が、法令コアの別候補へ静かに置き換わらない
- provider 名や外部情報源名だけの入力が、採用済み取得意図へ自動変換されない

## 関連

- [SOT-PROD-011: 統合法情報照会の製品範囲](11-unified-legal-query-scope.md)
- [SOT-PROD-009: 選択型法情報拡張パックの境界](09-selectable-legal-information-extension-packs.md)
- [SOT-PROD-010: 裁判例拡張パック](10-judicial-cases-extension-pack.md)
- [SOT-SCN-009: 日本語の法情報を統合照会する](../10-scenarios/09-query-legal-information.md)
- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](../30-architecture/23-unified-query-selection-and-hedging.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](../30-architecture/31-unified-query-intent-evidence-layer.md)
- [SOT-IF-051: MCP `query_legal_information`](../40-interfaces/51-mcp-query-legal-information.md)
