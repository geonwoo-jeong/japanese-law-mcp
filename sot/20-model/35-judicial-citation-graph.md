# SOT-MODEL-035: JudicialCitationGraph

- 状態: 有効

## 規定

`JudicialCitationGraph` は、一件の公表裁判例を起点に、確認済み引用、被引用候補、参照法条、原審関係、coverage および解決できなかった言及を、1-hop 限定の閉じた型で表す。

## 構造

### `JudicialCitationGraphResult`

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `status` | string | はい | `complete` または `partial` |
| `coverageNotice` | string | はい | `SOT-PROD-016` の固定注意文 |
| `graph` | `JudicialCitationGraph` | はい | 追跡結果 |
| `issues` | `JudicialCitationIssue[]` | はい | 失敗、制限または縮退の記録 |

### `JudicialCitationGraph`

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `rootNodeId` | string | はい | 起点裁判例ノードの ID |
| `nodes` | `JudicialCitationNode[]` | はい | 一意なノード集合 |
| `edges` | `JudicialCitationEdge[]` | はい | relation ごとの有向辺 |
| `unresolvedMentions` | `JudicialCitationUnresolvedMention[]` | はい | edge に昇格しなかった言及 |
| `summary` | `JudicialCitationSummary` | はい | 件数と観測分布 |
| `coverage` | `JudicialCitationCoverage` | はい | 実行した方向、失敗、縮退および観測範囲 |

### `JudicialCitationNode`

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `nodeId` | string | はい | 一つの結果内だけで有効な request-local ID |
| `nodeType` | string | はい | `judicial_decision`、`law_provision` または `judicial_decision_reference` |
| `label` | string | はい | 利用者向け表示名 |
| `ref` | `SourceResourceRef` | いいえ | 公式裁判例として同定できた場合の参照 |
| `decisionSummary` | `JudicialDecisionSummary` | いいえ | `nodeType=judicial_decision` のときの概要 |
| `lawReference` | `JudicialCitationLawReference` | いいえ | `nodeType=law_provision` のときの法令参照 |
| `referenceText` | string | いいえ | `nodeType=judicial_decision_reference` のときの原文 |

### `JudicialCitationLawReference`

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `lawId` | string | はい | 一意に解決できた法令 ID |
| `lawTitle` | string | はい | 法令名 |
| `location` | `LawArticleLocation` | はい | 正規化した条文位置 |

### `JudicialCitationEdge`

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `edgeId` | string | はい | 一つの結果内だけで有効な request-local ID |
| `fromNodeId` | string | はい | 始点ノード |
| `toNodeId` | string | はい | 終点ノード |
| `relationType` | string | はい | `cites_judicial_decision`、`possible_cites_judicial_decision`、`references_law_provision` または `has_lower_court_decision` |
| `evidence` | `JudicialCitationEvidence[]` | はい | 確認根拠の列挙順保存集合 |

### `JudicialCitationEvidence`

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `evidenceLevel` | string | はい | `official_metadata`、`exact_text_match` または `official_search_candidate` |
| `provenance` | `Provenance` | はい | 根拠の取得元 |
| `excerpt` | string | いいえ | 256 UTF-8 byte 以下の短い抜粋 |

原文位置は `Provenance.location` だけに保持し、同じ位置を別項目へ複製しない。

### `JudicialCitationUnresolvedMention`

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `mentionType` | string | はい | `judicial_decision` または `law_provision` |
| `mentionText` | string | はい | 原文 |
| `reason` | string | はい | edge に昇格しなかった閉じた理由 |
| `provenance` | `Provenance` | はい | 原文確認元 |

`reason` は `ambiguous_target`、`no_published_target_match`、`insufficient_identity`、
`unsupported_reference_form`、`unregistered_law_name`、`ambiguous_law_location` または
`fuzzy_match_only` のいずれかとする。text layer 自体を取得できない場合は原文 mention を
作らず、`JudicialCitationIssue.code=document_text_unavailable` と coverage に記録する。

### `JudicialCitationSummary`

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `confirmedOutgoingDecisionCount` | integer | はい | `cites_judicial_decision` の件数 |
| `incomingCandidateCount` | integer | はい | `possible_cites_judicial_decision` の件数 |
| `referencedProvisionCount` | integer | はい | `references_law_provision` の件数 |
| `lowerCourtRelationCount` | integer | はい | `has_lower_court_decision` の件数 |
| `unresolvedMentionCount` | integer | はい | 未解決言及数 |
| `incomingObservedYearBuckets` | `JudicialCitationYearBucket[]` | はい | 被引用候補の裁判年ごとの観測件数 |
| `incomingObservedCategoryBuckets` | `JudicialCitationCategoryBucket[]` | はい | 被引用候補の掲載カテゴリごとの観測件数 |

`JudicialCitationYearBucket` は `year: integer` と `count: integer`、
`JudicialCitationCategoryBucket` は `publicationCategory` と `count: integer` を持つ。
年は昇順、掲載カテゴリは `SOT-MODEL-020` の列挙順とし、件数 0 の bucket は作らない。
候補がない場合も二配列を `null` にせず空配列とする。

### `JudicialCitationCoverage`

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `requestedDirection` | string | はい | `outgoing`、`incoming` または `both` |
| `hopDepth` | integer | はい | 固定値 `1` |
| `outgoing` | `JudicialCitationDirectionCoverage` | はい | PDF 参照抽出の処理範囲 |
| `incoming` | `JudicialCitationDirectionCoverage` | はい | 被引用候補検索の処理範囲 |

`JudicialCitationDirectionCoverage` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `status` | string | はい | `complete`、`partial`、`unavailable` または `not_requested` |
| `methods` | string[] | はい | 完了した `official_detail_metadata`、`official_pdf_text` または `official_case_search` |
| `truncated` | boolean | はい | 情報源または処理上限により後続を返していないか |
| `limit` | integer | いいえ | incoming に適用した 1 以上 10 以下の上限 |
| `attemptedSearches` | integer | いいえ | incoming で試みた検索数。0 以上 2 以下 |
| `completedSearches` | integer | いいえ | incoming で成功した検索数。0 以上 `attemptedSearches` 以下 |

要求していない方向は `not_requested` と空の `methods` にする。詳細 metadata はどの方向でも
処理するが、`outgoing` の方向状態は判例参照 PDF 抽出、`incoming` は候補検索の完了状態を
表す。利用者が指定した `incomingLimit` で正常に切り詰めたことだけでは方向を `partial` に
せず、`complete` と `truncated: true` を両立できる。

### `JudicialCitationIssue`

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `direction` | string | はい | `outgoing` または `incoming` |
| `stage` | string | はい | `official_detail_metadata`、`official_pdf_text`、`official_case_search` または `law_reference_resolution` |
| `code` | string | はい | 制限または失敗の識別子 |
| `message` | string | はい | 利用者向け説明 |
| `retryable` | boolean | はい | 同じ入力で再試行する意味があるか |

## 制約

- `status` は要求した方向がすべて `complete` の場合だけ `complete` とする。一方向または一検索が失敗しても型付き結果を一つ以上保持できる場合は `partial` とし、一件以上の `issues` を必須とする。ルート取得失敗、全要求方向失敗または全域取消では成功モデルを返さない。利用者の `incomingLimit` による正常な切詰めだけでは `partial` にしない。
- `relationType`、`evidenceLevel`、`mentionType` および `reason` は本規定の閉じた値だけを使用する。
- `possible_cites_judicial_decision` は公式検索候補を表し、`cites_judicial_decision` と同じ意味へ縮約しない。
- `references_law_provision` は、法令名と `LawArticleLocation` を一意に解決できた場合だけ作成する。法令 revision は推測しない。
- `has_lower_court_decision` は公式詳細 HTML に原審メタデータが存在し、裁判所名と事件番号を同一の原審裁判例参照へ構成できる場合に限る。読めない場合は `judicial_decision_reference` または `unresolvedMentions` を使用する。
- 同じ `fromNodeId`、`toNodeId` および `relationType` を持つ重複 edge は一件に統合し、`evidence` の列挙順を保持する。
- 判例関係 edge は全 relation を合わせて 64 件、法条 edge は 32 件を超えない。上限到達時は決定的な先頭を保持し、coverage の `truncated` と issue で示す。
- `excerpt` は根拠確認に必要な最小限にとどめ、単一情報源から過度に長い原文を複製しない。
- `issues`、`message`、`excerpt` および `mentionText` に PDF 全文、検索語または未加工 HTML を格納しない。
- `nodeId` と `edgeId` は次のリクエスト入力、情報源 ID またはリクエスト間で安定した ID として使用しない。

## 確認

閉じた relation 種別、候補と確認済み引用の分離、法条正規化の一意条件、重複 edge 統合、evidence 順序保持、partial の条件、coverage 集計、未解決言及の保持および JSON 表現を単体テストで確認する。

## 関連

- [SOT-MODEL-012: Provenance](12-provenance.md)
- [SOT-MODEL-016: SourceResourceRef](16-source-resource-ref.md)
- [SOT-MODEL-018: LawArticleLocation](18-law-article-location.md)
- [SOT-MODEL-020: JudicialDecisionSummary](20-judicial-decision-summary.md)
- [SOT-PROD-016: 判例引用追跡拡張パック](../00-product/16-judicial-citations-extension-pack.md)
