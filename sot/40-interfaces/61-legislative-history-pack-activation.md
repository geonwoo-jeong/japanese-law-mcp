# SOT-IF-061: `legislative-history` 拡張パックの専門公開面

- 状態: 有効

## 規定

`legislative-history` 拡張パックは、設定ファイルの
`extensionPacks.legislative-history.enabled` が `true` の場合に限り、第一段階の
国会発言検索専門公開面を有効にする。

第一段階は `SOT-ARCH-041` の限定的な特則を明示的に採用し、
`query_legal_information` の profile、候補、実行 contribution または公開結果を
追加しない。

## 構造

```yaml
extensionPacks:
  legislative-history:
    enabled: true
```

`legislative-history` の object が持てる項目は boolean の `enabled` だけとする。
省略または `false` は無効、`true` は有効とする。未知の項目、型の不一致および
`null` を受理しない。

この規定は `SOT-IF-067` が定める `judicial-cases` および `judicial-citations` の構造と意味を変更せず、
有効な追加 pack ID として `legislative-history` だけを加える。`extensionPacks` の
入力元と優先順位は `SOT-IF-039` に従い、設定ファイル以外の環境変数または個別の
コマンドラインフラグを追加しない。

## 有効化する原子的集合

有効な場合に限り、次を一つの集合として構成する。

- 利用シナリオ: `SOT-SCN-014`
- capability: `parliament.speech.search@1`
- 専門 MCP ツール: `search_diet_speeches`
- provider と route: `SOT-IF-065` の `ndl-diet-speech-api` と primary route

一つでも構成できない場合は transport を開始しない。専門ツールだけ、provider
binding だけ、または route だけを公開しない。法令コア、`judicial-cases`、既存の
provider route および既存の公開ツールを変更しない。

## 統合照会との境界

第一段階では、国会発言固有の query profile、cue、候補生成規則、能力別 facade、
request materializer および result variant を構成しない。既存の core profile が
`legislative-history` または国会会議録を対象外として検出することは、国会発言検索を
実行する positive contribution ではなく、従来の非実行境界として維持する。

このため、`query_legal_information` は pack の有効状態にかかわらず従来どおり
`legislative-history` を未採用の対象外として扱い、国会発言 provider を呼び出さない。
統合照会へ参加させる場合は `SOT-ARCH-041` の第二段階を別の有効な SOT で採用する。

## 既定値と rollback

無設定起動では `legislative-history` を無効とし、従来の法令コア公開面だけを保持する。
`enabled: false` へ戻して再起動した場合は、`search_diet_speeches`、provider binding、
factory および route を実効構成から除く。統合照会は第一段階の構成要素ではないため、
profile set または公開結果を変更しない。

## 確認

省略、`false`、`true`、未知項目、型不一致および `null` を設定テストで確認する。
無効時は provider factory を呼ばず専門ツールと route が存在しないこと、有効時は
四つの構成要素が同時に追加されること、一部だけでは transport を開始しないこと、
および両状態で統合照会の profile set、候補、公開結果と外部呼出しが変わらないことを
構成テストで確認する。

## 関連

- [SOT-PROD-014: 立法過程拡張パックの国会発言検索](../00-product/14-legislative-history-extension-pack.md)
- [SOT-ARCH-019: 拡張パックの有効化境界](../30-architecture/19-extension-pack-activation-boundary.md)
- [SOT-ARCH-041: 拡張パックの専門公開面の段階採用](../30-architecture/41-staged-specialist-extension-surface.md)
- [SOT-IF-039: 設定ソースと優先順位 v2](39-configuration-sources-and-precedence-v2.md)
- [SOT-IF-040: `judicial-cases` 拡張パックの有効化](40-judicial-cases-pack-activation.md)
- [SOT-IF-065: 国立国会図書館の国会発言検索の組込み採用](65-ndl-diet-speech-built-in-adoption.md)
