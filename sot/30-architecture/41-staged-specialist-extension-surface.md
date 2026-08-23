# SOT-ARCH-041: 拡張パックの専門公開面の段階採用

- 状態: 有効

## 規定

選択型法情報拡張パックは、型付きの専門 MCP 公開面と、自然文を扱う統合照会
contribution を別の段階で採用できる。ただし、各段階の内部は原子的に構成し、
provider package、route、tool または query profile の一部だけを公開しない。

本規定は、`SOT-ARCH-019` が専門ツールと統合照会 contribution を同じ製品機能の
集合として有効化する規則に対する、採用時期だけの限定的な特則である。pack 固有の
有効な SOT が本規定を明示的に参照して第一段階を採用した場合は、統合照会
contribution の同時有効化に限って本規定を優先する。明示的な第一段階の採用がない
pack には、この特則を適用しない。

`SOT-ARCH-019` が定める拡張パック、capability および provider route の三つの構成軸、
ならびに各段階の内部の原子性は変更しない。統合照会を第二段階で採用した後の固定
profile set、無効時の `pack_disabled` および rollback も同 SOT に従う。

## 第一段階: 専門公開面

拡張パック固有の有効な SOT が第一段階を採用した場合は、次を一つの集合として
構成する。

- 拡張パックの有効化設定と rollback
- 採用済みの利用シナリオ、共通モデルおよび型付き capability
- capability を実装する provider binding と明示的な route
- capability をそのまま公開する専門 MCP tool

一つでも構成できない場合は transport を開始しない。拡張パックが無効な場合は、
provider factory、binding、route および専門 MCP tool を実効構成へ加えない。

第一段階は `query_legal_information` の製品範囲を広げない。統合照会の profile、cue、
candidate、能力別 facade、request materializer および result variant を追加せず、
当該 pack に関する自然文照会は既存の対象外分類を維持する。専門 tool が公開済みで
あっても、統合照会への採用を推測しない。

## 第二段階: 統合照会 contribution

統合照会へ参加させる場合は、対象となる task と resource、意味認識 contribution、
実行 contribution、公開 result variant、無効時の案内および検証方法を、別の有効な
SOT で採用する。

第二段階の採用では、profile metadata、cue、候補生成規則、能力別 facade、request
materializer および result variant を一つの集合として構成する。採用後の意味認識
contribution は pack の有効状態にかかわらず固定 profile set へ保持し、pack が無効な
場合は外部情報源を呼ばず `capability_unavailable` とする。

第二段階を採用していない pack は、専門 tool が利用可能でも統合照会では未採用のまま
とする。第一段階の provider-independent なモデル、capability および専門 tool の契約は、
第二段階の採否によって変更しない。

## 確認

第一段階では、pack の省略、無効および有効について、専門公開面の全構成要素が同時に
除外または追加されること、既存の統合照会 profile set、候補、公開 status および外部
呼出しが変わらないことを確認する。第二段階では、`SOT-ARCH-019` の固定 profile set、
原子的な実行 contribution、無効時の `capability_unavailable` および rollback を確認する。

## 関連

- [SOT-PROD-009: 選択型法情報拡張パックの境界](../00-product/09-selectable-legal-information-extension-packs.md)
- [SOT-PROD-011: 統合法情報照会の製品範囲](../00-product/11-unified-legal-query-scope.md)
- [SOT-PROD-012: 日本法情報照会の意図分類と受理境界](../00-product/12-japanese-legal-query-intent-taxonomy.md)
- [SOT-ARCH-019: 拡張パックの有効化境界](19-extension-pack-activation-boundary.md)
- [SOT-IF-061: `legislative-history` 拡張パックの専門公開面](../40-interfaces/61-legislative-history-pack-activation.md)
