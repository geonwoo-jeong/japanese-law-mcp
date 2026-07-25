# SOT-IF-017: 情報源エラーの正規化

- 状態: 有効

## 規定

プロバイダーアダプターは、外部の HTTP status、エラー本文、例外および構造不一致を、原因と次の対応を区別できる共通の情報源エラーへ変換する。

## エラー

| コード | 意味 | 既定の `retryable` |
|---|---|---:|
| `unsupported_capability` | プロバイダーが能力を宣言していない | `false` |
| `unsupported_query` | 共通 capability として有効な条件を、選択したプロバイダーの公式な対象範囲では処理できない | `false` |
| `configuration_required` | 必要なプロバイダー設定がない | `false` |
| `source_auth_failed` | 外部情報源の認証または認可に失敗した | `false` |
| `rate_limited` | 外部情報源が呼出し頻度を制限した | `true` |
| `source_timeout` | 外部情報源への接続、送信または受信が期限を超えた | `true` |
| `source_unavailable` | 外部情報源へ一時的に到達できない | `true` |
| `source_busy` | 現在実行中の同じプロバイダー同時実行 group が上限に達した | `true` |
| `source_contract_changed` | 公式スキーマまたは HTML 構造が確認済みの契約と一致しない | `false` |
| `invalid_source_response` | 外部レスポンスの値または形式が契約を満たさない | `false` |
| `source_response_too_large` | 応答または展開結果が安全上の上限を超えた | `false` |
| `source_processing_limit` | 展開または解析がローカルの安全な処理時間上限を超えた | `false` |
| `unsafe_source_content` | パス、XML、HTML または圧縮内容を安全に処理できない | `false` |

情報源エラーは、`code`、利用者向けの日本語 `message`、`providerId`、`sourceId`、`capabilityId`、`operation`、`retryable` および存在する場合の `retryAfter` を持つ。

## 制約

外部レスポンス本文、認証情報、内部 stack、ファイルパス、検索語または利用主体をエラーへ含めない。

外部情報源が明示した `Retry-After` または同等の値がある場合だけ `retryAfter` を返す。数値を推測しない。

`source_busy` は `SOT-ENG-016` の `providerId + concurrencyGroup` を単位として、外部呼出しまたは解析を開始する前に返し、完了済み呼出しの履歴を保持して判定しない。公開する `operation` は拒否された現在の operation とし、内部の group 名は公開 detail に追加しない。`source_processing_limit` は同じ取得済み内容を自動再試行しない。

`unsupported_query` は、共通 capability の入力検証を通過した値にだけ使用する。プロバイダー SOT が定義する期間、地域、文書種別、表現または公式機能の対象外であることを外部呼出し前に判定し、空結果、`not_found` または `invalid_argument` に読み替えない。

MCP ツールへ公開するコードとの対応は、対応するツールまたはエラー契約の SOT が定義する。未定義の情報源エラーを既存コードへ黙って縮約しない。

新しいプロバイダー能力を MCP ツールへ公開する前に、その能力から到達し得るすべての情報源エラーについて、`SOT-IF-027` と対象ツール SOT に公開する `ErrorResult` への対応を定義する。

## 関連

- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-ARCH-004: 情報源アダプター境界](../30-architecture/04-source-adapter-boundary.md)
- [SOT-ARCH-014: 外部原文の一時処理](../30-architecture/14-ephemeral-source-artifacts.md)
- [SOT-ENG-016: プロバイダー資源予算](../50-engineering/16-provider-resource-budgets.md)
