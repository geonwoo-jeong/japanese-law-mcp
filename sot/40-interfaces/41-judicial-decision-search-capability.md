# SOT-IF-041: `judicial-decision.search` capability v1

- 状態: 有効

## 規定

`judicial-decision.search@1` は、検索語に一致する公式掲載裁判例を共通モデルで返す、読取り専用の型付き capability とする。

## 能力識別子

| 項目 | 値 |
|---|---|
| `ProviderCapability.id` | `judicial-decision.search` |
| `ProviderCapability.majorVersion` | `1` |
| `ProviderCapability.level` | `extended` |
| `ProviderCapability.stability` | `stable` |

## 型付き入力

`JudicialDecisionSearchRequestV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `query` | string | はい | 裁判例を探す検索語 |
| `limit` | integer | いいえ | 返却上限。既定 20、最大 30 |
| `continuationToken` | string | いいえ | 同じ条件の続きを取得する不透明な継続トークン |

`query` は有効な UTF-8 とし、先頭末尾の Unicode whitespace を除いた後に 1 byte 以上 512 byte 以下で、ASCII 制御文字を含めない。内部の文字、全角半角および連続空白は変更しない。`limit` は 1 以上 30 以下とする。

## 型付き出力

`JudicialDecisionSearchPageV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `items` | `SourcedResource<JudicialDecisionSummary>[]` | はい | DOM 順の公式掲載裁判例 |
| `page` | `SourcePage` | はい | 返却件数、同じ item 単位で示せる場合の公式総件数および存在する場合の継続位置 |

各 item の `ref.key.resourceType` は `judicial-decision` とし、`ref.key.resourceId` は provider mapping が公式詳細 URL から作る canonical ID とする。同じ裁判例が複数の掲載カテゴリーに掲載されている場合は、カテゴリーごとの公式詳細 URL を別の item として保持する。検索結果を同じ裁判例識別子、事件番号、事件名または裁判日だけで統合、上書きまたは重複排除しない。

## 継続取得と空結果

継続取得は `SOT-IF-016` に従う。選択した情報源が安定した snapshot と決定的な並び順を提供できない場合は `page.nextToken` を発行しない。その provider が発行できるトークンがない状態で空でない `continuationToken` を受け取った場合は、外部呼出し前に `invalid_argument` とする。

情報源が総件数を明示する場合だけ `page.totalCount` と `totalRelation` を設定する。該当結果がない場合は `items: []`、`returnedCount: 0` の成功結果とし、`not_found` にしない。

## ポートと失敗

能力別ポートは `Search(context.Context, Request) (Page, error)` とし、外部 HTML、query parameter、CSS selector または provider 固有 DTO を公開しない。

共通入力違反は `invalid_argument` とする。プロバイダー境界からは `unsupported_capability`、`unsupported_query`、`configuration_required`、`source_auth_failed`、`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` および `unsafe_source_content` が到達し得る。

## 確認

検索語と上限の境界、空でない未発行 token の拒否、空結果、総件数の有無、DOM 順と重複保持、型付き port、情報源エラーおよび入力不変性を契約テストで確認する。

## 関連

- [SOT-SCN-006: 公表裁判例を検索する](../10-scenarios/06-search-judicial-cases.md)
- [SOT-MODEL-020: JudicialDecisionSummary](../20-model/20-judicial-decision-summary.md)
- [SOT-IF-016: 情報源の継続取得](16-source-continuation-contract.md)
- [SOT-IF-043: 裁判所「裁判例検索」HTML 情報源](43-source-courts-hanrei-html.md)
- [SOT-IF-047: MCP `search_judicial_cases`](47-mcp-search-judicial-cases.md)
