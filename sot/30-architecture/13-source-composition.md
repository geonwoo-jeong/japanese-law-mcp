# SOT-ARCH-013: 情報源の選択と組合せ

- 状態: 有効

## 規定

情報源の選択、fallback、集約および値の結合は、プロバイダーアダプターではなく、採用された利用シナリオのユースケースが明示的な規則に従って行う。

## 選択

選択方法は次のいずれかとする。

- `explicit`: 利用シナリオが指定した一つのプロバイダーだけを使用する。
- `primary`: 能力の SOT が定める既定プロバイダーを使用する。
- `aggregate`: 集約を採用した検索シナリオに限り、定義された複数のプロバイダーを使用する。

法令検索、法令本文およびリビジョンの既定プロバイダーは e-Gov 法令 API Version 2 とする。e-Gov 法令 API Version 1 の日付別更新一覧は別の能力であり、Version 2 の本文または時点検索の暗黙の fallback にしない。

`aggregate` は専用の利用シナリオ、結果モデルおよび MCP インターフェース SOT が採用された場合だけ使用する。既存の `search_laws` と `search_law_content` は単一の e-Gov 法令 API Version 2 を使用し、`aggregate` の結果契約として再解釈しない。

## 組合せ

正確な `SourceResourceKey` による取得は、別の情報源へ自動で fallback しない。fallback は、入力、意味、対象時点および欠落時の扱いが同じであることを能力別 SOT が定義した場合だけ許可する。

`SourceResourceRef` を入力に持つ取得は `explicit` とし、`ref.providerId` の provider だけを使用する。primary route は、既存 facade が情報源固有の識別子から最初の `SourceResourceRef` を組み立てる場合にだけ使用し、すでに存在する ref の provider または key を置き換えない。

異なる情報源の資源は、公式の共通識別子または確認済みの対応表がある場合だけ同一視する。名称、題名、住所、本文または検索順位の類似性だけで重複排除または結合を行わない。

複数情報源の値が競合する場合は、任意に一つを選ばず、それぞれの値と `Provenance` を保持する。フィールド単位の優先順位は、能力別 SOT が根拠とともに定義した場合だけ適用する。

集約検索では、情報源横断の正確な総件数、同一の関連度尺度または単一 offset を保証しない。一部の情報源が失敗した場合は空の成功として隠さず、結果が部分的であることと失敗した情報源を示す。

`SourcePage` と共通の継続トークンは、一つのプロバイダーによる取得に適用する。集約検索を採用する SOT は、部分結果、情報源ごとの問題、情報源ごとの継続位置、決定的な並び順および再開条件を持つ専用の結果モデルを定義する。このモデルがない間は `aggregate` route を登録しない。

## 関連

- [SOT-MODEL-011: SourceResourceKey](../20-model/11-source-resource-key.md)
- [SOT-MODEL-012: Provenance](../20-model/12-provenance.md)
- [SOT-MODEL-014: SourcePage](../20-model/14-source-page.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-PROD-005: 加工情報の区別](../00-product/05-derived-information.md)
- [SOT-IF-030: MCP `search_laws`](../40-interfaces/30-mcp-search-laws.md)
- [SOT-IF-033: MCP `search_law_content`](../40-interfaces/33-mcp-search-law-content.md)
