# SOT-IF-017: 情報源エラーの正規化

- 状態: 有効

## 規定

プロバイダーアダプターは、外部の HTTP status、エラー本文、例外および構造不一致を、原因と次の対応を区別できる共通の情報源エラーへ変換する。

## エラー

| コード | 意味 | 既定の `retryable` |
|---|---|---:|
| `unsupported_capability` | プロバイダーが能力を宣言していない | `false` |
| `configuration_required` | 必要なプロバイダー設定がない | `false` |
| `source_auth_failed` | 外部情報源の認証または認可に失敗した | `false` |
| `rate_limited` | 外部情報源が呼出し頻度を制限した | `true` |
| `source_timeout` | 外部情報源または解析が期限を超えた | `true` |
| `source_unavailable` | 外部情報源へ一時的に到達できない | `true` |
| `source_contract_changed` | 公式スキーマまたは HTML 構造が確認済みの契約と一致しない | `false` |
| `invalid_source_response` | 外部レスポンスの値または形式が契約を満たさない | `false` |
| `source_response_too_large` | 応答または展開結果が安全上の上限を超えた | `false` |
| `unsafe_source_content` | パス、XML、HTML または圧縮内容を安全に処理できない | `false` |

情報源エラーは、`code`、利用者向けの日本語 `message`、`providerId`、`sourceId`、`capabilityId`、`operation`、`retryable` および存在する場合の `retryAfter` を持つ。

## 制約

外部レスポンス本文、認証情報、内部 stack、ファイルパス、検索語または利用主体をエラーへ含めない。

外部情報源が明示した `Retry-After` または同等の値がある場合だけ `retryAfter` を返す。数値を推測しない。

MCP ツールへ公開するコードとの対応は、対応するツールまたはエラー契約の SOT が定義する。未定義の情報源エラーを既存コードへ黙って縮約しない。

新しいプロバイダー能力を MCP ツールへ公開する前に、その能力から到達し得るすべての情報源エラーについて、公開する `ErrorResult` またはプロトコルエラーへの対応を定義する。既存 `SOT-IF-006` に対応先がない場合は、後継の公開エラー契約を先に採用する。

## 関連

- [SOT-IF-006: エラー契約](06-error-contract.md)
- [SOT-ARCH-004: 情報源アダプター境界](../30-architecture/04-source-adapter-boundary.md)
- [SOT-ARCH-014: 外部原文の一時処理](../30-architecture/14-ephemeral-source-artifacts.md)
