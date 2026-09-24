# SOT-DEL-015: ローカル HTTP の browser CORS

- 状態: 有効

## 規定

ローカル Streamable HTTP は、設定で許可した HTTPS Origin の browser が資格情報なしで MCP の POST 応答を読めるよう、`/mcp` で CORS preflight と応答共有を提供する。この規定は `SOT-DEL-013` を補い、transport の提供範囲や認可境界を変更しない。

## Origin と応答共有

- `allowedOrigins` の設定と起動時検証は `SOT-IF-029` に従う。受信した `Origin` は一つの値だけを許し、設定値と文字列で厳密一致させる。大小文字、port、末尾の `/` を正規化せず、wildcard、`null`、部分一致、複数値を許可しない。
- `/mcp` では method の判定より先に Origin を照合する。Origin の不一致、不正値、空値または複数値は `403 Forbidden` とし、`Access-Control-Allow-*` を返さず MCP 処理へ渡さない。空の許可リストでの拒否、および Origin のない非 browser POST の受入れは `SOT-IF-029` に従う。
- 許可 Origin の実 POST と OPTIONS 以外の method への応答には、成功・失敗にかかわらず `Access-Control-Allow-Origin` に一致した Origin を一つ返す。本文・protocol version・session・同時実行制限による拒否、SDK が返すエラーも対象とする。Origin のない応答にはこの header を付けない。
- `/mcp` の全応答に `Vary: Origin` を含める。OPTIONS の全応答には `Access-Control-Request-Method` と `Access-Control-Request-Headers` も `Vary` に含める。既存の `Vary` 値は保持する。
- `/mcp` 以外は CORS の対象とせず `404 Not Found` とする。Origin 検証を通過した GET など POST と OPTIONS 以外の method は `405 Method Not Allowed` とし、`Allow: POST, OPTIONS` を返す。

## Preflight

- OPTIONS は HTTP の事前確認だけを行い、本文を MCP message として読み取らず、SDK、tool、外部 API および tool 同時実行枠を使用しない。
- 許可された一つの Origin と、一つの `Access-Control-Request-Method: POST` を必須とする。method は大小文字を含め厳密一致とする。
- `Access-Control-Request-Headers` は省略できる。存在する場合、カンマ区切りの各名前を前後の SP・HTAB だけ除いて大小文字を区別せず照合し、`Accept`、`Content-Type`、`MCP-Protocol-Version` だけを許可する。複数 field 行は同じリストとして扱い、同じ名前の重複は許す。空の値・要素、不正な名前、未知の名前、`Authorization`、`Mcp-Session-Id`、`Last-Event-ID` および wildcard は拒否する。要求 header を無条件に反射しない。
- Origin がない場合、method または要求 header が上記に合わない場合、および OPTIONS 自体が `Mcp-Session-Id` を含む場合は `400 Bad Request` とし、`Access-Control-Allow-*` を返さない。Origin が存在して不正な場合は前節の `403` を優先する。
- 成功時は本文のない `204 No Content` とし、`Access-Control-Allow-Origin` に一致した Origin、`Access-Control-Allow-Methods` に `POST`、`Access-Control-Allow-Headers` に固定値 `Accept, Content-Type, MCP-Protocol-Version` を返す。`Access-Control-Max-Age` は送らず、server に preflight の履歴を保存しない。
- preflight は header 名だけを検証する。実 POST での media type、protocol version、session、本文および同時実行制限の検証は省略しない。

## 資格情報と適用範囲

browser client は `fetch` の `credentials: "omit"` を使う。cookie 認証や HTTP 認証を導入せず、`Access-Control-Allow-Credentials` と `Access-Control-Expose-Headers` を送らない。認証用 header の preflight 許可、session ID の公開および cookie の発行を行わない。CORS は資格情報の到着や非 browser client を認可する仕組みではなく、受信した cookie 等を認証の根拠にしない。

browser 固有のローカルネットワーク接続許可は CORS と別の条件であり、この規定はその許可や回避を提供しない。

## 確認

transport の対象検査で正常 preflight、Origin の厳密一致と拒否、method・header の制限、`Vary`、許可 Origin から読める成功・エラー応答、Origin のない既存 POST、および事前確認が MCP 処理を開始しないことを確認する。実 binary を loopback で起動し、HTTPS Origin 付きの preflight、initialize、tools/list を順に確認する。

## 関連

- [SOT-DEL-013: ローカル Streamable HTTP](13-local-streamable-http.md)
- [SOT-DEL-008: HTTP リソース制限](08-http-resource-limits.md)
- [SOT-IF-029: ローカル実行設定](../40-interfaces/29-local-runtime-configuration.md)
- [SOT-IF-013: MCP プロトコル基準](../40-interfaces/13-mcp-protocol-version.md)
- [WHATWG Fetch: CORS protocol](https://fetch.spec.whatwg.org/#http-cors-protocol)
- [WHATWG Fetch: CORS protocol and HTTP caches](https://fetch.spec.whatwg.org/#cors-protocol-and-http-caches)
- [MCP 2025-11-25: Transports](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports)
