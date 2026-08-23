# SOT-ENG-025: 統合照会のパッケージ構成

- 状態: 有効

## 規定

統合法情報照会の実装は、計画と実行を所有するアプリケーションパッケージ、能力群ごとの query profile、既存の共通前処理および薄い MCP adapter に分ける。

## 基準構成

```text
internal/
├── application/
│   ├── resourceinput/
│   │   └── law_ref.go
│   └── legalquery/
│       ├── request.go
│       ├── candidate.go
│       ├── candidate_composer.go
│       ├── planner.go
│       ├── selector.go
│       ├── materializer.go
│       ├── executor.go
│       ├── result.go
│       ├── service.go
│       └── ports.go
├── queryprofile/
│   ├── cueartifact/
│   ├── metadataartifact/
│   ├── core/
│   │   ├── profile.go
│   │   ├── cues.go
│   │   └── data/
│   │       ├── profile.json
│   │       ├── cues.json
│   │       └── legal-concepts.json
│   └── judicialcases/
│       ├── profile.go
│       ├── cues.go
│       └── data/
│           ├── profile.json
│           └── cues.json
├── querypreprocess/
├── nlp/
│   └── kagome/
├── lawnamelexicon/
├── legalquerycorpus/
└── mcp/
    ├── query_legal_information_tool.go
    └── query_legal_information_schema.go

cmd/
└── legal-query-eval/
    └── main.go

testdata/
└── legalquery/
    ├── schemas/
    ├── corpus-v*/
    └── baselines/
```

これは採用する責務境界を示す基準構成であり、同じ責務を無理に一ファイルへ集約することを要求しない。ファイルは `SOT-ENG-019` の規模と lint 規則に従って小さく分ける。

## `application/legalquery`

- `request.go`: transport 非依存の構文、上限および共通 `ref` 構造を検証した request。採用済み provider/source と read capability の照合は持たない
- `candidate.go`: `LegalQueryCandidate` と型付き step の組立て
- `candidate_composer.go`: profile が明示した必須 member の検証、原文順の合成および構成元候補の置換
- `planner.go`: profile と前処理結果からの候補生成
- `selector.go`: score、margin、pack および対象外の分離判定
- `materializer.go`: `SOT-ARCH-026` の選択済み binding metadata から既存 capability request を作る能力別 materializer
- `executor.go`: item 予算の事前配分、計画順、限定並列、context、部分失敗、呼出し予算および能力結果から上限内の型付き attempt への mapping
- `result.go`: 検証済み attempt を再切出しせず、状態、固定 notice および明確化質問を導出する `LegalQueryResult` への lossless な組立て
- `service.go`: 一回の照会を調整する公開アプリケーション入口
- `ports.go`: 前処理、profile、pack 状態および能力別ユースケースの必要最小 interface

`resourceinput` は、複数の能力 request と logical input が共有する provider 非依存の法令参照および不透明 ID の構造検証だけを持つ。provider/source の採用状態、route、pack、外部 DTO または provider 固有制約を持たない。

`legalquery` は、既存能力の request/result 型、共通の `SourceResourceRef`、provider 非依存の `resourceinput` と自身が所有する interface にだけ依存し、MCP SDK、`internal/source/...` または provider descriptor の具象型を import しない。

能力別 facade と materializer は、`SOT-ARCH-026` の binding interface と変換規則に従う。planner と profile は provider ID を生成しない。

入力 `ref` は `request.go` で `law` または `judicial-decision` の共通構造まで検証し、能力別 facade、registry および materializer で採用済み provider/source metadata、pack 状態および選択した read capability との一致を外部呼出し前に検証する。前段と後段の検証を一方へ省略または重複実装しない。

## query profile

`queryprofile/core` は法令コアの task/resource、根拠、重み、閾値、tie-break および法概念辞書を実装する。`queryprofile/judicialcases` は裁判例固有の語彙、事件参照および plan 規則を実装する。意味認識 contribution は採用後つねに固定 profile set へ注入し、pack が有効な場合だけ裁判例の facade、request materializer、result mapper、binding および route から成る実行 contribution を注入する。

`queryprofile/cueartifact` は、`SOT-ENG-030` の共通 cue 成果物構造、安全境界、
語彙衝突および profile metadata との整合だけを検証する。category、value、
signal、task/resource、score または採用範囲を共通 loader で決めない。
`queryprofile/metadataartifact` は、`SOT-ENG-035` の schema version 別の
閉じた typed decoder、安全境界および共通 metadata model への変換だけを
所有する。profile ID、target の固定順、cue・辞書 version および
条件付き tie-break の採否と完全順は、各 profile 固有 loader が検証する。
固定 profile set 全体の整合は `application/legalquery` が検証する。
`querypreprocess` は、注入された検証済み語彙と一回の Kagome 解析から
`SOT-MODEL-025` の provider 非依存事実を作り、profile の意味候補を生成しない。

`application/legalquery` が前処理結果から profile 用入力を作る共通 constructor
は、原文全体、任意の部分文字列、比較用正規化値または token 列を profile
interface へ公開しない。`SOT-ARCH-021` と `SOT-ARCH-025` が定める閉じた
separator 検証については、既存の位置付き出現と `direct_task` relation を入力に、
条件を満たした `SOT-MODEL-031` の `SharedTerminalSequence` だけを不変な値として
渡せる。
profile 固有の task/resource、演算子、score または候補を共通 constructor で
決めない。

`CandidateGenerationInput` と `SharedTerminalSequence` の getter、要素 accessor、
上限および深い複製は `SOT-MODEL-031` を定義元とする。profile は sidecar を
再構築せず、検証済み値を読むだけとする。

同じ profile に属する複数の明示 task/resource は、その profile が
`SOT-ARCH-025` に従って一候補の複数 step へまとめる。
`queryprofile/judicialcases` は、検証済み入力 `ref` の
`judicial_decision_read` と別に明示された `judicial_decision_search` を
原文順の一候補へまとめる責務を持つ。`SOT-ARCH-027` の composer は異なる
profile の member だけを合成し、同じ profile の search/read を組み立てない。

profile が実装する interface と共通の enum は `application/legalquery` が所有する。core profile と pack profile は互いを import せず、composition root が決定的な順序で一つの不変 profile set として組み立てる。

各 profile は、`SOT-ARCH-031` の意図根拠レイヤと `SOT-ARCH-037` の
evidence cluster を、候補 draft の生成中だけ使う profile-private な型または
関数へ分けられる。その一時データを共通前処理、`application/legalquery` の
公開 interface、別 profile、provider または MCP schema に追加しない。

pack 無効を認識する最小 cue と、入力された `SourceResourceRef` を採用済み provider/source/resource として構造検証する metadata は、core 側の予約済み pack metadata として保持できる。無効な pack の request builder、binding、provider route または result mapper は構成しない。

各 profile directory は `data/profile.json` を所有する。その成果物の閉じた構造、
schema version、`branchRetentionMargin` の存在状態、loader、安全境界および
固定 profile set 内の整合は `SOT-ENG-035` を定義元とする。本規定は、
`profile.json` を各 profile package に配置する責務境界だけを定め、成果物契約を
重複して定義しない。

`data/cues.json` は出典不要の構文 cue と予約語だけを持ち、法概念と法令名の
出典付きデータを混在させない。

profile は `SOT-MODEL-026` の contribution として候補、信号、selection
mode、hedge pair および `SOT-MODEL-028` の composition member を返す。
profile set は異なる profile version を独立に保持できるが、ranking version
と実際の校正値が一致する contribution だけを回収する。`SOT-ARCH-027` の
composer が必要な member を合成した後に stable な最終順位を作り、
composition version を含む set 全体の不透明な plan profile version を
決定的に作る。

将来 pack は `internal/queryprofile/<packId からハイフンを除いた名前>/` を独立単位とし、同じ `profile.json` schema、固有の data、loader test および評価カテゴリを持つ。他 pack の Go package または data file を import しない。

## 既存共通モジュール

Kagome tokenizer、Unicode 比較用正規化、法令名辞書および誤記候補判定は既存の provider 非依存モジュールを再利用する。法概念辞書は `SOT-ENG-023` に従い法令名辞書と別 loader を持つ。

`legalquerycorpus` は `SOT-ENG-026` の版付き成果物型と厳格な loader だけを所有する。`model`、`application/legalquery` および provider 非依存の比較用正規化を参照できるが、evaluator、query profile、executor、provider、`source` または MCP を参照しない。

MCP package は tool schema、入力変換および `CallToolResult` 変換だけを扱う。`query_legal_information_schema.go` は `SOT-MODEL-024` の concrete result/attempt 型を `oneOf`、const discriminator および `additionalProperties: false` で表し、起動時に schema 自身を検証する。単一の generic `result: any` から自動推論しない。

planner の debug tool、private endpoint または MCP-to-MCP 呼出しを追加しない。

## 不変性と並行性

composition root は tokenizer、辞書、profile、pack 状態、route および limiter を起動時に検証して構築し、その後変更しない。各 request は新しい candidate、plan、attempt および result を作り、共有 slice、map または builder を変更しない。

executor は root context と固定 budget を各 step へ渡す。結果は通信完了順ではなく plan 順に新しい配列へ組み立てる。

## 初回構築の実装順序

統合照会を最初に構築するときは次の依存順に分け、各段階で test-first の検証を
通す。この順序は package の初回構築順であり、採用済み意味判定を変更する
rollout の段階、進行条件または commit 境界を定義しない。後者の定義元は
`SOT-ENG-039` だけとする。

1. logical input、candidate、plan、item 配分式、concrete result 型および JSON Schema
2. 固定 corpus、evaluator、core profile、法概念辞書および共通前処理 port
3. selector、能力別 request materializer、fake 能力 port および executor
4. core の `query_legal_information` MCP handler と、七つの専門ツールを含む八ツール登録
5. `judicial-cases` profile contribution、`ref` read、result variant と十ツール登録
6. 全評価、race、契約、既存専門ツール回帰および中央品質ゲート

前段の型または評価基準を後段の都合で黙って変更せず、意味変更が必要な場合は関係 SOT と corpus expectation を同じ変更で review する。

## 確認

package import test で `legalquery` と profile から `source` および MCP SDK への依存がないこと、provider から profile、辞書および `legalquery` への依存がないことを確認する。

core profile だけ、core と judicial profile、fake profile、fake ability ports および race detector を使い、pack 分離、不変性、request materialization、item 配分、決定的順序、context cancellation および既存専門ツールとの独立性を確認する。

profile metadata は `SOT-ENG-035` の固定 test ID で確認し、本規定の package
依存検査から成果物構造を重複定義しない。
閉じた separator 検証では、原文または任意の byte 列を
profile interface へ公開せず、同じ span の複数意味を重複主題にせず、検証済み
`SharedTerminalSequence` だけを深く複製して渡すことを確認する。

`queryprofile/judicialcases` が、入力 `ref` の read と別に明示された search を
同じ contribution の一候補へまとめ、composer を呼ばなくても原文順の二 step を
保持することも profile test で確認する。

MCP schema の全 `oneOf` variant、状態と decision の許可された組合せ、状態ごとの interpretation 件数と availability、未知項目拒否、公開 `ref` 供給元の往復、法令専門ツールで `ref` を公開しない互換性、七専門ツールを含む八ツールの core 登録、`judicial-cases` 有効時の十ツール登録、無効へ戻した場合の八ツール rollback、および stdio/HTTP の schema 一致を golden test で確認する。

## 関連

- [SOT-ARCH-007: 依存方向](../30-architecture/07-dependency-direction.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](../30-architecture/22-unified-query-planning-pipeline.md)
- [SOT-ARCH-024: 統合照会の内部境界と公開境界](../30-architecture/24-unified-query-internal-public-boundary.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](../30-architecture/25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-026: 統合照会の request materialization](../30-architecture/26-unified-query-request-materialization.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](../30-architecture/27-unified-query-cross-profile-composition.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](../30-architecture/31-unified-query-intent-evidence-layer.md)
- [SOT-ARCH-037: 統合照会の正規化済み限定分岐保持](../30-architecture/37-unified-query-normalized-branch-retention.md)
- [SOT-ARCH-033: 統合照会の意味判定 profile set 採用境界](../30-architecture/33-unified-query-profile-set-adoption-boundary.md)
- [SOT-MODEL-026: QueryProfileContribution](../20-model/26-query-profile-contribution.md)
- [SOT-MODEL-028: QueryCandidateCompositionMember](../20-model/28-query-candidate-composition-member.md)
- [SOT-MODEL-031: SharedTerminalSequence](../20-model/31-shared-terminal-sequence.md)
- [SOT-ENG-001: Go パッケージ構成](01-go-package-layout.md)
- [SOT-ENG-012: プロバイダーパッケージ構成](12-provider-package-layout.md)
- [SOT-ENG-019: 静的解析とコーディングスタイル](19-static-analysis-and-coding-style.md)
- [SOT-ENG-028: 統合照会の対象外意図 cue セット](28-unified-query-unsupported-intent-cues.md)
- [SOT-ENG-039: 内容固定済み候補による統合照会の導入段階と変更順序](39-content-bound-unified-query-rollout-stages.md)
- [SOT-ENG-035: 統合照会 profile metadata 成果物契約](35-unified-query-profile-metadata-artifact-contract.md)
