# SOT-ENG-034: 統合照会の意味判定変更における導入段階と変更順序

- 状態: 有効

## 規定

統合照会の採用済み意味判定 profile set を変更する導入作業は、relation 生成、
profile 内適用、限定分岐、評価成果物準備、原子的採用、独立後続変更および
案内同期の七段階を、この順序で一段階ずつ進め、前段階を飛ばして公開既定動作を
切り替えない。

## 適用範囲

本規定は、`query_legal_information` の採用済み意味判定 profile set、関連
corpus/baseline、検索例カタログおよび同じ観測動作を持つ公開既定値を変更する
導入作業の順序を定める。

とくに、relation 生成と relation 依存の meaning 変更を段階導入する作業を
対象とする。provider parser、HTTP transport、MCP schema 互換性または意味判定と
独立した文書整備まで、この七段階へ機械的に当てはめない。

日常の軽微な文言修正、誤字修正または公開観測を変えない説明追加には適用しない。
一方で、cue role、relation 依存の signal、候補保持、複数主題分離、明確化条件、
selection、標準 corpus、baseline、採用 manifest または公開検索例の観測結果を
変える変更には適用する。

## 段階

### 第 1 段階: 構造準備

不変 model、cue schema、共通 loader、固定 profile 所有境界および active 成果物の不変条件を整える。ここでは、公開既定の meaning、decision、step、reason または外部呼出し境界を変えない。

### 第 2 段階: profile 内の relation 適用

positive task cue の role をそろえ、共通前処理で relation を生成し、各 profile が意図根拠レイヤと対象外候補 scope を内部で適用できるようにする。ただし、この段階では未採用の next profile set を production から選択可能にしない。

### 第 3 段階: 限定分岐と列挙境界

`SOT-ARCH-025` と `SOT-ARCH-032` が定める、共有末尾 cue による主題分離と
evidence cluster 単位の限定分岐保持を、profile 内の規則として完成させる。
共有末尾 cue の構造確認は `SOT-ARCH-021` の閉じた separator 検証を使い、
`SOT-MODEL-031` の `SharedTerminalSequence` だけを profile へ渡し、
原文または token 列を公開しない。限定分岐は
`SOT-ENG-035` の独立した `branchRetentionMargin` を使い、`singleMargin`
または `hedgeMargin` を代用しない。この段階の `branchRetentionMargin` は
test 専用 set の暫定値に限り、active metadata、標準 command または baseline を
更新しない。

第 3 段階の内部変更は、次の順で一つずつ行う。

1. 次版 profile 用に、schema version 2 の metadata model、loader、
   `branchRetentionMargin` の存在状態および固定 set の整合検証を準備する
2. production-neutral な `SharedTerminalSequence` sidecar を構築し、active
   profile が意味候補へ使用しないことを確認する
3. profile ごとの private evidence mapping と evidence cluster を構築する
4. core profile だけが sidecar を消費して共有末尾 cue の複数主題 step と
   限定分岐を作る test 専用経路を完成させる
5. `judicial-cases` profile は sidecar を消費せず、自身の位置付き出現による
   根拠対応と cluster だけで限定分岐を適用する
6. core と `judicial-cases` を production と同じ固定順で組み立て、全 profile が
   schema version 2、同じ ranking version および同じ
   `branchRetentionMargin` を持つ一つの test 専用固定 profile set を完成させる

1 から 5 は個別 model、loader、profile または integration test の準備であり、
それだけでは校正、holdout、標準 command、baseline、production または採用候補の
固定 profile set とみなさない。6 を満たした固定 set だけを第 4 段階の development
校正へ渡せる。

この段階は意味規則そのものを再定義せず、それらの採用順序上の変更単位だけを
固定する。

### 第 4 段階: 評価成果物の準備

現行採用済み profile set を変えずに、次の順で成果物を準備する。

1. production、標準 command、現行 corpus、baseline および観測動作を変えず、
   現行採用集合を `SOT-ENG-033` の初回 history manifest と
   `current.json` へ固定する
2. profile、辞書および既存期待値を変更しない独立変更で、新しい corpus と
   holdout を review し、digest を固定する
3. holdout の内容または結果を参照せず、development 集合だけで第 3 段階の
   test 専用固定 profile set を校正し、profile version と ranking version を固定する
4. holdout の内容または結果を読まずに、`SOT-ENG-036` の schema、固定 evaluator、
   次の `baselineVersion` および候補 file の出力先だけを準備する。この時点では
   holdout の測定値を持つ baseline を生成しない
5. 固定した holdout を採用判定に一回だけ使い、`SOT-ENG-024` の全受入基準を
   検査する。合格した一回の report byte だけを 4 で予約した version file の
   baseline 候補として保存し、corpus、profile set、metric 計算および privacy
   境界を独立 review する
初回の relation 依存変更では、2 の候補名を `corpus-v10`、4 の予約名を
`default-2` とする。これらは今回の初回候補を識別する名前であり、将来の導入で
常に固定する版ではない。初回導入後の現行値は current adoption tuple を
定義元とする。

holdout の結果を見て同じ候補 profile set の値、辞書、規則または期待値を
調整しない。受入基準を満たさない場合は第 5 段階へ進まず、失敗した候補を
採用対象から外して、新しい準備変更として第 3 段階以降をやり直す。失敗結果を
後続候補の変更判断に利用した場合、その holdout digest を再び採用判定へ使わない。
失敗した holdout は履歴成果物として保存し、同じ `leakageGroupId` を含まない
新しい holdout を独立 review してから、次の第 4 段階を行う。

### 第 5 段階: 原子的採用

`SOT-ARCH-033` の「原子的な採用」に列挙された全要素と、
`SOT-ENG-033` の current adoption tuple を、同じ採用変更で新しい profile set へ
切り替える。本規定では完全な採用要素を重複して列挙せず、いずれか一部だけを
先行切替しないという順序上の制約だけを定める。

### 第 6 段階: 意味判定から独立した後続変更

e-Gov parser の error 分類、canonical target 優先、provider mapping 調整など、意味判定 profile set と独立に進めるべき変更を別変更単位として扱う。第 5 段階へ混在させない。

### 第 7 段階: 案内と scenario 同期

前段の原子的採用義務に含まれない非実行案内、利用シナリオおよび説明文書を、現行標準の観測結果へ同期する。将来状態を先回りして「実装済み」と書かない。

## 段階ごとの進行条件

各段階は、次を満たした場合だけ次段階へ進める。

1. 対象 SOT に直接ひも付く必要最小限の検証が成功している
2. 独立 reviewer の評価が 8.0 / 10 以上で、blocker がない
3. review 指摘を反映した後に、同じ段階境界を再確認している
4. 段階の変更単位が、前後の段階と混ざらない

公開既定動作を切り替えない段階では、`SOT-ARCH-033` に従い、active artifact、
production composition root、標準 corpus、baseline および検索例カタログを現行のまま保つ。

必要最小限の検証内容は `SOT-ENG-027` を定義元とし、公開既定動作を切り替える
第 5 段階では `SOT-ENG-020` と `SOT-ENG-024` の中央品質ゲートを省略しない。

## 段階間の禁止事項

- 第 2 段階または第 3 段階で、未採用の next profile set を CLI、設定、環境変数、MCP または transport から選択可能にすること
- 第 4 段階で、次版 corpus や baseline 候補を標準 command や中央品質ゲートの現行参照先へ切り替えること
- 第 4 段階で、holdout の内容または結果を `branchRetentionMargin`、重み、閾値、
  辞書、規則若しくは期待値の調整へ使用すること
- 第 5 段階で、`SOT-ARCH-033` の採用要素または `SOT-ENG-033` の tuple の
  一部だけを先行採用すること
- 第 6 段階の provider parser 変更を、第 5 段階の meaning 変更に便乗させること
- 第 7 段階より前に、将来の案内文や scenario を現行確認済みのように記載すること

## Wiki との関係

実装差分、進捗、review 点数、確認日および段階の現在地は Wiki で追跡できる。ただし、段階そのものの定義、順序および進行条件の定義元は本 SOT とする。Wiki が本規定と異なる場合は、本規定を優先する。

## 確認

少なくとも次を確認する。

- 段階 1 から 7 の定義が、`query_legal_information` の意味判定変更単位を過不足なく分離している
- 現行標準を変えない段階で、production composition root、standard corpus および baseline が不変である
- 原子的採用段階で、`SOT-ARCH-033` の全採用要素と `SOT-ENG-033` の
  current tuple の同期が要求される
- provider parser などの独立変更が、meaning 変更段階から切り離されている
- review 8.0 / 10 以上、blocker なし、および段階ごとの必要最小限検証が、次段階へ進む前提として明示されている

## 関連

- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](../30-architecture/25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](../30-architecture/31-unified-query-intent-evidence-layer.md)
- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](../30-architecture/33-unified-query-profile-set-adoption-boundary.md)
- [SOT-ARCH-032: 統合照会の限定分岐保持](../30-architecture/32-unified-query-bounded-branch-retention.md)
- [SOT-MODEL-031: SharedTerminalSequence](../20-model/31-shared-terminal-sequence.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-ENG-025: 統合照会のパッケージ構成](25-unified-query-package-layout.md)
- [SOT-ENG-026: 統合照会の評価コーパス成果物契約](26-legal-query-corpus-artifact-contract.md)
- [SOT-ENG-027: 資源制約を踏まえた検証段階](27-resource-aware-verification-stages.md)
- [SOT-ENG-029: 統合照会の検索例カタログ](29-unified-query-example-catalog.md)
- [SOT-ENG-033: 統合照会 profile set 採用 manifest](33-unified-query-profile-set-adoption-manifest.md)
- [SOT-ENG-035: 統合照会 profile metadata 成果物契約](35-unified-query-profile-metadata-artifact-contract.md)
- [SOT-ENG-036: 統合照会の評価 baseline 成果物契約](36-unified-query-evaluation-baseline-artifact-contract.md)
