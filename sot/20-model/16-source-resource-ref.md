# SOT-MODEL-016: SourceResourceRef

- 状態: 有効

## 規定

`SourceResourceRef` は、共通 capability が返した情報の根拠となる主資源を、取得に使用したプロバイダーと情報源上の資源キーを変えずに後続の capability へ渡すための参照を表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `providerId` | string | はい | 資源を取得した `ProviderDescriptor.providerId` |
| `key` | `SourceResourceKey` | はい | 情報源上の資源と版 |

## 制約

`providerId` が示す `ProviderDescriptor.source.id` と `key.sourceId` は一致しなければならない。

後続の capability は、`providerId`、`sourceId`、`resourceType`、`resourceId` および存在する場合の `versionId` を別の値へ読み替えない。入力した provider が対象 capability を実装しない場合は別の provider へ fallback せず `unsupported_capability` とする。

`SourceResourceRef` は、常に結果配列の一要素を一意に識別する item ID ではない。検索一致、統計観測その他の一つの主資源から複数の item が返る能力は、能力別 SOT が item identity を別に定義し、`ref` だけで重複排除しない。

この参照の往復保証は、`SourcedResource<T>` を入出力に用いる内部 capability 境界に適用する。公開 MCP 境界での `ref` の公開有無、供給元および往復範囲は各 MCP interface SOT を定義元とし、このモデルだけからすべての facade が `ref` を公開すると解釈しない。

`SourceResourceRef` は発行元の署名、nonce またはプロセス識別子を持たず、過去に同じ MCP プロセスが返したことを証明しない。公開入力で受け取る場合は、構造、採用済み provider と source の metadata、resource type および対象 capability との一致を検証し、同一プロセスによる発行証明として扱わない。

未知の `providerId`、`providerId` と `sourceId` の不一致、対象 capability が許可しない `resourceType` および空の識別子は `invalid_argument` とし、外部呼出しを行わない。登録済みだが無効化されている provider は `configuration_required` とする。

## 確認

内部検索 capability の `SourceResourceRef` を内部読み取り capability へそのまま渡し、同じ provider、情報源、資源および版を取得することを確認する。provider、source、resource type または version を変更した入力と、無効化した provider の参照を外部呼出し前に拒否する。既存 MCP facade の互換投影を、この往復試験の入口または出口に使用しない。

## 関連

- [SOT-MODEL-011: SourceResourceKey](11-source-resource-key.md)
- [SOT-IF-014: ProviderDescriptor](../40-interfaces/14-provider-descriptor.md)
- [SOT-IF-015: 情報源操作の共通契約](../40-interfaces/15-source-operation-contract.md)
- [SOT-IF-047: MCP `search_judicial_cases`](../40-interfaces/47-mcp-search-judicial-cases.md)
- [SOT-IF-048: MCP `get_judicial_case`](../40-interfaces/48-mcp-get-judicial-case.md)
- [SOT-IF-051: MCP `query_legal_information`](../40-interfaces/51-mcp-query-legal-information.md)
- [SOT-ARCH-013: 情報源の選択と組合せ](../30-architecture/13-source-composition.md)
