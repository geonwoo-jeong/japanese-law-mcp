# SOT-PROD-011: 統合法情報照会の製品範囲

- 状態: 有効

## 規定

Japanese Law MCP は、既存の専門 MCP ツールを維持したまま、日本語の照会文を採用済みの公式法情報取得能力へ結び付ける `query_legal_information` を、追加の公開入口として提供する。

## 利用目的

統合法情報照会は、利用者が能力名やプロバイダー名を知らなくても、法令名、法令番号、条文参照、事件番号、日付または法概念を含む日本語から、取得対象と処理を安全に選べるようにするための機能である。

この機能が行うのは、公式情報の検索、読取りおよび更新一覧取得である。取得した資料を根拠に法的な結論を生成する機能、相談内容へ法を適用する機能、または回答文を合成する機能ではない。

裁判例詳細の読取りは、採用済みの provider、source および resource type として構造検証できる `SourceResourceRef` を利用者が入力に渡した場合だけ行う。参照が過去に同じプロセスから発行されたことは要件または検証対象にせず、事件番号、題名または URL だけから検索結果の一件を推測して読まない。

## 初期の実行範囲

初期版が実行できる task と resource の組合せは次に限定する。

| task | resource | 使用する能力 | 利用条件 |
|---|---|---|---|
| `search` | `law` | `law.search@1` | 法令コア |
| `search` | `law_provision` | `law.content.search@1` | 法令コア |
| `read` | `law` | `law.document.read@1` | 法令コア |
| `read` | `law_provision` | `law.article.read@1` | 法令コア |
| `list_updates` | `law` | `law.update.list@1` | 法令コア |
| `search` | `judicial_decision` | `judicial-decision.search@1` | `judicial-cases` が有効な場合だけ |
| `read` | `judicial_decision` | `judicial-decision.read@1` | `judicial-cases` が有効な場合だけ |

法令のリビジョン、基準日、法令番号、条、項、事件番号および情報源参照は resource ではなく、上表の能力へ渡す型付き条件として扱う。

## 既存専門ツールとの関係

`query_legal_information` は、`search_laws`、`search_law_content`、`get_law`、`get_article`、`list_law_updates`、`search_judicial_cases` および `get_judicial_case` を置き換えない。

入力と取得対象を決定的に指定できる利用者は、ページ継続、プロバイダー固有の互換入力または厳密な結果型を持つ既存専門ツールを使用する。統合法情報照会は、自然文から適切な能力へ到達するための facade とし、専門ツールの契約や経路を変更しない。

## 対象外

初期版は次を行わない。

- 法的助言、適法性評価、勝敗予測、先例性の判断、手続要件の判定または利用者の事実への法適用
- 私的データベース、二次資料、一般ウェブ記事または全国の条例を対象とする横断調査
- 翻訳、多言語分類または日本語以外の意味解釈
- `compare`、`trace`、改正履歴の連続比較または立法理由の復元
- 未採用の `legislative-history`、`tax` または `labor` 拡張パックの実行
- すべてのプロバイダーへの無条件 fan-out、空結果を理由にした別 resource への再分類、または検索結果からの法的回答合成

上記の task、resource または拡張パックを追加する場合は、製品範囲、利用シナリオ、型付き能力、公開 MCP 契約、情報源の権威境界および検証方法を定める別の有効な SOT を先に採用する。

実行可能な情報取得と、法的助言、翻訳または未採用 task/resource が一つの照会文に混在する場合は、情報取得部分だけを実行しない。照会全体を対象外として外部情報源を呼ばず、取得要求だけを分けて専門ツールまたは統合照会へ入力するよう案内する。

## 利用上の境界

空結果は、該当する法令、裁判例または資料が存在しないことを証明しない。裁判例結果には `SOT-PROD-010` の収録範囲注意を保持し、統合法情報照会によって完全性、現在の有効性または先例性を追加で保証しない。

## 確認

公開ツールの契約テストで上表以外の組合せを実行しないこと、既存専門ツールの schema と経路が変わらないこと、対象外との混在要求が外部情報源を呼ばないこと、裁判例 read が検証済み `ref` を必要とすること、および裁判例の収録範囲注意を失わないことを確認する。

## 関連

- [SOT-PROD-008: e-Gov 法令コアの製品範囲](08-egov-law-core-scope.md)
- [SOT-PROD-009: 選択型法情報拡張パックの境界](09-selectable-legal-information-extension-packs.md)
- [SOT-PROD-010: 裁判例拡張パック](10-judicial-cases-extension-pack.md)
- [SOT-SCN-009: 日本語の法情報を統合照会する](../10-scenarios/09-query-legal-information.md)
- [SOT-ARCH-024: 統合照会の内部境界と公開境界](../30-architecture/24-unified-query-internal-public-boundary.md)
- [SOT-IF-051: MCP `query_legal_information`](../40-interfaces/51-mcp-query-legal-information.md)
