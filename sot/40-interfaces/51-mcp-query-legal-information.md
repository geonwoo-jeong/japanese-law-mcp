# SOT-IF-051: MCP `query_legal_information`

- 状態: 有効

## 規定

`query_legal_information` は、一つの日本語照会文と任意の既取得資源参照から採用済みの法情報能力を選択し、解釈単位の型付き結果を返す公開 MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 |
|---|---|---:|---|
| `query` | string | はい | trim 後に UTF-8 で 1 byte 以上 2048 byte 以下 |
| `ref` | `SourceResourceRef` | いいえ | 採用済み provider、source および resource type として検証する法令または裁判例の不透明な参照 |
| `limitPerAttempt` | integer | いいえ | 1 以上 20 以下、既定値 10 |

`query` は有効な UTF-8 でなければならず、U+0000 から U+001F および U+007F の ASCII 制御文字を含めない。先頭と末尾の Unicode White_Space を除いた値を検証済み原文とし、比較用正規化値で置き換えない。

`ref` を指定する場合は、`SOT-MODEL-016` の構造、採用済み provider/source metadata、resource type および入力から選んだ read resource との一致を外部呼出し前に検証する。pack が無効でも、その pack が採用した provider/source metadata との一致は検証でき、binding がないことを `invalid_argument` にせず `capability_unavailable` の判定へ渡す。

transport 非依存 request の作成時には、共通構造と `law` または `judicial-decision` resource type までを検証する。採用済み metadata、pack 状態および選択した read resource との一致は、計画と route の確定後、request materialization より前に追加検証する。いずれか一方だけで上記の検証を完了したとは扱わない。

`ref` は検索または一覧 step に使わず、provider route の任意選択、fallback 指定または version override として解釈しない。未採用または未知の provider/source を持つ `ref` は `invalid_argument` とする。

欠落、`null`、型不一致、上限超過、未知の入力項目、整数へ正確に変換できない number および不正な `ref` は、外部情報源を呼び出す前に `invalid_argument` とする。

task、resource、score 閾値、候補数、pack、offset、continuation、`asOf` または任意の filter を追加の入力項目として受け付けない。決定的な指定や継続取得には既存専門ツールを使用する。

## 公開 `ref` の供給元

採用後の公開面で、このツールの read 入力へ渡せる `ref` の供給元は次に限定する。

- `query_legal_information` の法令検索、法令本文検索、法令読取りまたは条文読取り result にある `ref`
- `query_legal_information` の裁判例検索または裁判例読取り result にある `ref`
- `search_judicial_cases` の `items[].ref`。`get_judicial_case` と `query_legal_information` の裁判例読取り result は、入力した値を同じ resource の `ref` として往復する

既存の法令専門ツールである `search_laws`、`search_law_content`、`get_law`、`get_article` および `list_law_updates` は `ref` を公開しない。そのため、公開境界で法令の `ref` を新しく得られる入口は `query_legal_information` だけである。

入力値が上記の公開面から得られたかどうかを履歴または署名で証明しない。`SOT-MODEL-016` の構造、採用済み metadata、resource type および read capability との一致だけを検証し、「同じ MCP が以前返した」という発行証明へ読み替えない。

## 日本語の境界

このツールは翻訳または一般的な言語判定を行わない。入力の実行適格性は次の決定的な規則で判定する。

- 照会文は、Unicode Script property が `Hiragana`、`Katakana` または `Han` の scalar value を一文字以上含まなければならない。
- 日本語部分以外の自然言語表現を翻訳または意味分類しない。非日本語の span は、版付きの構造化 parser が e-Gov の `lawId` 若しくは `revisionId`、ISO 日付または句読点として完全一致で検証できる場合だけ補助値にできる。
- 上記を満たさない非日本語入力、および候補を作るために非日本語 span の意味解釈が必要な混在入力は、外部情報源を呼ばず成功結果の `status=unsupported` とする。

URL、capability ID、provider ID、source ID、e-Gov の `lawId` 若しくは `revisionId`、裁判例の canonical path、事件番号または ISO 日付だけの入力は、日本語照会文として扱わない。決定的な識別子だけを指定する利用者は対応する専門ツールを使用する。`ref` は別の構造化入力であり、`query` の日本語境界を単独では満たさない。

漢字だけの短い一般語を日本語であると推測して自動実行せず、意味候補が基準に達しない場合は `needs_clarification` とする。

## 対象外意図との混在

法的助言、適法性評価、勝敗予測、翻訳または未採用 task/resource を明示的に求める span が一つでもある場合は、同じ照会文に実行可能な情報取得意図があっても一部だけを実行しない。外部情報源を呼ばず `status=unsupported` とし、専門ツールで取得要求だけを明示するよう案内する。

## 出力

成功時は `SOT-MODEL-024` の `LegalQueryResult` を `structuredContent` として返す。`outputSchema` は同 SOT の concrete result と attempt を `oneOf`、const discriminator および `additionalProperties: false` で表す。

| `status` | 外部呼出し | 意味 |
|---|---:|---|
| `completed` | あり | 一つ以上の非空 result が成功し、失敗がない |
| `empty` | あり | 選択した全 collection step が成功し、結果が空 |
| `partial` | あり | 実行した step に成功と失敗が一つ以上ずつある |
| `needs_clarification` | なし | 安全に選べる候補がない |
| `capability_unavailable` | なし | 採用済みだが必要な拡張パックが無効 |
| `unsupported` | なし | 対象外、非日本語境界違反または未採用 task/resource |

`needs_clarification`、`capability_unavailable` および `unsupported` は、plan が正常に非実行を選んだ成功結果であり、`internal_error` または空結果へ変換しない。

数値 score、候補 trace、provider route の理由および未選択候補の全列挙を返さない。`confidence`、`evidenceCodes` および法概念を根拠にした公的資料だけを公開する。

## 実行

このツールは、`SOT-ARCH-022` のアプリケーションサービスを一回呼び出す。planner 自体を provider capability または MCP tool として呼び出さない。

`limitPerAttempt` は collection step の希望上限であり、確定値ではない。executor は、read item の予約と選択済み collection step 数を使う `SOT-MODEL-023` の式で、全 collection step の `effectiveLimit` を外部呼出し前に確定する。

検索 capability には `effectiveLimit` を渡し、read capability では limit を使わない。`law.update.list@1` は完全一覧という既存契約を変更せず、executor の能力結果 mapping で attempt の公開結果だけを `effectiveLimit` 件へ切り詰め、`hasMore` と正確な総件数を保持する。最終 result assembler は検証済み attempt を再切出ししない。空結果や失敗後に item 予算を再配分しない。

全体では候補二件、ranked candidate 十六件、capability 呼出し四回、返却 item 四十件および最初の page だけという固定予算を超えない。

検索結果の継続トークンまたは offset は公開結果へ含めない。続きまたは provider 互換入力が必要な場合は対応する専門ツールを使用する。

## 部分失敗とツールエラー

少なくとも一つの実行済み step が型付き結果を返した場合は、別の実行済み step の情報源エラーを `LegalQueryFailedAttempt` に保持し、`status=partial` の成功結果を返せる。pack 無効または対象外による非実行を `partial` にしない。

実行したすべての step が失敗した場合、入力検証、計画、request materialization、結果変換または不変条件の検査が失敗した場合は、`SOT-IF-007` に従い `isError: true` のツール結果を返す。

複数の失敗から tool error を一つ選ぶ場合は、計画順で最初の step の公開 error を使用する。結果の成否と順序を通信完了順で変えない。

## 起動時構成と到達し得るエラー

法令コアの五 capability route、各 route の binding・materializer、および有効な pack が必要とする route は、transport 開始前に検証する。欠落、不一致または必要設定不足は起動エラーとし、正常起動後の照会で半端な公開面を作らない。

実行時には `invalid_argument`、`not_found`、`ambiguous_location`、`unsupported_query`、`source_auth_failed`、`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit`、`unsafe_source_content` および `internal_error` が到達し得る。

防御的な不変条件検査で `unsupported_capability` または `configuration_required` 相当を検出した場合は、外部呼出しを行わず `internal_error` として fail closed し、次回起動時の構成検証で拒否できるようにする。利用者入力や意味候補の status に変換しない。

`unsupported_query` は、実行対象として選択した採用済み capability が、型として有効な条件を公式な提供範囲外として拒否した場合に使用する。計画段階で対象外と判定した要求は、成功結果の `status=unsupported` とする。

公開 code、`retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 登録と互換性

`query_legal_information` は法令コアの公開ツールとして、stdio と Streamable HTTP の両方へ同じ schema で常時登録する。拡張パックの有効化はこの tool の登録有無を変えず、実行できる profile contribution と result variant だけを変える。

既存専門ツールの名前、schema、pagination、provider route およびエラー契約を変更しない。公開 tool 数と pack ごとの構成は `SOT-IF-040` を定義元とする。

## 確認

入力の全境界、公開供給元ごとの `ref` の往復と不一致、法令専門ツールが `ref` を公開しない互換性、未知項目、非日本語、識別子だけの入力、構造化識別子を含む日本語、混在言語、対象外意図との混在、単一候補、二候補、明確化、pack 無効、空結果、部分失敗、全失敗、全 `R/C` item 配分、四呼出し上限、四十 item 上限、一ページ制約、起動時 route 不備および transport 間の schema 一致を MCP 契約テストで確認する。

`民法第709条を見せて。私の場合は違法ですか` のような混在要求で外部呼出しを行わないこと、内部 score と trace が公開されないこと、法概念の公的資料、裁判例 notice と全 item の provenance が保持されること、および既存専門ツールの契約 fixture が変わらないことを確認する。

## 関連

- [SOT-PROD-011: 統合法情報照会の製品範囲](../00-product/11-unified-legal-query-scope.md)
- [SOT-SCN-009: 日本語の法情報を統合照会する](../10-scenarios/09-query-legal-information.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-MODEL-024: LegalQueryResult](../20-model/24-legal-query-result.md)
- [SOT-ARCH-022: 統合照会の計画パイプライン](../30-architecture/22-unified-query-planning-pipeline.md)
- [SOT-ARCH-024: 統合照会の内部境界と公開境界](../30-architecture/24-unified-query-internal-public-boundary.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-026: プロバイダールーティング設定](26-provider-routing-configuration.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-040: `judicial-cases` 拡張パックの有効化](40-judicial-cases-pack-activation.md)
