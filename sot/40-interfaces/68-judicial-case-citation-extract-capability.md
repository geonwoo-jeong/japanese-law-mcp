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
| `ref` | `SourceResourceRef` | はい | 解析対象の裁判例参照 |
| `documentLink` | `JudicialDocumentLink` | はい | 同じ裁判例詳細が示した `full_text` PDF |

`documentLink.kind` は `full_text`、`mediaType` は `application/pdf`、`url` は `https://www.courts.go.jp/assets/hanrei/` 配下の HTTPS URL でなければならない。任意 URL、`summary`、`attachment`、別 origin、別 path、または詳細 HTML が直接示していない PDF を受理しない。

## 型付き出力

`JudicialDecisionCaseCitationExtractResultV1` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `confirmedDecisionMentions` | `JudicialCitationDecisionMention[]` | はい | 明示的に確認した判例参照 |
| `unresolvedMentions` | `JudicialCitationUnresolvedMention[]` | はい | edge に昇格しなかった言及 |
| `documentTextAvailable` | boolean | はい | PDF text layer の有無 |
| `provenance` | `Provenance` | はい | PDF を根拠とする取得元 |

`JudicialCitationDecisionMention` は、少なくとも `referenceText`、存在する場合の `ref`、`evidenceLevel=exact_text_match`、`excerpt` および `location` を持つ。確認済み引用に昇格できない場合は `unresolvedMentions` へ残す。

## 資源境界と失敗

この capability は OCR、画像復元、別紙復元、外部 font・resource 取得、自動再試行またはリクエスト間キャッシュを行わない。到達し得る失敗は `invalid_argument`、`not_found`、`unsupported_capability`、`configuration_required`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` および `unsafe_source_content` とする。

PDF に text layer がない場合は成功結果とし、`documentTextAvailable=false`、`confirmedDecisionMentions=[]` とする。

## 確認

入力が `full_text` PDF だけを受理すること、別 origin 拒否、text layer あり・なし、重複言及、曖昧な言及の未解決化、抜粋と位置、原文非保存、および資源上限を契約テストで確認する。

## 関連

- [SOT-MODEL-035: JudicialCitationGraph](../20-model/35-judicial-citation-graph.md)
- [SOT-IF-070: 裁判所「裁判例検索」PDF 情報源](70-source-courts-hanrei-pdf.md)
- [SOT-IF-071: 裁判所 PDF の判例引用抽出マッピング](71-courts-hanrei-pdf-extract-mapping.md)
