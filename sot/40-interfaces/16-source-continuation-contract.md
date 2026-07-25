# SOT-IF-016: 情報源の継続取得

- 状態: 有効

## 規定

新しい能力別の一覧および検索インターフェースは、初回の `limit` と、同じ条件の続きを取得するための不透明な `continuationToken` を共通の継続取得契約とする。

## 入力

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `limit` | integer | いいえ | 今回返す上限。能力別 SOT が既定値と最大値を定義する |
| `continuationToken` | string | いいえ | 直前の結果が返した次の取得位置 |

`continuationToken` を指定した場合は、初回と同じ検索条件および `limit` を使用する。条件、`limit`、`adapterContractVersion`、provider configuration scope または継続順序の前提が一致しない場合、期限切れの場合または改変を検出した場合は `invalid_argument` とする。

`continuationToken` は UTF-8 で 4096 byte 以下とし、超過した値は decode または検証の前に `invalid_argument` とする。

## トークン

トークンは `v1.{payload}.{mac}` の三部分とする。`payload` は、次の key だけを持つ JSON object を RFC 8785 の JSON Canonicalization Scheme で UTF-8 byte 列にし、padding なしの base64url にした値とする。

| key | 型 | 意味 |
|---|---|---|
| `providerId` | string | 選択したプロバイダー |
| `capabilityId` | string | 能力 ID |
| `majorVersion` | integer | 能力のメジャーバージョン |
| `limit` | integer | 正規化後のページ上限 |
| `adapterContractVersion` | string | 発行時のアダプター契約版 |
| `position` | JSON value | アダプターだけが解釈する外部取得位置 |
| `issuedAt` | integer | UTC の Unix time seconds |
| `expiresAt` | integer | UTC の Unix time seconds |
| `conditionFingerprint` | string | 正規化済み検索条件の鍵付き fingerprint |
| `configFingerprint` | string | provider configuration scope の鍵付き fingerprint |
| `snapshot` | JSON value | 情報源の snapshot を固定する場合だけ使用する marker |
| `sort` | JSON value | 決定的な並び順の検証に必要な場合だけ使用する marker |

`snapshot` と `sort` 以外の key は必須とし、未知の key、型の不一致、重複 key および非 canonical な payload は `invalid_argument` とする。`position`、`snapshot` および `sort` の具体的な構造は mapping SOT が定義する。

`mac` は、継続トークン鍵を使い、ASCII の `continuation-token-v1`、`0x00`、`v1.{payload}` の ASCII byte 列の順に連結した値へ HMAC-SHA-256 を適用し、結果を padding なしの base64url にした値とする。MAC は一定時間比較し、不一致を `invalid_argument` とする。

検索条件は、能力 SOT が定義する正規化済みの JSON object を RFC 8785 で canonicalize し、継続トークン鍵を使う HMAC-SHA-256 で fingerprint にする。HMAC 入力は ASCII の `continuation-condition-v1`、`0x00`、canonical JSON の順に連結し、出力は padding なしの base64url とする。検索語の原文または鍵なしの hash はトークンへ含めない。

継続トークン鍵は、process の起動ごとに CSPRNG から 32 byte を生成してメモリー内だけに保持する。生成に失敗した場合は起動を失敗させる。鍵をファイル、設定、ログ、エラーまたは診断へ保存せず、process 終了時に破棄する。再起動前に発行したトークンは、新しい process では `invalid_argument` とする。

能力別 SOT はトークンの有効期限上限を定義する。`expiresAt` は `issuedAt` より後で、その上限以内でなければならない。期限切れ、未来の `issuedAt` および process の現在時刻より前の `expiresAt` を受け入れない。

offset、`startRecord`、`NEXT_KEY`、`divideNumber`、日付内の取得位置またはプロバイダー別の継続値は、各アダプターがトークンとの間で変換する。初回条件そのものに意味がある日付、統計の次元または地理タイルは継続位置へ置き換えない。

外部情報源が同じ条件で安定した snapshot と決定的な並び順の両方を提供できない場合は、`nextToken` を発行しない。再開時に snapshot marker、sort marker、provider configuration scope の fingerprint または `adapterContractVersion` が一致しない場合は拒否する。

既存の MCP ツールが定義する `offset` と `nextOffset` は、公開インターフェースを別の SOT で変更するまで維持する。

## 関連

- [SOT-MODEL-014: SourcePage](../20-model/14-source-page.md)
- [SOT-ARCH-005: リクエスト情報の一時性](../30-architecture/05-ephemeral-request-lifecycle.md)
- [SOT-IF-030: MCP `search_laws`](30-mcp-search-laws.md)
