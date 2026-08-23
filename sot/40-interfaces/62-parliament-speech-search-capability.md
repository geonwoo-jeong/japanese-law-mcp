# SOT-IF-062: `parliament.speech.search` capability v1

- 状態: 有効

## 規定

`parliament.speech.search@1` は、発言本文、発言者、会議、院または開催日に一致する公式国会発言を、発言と会議の意味を保持した共通モデルで返す読取り専用の型付き capability とする。

## 能力識別子

| 項目 | 値 |
|---|---|
| `ProviderCapability.id` | `parliament.speech.search` |
| `ProviderCapability.majorVersion` | `1` |
| `ProviderCapability.level` | `extended` |
| `ProviderCapability.stability` | `stable` |

## 型付き入力

`ParliamentSpeechSearchRequestV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `query` | string | いいえ | 発言本文等に含まれる検索語 |
| `speaker` | string | いいえ | 発言者名の検索条件 |
| `meetingName` | string | いいえ | 会議名の検索条件 |
| `house` | string | いいえ | 院名の正規化済み列挙値 |
| `fromDate` | date | いいえ | 会議開催日の下限。境界を含む |
| `untilDate` | date | いいえ | 会議開催日の上限。境界を含む |
| `limit` | integer | いいえ | 返却上限。既定 20、最大 30 |
| `continuationToken` | string | いいえ | 同じ条件の続きに使う不透明な継続トークン |

`query`、`speaker`、`meetingName`、`house`、`fromDate` または `untilDate` のうち一つ以上を必須とする。`limit` と `continuationToken` だけを指定した要求は `invalid_argument` とする。

`query`、`speaker` および `meetingName` は有効な UTF-8 とし、先頭末尾の Unicode whitespace を除いた後に 1 byte 以上 512 byte 以下で、ASCII 制御文字を含めない。内部の文字、全角半角、連続空白および語順を変更しない。

`query` に半角空白で区切った複数語がある場合は全ての語を含む発言を検索し、`speaker` または `meetingName` に半角空白で区切った複数語がある場合はいずれかの語を含む値を検索する。この意味を共通層で別の検索式へ書き換えず、同じ capability を実装する provider は同じ包含条件を満たす。

`house` は次のいずれかとする。

- `house_of_representatives`
- `house_of_councillors`
- `both_houses`
- `conference_of_both_houses`

`both_houses` と `conference_of_both_houses` は利用者が入力した院名の意味を保持する
ため別の列挙値とする。現在の NDL API では同じ検索結果になるが、共通 capability で
二つを同義語へ統合したり、一方へ書き換えたりしない。

`fromDate` と `untilDate` は実在する `YYYY-MM-DD` とし、両方を指定した場合は `fromDate` を `untilDate` 以下とする。`limit` は 1 以上 30 以下とする。

## 型付き出力

`ParliamentSpeechSearchPageV1` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `items` | `SourcedResource<ParliamentSpeech>[]` | はい | 情報源が返した順序の国会発言 |
| `page` | `SourcePage` | はい | 今回の返却件数、正確な総件数および採用できる場合の継続位置 |

各 item の `ref.key.resourceType` は `parliament-speech` とし、`resourceId` は provider mapping が公式の発言 ID から変更せず作る。発言者名、会議名、会議日または本文の類似性によって item を統合、上書きまたは重複排除しない。

## 空結果と継続取得

該当発言がない場合は `items: []`、`returnedCount: 0`、`totalCount: 0` および `totalRelation: exact` の成功結果とする。

継続取得は `SOT-IF-016` に従う。provider が安定した snapshot と決定的な順序の両方を公式仕様から確認できない場合は `nextToken` を発行せず、空でない `continuationToken` を外部呼出し前に `invalid_argument` とする。現在採用する `ndl-diet-speech-api` はこの制約に該当するため、最大 30 件の最初のページだけを返す。

## ポートと失敗

能力別ポートは `Search(context.Context, Request) (Page, error)` とし、外部 query parameter、JSON DTO、開始位置または provider 固有の error body を公開しない。

共通入力違反は `invalid_argument` とする。provider 境界からは `unsupported_capability`、`unsupported_query`、`configuration_required`、`source_auth_failed`、`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` および `unsafe_source_content` が到達し得る。

## 確認

検索条件の一つ以上指定、各文字列、四つの院名、日付境界、上限、空でない token の外部呼出し前拒否、空結果、正確な件数、最大 30 件、順序と重複の保持、型付き port、出典および情報源エラーを契約テストで確認する。

## 関連

- [SOT-SCN-014: 国会会議録の発言を検索する](../10-scenarios/14-search-diet-speeches.md)
- [SOT-MODEL-034: ParliamentSpeech](../20-model/34-parliament-speech.md)
- [SOT-IF-016: 情報源の継続取得](16-source-continuation-contract.md)
- [SOT-IF-063: 国立国会図書館の国会発言検索 API 情報源](63-source-ndl-diet-speech-api.md)
- [SOT-IF-066: MCP `search_diet_speeches`](66-mcp-search-diet-speeches.md)
