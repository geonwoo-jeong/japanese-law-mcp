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
| `ref` | `SourceResourceRef` | はい | 被引用対象の裁判例参照 |
| `caseNumber` | string | はい | 公式事件番号 |
| `reporterCitation` | string | いいえ | 公式判例集等表記 |
| `limit` | integer | いいえ | 既定 5、最大 10 |

`caseNumber` は有効な UTF-8 で 1 byte 以上 128 byte 以下とし、空にしない。`reporterCitation` を指定する場合も有効な UTF-8 とし、1 byte 以上 256 byte 以下とする。

## 型付き出力

`JudicialDecisionCitingCandidateSearchResultV1` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `items` | `JudicialCitationCandidate[]` | はい | 公式検索で観測した候補 |
| `searchesPerformed` | integer | はい | 実行した公式検索回数。0 以上 2 以下 |

`JudicialCitationCandidate` は、少なくとも `ref`、`decisionSummary`、`matchedQuery`、`evidenceLevel=official_search_candidate` および `provenance` を持つ。

## 制約

- 検索は、事件番号で一回、`reporterCitation` が存在する場合だけ追加で一回の最大二回とする。
- 候補は公式 DOM 順を維持し、`ref` 基準で重複排除する。
- 自己参照は除外する。
- 候補 PDF を連鎖取得しない。
- 返却上限は 10 件を超えない。

## 失敗

到達し得る失敗は `invalid_argument`、`unsupported_capability`、`configuration_required`、`unsupported_query`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` および `unsafe_source_content` とする。

## 確認

事件番号必須、上限境界、最大二回検索、DOM 順保持、自己参照除外、重複排除、追加 PDF 未取得、空結果および全失敗対応を契約テストで確認する。

## 関連

- [SOT-IF-072: 裁判所「裁判例検索」HTML 情報源 v2](72-source-courts-hanrei-html-v2.md)
- [SOT-IF-073: 裁判所検索の被引用候補マッピング](73-courts-hanrei-citing-candidate-search-mapping.md)
