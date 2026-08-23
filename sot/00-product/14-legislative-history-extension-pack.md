# SOT-PROD-014: 立法過程拡張パックの国会発言検索

- 状態: 有効

## 規定

`legislative-history` の最初の機能範囲として、国立国会図書館の国会会議録検索システムに登録された発言を検索し、発言本文、発言者、会議情報および公式参照先を確認する読取り専用機能を採用する。

この採用は `SOT-ARCH-041` の第一段階とし、`SOT-IF-061` に従って専門 MCP ツール、
capability、provider binding および route を公開できる。`query_legal_information` への
参加は採用しない。同ツールでは、`SOT-PROD-011` と `SOT-PROD-012` が定める
`legislative-history` の対象外分類を維持し、後続の統合照会 SOT が有効になるまで
国会発言検索へ読み替えない。

## 利用目的

利用者または AI は、次の条件を単独または組み合わせて、国会で行われた発言を公式会議録へ結び付けて確認できる。

- 発言本文に含まれる検索語
- 発言者名
- 会議名
- 院名
- 会議開催日の範囲

検索結果は発言の事実と公式掲載位置を示す。発言を、現行法令、成立した議案、国会全体の意思、立法者意思、行政解釈または法的結論として扱わない。

## 公式情報源と範囲

対象は、国立国会図書館が公開する[国会会議録検索システムの検索用 API](https://kokkai.ndl.go.jp/api.html)のうち、JSON を返す `GET https://kokkai.ndl.go.jp/api/speech` に限る。

初期範囲は発言単位出力だけとし、次を含めない。

- `/api/meeting_list` による会議一覧
- `/api/meeting` による一会議の全発言取得
- 衆議院または参議院の議案、審議経過および公表文書
- 発言と議案、成立法令または法令改正との自動対応
- 発言の要約、立法趣旨、賛否、影響または法的評価の生成
- 全件収集、バックグラウンド取得、恒久キャッシュまたは稼働監視

発言、会議、議案および成立後の法令を一つの資源型、識別子、日付、状態または本文へ平坦化しない。初期範囲では `ParliamentSpeech` が発言を表し、その会議情報は発言に従属する型付き参照として保持する。

## 日本法令索引の扱い

日本法令索引の provider 採用保留境界は `SOT-PROD-015` に従う。したがって、
この第一段階では日本法令索引を構造化 provider として扱わず、国会発言検索の
能力、tool、binding または route に含めない。

## 利用条件と固定注意

取得と表示は、検索用 API が示す利用条件および[国立国会図書館ウェブサイトのコンテンツ利用規約](https://www.ndl.go.jp/sitepolicy/terms)に従う。検索結果の成功応答には、次の文字列を `usageNotice` として変更せず含める。

> 国会会議録の発言は発言者等が著作権を有する場合があります。利用条件を確認してください。発言は現行法令または法的結論を示すものではありません。

この注意は、公式文を転載したものではなく、確認済みの利用条件を基に本製品が固定した
注意文である。権利処理の要否、引用の適法性または発言の法的意味を判定しない。
発言本文、発言者名および会議情報を情報源にない表現で補完しない。

## 確認

公式 API 仕様、API 一覧、日本法令索引のヘルプ、連携終了の告知および利用規約へ到達できることを確認する。契約テストでは、発言だけを型付き結果として返すこと、会議情報を独立した従属構造として保持すること、固定注意を変更しないこと、対象外の議案・法令対応および法的結論を生成しないことを確認する。

## 関連

- [SOT-PROD-009: 選択型法情報拡張パックの境界](09-selectable-legal-information-extension-packs.md)
- [SOT-PROD-015: 日本法令索引の採用保留境界](15-japanese-law-index-adoption-boundary.md)
- [SOT-SCN-014: 国会会議録の発言を検索する](../10-scenarios/14-search-diet-speeches.md)
- [SOT-MODEL-034: ParliamentSpeech](../20-model/34-parliament-speech.md)
- [SOT-IF-061: `legislative-history` 拡張パックの専門公開面](../40-interfaces/61-legislative-history-pack-activation.md)
- [SOT-IF-063: 国立国会図書館の国会発言検索 API 情報源](../40-interfaces/63-source-ndl-diet-speech-api.md)
- [SOT-ARCH-019: 拡張パックの有効化境界](../30-architecture/19-extension-pack-activation-boundary.md)
- [SOT-ARCH-041: 拡張パックの専門公開面の段階採用](../30-architecture/41-staged-specialist-extension-surface.md)
