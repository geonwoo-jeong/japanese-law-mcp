# SOT-DEL-006: HTTP 認可

- 状態: 廃止

## 規定

ループバック以外へ公開する Streamable HTTP は、TLS 上で MCP 仕様 `2025-11-25` の HTTP 認可に適合する保護リソースとして提供する。

公式実行方式を利用者のローカル環境へ限定したため、この規定を廃止する。非 loopback の HTTP 公開は公式機能として提供しない。

## 適用

- Bearer token は認可サーバーが発行した署名付きトークンとして検証する。
- `alg` は明示した許可リストだけを受け入れ、`none` を含む未署名トークンを拒否する。
- `iss` は採用した認可サーバーの issuer と完全一致させ、署名検証に使用する JWKS または同等の鍵取得先は、その issuer に対応する事前定義の HTTPS origin へ固定する。
- `aud` または同等の resource indicator は Japanese Law MCP の保護対象と完全一致させ、別の API または別の環境向けトークンを拒否する。
- `exp` と `nbf` を必須で検証し、許容する clock skew の上限を固定する。上限を超えて期限前または期限切れと判定されるトークンを受け入れない。
- 対象リソースと権限が Japanese Law MCP のエンドポイントに適合しないトークンを拒否する。
- 鍵の取得、更新または選択に失敗した場合は fail-closed で拒否し、未検証の鍵、期限切れの鍵または別 origin の鍵へ自動でフォールバックしない。
- 認証情報はリクエストの検証中だけ扱い、アプリケーションのストレージ、キャッシュまたはログへ保存しない。
- ループバックだけに公開するローカル HTTP は、ホスト境界と Origin 検証を接続境界として使用できる。
- stdio には MCP の HTTP 認可を適用しない。

## 確認

有効なトークン、期限切れのトークン、対象リソースが異なるトークン、トークンなしの各接続をテストし、非ループバック公開で未認可のリクエストがツールへ到達しないことを確認する。

## 関連

- [MCP Authorization](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)
- [SOT-DEL-002: Streamable HTTP](02-streamable-http.md)
- [SOT-ARCH-005: リクエスト情報の一時性](../30-architecture/05-ephemeral-request-lifecycle.md)
