# SOT-SCN-010: 統合照会の非実行案内を使って再照会する

- 状態: 有効

## 規定

利用者または MCP クライアントは、統合照会が安全のため実行しなかった場合、
状態ごとに定めた日本語の案内から、不足する指定、利用できない拡張または
対象外の要求を確認し、取得要求を分けるか具体化して再照会できる。

## 開始条件

`query_legal_information` が `needs_clarification`、
`capability_unavailable` または `unsupported` の成功結果を返している。
三状態はいずれも `decision=no_execution`、空の `attempts` および
外部情報源呼出し零回である。

## 状態ごとの案内

| `status` | 案内の定義元 | 利用者が確認する内容 |
|---|---|---|
| `needs_clarification` | `clarification.reasonCodes` と `clarification.questions` | task、resource、対象法令・条番号・裁判例、または四件以下への分割 |
| `capability_unavailable` | `notices` と `interpretations[].requiredPacks` | 必要な採用済み拡張パックと、その現在の無効状態 |
| `unsupported` | `notices` | 非日本語、構造化値だけの入力、対象外との混在、または未採用 task/resource |

`needs_clarification` の行動案内の定義元は `SOT-MODEL-024` の
`clarification.questions` とし、`notices` が空であることを案内欠落と
みなさない。質問は同 SOT の固定集合から一件以上二件以下を返す。

`capability_unavailable` と `unsupported` の行動案内の定義元は
`SOT-MODEL-024` の `notices` とし、状態と plan の理由に対応する固定 notice を
一件以上返す。内部 score、候補 ID、辞書 entry、provider route、
未選択候補または外部応答を説明に加えない。

## 再照会

- `needs_clarification` では、質問が求める task、resource または対象を
  利用者が明示する。四 step 上限の場合は要求を四件以下の複数照会へ分ける。
- `capability_unavailable` では、利用者が必要な pack を起動時設定で有効にするか、
  pack を必要としない別の取得要求を新しく指定する。システムが照会中に pack を
  自動有効化しない。
- `unsupported` では、法的助言、翻訳、比較、影響分析その他の対象外 task と、
  法令または裁判例の取得要求を別の照会へ分ける。対象外部分を言い換えて
  暗黙実行させない。

再照会は新しい独立リクエストとし、前回の候補、質問への回答、pack 状態、
入力本文または結果を server が session state として保持しない。MCP client は
必要な指定を新しい `query` と任意の `ref` に明示する。

## 完了条件

非実行結果だけから、なぜ外部情報源を呼ばなかったかと、次の照会で何を
具体化、分離または有効化すべきかを、日本語の公開項目で確認できる。

案内に従わず終了することも正常な完了とする。一般的な法的助言、翻訳または
設定変更を server が案内文の生成を理由に実行しない。

## 確認

少なくとも次を MCP 契約 test で確認する。

- 曖昧な法概念は `needs_clarification`、空の `notices`、一件以上の
  `clarification.questions` および外部呼出し零回となる。
- 五つ以上の明示主題は、四件以下に分ける固定質問一件だけを返す。
- 無効な `judicial-cases` 要求は `capability_unavailable` とし、固定 notice と
  `requiredPacks=["judicial-cases"]` を返す。
- 取得要求と比較または影響グラフが混在する場合は `unsupported` とし、
  要求を分ける固定 notice を返す。
- `賃金が支払われません。どうすればよいですか。` は対象外の法的助言として
  `unsupported` とし、空の案内を持つ `needs_clarification` にしない。
- 三状態の `content` は `SOT-IF-007` に従い、同じ
  `structuredContent` の JSON を変更せず表す。

## 関連

- [SOT-SCN-009: 日本語の法情報を統合照会する](09-query-legal-information.md)
- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-MODEL-024: LegalQueryResult](../20-model/24-legal-query-result.md)
- [SOT-IF-007: MCP ツール結果](../40-interfaces/07-mcp-tool-result.md)
- [SOT-IF-051: MCP `query_legal_information`](../40-interfaces/51-mcp-query-legal-information.md)
- [SOT-ENG-028: 統合照会の対象外意図 cue セット](../50-engineering/28-unified-query-unsupported-intent-cues.md)
