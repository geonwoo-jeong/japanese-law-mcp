# SOT-DEL-006: HTTP 認可

- 状態: 有効

## 規定

ループバック以外へ公開する Streamable HTTP は、TLS 上で MCP 仕様 `2025-11-25` の HTTP 認可に適合する保護リソースとして提供する。

## 適用

- Bearer token は認可サーバーが発行した署名付きトークンとして検証する。
- 対象リソースと権限が Japanese Law MCP のエンドポイントに適合しないトークンを拒否する。
- 認証情報はリクエストの検証中だけ扱い、アプリケーションのストレージ、キャッシュまたはログへ保存しない。
- ループバックだけに公開するローカル HTTP は、ホスト境界と Origin 検証を接続境界として使用できる。
- stdio には MCP の HTTP 認可を適用しない。

## 確認

有効なトークン、期限切れのトークン、対象リソースが異なるトークン、トークンなしの各接続をテストし、非ループバック公開で未認可のリクエストがツールへ到達しないことを確認する。

## 関連

- [MCP Authorization](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)
- [SOT-DEL-002: Streamable HTTP](02-streamable-http.md)
- [SOT-ARCH-005: リクエスト情報の一時性](../30-architecture/05-ephemeral-request-lifecycle.md)
