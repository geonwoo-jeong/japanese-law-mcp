# SOT-ARCH-019: 拡張パックの有効化境界

- 状態: 有効

## 規定

選択型法情報拡張パックの有効化、能力の意味、および能力を実装するプロバイダーの選択を、それぞれ独立した構成軸として扱う。

## 三つの構成軸

| 構成軸 | 識別単位 | 責務 |
|---|---|---|
| 製品機能 | 拡張パック | 利用者が有効にする公開ツールと利用シナリオの集合を決める |
| 意味契約 | `ProviderCapability` の ID とメジャーバージョン | 型付き入力、出力、欠落、継続取得およびエラーの意味を決める |
| 実装選択 | capability ごとの provider route | 有効な能力をどのプロバイダー binding で実行するかを決める |

拡張パックの安定した識別子を次のとおり予約する。

| 拡張パック | `packId` |
|---|---|
| 判例 | `judicial-cases` |
| 立法過程 | `legislative-history` |
| 税務 | `tax` |
| 労働 | `labor` |

法令コアは既定の製品機能であり、拡張パックとして扱わない。

上表は独立した法情報領域の pack ID を示す。同じ法情報領域の内部で既存 pack に依存する機能パックは、依存関係、起動時エラー、原子的構成および rollback を別の有効な SOT で定義した場合に限って追加できる。`judicial-citations` は `SOT-ARCH-042` が定義する `judicial-cases` の従属機能パックであり、第五の独立した法情報領域として扱わない。

拡張パックの有効化は、同じ capability ID の provider route を置き換える操作ではない。provider route は有効化済みの能力について binding を選ぶだけとし、拡張パックを有効にしたことで既存の法令コアの既定 route、結果または provider を変更しない。

税務または労働の拡張パックが法令コアの `law.*` 能力を利用する場合は、既存の法令コアをそのまま利用する。分野名だけを理由に `tax.law.*`、`labor.law.*` または同じ意味を持つ別能力を作らず、別の法令プロバイダーを法令コアの既定 route へ暗黙に参加させない。

一つのプロバイダーを登録または有効化したことだけで、そのプロバイダーに関係する拡張パックを有効にしない。同様に、一つの拡張パックを有効にしたことだけで、採用 SOT に含まれない provider、capability または公開ツールを有効にしない。

## 統合照会への contribution

統合法情報照会の公開ツール自体は法令コアに属し、拡張パックの有効化で登録または解除しない。拡張パックは、固有の専門ツールと capability に加え、統合照会の能力別 facade、request materializer および型付き result variant を同じ製品機能の集合として有効化する。

query profile の意味認識 contribution と、能力別 facade、request materializer、result variant、binding および route から成る実行 contribution は分けて扱う。採用済み pack の意味認識 contribution は、pack の有効状態にかかわらず composition root の固定 profile set へ含める。selector は `SOT-ARCH-023` に従って意味順位を先に確定し、その後で pack の実行可否を付与する。無効な pack を `capability_unavailable` とするために必要な型付き候補は、`SOT-MODEL-026` に従ってその意味を所有する profile が生成し、core や selector が候補を捏造しない。

この常設部分には、採用済み pack の profile metadata、cue、候補生成規則、共有辞書版、およびその pack が既に公開結果へ付与した `SourceResourceRef` を構造検証する provider/source metadata を含められる。これらは外部呼出しを行わない統合照会の意味契約であり、起動時に法令コア profile と同じく検証する。不整合を無視して profile set または `profileVersion` を pack の有効状態ごとに変えず、検証に失敗した binary は transport を開始しない。

無効な pack では、意味認識後の選択を `pack_disabled` とし、能力別 facade、request materializer、binding、provider route、結果取得および専門ツールを構成しない。常設した意味認識 contribution は capability、binding、provider route または結果取得を有効にするものではなく、意味の弱い別 resource への誤った切替えと既知 `ref` の誤った入力エラーを防ぐためだけに使う。

query profile contribution は capability ID または provider route の代替識別子にしない。profile は capability を要求し、provider binding の選択は既存の route が行う。

## 新しい拡張パックの公開前に定義する事項

新しい拡張パックを実装する前に、その pack の採用 SOT と同じ変更単位で、次をインターフェース SOT に定義する。

- 有効化設定の入力名、型、既定値および未知の `packId` の拒否
- 有効化によって公開する MCP ツール、利用シナリオおよび capability の集合
- 必要な provider binding と route、および設定不足時の起動または実行エラー
- 法令コアと他の拡張パックへ影響しないことの検証方法
- 拡張パックを無効へ戻した場合の公開面と設定の rollback

## 禁止する形

- `packId` を capability ID または provider ID の代わりに使用すること
- provider route の変更だけを拡張パックの有効化として扱うこと
- 一つのプロバイダーを有効にしただけで複数の公開ツールを暗黙に追加すること
- 拡張パックのために既存の法令コアの既定 provider を置き換えること
- 無効な拡張パックの profile から capability または provider を呼び出すこと

## 確認

各拡張パックの採用時に、無設定起動では法令コアの公開面と既定 route が変わらず、採用済み profile の意味認識 contribution だけが外部呼出しなしで固定され、明示的に有効化した pack だけの専門ツール、能力別 facade、request materializer、result variant および binding が追加されることを設定テストと composition root のテストで確認する。無効な pack の照会が `capability_unavailable` となり、法令コアまたは無効な pack の provider を呼び出さないことも確認する。

## 関連

- [SOT-PROD-009: 選択型法情報拡張パックの境界](../00-product/09-selectable-legal-information-extension-packs.md)
- [SOT-ARCH-012: プロバイダーの登録](12-provider-registry.md)
- [SOT-ARCH-013: 情報源の選択と組合せ](13-source-composition.md)
- [SOT-ARCH-017: 採用可能な能力群](17-approved-capability-families.md)
- [SOT-IF-026: プロバイダールーティング設定](../40-interfaces/26-provider-routing-configuration.md)
- [SOT-ARCH-024: 統合照会の内部境界と公開境界](24-unified-query-internal-public-boundary.md)
- [SOT-ARCH-042: 判例引用追跡拡張パックの従属有効化](42-judicial-citations-pack-dependency.md)
