# SOT-ARCH-009: 能力別情報源ポート

- 状態: 有効

## 規定

アプリケーションは、すべてのプロバイダー操作を一つの汎用メソッドへ集約せず、利用目的と情報の意味ごとに型を定めた小さな情報源ポートを所有する。

## 能力の境界

接続候補を次の能力群へ分ける。各能力の公開採用時には、対応するシナリオ、情報モデルおよびインターフェース SOT で入出力を定義する。

| 能力群 | 能力 ID の例 | 保持する意味 |
|---|---|---|
| 法令 | `law.search`、`law.content.search`、`law.document.read`、`law.article.read`、`law.revision.list`、`law.update.list` | 法令、リビジョン、条文および更新 |
| 国会会議録 | `parliament.meeting.search`、`parliament.meeting.read`、`parliament.speech.search`、`parliament.speech.read` | 会議と発言 |
| 議案 | `bill.list`、`bill.read`、`bill.progress.read`、`bill.document.read` | 提出回次、議案、審議経過および文書 |
| 行政資料 | `guidance.search`、`guidance.read` | 通達、通知および指針 |
| 行政裁決 | `administrative-decision.search`、`administrative-decision.read` | 裁決要旨および公表裁決 |
| 統計 | `statistics.table.search`、`statistics.metadata.read`、`statistics.data.read` | 統計表、次元、単位および観測値 |
| 法人 | `corporation.search`、`corporation.read`、`corporation.change.list` | 法人情報と変更情報 |
| 適格請求書 | `invoice.registration.read`、`invoice.validity.check`、`invoice.change.list` | 登録情報、指定日判定および変更情報 |
| 不動産 | `real-estate.transaction.search`、`real-estate.geospatial.read` | 取引情報と空間情報 |
| 開示 | `disclosure.filing.list`、`disclosure.artifact.read`、`xbrl.fact.read` | 提出書類、配布物および XBRL fact |

一つのプロバイダーは、実際に同じ意味を提供できるポートだけを実装する。未対応の能力を空の結果、既定値または推測値で実装しない。

`law.content.search` を共通能力として採用する場合は、検索語、論理条件、対象範囲および一致位置をプロバイダーに依存しない型で別の SOT に定義する。e-Gov の検索式は既存 `search_law_content` の互換インターフェースに留め、他のプロバイダーへ e-Gov の文法を要求しない。

既存の MCP ツール名、入力および出力は、自動的に同名または類似名の共通能力契約にはならない。公開ツールと能力別ポートの対応は mapping SOT で明示する。

## 禁止する形

- `Execute(operation, map)` のように操作名と無型の値だけを受けるポート
- すべての情報を `Search`、`Get` または `content` だけへ縮約するポート
- プロバイダー固有 DTO、HTML 要素、XML 要素または XBRL 要素をアプリケーションへ公開するポート

## 関連

- [SOT-ARCH-003: ユースケース境界](03-application-boundary.md)
- [SOT-MODEL-013: ProviderCapability](../20-model/13-provider-capability.md)
- [SOT-PROD-006: 関連公的情報の統合境界](../00-product/06-related-public-information-boundary.md)
- [SOT-IF-008: MCP `search_law_content`](../40-interfaces/08-mcp-search-law-content.md)
