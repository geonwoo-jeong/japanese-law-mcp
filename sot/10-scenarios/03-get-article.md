# SOT-SCN-003: 条文を取得する

- 状態: 有効

## 規定

利用者は、法令識別子と条文位置を指定し、該当する法令部分とその位置を確認できる情報を取得できる。

## 開始条件

利用者が法令識別子、本則または附則の区分、および条番号を指定している。

## 基本フロー

1. MCP クライアントが法令識別子と条文位置を送信する。
2. Japanese Law MCP が公式情報源から対象リビジョンの法令 XML を取得する。
3. 指定された本則または附則の直下にある条と項を、法令 XML の `Num` 属性で特定する。
4. 出所と位置情報を含む結果を返す。

## 分岐

- 指定した条文が存在しない場合は、該当結果なしとして返す。
- 条文位置を一意に特定できない場合は、推測せず `ambiguous_location` を返す。
- 入れ子になった別の `Article` を対象条文として扱わない。

## 完了条件

返された内容がどの法令のどの位置に由来するかを利用者が確認できる。

## 関連

- [SOT-IF-003: MCP `get_article`](../40-interfaces/03-mcp-get-article.md)
- [SOT-MODEL-004: Citation](../20-model/04-citation.md)
- [SOT-IF-012: e-Gov 条文取得マッピング](../40-interfaces/12-egov-article-mapping.md)
- [SOT-ARCH-005: リクエスト情報の一時性](../30-architecture/05-ephemeral-request-lifecycle.md)
