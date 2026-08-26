# SOT-IF-069: `judicial-decision.citing-candidate.search` capability v1

- 状態: 有効

## 規定

`judicial-decision.citing-candidate.search@1` は、指定した公表裁判例を指している可能性がある公表裁判例候補を、裁判所の公式検索結果だけから返す読取り専用の型付き capability とする。

## 能力識別子

| 項目 | 値 |
|---|---|
| `ProviderCapability.id` | `judicial-decision.citing-candidate.search` |
| `ProviderCapability.majorVersion` | `1` |
| `ProviderCapability.level` | `extended` |
| `ProviderCapability.stability` | `stable` |

## 型付き入力

`JudicialDecisionCitingCandidateSearchRequestV1` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `target` | `SourcedResource<JudicialDecisionDetails>` | はい | 引用される側の公式掲載裁判例 |
| `limit` | integer | いいえ | 既定 5、最大 10 |

`target.ref.key.resourceType` は `judicial-decision`、`versionId` は未設定とし、ref、summary、source および最後の provenance を検証する。`target.data.summary.caseNumber` は必須とし、検索値はこの事件番号と、存在する場合の `target.data.reporterCitation` だけから作る。任意の検索式、URL、別名、分かち書き、誤記補正、類義語、事件名、日付または LLM 生成語を受け取らない。

## 型付き出力

一つ以上の検索が成功した場合は `JudicialDecisionCitingCandidateSearchResultV1` を返す。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `status` | string | はい | `complete` または `partial` |
| `items` | `JudicialCitationCandidate[]` | はい | 公式検索で観測した候補 |
| `coverage` | `JudicialCitingCandidateSearchCoverage` | はい | 計画した検索と観測範囲 |
| `issues` | `JudicialCitingCandidateSearchIssue[]` | はい | 失敗した検索。全成功時は空配列 |

`JudicialCitationCandidate` は、`decision: SourcedResource<JudicialDecisionSummary>` と、`evidenceLevel=official_search_candidate` の `evidence` を持つ。検索語そのものを出力へ含めず、確認済み引用の `exact_text_match` へ昇格させない。

`JudicialCitingCandidateSearchCoverage` は、事件番号、次いで判例集表記の順に `searchKind` と `status: complete | failed` を保持する `attempts`、成功結果で観測した item 数、自己参照・重複除外後の候補数および `truncated` を持つ。issue は失敗した `searchKind` と共通情報源エラーを持ち、検索語、URL query または HTML 本文を持たない。

## 制約

- 検索は、事件番号で一回、`reporterCitation` が存在する場合だけ追加で一回の最大二回とする。
- 候補は公式 DOM 順を維持し、`ref` 基準で重複排除する。
- 自己参照は除外する。
- 候補 PDF を連鎖取得しない。
- 返却上限は 10 件を超えない。
- 二検索を予定した場合、一方が失敗しても context が有効なら残る検索を一回だけ実行する。一つ以上が成功した場合は成功候補を捨てず `status=partial` とし、両方が失敗した場合又は全域 cancellation の場合だけ capability error とする。

## 失敗

到達し得る失敗は `invalid_argument`、`unsupported_capability`、`configuration_required`、`unsupported_query`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` および `unsafe_source_content` とする。

## 確認

事件番号必須、上限境界、最大二回検索、DOM 順保持、自己参照除外、重複 evidence の順序保持、追加 PDF 未取得、空結果、一検索失敗の部分結果、全検索失敗および cancellation を契約テストで確認する。

## 関連

- [SOT-IF-072: 裁判所「裁判例検索」HTML 情報源 v2](72-source-courts-hanrei-html-v2.md)
- [SOT-IF-073: 裁判所検索の被引用候補マッピング](73-courts-hanrei-citing-candidate-search-mapping.md)
