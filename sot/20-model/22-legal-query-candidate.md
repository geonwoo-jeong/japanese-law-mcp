# SOT-MODEL-022: LegalQueryCandidate

- 状態: 有効

## 規定

`LegalQueryCandidate` は、一つの照会文から導いた一つの意味解釈を、根拠、意味 score および provider を選ぶ前の型付き logical step 列によって表す内部モデルとする。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `candidateId` | string | はい | 一つの plan 内で一意な不透明識別子 |
| `semanticScore` | integer | はい | 同じ profile version 内で意味一致を比較する内部値 |
| `confidence` | string | はい | `high`、`medium` または `low` |
| `evidenceCodes` | `string[]` | はい | 候補を支持する根拠分類 |
| `conceptSources` | `LegalConceptSource[]` | はい | 法概念を根拠にした場合の公的資料 |
| `requiredPacks` | `string[]` | はい | 実行に必要な拡張パック ID |
| `steps` | `LegalQueryCandidateStep[]` | はい | 一つ以上四つ以下の logical step |

`candidateId`、`semanticScore` および未選択候補の全列挙は内部情報とし、公開 MCP 結果へそのまま露出しない。

`candidateId` と `stepId` は、照会ごとに内部で新しく作る 1 byte 以上 64 byte 以下の ASCII 識別子とする。小文字英数字の segment を一つの `-` で連結し、先頭と末尾は小文字英数字とする。入力本文または入力断片を識別子へ埋め込まず、利用側はその構造を解釈しない。`candidateId` と `stepId` の plan 全体での一意性は `LegalQueryPlan` が確認し、`LegalQueryCandidate` は自身の `steps` に重複する `stepId` がないことを確認する。

## 根拠コード

`evidenceCodes` は次の値だけを使用し、強い根拠から弱い根拠の順に評価する。

1. `official_identifier`: 公式の法令 ID、リビジョン ID または入力の `SourceResourceRef`
2. `structured_reference`: 法令番号、事件番号、条、項または完全な暦日
3. `explicit_task`: 検索、読取りまたは更新一覧を明示する語
4. `explicit_resource`: 法令、条文または裁判例を明示する語
5. `official_alias`: 出典を持つ正式名称、略称または別名
6. `legal_concept`: 出典を持つ法概念辞書の一致
7. `morphological_context`: Kagome と周辺語から得た文脈
8. `unique_typo_correction`: 一意に確定した軽微な誤記候補
9. `general_term`: それだけでは対象を決められない一般語

`evidenceCodes` は一件以上とし、同じ値を重複させず、上記の強い根拠から弱い根拠の順に保持する。

`semanticScore` は成功確率ではなく、同じ profile version の候補を順位付けするためだけに使う。数値範囲、重みおよび閾値は評価済み profile が所有する。

`evidenceCodes` に `legal_concept` がある場合は `conceptSources` を一件以上持ち、それ以外では空配列とする。`LegalConceptSource` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `conceptId` | string | はい | 法概念辞書内で安定した不透明識別子 |
| `title` | string | はい | 対応を確認した公的資料名 |
| `url` | string | はい | 利用者情報を含まない絶対 HTTPS URL |
| `confirmedOn` | date | はい | 対応を確認した暦日 |

`conceptSources` は `conceptId` を重複させない。辞書の内部 weight、候補 template または入力断片は持たない。

`requiredPacks` は、1 byte 以上 64 byte 以下で小文字英数字の segment を一つの `-` で連結した pack ID だけを持つ。同じ pack ID を重複させず、pack ID の昇順に保持する。step と pack ID の意味対応は candidate model に固定せず、採用済み query profile が候補生成規則として所有し、selector が pack の有効状態を判定する。

## logical step

`LegalQueryCandidateStep` は次を持つ。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `stepId` | string | はい | plan 内で一意な step ID |
| `task` | string | はい | `search`、`read` または `list_updates` |
| `resource` | string | はい | `law`、`law_provision` または `judicial_decision` |
| `capabilityId` | string | はい | `ProviderCapability.id` |
| `capabilityMajorVersion` | integer | はい | `ProviderCapability.majorVersion` |
| `inputKind` | string | はい | logical input の variant |
| `logicalInput` | logical input variant | はい | provider を選ぶ前の取得条件 |

`capabilityId` に `@1` を連結せず、major version を別項目で保持する。許可する対応は次に限定する。

| task | resource | `capabilityId` | major | `inputKind` |
|---|---|---|---:|---|
| `search` | `law` | `law.search` | `1` | `law_search` |
| `search` | `law_provision` | `law.content.search` | `1` | `law_content_search` |
| `read` | `law` | `law.document.read` | `1` | `law_read` |
| `read` | `law_provision` | `law.article.read` | `1` | `law_article_read` |
| `list_updates` | `law` | `law.update.list` | `1` | `law_updates` |
| `search` | `judicial_decision` | `judicial-decision.search` | `1` | `judicial_decision_search` |
| `read` | `judicial_decision` | `judicial-decision.read` | `1` | `judicial_decision_read` |

## logical input variant

- `LawSearchIntentV1`: 検証済み検索語と任意の `asOf`
- `LawContentSearchIntentV1`: `allTerms`、`anyTerms`、`excludeTerms` および任意の `asOf`
- `LawReadIntentV1`: 一意な `lawId` と任意の `revisionId`・`asOf`、または入力で受け取った法令の `ref`
- `LawArticleReadIntentV1`: 一意な `lawId` または法令の `ref`、`LawArticleLocation` および任意の `asOf`
- `LawUpdateListIntentV1`: 一つの `date`
- `JudicialDecisionSearchIntentV1`: 検証済み検索語
- `JudicialDecisionReadIntentV1`: 入力で受け取った裁判例の `ref`

各 logical input は、対応する capability v1 のうち registry なしで決定できる provider 非依存入力制約を candidate の作成時に満たす。検索語、構造化検索語、日付、不透明な法令 ID、リビジョン ID、`LawArticleLocation` および `SourceResourceRef` は、materialize するまで構造的に不正な値を保持しない。`ref` の provider と source が採用済みの対応か、pack が有効か、および選択した read capability と一致するかは candidate で推測せず、registry と materializer が外部呼出し前に検証する。

`LawReadIntentV1` は、次のどちらか一方だけを持つ。

- `lawId` と、任意の `revisionId` または任意の `asOf`
- 入力で受け取った `resourceType=law` の `ref`

`revisionId` と `asOf` は同時に使わない。`ref` を使う形では `lawId`、`revisionId` および `asOf` を持たない。

`LawArticleReadIntentV1` は `lawId` または入力で受け取った `resourceType=law` の `ref` のどちらか一方、必須の `LawArticleLocation` および任意の `asOf` を持つ。版を含む `ref` と `asOf` は同時に使わない。

一つの read logical input で、ID と `ref` を同時に使わない。裁判例の事件番号、題名または URL だけから read step を作らず、これらは検索候補とする。裁判例 read は、`judicial-decision` resource として構造検証でき、version を持たない `SourceResourceRef` が入力にある場合だけ作る。provider と source の採用済み対応は前段で同一文字列として仮定せず、registry metadata で後段検証する。

logical input は `limit`、offset または continuation を持たない。executor が plan 全体の固定予算を確定した後に、能力別 request materializer が logical input を既存 capability request へ変換する。

能力別 request materializer は、route 選択後の binding と共通モデルを使い、必要な `SourceResourceRef` と検索上限を組み立てる。planner と profile は provider ID を選択または生成しない。利用者が渡した `ref` に含まれる provider ID は不透明な共通参照として保持し、registry と resource type の検証にだけ使う。

候補モデルに provider DTO、外部 query parameter、HTML selector、任意の `map[string]any`、生の MCP JSON または意味を定義しない `filters` を持たせない。

`steps` の配列順を計画上の実行順とする。複数 step は照会文に複数の明示意図がある場合だけ作る。検索結果の順位を使って後続 step の識別子を補わず、read に必要な ID または `ref` を計画確定前に一意に得られなければ read step を作らない。

## 拡張

新しい task、resource、根拠コードまたは logical input variant は、対応する製品、利用シナリオ、capability、materializer および評価 SOT を採用した変更でだけ追加する。

## 確認

許可した七組の対応、ID と major version の分離、logical input と capability の不一致、四 step 超過、provider DTO の混入、`ref` の構造と resource type の不一致、誤記候補の一意性および検索結果に依存する暗黙 read の禁止をモデルテストで確認する。未知の provider と採用済み provider・source metadata の不一致は、registry と materializer のテストで外部呼出し前に拒否することを確認する。

## 関連

- [SOT-MODEL-013: ProviderCapability](13-provider-capability.md)
- [SOT-MODEL-016: SourceResourceRef](16-source-resource-ref.md)
- [SOT-MODEL-018: LawArticleLocation](18-law-article-location.md)
- [SOT-MODEL-023: LegalQueryPlan](23-legal-query-plan.md)
- [SOT-ARCH-018: 拡張パック単位の正規化境界](../30-architecture/18-pack-scoped-normalization-boundary.md)
- [SOT-IF-022: law.search capability v1](../40-interfaces/22-law-search-capability.md)
- [SOT-IF-023: law.content.search capability v1](../40-interfaces/23-law-content-search-capability.md)
- [SOT-IF-024: law.document.read capability v1](../40-interfaces/24-law-document-read-capability.md)
- [SOT-IF-025: law.article.read capability v1](../40-interfaces/25-law-article-read-capability.md)
- [SOT-IF-034: law.update.list capability v1](../40-interfaces/34-law-update-list-capability.md)
- [SOT-IF-041: judicial-decision.search capability v1](../40-interfaces/41-judicial-decision-search-capability.md)
- [SOT-IF-042: judicial-decision.read capability v1](../40-interfaces/42-judicial-decision-read-capability.md)
