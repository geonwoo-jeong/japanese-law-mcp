# SOT-ARCH-021: プロバイダー非依存の検索語前処理

- 状態: 有効

## 規定

検索語の解析、辞書照合および安全な誤記候補の判定は、MCP transport と情報源アダプターから独立したアプリケーション層の前処理として実装し、検索能力ごとのプロファイルを注入して再利用する。

## 境界

共通前処理は、Unicode の比較用正規化、日本語の形態素解析、辞書語の抽出、編集距離、候補の一意性判定、provider に依存せず採用済みの情報モデル SOT が定義した構造化表記を抽出する型付き parser、および原文位置を持つ一般検索語の抽出を提供する。法令、裁判例その他の能力に固有の正式名称、採用する別名または資源対応は共通前処理へ埋め込まず、能力別プロファイルまたは辞書から渡す。事件番号 parser は `SOT-MODEL-027` の構文だけを実装し、裁判所が公開する符号一覧の allowlist または一意な裁判例との対応を持たない。

一つの照会文に対する Kagome の解析は一回だけ行い、その token 列を辞書出現、
誤記候補、形態素検索語および `SOT-MODEL-029` の cue task relation で共有する。
引用句と `SOT-MODEL-027` の事件番号も同じ前処理呼出しの中で原文から抽出する。
節 span は `SOT-MODEL-029` の固定境界文字を原文 byte 列から一回だけ走査して
作り、token の byte span、Kagome の品詞、引用句 span および cue span から
同 SOT の閉じた手順で relation を作る。「説明らしい」などの意味推定や、
未登録の述語を補う規則を加えない。
query profile は位置付きの前処理結果を利用し、原文を別の tokenizer、正規表現、
節分割器または独自の規則で再解析して検索語、助詞、述語若しくは task relation を
補わない。

前処理結果から profile 用入力を作る共通 constructor は、`SOT-MODEL-026` の `standalone_structured_query` に該当するかを、位置付きの公式識別子、事件番号、日付および区切りだけから一回だけ導出する。各 query profile はこの不変な判定結果を読み、原文または span を独自に再評価して同じ境界を実装しない。

能力別プロファイルは、その能力に必要な法令名、法概念、cue の語彙および
`SOT-MODEL-029` の閉じた `syntaxRole` だけを個別に注入できる。共通前処理は
role、原文 span、一回の token 列および引用境界から構文 relation を作るが、
cue の intent group、signal、採用範囲、score、task/resource または pack を
解釈しない。法令名辞書を持たない profile も同じ共通前処理を利用できるものとし、
特定の能力の語彙を共通前処理の必須起動条件にしない。

共通前処理は同じ profile ID の cue だけを relation にする。query profile は
検証済み relation を自身の採用範囲と signal へ対応させる。profile 横断の候補
合成は `SOT-ARCH-027` の責務とし、前処理が cue の近接から意味候補を合成しない。
relation 対応経路は `SOT-ENG-028` の cue schema version 3 だけを受理し、
固定 profile set の一部だけを relation 対応にして旧 schema と混在させない。

法令名検索プロファイルは `search_laws` のユースケースへ注入する。e-Gov その他の情報源アダプターは Kagome、法令名辞書または誤記判定へ依存せず、ユースケースが選択した検証済みの検索語だけを各情報源の入力へ対応させる。

共通前処理を別の検索能力へ適用する場合は、その能力の有効な利用シナリオとインターフェース SOT が、辞書の対象、候補の優先順位、曖昧性および再検索条件を定義した後に、別のプロファイルとして接続する。法令名プロファイルを裁判例本文、法令本文その他の検索へ暗黙に適用しない。

統合法情報照会は、同じ共通前処理 primitive を再利用し、法令コアと拡張パックごとの query profile を別に注入する。統合照会の法概念辞書、意図 score および限定並列の判断を、`search_laws` の法令名解決 profile へ逆流させない。逆に、`search_laws` の空結果後だけ行う再検索規則を統合照会の候補選択へ流用しない。

## 不変性

辞書、索引、Kagome tokenizer および設定は起動時に検証して構築し、その後は変更しない。リクエスト間で利用者の検索語、選択した候補または検索結果を保持しない。

比較用の正規化値と解析 token は内部判定にだけ使用する。位置付きの引用句または最大形態素句は query profile の候補材料であって、外部情報源へ送ることを許可する値ではない。情報源へ送る値は、利用者の検証済み原文 span または辞書で出典を確認した正式名称から作り、選んだ logical input の能力別制約を満たすものに限る。編集途中の文字列、距離だけで生成した未知の文字列または token の断片を送らない。

## エラー境界

原検索の情報源エラーを前処理による再検索で隠さない。起動時の辞書または tokenizer 構築に失敗した場合は、機能を部分的に無効化して起動せず、構成エラーとして起動を失敗させる。

## 確認

共通前処理の単体テストを外部ネットワークなしで実行し、能力別辞書と
syntax role を差し替えられること、完全な事件番号を型付き出現として一度だけ
抽出して一般検索語から除外すること、同じ token 列から cue task relation を
決定的に作ること、query profile が原文を再解析しないこと、e-Gov アダプターが
Kagome、辞書および relation model の package を import しないこと、および
並行リクエストで共有する不変オブジェクトに data race がないことを確認する。

## 関連

- [SOT-SCN-011: 解決済み法令を検索結果で優先する](../10-scenarios/11-prioritize-resolved-law-search-result.md)
- [SOT-ARCH-005: 一時的なリクエスト処理](05-ephemeral-request-lifecycle.md)
- [SOT-ARCH-010: プロバイダーの分離](10-provider-isolation.md)
- [SOT-ARCH-018: 拡張パック単位の正規化境界](18-pack-scoped-normalization-boundary.md)
- [SOT-ARCH-020: 採用済みユースケース境界](20-adopted-use-case-boundary.md)
- [SOT-ARCH-030: 解決済み法令対象の検索結果優先順位](30-canonical-law-target-priority.md)
- [SOT-ENG-022: 法令名検索辞書](../50-engineering/22-law-name-search-lexicon.md)
- [SOT-ENG-023: 統合法情報照会の法概念辞書](../50-engineering/23-unified-query-concept-lexicon.md)
- [SOT-MODEL-027: JudicialCaseNumberMention](../20-model/27-judicial-case-number-mention.md)
- [SOT-MODEL-029: CueTaskRelation](../20-model/29-cue-task-relation.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](22-unified-query-planning-pipeline.md)
