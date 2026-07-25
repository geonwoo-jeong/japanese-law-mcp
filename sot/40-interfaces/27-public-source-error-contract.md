# SOT-IF-027: 公開情報源エラー契約

- 状態: 有効

## 規定

MCP ツールが公開する `ErrorResult` は、既存の入力エラーと結果欠落に加えて、`SOT-IF-017` が定義するすべての情報源エラーを、利用者が再試行、設定修正または保守対応の要否を判断できる公開コードへ対応づけて返す。

## 適用優先順位

`SOT-IF-027` は、廃止した `SOT-IF-006` を置き換える公開エラーの定義元である。すべての MCP ツールと公開インターフェースは、この文書を適用する。

ツール固有の SOT は `SOT-IF-027` が許可する公開コードと追加情報を、対象ツールの範囲でさらに絞り込めるが、未定義の縮約や秘密情報の追加をしてはならない。

## 公開コード

| コード | 意味 | `retryable` | 許可する `details` |
|---|---|---:|---|
| `invalid_argument` | 入力がツール契約を満たさない | `false` | `field`, `reason` |
| `not_found` | 指定条件に該当する情報がない | `false` | なし |
| `ambiguous_location` | 対象位置を一意に決定できない | `false` | `candidates` |
| `unsupported_capability` | 選択したプロバイダーまたは route が能力を公開していない | `false` | `providerId`, `sourceId`, `capabilityId` |
| `unsupported_query` | 有効な共通条件が、選択したプロバイダーの公式な対象範囲外である | `false` | `providerId`, `sourceId`, `capabilityId`, `field`, `constraint` |
| `configuration_required` | 必要なプロバイダー設定または認証参照がない | `false` | `providerId`, `sourceId`, `capabilityId`, `missing` |
| `source_auth_failed` | 外部情報源の認証または認可に失敗した | `false` | `providerId`, `sourceId`, `capabilityId`, `operation` |
| `rate_limited` | 外部情報源が呼出し頻度を制限した | `true` | `providerId`, `sourceId`, `capabilityId`, `operation`, `retryAfter` |
| `source_timeout` | 外部情報源への接続、送信または受信が期限を超えた | `true` | `providerId`, `sourceId`, `capabilityId`, `operation` |
| `source_unavailable` | 外部情報源へ一時的に到達できない | `true` | `providerId`, `sourceId`, `capabilityId`, `operation`, `retryAfter` |
| `source_busy` | 現在のローカルプロセスで同じプロバイダー同時実行 group の上限に達した | `true` | `providerId`, `sourceId`, `capabilityId`, `operation` |
| `source_contract_changed` | 公式スキーマまたは HTML 構造が確認済み契約と一致しない | `false` | `providerId`, `sourceId`, `capabilityId`, `operation` |
| `invalid_source_response` | 外部レスポンスの値または形式が契約を満たさない | `false` | `providerId`, `sourceId`, `capabilityId`, `operation` |
| `source_response_too_large` | 応答または展開結果が安全上の上限を超えた | `false` | `providerId`, `sourceId`, `capabilityId`, `operation`, `limitName` |
| `source_processing_limit` | 展開または解析がローカルの安全な処理時間上限を超えた | `false` | `providerId`, `sourceId`, `capabilityId`, `operation`, `limitName` |
| `unsafe_source_content` | 外部内容を安全に処理できない | `false` | `providerId`, `sourceId`, `capabilityId`, `operation` |
| `internal_error` | 内部処理を完了できない | `false` | なし |

`message` は、各コードに対して次の判断を助ける日本語とする。

- `invalid_argument`: 利用者がどの入力を直すべきか分かる。
- `not_found` と `ambiguous_location`: 結果の不存在または位置の曖昧さが分かる。
- `unsupported_query`: 入力形式は有効だが、選択した情報源の対象期間または対象範囲では処理できないと分かる。
- `configuration_required`: 実行環境の設定不足であり、入力再送だけでは解決しないと分かる。
- `source_auth_failed`: 認証または利用権限の確認が必要と分かる。
- `rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`: 同じ入力で後から再試行する意味があると分かる。
- `source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit`、`unsafe_source_content`: 利用者入力ではなく情報源契約または安全境界の問題だと分かる。
- `internal_error`: 入力を変えずに再試行しても通常は改善しないと分かる。

## 公開対応

`SOT-IF-017` の情報源エラーは、次の対応で公開する。

| 情報源または内部の失敗 | 公開する `ErrorResult.code` |
|---|---|
| `unsupported_capability` | `unsupported_capability` |
| `unsupported_query` | `unsupported_query` |
| `configuration_required` | `configuration_required` |
| `source_auth_failed` | `source_auth_failed` |
| `rate_limited` | `rate_limited` |
| `source_timeout` | `source_timeout` |
| `source_unavailable` | `source_unavailable` |
| `source_busy` | `source_busy` |
| `source_contract_changed` | `source_contract_changed` |
| `invalid_source_response` | `invalid_source_response` |
| `source_response_too_large` | `source_response_too_large` |
| `source_processing_limit` | `source_processing_limit` |
| `unsafe_source_content` | `unsafe_source_content` |
| ツール入力検証の失敗 | `invalid_argument` |
| 正確な識別子による取得対象の不存在 | `not_found` |
| 位置選択の候補複数 | `ambiguous_location` |
| 上記へ分類できない内部処理の失敗 | `internal_error` |

未分類の情報源エラーを `source_unavailable` または `internal_error` へ黙って縮約してはならない。新しい情報源エラーを追加する場合は、公開するツールより先に、この文書またはその後継 SOT へ対応を追加する。

## `retryAfter`

`retryAfter` は `details.retryAfter` の文字列として返す。外部情報源が `Retry-After` または同等の待機値を明示した場合だけ含め、値を推測、丸め、合成または既定化しない。

`retryAfter` を返せるのは `rate_limited` と `source_unavailable` だけとする。`source_timeout` は `retryable: true` であっても `retryAfter` を付けない。

## 禁止事項

認証情報、利用者入力全文、検索語、継続トークン、外部レスポンス全文、HTML・XML・PDF・ZIP・XBRL 等の原文、内部 stack、ファイルパス、proxy 情報および秘密値の参照名を `message` または `details` へ含めない。

`details` には、この文書で許可したキーだけを含める。`providerId`、`sourceId`、`capabilityId` および `operation` は定義済み識別子または操作名をそのまま使い、説明文や未加工レスポンスを埋め込まない。

## ツールとの関係

新しいプロバイダー能力を公開するツール SOT は、`SOT-IF-027` を参照し、そのツールが到達し得る公開コードを列挙する。

ツール SOT は、そのツールが到達し得る公開コードを `## エラー` で列挙する。ツール SOT が narrower な公開範囲を定義した場合も、内部の情報源エラー対応はこの文書を定義元とする。

## 関連

- [SOT-MODEL-005: ErrorResult](../20-model/05-error-result.md)
- [SOT-IF-006: エラー契約](06-error-contract.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-017: 情報源エラーの正規化](17-source-error-normalization.md)
- [SOT-ENG-003: 明示的なエラー処理](../50-engineering/03-explicit-error-handling.md)
