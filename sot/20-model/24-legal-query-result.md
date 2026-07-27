# SOT-MODEL-024: LegalQueryResult

- 状態: 有効

## 規定

`LegalQueryResult` は、統合法情報照会の公開成功結果を、状態ごとの concrete object、公開解釈および能力別の型を保った attempt variant によって表す。

## JSON schema

公開 `outputSchema` は JSON Schema の `oneOf` を使い、次の六つの concrete result object を判別する。各 object の `status` と非実行時の `decision` は表の定数、実行時の `decision` は表の enum とし、`additionalProperties: false` とする。

| concrete object | `status` | `decision` | `interpretations` | `attempts` | 固有項目 |
|---|---|---|---|---|---|
| `LegalQueryCompletedResult` | `completed` | `single` または `hedged` | 一件または二件の `available` | 一件以上 | なし |
| `LegalQueryEmptyResult` | `empty` | `single` または `hedged` | 一件または二件の `available` | 一件以上 | なし |
| `LegalQueryPartialResult` | `partial` | `single` または `hedged` | 一件または二件の `available` | 二件以上 | なし |
| `LegalQueryNeedsClarificationResult` | `needs_clarification` | `no_execution` | 二件以下の `available` | 空 | `clarification` |
| `LegalQueryCapabilityUnavailableResult` | `capability_unavailable` | `no_execution` | 一件または二件の `pack_disabled` | 空 | なし |
| `LegalQueryUnsupportedResult` | `unsupported` | `no_execution` | 空 | 空 | なし |

各 concrete object は次の共通項目を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `status` | 上表の const | はい | 結果状態 |
| `decision` | 上表の const または enum | はい | 実行方法または非実行 |
| `language` | string | はい | 固定値 `ja` |
| `interpretations` | `LegalQueryInterpretation[]` | はい | 公開する解釈。二件以下 |
| `attempts` | `LegalQueryAttempt[]` | はい | 実行した step。四件以下 |
| `notices` | `string[]` | はい | 全体に適用する注意 |

数値の意味 score、候補生成 token、文字位置、未選択候補の全列挙、重み、閾値、provider route の選択理由および内部 trace は含めない。

## 全体 notice

`notices` は任意の文字列を受け取らず、result assembler が状態、attempt 数、page preview および plan の理由から次の固定文字列だけを表の順序で重複なく導出する。零件以上三件以下とし、該当しない場合も空配列を持つ。

| 順序 | 固定文字列 | 付与条件 |
|---:|---|---|
| 1 | `最初のページだけを返しています。続きが必要な場合は対応する専門ツールを使用してください。` | 一つ以上の page preview が `hasMore=true` |
| 2 | `複数の解釈または step の件数、順位および継続位置は統合していません。` | attempt が二件以上 |
| 3 | `一部の step が失敗しました。attempts の error を確認してください。` | `status=partial` |
| 4 | `必要な拡張パックが無効です。interpretations の requiredPacks を確認してください。` | `status=capability_unavailable` |
| 5 | `日本語の法情報取得要求を入力してください。` | `status=unsupported` で plan の理由が `non_japanese_query` |
| 6 | `法情報の取得要求と法的助言、翻訳または対象外の要求を分けて入力してください。` | `status=unsupported` で plan の理由が `mixed_unsupported_intent` |
| 7 | `指定した task または resource は統合照会の採用範囲外です。対応する専門ツールを確認してください。` | `status=unsupported` で plan の理由が `unsupported_task_or_resource` |

`completed` と `empty` は一番目と二番目だけを条件付きで持つ。`partial` は三番目を必ず持ち、一番目と二番目も条件に従って持てる。`needs_clarification` は空配列、`capability_unavailable` は四番目だけ、`unsupported` は plan の理由に対応する五番目から七番目を一件以上三件以下持つ。

## 公開する解釈

`LegalQueryInterpretation` は次を持ち、`additionalProperties: false` とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `interpretationId` | string | はい | 結果内で一意な不透明識別子 |
| `confidence` | string | はい | `high`、`medium` または `low` |
| `evidenceCodes` | `string[]` | はい | `SOT-MODEL-022` の公開可能な根拠分類 |
| `conceptSources` | `LegalConceptSource[]` | はい | 法概念一致を検証する公的資料 |
| `availability` | string | はい | `available` または `pack_disabled` |
| `requiredPacks` | `string[]` | はい | 必要な拡張パック ID。空配列を許す |
| `steps` | `LegalQueryStepSummary[]` | はい | step ID、task、resource、capability ID と major version |

`LegalConceptSource` の項目と従属条件は `SOT-MODEL-022` をそのまま使用し、公開用に別定義を作らない。`requiredPacks` と availability の従属条件は `SOT-MODEL-023` の selection をそのまま使用する。`outputSchema` では同じ型を `$defs` から参照する。

公開用 `interpretationId` は selection 順に `interpretation-1` または `interpretation-2` とし、内部 `candidateId`、score、辞書 entry または入力断片を埋め込まない。result 内で重複させない。

実行結果では `decision=single` のとき interpretation を一件、`decision=hedged` のとき二件とし、全 attempt の `interpretationId` はその配列の一件を参照する。非実行結果では `decision=no_execution` 以外を schema で拒否する。これらの組合せと表の件数・availability 制約は説明文だけにせず、各 concrete result object の JSON Schema で固定する。

## attempt の concrete variant

`LegalQueryAttempt` も JSON Schema の `oneOf` とする。成功 variant は共通の `interpretationId`、`stepId`、`capabilityId`、`capabilityMajorVersion`、`outcome`、`resultKind` および型付き `result` を持つ。`capabilityId` と major version を一つの文字列へ連結しない。

| concrete attempt | `resultKind` の const | `result` |
|---|---|---|
| `LegalQueryLawSearchAttempt` | `law_search` | `page: LegalQueryPagePreview` と `items: SourcedResource<LawSummary>[]` |
| `LegalQueryLawContentSearchAttempt` | `law_content_search` | `page: LegalQueryPagePreview` と `items: SourcedResource<LawContentMatch>[]` |
| `LegalQueryLawDocumentAttempt` | `law_document` | `item: SourcedResource<LawDocumentRepresentation>` |
| `LegalQueryLawArticleAttempt` | `law_article` | `item: SourcedResource<LawArticleFragment>` |
| `LegalQueryLawUpdatesAttempt` | `law_updates` | `page: LegalQueryPagePreview` と `items: SourcedResource<LawUpdate>[]` |
| `LegalQueryJudicialSearchAttempt` | `judicial_decision_search` | `coverageNotice`、`page: LegalQueryPagePreview` と `items: SourcedResource<JudicialDecisionSummary>[]` |
| `LegalQueryJudicialDecisionAttempt` | `judicial_decision` | `coverageNotice` と `item: SourcedResource<JudicialDecisionDetails>` |

成功 variant の `outcome` は `completed` または `empty` とし、`error` を持たない。read variant は `completed` だけを許可する。collection variant は item が空なら `empty`、一件以上なら `completed` とする。

`LegalQueryFailedAttempt` は、共通の識別項目、`outcome: failed` および `error: ErrorResult` だけを持ち、`resultKind` と `result` を持たない。

failed attempt の `error.code` は、実行済み step から到達し得る `not_found`、`ambiguous_location`、`unsupported_query`、`source_auth_failed`、`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit`、`unsafe_source_content` または `internal_error` に限定する。入力検証または materialization の `invalid_argument`、起動時に排除する `unsupported_capability` および `configuration_required` を partial result に入れない。

すべての attempt object と各 `result` object は `additionalProperties: false` とし、`resultKind` に対応しない payload の共存を schema で拒否する。異なる variant を一つの `items: object[]`、`content`、`date` または共通文書へ平坦化しない。

attempt の `interpretationId` は同じ result の `interpretations` の一件を参照し、`stepId`、`capabilityId` および `capabilityMajorVersion` はその interpretation の一つの step と一致する。attempt は interpretation 順と各 interpretation の step 順からなる計画順の部分列とし、同じ step を重複させない。

裁判例 attempt の `ref.key.resourceId` は、`SOT-MODEL-016` と `SOT-IF-041` に従う provider mapping 所有の canonical ID を変更せず保持する。共通 result assembler は `resourceId` を `decisionId` または `detailUrl` から再導出せず、両者の文字列一致を要求しない。公式詳細 URL から canonical ID を作る規則とその対応は各 provider mapping の SOT と適合試験で確認する。

裁判例の二 attempt が持つ `coverageNotice` は `SOT-IF-047` の固定文字列を省略、要約または変更せず使用し、全体の `notices` へ重複転記しない。

## 一ページの表示

`LegalQueryPagePreview` は次を持ち、`additionalProperties: false` とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `returnedCount` | integer | はい | この attempt で公開した item 数 |
| `hasMore` | boolean | いいえ | 情報源の page 情報から続きを確定できる場合の有無 |
| `totalCount` | integer | いいえ | 情報源が示した件数 |
| `totalRelation` | string | 条件付き | `totalCount` がある場合の `exact` または `lower_bound` |

統合照会は最初の page だけを返し、`nextToken`、`nextOffset` または provider 固有の継続位置を返さない。続きが必要な利用者は、対応する専門ツールを明示的に使用する。

`hasMore` は、内部 page に次の位置がある場合、または完全一覧を公開上限で切り詰めた場合は `true` とする。正確な件数等から続きがないと確定できる場合は `false` とし、判定できない場合は省略する。

`returnedCount` と `totalCount` は零以上とし、attempt の `returnedCount` は同じ result の `items` 件数と一致する。`totalCount` がある場合は `returnedCount` 以上とし、`totalRelation` を必須とする。`totalCount` がない場合は `totalRelation` を持たない。`totalRelation=exact` の場合は `hasMore` を必須とし、`returnedCount < totalCount` なら `true`、等しければ `false` とする。`totalRelation=lower_bound` でも `returnedCount < totalCount` の場合は、少なくとも未公開の item があるため `hasMore=true` を必須とする。`totalRelation=lower_bound` で両件数が等しい場合は、情報源の page 情報または公開上限によって続きを確定できるときだけ `hasMore` を指定し、判定できない場合は省略できる。

## 状態

- `completed`: 一つ以上の attempt が非空で成功し、失敗がない。
- `empty`: 実行した全 attempt が collection 成功で、すべての item が空。
- `partial`: 実行した attempt のうち一つ以上が成功し、一つ以上が `failed`。
- `needs_clarification`、`capability_unavailable` および `unsupported`: 外部情報源を呼び出さず、attempt は空。

pack 無効の非実行を `partial` にしない。実行した全 attempt が失敗した場合は `LegalQueryResult` を返さず、`SOT-IF-007` に従う MCP ツールエラーとする。

失敗がなく、非空の成功と空の collection 成功が混在する場合は `completed` とする。空の collection 成功と `failed` が混在する場合は `partial` とする。`partial` の成功には `completed` と `empty` のどちらの outcome も数え、すべてが `failed` の配列は拒否する。

候補または provider をまたいだ `totalCount`、関連度、並び順、継続位置および類似重複を一つに合成しない。各 item の `ref`、`provenance`、`Citation` および `coverageNotice` を保持する。

## 明確化

`LegalQueryClarification` は、一件以上二件以下の `reasonCodes` と一件以上二件以下の決定的な `questions` を持ち、`additionalProperties: false` とする。`reasonCodes` は `SOT-MODEL-023` の `needs_clarification` が許す `below_execution_threshold` と `ambiguous_candidates` だけを、同 SOT の順序で重複なく持つ。

`questions` は次の固定文字列だけを表の順序で重複なく持つ。

1. `検索、読取りまたは更新一覧のどれを行うか指定してください。`
2. `法令、条文または裁判例のどれを対象にするか指定してください。`
3. `対象の法令名、法令 ID または条番号を指定してください。`
4. `対象の裁判例を検索する語または裁判例の ref を指定してください。`

質問は入力本文、内部 token または score を反復せず、task、resource、法令若しくは裁判例のどれを指定すべきかだけを案内する。

## 確認

六つの result object、七つの成功 attempt と失敗 attempt、`oneOf`、const discriminator、`additionalProperties: false`、状態と decision の組合せ、状態ごとの interpretation 件数・availability・必須項目、型の排他性、一ページ制約、件数の非集約、固定 notice と質問、attempt の参照と計画順、法概念出典、provenance、裁判例 notice、部分失敗、全失敗時のツールエラーおよび内部 score の非公開を JSON Schema とモデルテストで確認する。

## 関連

- [SOT-MODEL-005: ErrorResult](05-error-result.md)
- [SOT-MODEL-012: Provenance](12-provenance.md)
- [SOT-MODEL-014: SourcePage](14-source-page.md)
- [SOT-MODEL-016: SourceResourceRef](16-source-resource-ref.md)
- [SOT-MODEL-022: LegalQueryCandidate](22-legal-query-candidate.md)
- [SOT-MODEL-023: LegalQueryPlan](23-legal-query-plan.md)
- [SOT-IF-007: MCP ツール結果](../40-interfaces/07-mcp-tool-result.md)
- [SOT-IF-051: MCP `query_legal_information`](../40-interfaces/51-mcp-query-legal-information.md)
