# SOT-ENG-028: 統合照会の対象外意図 cue セット

- 状態: 有効

## 規定

統合照会は、製品範囲外の task、resource、法的助言および翻訳を、版付きの
日本語 cue セットから決定的に検出し、採用済みの取得意図だけへ縮約して
実行しない。

## 定義元と適用範囲

採用範囲の定義元は `SOT-PROD-011` とし、本規定はその範囲を拡張または
縮小しない。本規定は、対象外意図を `SOT-MODEL-026` の signal へ変換する
語彙、照合境界、版管理および回帰確認だけを定義する。

法令名、法概念、provider 固有の検索語および一般検索語を、この cue セットへ
混在させない。対象外意図の cue は法的な同義語辞書ではなく、外部呼出しを
安全に止めるための閉じた task 語彙とする。

## 必須の意図群

cue セットは少なくとも次の意図群を持つ。

| 意図群 | `QueryProfileSignal` | 最低限認識する表現 |
|---|---|---|
| 個別事案への法的助言・法適用 | `unsupported_legal_advice` | `どうすればよいですか`、`違法ですか`、`適法か判断`、`勝てますか`、`どちらが有利` |
| 翻訳 | `unsupported_translation` | `翻訳`、`英訳`、`和訳` |
| 版比較・差分・追跡 | `unsupported_task_or_resource` | `比較して`、`比較してください`、`差分`、`二時点を比較`、`改正前後を比較`、`変更履歴を追う` |
| 引用・影響関係の分析または可視化 | `unsupported_task_or_resource` | `影響グラフ`、`影響マップ`、`引用関係図`、`引用関係を可視化`、`引用する裁判例のグラフ` |
| 未採用の情報種別または拡張 | `unsupported_task_or_resource` | `立法理由`、`国会審議`、`自治体条例`、`行政規則`、`税務相談`、`労務相談` |

表の表現は必須の回帰境界であり、同じ意図群の活用形、送り仮名、全角・半角、
句読点および敬語差を、比較用正規化または Kagome の語境界によって追加できる。
単なる部分文字列の一致で、より長い別語の内部を cue にしない。

未採用だった task または resource を後に採用する場合は、製品範囲と対応する
profile を先に採用し、対象外 cue から同じ意味群を除いた新しい
`cueSetVersion` と profile version を割り当てる。pack の有効・無効だけで
cue セットを変えない。

## 照合境界

- cue は登録表現との完全一致、比較用正規化一致または Kagome が確認した
  登録語の span だけから作る。対象外 task を誤記補正で新しく作らない。
- `queryTermMentions.kind=quoted_phrase` の内側に完全に含まれる表現は、
  検索対象の文字列であり task ではないため、対象外 signal にしない。
- task として使われた表現の span、または対象外の目的語と task 述語を
  一つの節で結び付けられる表現だけを signal の根拠にする。別の引用句、
  別の節または単なる説明語へ関係を広げない。
- 複数の profile が同じ対象外表現を認識しても、profile set は
  `SOT-MODEL-026` の固定順で同じ signal を一件にする。

対象外 signal が一件でもある場合は、selector が score、候補順位、pack 状態
および provider route を評価して実行対象を選ぶ前に適用する。採用済みの
取得候補が同じ照会文にあれば `mixed_unsupported_intent`、なければ
`unsupported_task_or_resource` を含む `unsupported` plan とし、外部情報源を
呼び出さない。法的助言または翻訳の signal 名を、公開 reason code として
新しく追加しない。

## 成果物と版

cue データは profile ごとの `data/cues.json` に置き、schema version、
profile ID、`cueSetVersion`、cue ID、意図群、signal 値および登録表現を
閉じた JSON として保持する。配列順と cue ID は決定的にし、重複する
正規化表現が異なる signal へ対応する場合は起動を失敗させる。

登録表現、照合境界または signal 対応を変更した場合は `cueSetVersion`、
対象 profile の profile version および統合 profile set version を変更する。
重み、閾値または ranking scale を変えない限り ranking version は変更しない。

## 確認

少なくとも次を、外部ネットワークを使わない profile fixture と
`SOT-ENG-024` の評価で確認する。

- `民法第103条を引用する裁判例の影響グラフを作成してください。` は、
  法令または条文の候補を内部に保持できても `mixed_unsupported_intent` で
  外部呼出しを零回とする。
- `2020年1月1日と2025年11月1日の個人情報保護法を比較してください。` は、
  一方の日付だけの読取りへ縮約せず `mixed_unsupported_intent` とする。
- `賃金が支払われません。どうすればよいですか。` は法的助言として
  `unsupported` とし、曖昧な取得要求の `needs_clarification` にしない。
- `「比較」を含む条文を検索してください。` は引用句内の `比較` を
  対象外 task にせず、通常の本文検索候補を作る。
- cue セットの順序、重複、未知の signal、profile ID 不一致および
  版不整合を起動時に拒否する。

## 関連

- [SOT-PROD-011: 統合法情報照会の製品範囲](../00-product/11-unified-legal-query-scope.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](../20-model/25-legal-query-preprocess-result.md)
- [SOT-MODEL-026: QueryProfileContribution](../20-model/26-query-profile-contribution.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-025: 統合照会のパッケージ構成](25-unified-query-package-layout.md)
