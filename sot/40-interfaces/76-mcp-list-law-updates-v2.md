# SOT-IF-076: MCP `list_law_updates` v2

- 状態: 有効

## 規定

`list_law_updates` は、一つの暦日と任意の返却上限を受け取り、その日に公式情報源の更新一覧へ掲載された法令の正確な総件数、返した件数および省略した件数を、公式順序の先頭項目とともに返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `date` | string | はい | 実在する `YYYY-MM-DD` 形式の暦日 | 更新一覧の対象日 |
| `limit` | integer | いいえ | 1 以上 512 以下、既定値 50 | 公開結果へ含める先頭項目の上限 |

欠落した `date`、各入力の `null`、型不正、形式不正、実在しない暦日、範囲外の `limit` および定義していない入力項目は受け付けず、外部情報源を呼び出す前に `invalid_argument` とする。収録開始日、現在日その他のプロバイダー固有の提供範囲は公開入力の形式制約に含めない。

## 出力

成功時の `structuredContent` は、次の JSON object とする。

| 名前 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `date` | date | はい | 更新一覧の対象日 |
| `totalCount` | integer | はい | 公式情報源の一日分の正確な総件数 |
| `returnedCount` | integer | はい | `items` に返した件数 |
| `omittedCount` | integer | はい | `totalCount - returnedCount` |
| `truncated` | boolean | はい | `omittedCount` が 1 以上なら `true`、0 なら `false` |
| `items` | object[] | はい | 公開用に投影した `LawUpdate` の先頭一覧 |

`items` の各 object は、`SOT-MODEL-019` の JSON 表現を変更せず使用する。

| 名前 | 型 | 必須 |
|---|---|---:|
| `updatedOn` | date | はい |
| `lawId` | string | はい |
| `title` | string | はい |
| `lawType` | string | いいえ |
| `lawNumber` | string | いいえ |
| `titleKana` | string | いいえ |
| `previousTitle` | string | いいえ |
| `promulgationDate` | date | いいえ |
| `amendmentTitle` | string | いいえ |
| `amendmentLawNumber` | string | いいえ |
| `amendmentPromulgationDate` | date | いいえ |
| `effectiveDate` | date | いいえ |
| `effectiveDateNote` | string | いいえ |
| `documentUrl` | string | いいえ |
| `enforcementPending` | boolean | いいえ |
| `authorityReviewPending` | boolean | いいえ |
| `source` | `LegalSource` | はい |

`source` は `SOT-MODEL-003` の `id`、`name`、`authority` および `serviceUrl` を持つ JSON object とする。省略可能な値が存在しない項目は省略し、`null`、空文字列または推測値で補わない。情報源が boolean の `false` を明示した場合は項目を `false` として保持し、値の欠落と区別する。

## 共通 capability からの投影

入力の `date` だけを変更せず `law.update.list@1` の型付き入力へ渡し、`limit` は公式情報源へ送らない。primary route から一日分の完全な `SourcedResource<LawUpdate>` 一覧と正確な総件数を取得して共通契約を検証した後、公式情報源の順序を変えず、先頭から実効 `limit` 件までの `data` だけを `items` へ投影する。公開上限の外にある内部 item も検証し、不正な item を切捨てで隠さない。

内部の `ref`、`provenance` および `SourcePage` は公開結果へ含めない。公開結果に `offset`、`nextOffset`、`nextToken`、`continuationToken` その他の継続位置を追加しない。同じ日付を後から再取得したときに同じ一覧 snapshot であることを公式情報源が保証しないため、継続取得を合成せず、省略の有無と件数を各応答で明示する。

## 日付、件数、省略および空結果

- 出力の `date` とすべての `items[].updatedOn` は入力の `date` と一致させる。
- 内部ページの `totalRelation` は `exact` とし、内部 `totalCount`、`returnedCount` および完全な内部 item 数を一致させる。不一致は `invalid_source_response` とする。
- 公開 `totalCount` は内部の正確な総件数を変更せず保持する。
- 公開 `returnedCount` は `items` の配列長と一致し、`totalCount` と実効 `limit` の小さい方とする。
- `omittedCount` は `totalCount - returnedCount` とし、負の値を許さない。
- `truncated` は `omittedCount > 0` と同値にする。
- 該当する法令がない場合は、入力と同じ `date`、三つの件数がすべて `0`、`truncated: false` および `items: []` を持つ成功結果とする。`not_found` へ変換しない。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 有効な暦日が選択したプロバイダーの公式な提供範囲外である場合は `unsupported_query` を返す。
- route または設定、認証に関する失敗は、原因に応じて `unsupported_capability`、`configuration_required` または `source_auth_failed` を返す。
- 外部情報源の制限、一時障害または現在のローカル同時実行上限は、原因に応じて `rate_limited`、`source_timeout`、`source_unavailable` または `source_busy` を返す。
- 外部契約、応答または安全上限の問題は、原因に応じて `source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` または `unsafe_source_content` を返す。
- 上記へ分類できない内部処理の失敗は `internal_error` を返す。

内部の情報源エラーを別の code へ縮約せず、各 code の `retryable`、`details` および秘密情報の禁止を `SOT-IF-027` に従って保持する。

## 公開ツール構成

公開 tool 集合における `list_law_updates` の位置付けと件数は `SOT-IF-067` を定義元とし、統合照会の登録は `SOT-IF-051` を定義元とする。

## 確認

少なくとも次を契約テストで確認する。

- `limit` の省略時に 50、明示時に 1 以上 512 以下を使用し、範囲外または不正な入力では外部情報源を呼び出さないこと
- プロバイダー固有の提供範囲外を `unsupported_query` として保持すること
- 208 件の内部完全一覧を既定入力で取得した場合、公式順序の先頭 50 件、`totalCount: 208`、`returnedCount: 50`、`omittedCount: 158` および `truncated: true` を返すこと
- 同じ 208 件に `limit: 208` を指定した場合、208 件すべて、`omittedCount: 0` および `truncated: false` を返すこと
- 公開上限外を含む全内部 item を検証し、不正な項目を省略として隠さないこと
- 全 `LawUpdate` 項目と `source` を公開し、省略可能値、明示された boolean の `false` および値の欠落を区別すること
- 入力日、出力日および全公開 item の対象日が一致すること
- 空の一覧で三つの件数、`truncated` および空配列が整合すること
- `ref`、`provenance`、内部ページおよび continuation に関する項目を公開しないこと
- `SOT-IF-027` の情報源エラー code と公開情報を変更せず返すこと

## 関連

- [SOT-SCN-016: 上限と省略件数を伴う法令更新一覧を取得する](../10-scenarios/16-list-law-updates-v2.md)
- [SOT-MODEL-003: LegalSource](../20-model/03-legal-source.md)
- [SOT-MODEL-019: LawUpdate](../20-model/19-law-update.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-034: `law.update.list` capability v1](34-law-update-list-capability.md)
- [SOT-IF-035: e-Gov 法令 API Version 1 更新一覧](35-source-egov-law-api-v1.md)
- [SOT-IF-037: e-Gov 法令 API Version 1 の組込み採用](37-egov-v1-built-in-adoption.md)
- [SOT-IF-067: `judicial-cases` と `judicial-citations` の有効化](67-judicial-citations-pack-activation.md)
- [SOT-IF-051: MCP `query_legal_information`](51-mcp-query-legal-information.md)
