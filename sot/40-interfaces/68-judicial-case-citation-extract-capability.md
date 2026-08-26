# SOT-IF-068: `judicial-decision.case-citation.extract` capability v1

- 状態: 有効

## 規定

`judicial-decision.case-citation.extract@1` は、裁判例詳細が直接示した公式 `full_text` PDF を一度だけ解析し、本文中で明示的に確認できた判例参照と未解決言及を返す読取り専用の型付き capability とする。

## 能力識別子

| 項目 | 値 |
|---|---|
| `ProviderCapability.id` | `judicial-decision.case-citation.extract` |
| `ProviderCapability.majorVersion` | `1` |
| `ProviderCapability.level` | `extended` |
| `ProviderCapability.stability` | `stable` |

## 型付き入力

`JudicialDecisionCaseCitationExtractRequestV1` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `decision` | `SourcedResource<JudicialDecisionDetails>` | はい | 同じ request 内で詳細取得 capability が返した対象裁判例 |
| `document` | `JudicialDocumentLink` | はい | 上記詳細に含まれる抽出対象の全文 PDF |

`decision.ref.key.resourceType` は `judicial-decision`、`versionId` は未設定とし、summary、ref、source および最後の provenance の整合を検証する。`document` は `decision.data.summary.documents` に値が完全一致する要素として一回だけ現れ、`kind` が `full_text`、`mediaType` が `application/pdf` でなければならない。

MCP 入力から URL、PDF bytes、検索式、最大深さ、parser 名または再試行条件を受け取らない。application service は同じ request の詳細取得結果からこの入力を作り、所属と構造の違反を PDF 取得前に `invalid_argument` とする。

## 型付き出力

`JudicialDecisionCaseCitationExtractResultV1` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `confirmedDecisionMentions` | `JudicialCitationDecisionMention[]` | はい | 明示的に確認した判例参照の出現 |
| `unresolvedMentions` | `JudicialCitationUnresolvedMention[]` | はい | edge に昇格しなかった言及 |
| `documentTextStatus` | string | はい | `available` または `document_text_unavailable` |
| `examinedPageCount` | integer | はい | text layer を検査した page 数 |
| `occurrenceCount` | integer | はい | 二つの mention 配列の合計 |
| `truncated` | boolean | はい | 出現上限で後続を返さなかったか |

`JudicialCitationDecisionMention` は `referenceText`、厳密な表記から作った裁判例同一性、および `evidenceLevel=exact_text_match` の `evidence` を持つ。evidence は PDF provenance と 256 UTF-8 byte 以下の連続した原文抜粋を持つ。確認済み引用に昇格できない場合は `unresolvedMentions` へ残す。

capability は同じ対象の反復を統合せず、各出現を PDF の page 順と text object 順に保持する。先頭 256 occurrence を返し、257 件目を確認した場合は `truncated: true` とする。edge の重複統合と evidence 順序の結合は application service が行う。

## 資源境界と失敗

この capability は OCR、画像復元、別紙復元、外部 font・resource 取得、自動再試行またはリクエスト間キャッシュを行わない。到達し得る失敗は `invalid_argument`、`not_found`、`unsupported_capability`、`configuration_required`、`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` および `unsafe_source_content` とする。

PDF に text layer がない場合は成功結果とし、`documentTextStatus=document_text_unavailable`、二つの mention 配列を空、`occurrenceCount=0` とする。これを「引用なし」と解釈しない。暗号化され復号を要求する PDF は `unsafe_source_content` とする。

## 確認

入力の詳細所属を外部呼出し前に確認すること、`full_text` PDF だけを受理すること、別 origin 拒否、text layer あり・なし、重複言及、曖昧な言及の未解決化、256 occurrence と 256 byte の境界、原文非保存、および資源上限を契約テストで確認する。

## 関連

- [SOT-MODEL-035: JudicialCitationGraph](../20-model/35-judicial-citation-graph.md)
- [SOT-IF-070: 裁判所「裁判例検索」PDF 情報源](70-source-courts-hanrei-pdf.md)
- [SOT-IF-071: 裁判所 PDF の判例引用抽出マッピング](71-courts-hanrei-pdf-extract-mapping.md)
