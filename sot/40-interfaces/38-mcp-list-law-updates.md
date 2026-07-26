# SOT-IF-038: MCP `list_law_updates`

- 状態: 有効

## 規定

`list_law_updates` は、一つの暦日を受け取り、その日に公式情報源の更新一覧へ掲載された法令を、正確な総件数を伴う完全な一覧として返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `date` | string | はい | 実在する `YYYY-MM-DD` 形式の暦日 | 更新一覧の対象日 |

欠落、`null`、形式不正、実在しない暦日および定義していない入力項目は受け付けず、外部情報源を呼び出す前に `invalid_argument` とする。収録開始日、現在日その他のプロバイダー固有の提供範囲は公開入力の形式制約に含めない。

## 出力

成功時の `structuredContent` は、次の JSON object とする。

| 名前 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `date` | date | はい | 更新一覧の対象日 |
| `totalCount` | integer | はい | `items` の正確な総件数 |
| `items` | object[] | はい | 公開用に投影した `LawUpdate` の完全な一覧 |

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

このツールは、入力の `date` を変更せず `law.update.list@1` の型付き入力へ渡す。primary route から返された `SourcedResource<LawUpdate>` は、共通契約に従って検証した後、`data` だけを `items` へ投影する。

内部の `ref`、`provenance`、`SourcePage` および continuation に関する項目は公開結果へ含めない。公開結果に `nextToken`、`continuationToken` その他の継続位置を追加しない。

## 日付、件数および空結果

- 出力の `date` とすべての `items[].updatedOn` は入力の `date` と一致させる。
- `totalCount` は `items` の配列長と一致する 0 以上の正確な件数とする。
- 内部ページの対象日、`totalRelation: exact`、`totalCount`、`returnedCount` または item 数がこれらの条件と一致しない場合は `invalid_source_response` とする。
- 該当する法令がない場合は、入力と同じ `date`、`totalCount: 0` および `items: []` を持つ成功結果とする。`not_found` へ変換しない。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 有効な暦日が選択したプロバイダーの公式な提供範囲外である場合は `unsupported_query` を返す。
- route または設定、認証に関する失敗は、原因に応じて `unsupported_capability`、`configuration_required` または `source_auth_failed` を返す。
- 外部情報源の制限、一時障害または現在のローカル同時実行上限は、原因に応じて `rate_limited`、`source_timeout`、`source_unavailable` または `source_busy` を返す。
- 外部契約、応答または安全上限の問題は、原因に応じて `source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` または `unsafe_source_content` を返す。
- 上記へ分類できない内部処理の失敗は `internal_error` を返す。

内部の情報源エラーを別のコードへ縮約せず、各コードの `retryable`、`details` および秘密情報の禁止を `SOT-IF-027` に従って保持する。

## 公開ツール構成

この SOT は、既存の `search_laws`、`get_law`、`get_article` および `search_law_content` に `list_law_updates` を追加する。stdio MCP の公開ツールは、この五つとする。

## 確認

少なくとも次を契約テストで確認する。

- 有効な暦日だけを受理し、不正な入力では外部情報源を呼び出さないこと
- プロバイダー固有の提供範囲外を `unsupported_query` として保持すること
- 全 `LawUpdate` 項目と `source` を公開し、省略可能値、明示された boolean の `false` および値の欠落を区別すること
- 入力日、出力日および全 item の対象日が一致すること
- `totalCount` が item 数と一致し、空の一覧を成功として返すこと
- `ref`、`provenance`、内部ページおよび continuation に関する項目を公開しないこと
- `SOT-IF-027` の情報源エラーコードと公開情報を変更せず返すこと

## 関連

- [SOT-SCN-005: 更新日から法令更新一覧を取得する](../10-scenarios/05-list-law-updates.md)
- [SOT-MODEL-003: LegalSource](../20-model/03-legal-source.md)
- [SOT-MODEL-019: LawUpdate](../20-model/19-law-update.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-034: `law.update.list` capability v1](34-law-update-list-capability.md)
- [SOT-IF-037: e-Gov 法令 API Version 1 の組込み採用](37-egov-v1-built-in-adoption.md)
