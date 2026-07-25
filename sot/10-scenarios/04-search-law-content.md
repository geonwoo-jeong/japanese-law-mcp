# SOT-SCN-004: 法令本文を検索する

- 状態: 有効

## 規定

利用者は、法令本文を対象とする検索式を指定し、該当箇所の本文、法令内の位置および出典を取得できる。

## 開始条件

利用者が空白以外の検索式を指定している。

## 基本フロー

1. MCP クライアントが本文の検索条件を送信する。
2. Japanese Law MCP が公式情報源の法令本文を検索する。
3. 法令情報と一致箇所を `LawContentMatch` に変換する。
4. 総件数、次の取得位置および一致箇所を含む `LawContentSearchResult` を返す。

## 分岐

- 一致箇所がない場合は、成功した空の結果として返す。
- 検索式が公式情報源の検索構文に適合しない場合は、入力エラーとして返す。
- 情報源を利用できない場合は、情報源エラーとして返す。
- 一致箇所に付与された強調表示は除去し、法令本文と位置情報は変更しない。

## 完了条件

返された各一致箇所から、利用者が対象法令、リビジョンおよび法令内の位置を確認できる。

## 関連

- [SOT-IF-033: MCP `search_law_content`](../40-interfaces/33-mcp-search-law-content.md)
- [SOT-MODEL-008: LawContentSearchResult](../20-model/08-law-content-search-result.md)
- [SOT-ARCH-005: リクエスト情報の一時性](../30-architecture/05-ephemeral-request-lifecycle.md)
