# SOT-SCN-014: 国会会議録の発言を検索する

- 状態: 有効

## 規定

利用者は、発言本文、発言者、会議、院または開催日の一つ以上を条件として、国会会議録検索システムに登録された発言を、公式参照先と利用上の注意とともに検索できる。

## 開始条件

`query`、`speaker`、`meetingName`、`house`、`fromDate` または `untilDate` のうち一つ以上が有効であり、`SOT-IF-061` に従って `legislative-history` の専門公開面が有効である。

`limit` と `continuationToken` は検索条件に数えない。

## 基本フロー

1. MCP クライアントが一つ以上の検索条件と任意の返却上限を送信する。
2. Japanese Law MCP が `parliament.speech.search@1` の primary route を通じて、国立国会図書館の発言単位 JSON API を一回呼び出す。
3. JSON の一つの `speechRecord` を一つの `ParliamentSpeech` へ対応させ、API の順序を変更せず最大 30 件返す。
4. 各発言の情報源参照、取得経路、正確な該当総数、今回の返却件数および `SOT-PROD-014` の固定注意を返す。

## 分岐

- 検索条件が一つもない場合、文字列、院名、日付または上限が制約を満たさない場合は、外部情報源を呼び出さず `invalid_argument` とする。
- `fromDate` が `untilDate` より後の場合は、外部情報源を呼び出さず `invalid_argument` とする。
- 現在の組込み provider は安定した snapshot と開催日が同じ結果の決定的な順序を公式仕様から確認できないため、空でない `continuationToken` を外部情報源を呼び出さず `invalid_argument` とする。
- 該当発言がない場合は、正確な総数が 0 の成功した空一覧として返す。
- 情報源が示していない発言者属性、会議情報、PDF または法令との関係を推測しない。
- 情報源を利用できない場合、JSON が採用した契約を満たさない場合または資源上限を超えた場合は、原因を保持した情報源エラーとして返す。

## 完了条件

各結果の `speechId`、会議録 ID、発言 URL、会議録 URL、発言本文、発言者情報および会議情報を同じ公式 JSON または公式 URL で確認できる。`SourcePage.returnedCount` は item 数と一致し、`totalCount` と `totalRelation: exact` は API の総結果件数を表し、`nextToken` は存在しない。

## 関連

- [SOT-PROD-014: 立法過程拡張パックの国会発言検索](../00-product/14-legislative-history-extension-pack.md)
- [SOT-MODEL-034: ParliamentSpeech](../20-model/34-parliament-speech.md)
- [SOT-IF-062: `parliament.speech.search` capability v1](../40-interfaces/62-parliament-speech-search-capability.md)
- [SOT-IF-066: MCP `search_diet_speeches`](../40-interfaces/66-mcp-search-diet-speeches.md)
