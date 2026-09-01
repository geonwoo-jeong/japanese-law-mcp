# SOT-IF-057: e-Gov 法令改正履歴のマッピングと組込み採用

- 状態: 有効

## 規定

e-Gov 法令 API Version 2 の `GET /law_revisions/{law_id_or_num}` を
`law.revision.list@1` へ対応させ、無設定時の法令コアで
`list_law_revisions` として公開する。

## 外部リクエスト

`lawIdOrNumber` は一つの path segment として UTF-8 から percent-encoding し、
文字列連結によって未処理の値を URL へ埋め込まない。query parameter は
`response_format=json` だけとし、公式 API が受理する絞込み条件を暗黙に
追加しない。接続 origin、base path、proxy、再試行、開始間隔および
同時実行枠は `SOT-IF-004` に従う。

HTTP `200` と `application/json` の組合せだけを成功応答とする。HTTP `404` は
`not_found` とし、その他の status は `SOT-IF-004` と `SOT-IF-017` に従って
正規化する。

## 応答と項目対応

最上位の `law_info` object と `revisions` array は必須とする。`law_info.law_id`、
各 `revisions[].law_revision_id` および `revisions[].law_title` は空でない string
を必須とする。`revisions` の順序は公式応答の新しい法令履歴 ID からの順序を
変更しない。

| e-Gov JSON | 共通値 |
|---|---|
| `law_info.law_id` | page と各 item の `lawId` |
| `law_info.law_num` | `lawNumber` |
| `law_info.promulgation_date` | `promulgationDate` |
| `revisions[].law_revision_id` | `revisionId` |
| `revisions[].law_type` | `lawType` |
| `revisions[].law_title` | `title` |
| `revisions[].law_title_kana` | `titleKana` |
| `revisions[].abbrev` | `abbreviation` |
| `revisions[].category` | `category` |
| `revisions[].updated` | `sourceUpdatedAt` |
| `revisions[].amendment_promulgate_date` | `amendmentPromulgationDate` |
| `revisions[].amendment_enforcement_date` | `effectiveDate` |
| `revisions[].amendment_enforcement_comment` | `effectiveDateNote` |
| `revisions[].amendment_scheduled_enforcement_date` | `scheduledEffectiveDate` |
| `revisions[].amendment_law_id` | `amendmentLawId` |
| `revisions[].amendment_law_title` | `amendmentLawTitle` |
| `revisions[].amendment_law_title_kana` | `amendmentLawTitleKana` |
| `revisions[].amendment_law_num` | `amendmentLawNumber` |
| `revisions[].repeal_date` | `repealRecordedDate` |
| `revisions[].remain_in_force` | `remainInForce` |

省略可能な string が欠落、`null` または空文字の場合は共通値を省略する。
省略可能な boolean の `false` は保持し、`null` と欠落だけを省略する。日付、
日時、boolean または enum の型が異なる場合は `invalid_source_response` とし、
別項目またはローカル計算から補わない。

## 列挙値の正規化

`revisionKind` は次の優先順位で一つに決定する。

1. `amendment_type` が `8`、または `repeal_status` が `Repeal`、`Expire`、
   `LossOfEffectiveness` の場合は `repeal`
2. `mission` が `Partial` の場合は `partial_amendment`
3. `amendment_type` が `3` かつ `mission` が `New` の場合は `affected_law`
4. `amendment_type` が `1` かつ `mission` が `New` の場合は `enactment`

値が省略されている場合は `revisionKind` を省略できる。値が存在するのに上記の
組合せへ対応できない場合、および公式 enum にない値は
`invalid_source_response` とする。

| e-Gov `repeal_status` | `repealStatus` |
|---|---|
| `None` | `none` |
| `Repeal` | `repealed` |
| `Expire` | `expired` |
| `Suspend` | `suspended` |
| `LossOfEffectiveness` | `loss_of_effectiveness` |

`Expire` と `LossOfEffectiveness` の `repeal_date` は公式仕様上、実際の法的な
廃止日ではなくデータ廃止処理日となり得るため、共通値を
`repealRecordedDate` とし、法的効力発生日へ読み替えない。

| e-Gov `current_revision_status` | `currentStatus` |
|---|---|
| `CurrentEnforced` | `current` |
| `UnEnforced` | `future` |
| `PreviousEnforced` | `previous` |
| `Repeal` | `repealed` |

## 参照、出典および完全一覧

各 item は次を持つ。

- `ref.providerId` と `ref.key.sourceId`: `e-gov-law-api-v2`
- `ref.key.resourceType`: `law`
- `ref.key.resourceId`: `law_info.law_id`
- `ref.key.versionId`: `revisions[].law_revision_id`
- `provenance.url`: 実際に呼び出した固定 endpoint
- `provenance.mediaType`: `application/json`
- `provenance.location`: 対応する `revisions[<zero-based index>]`
- `provenance.transformation`: `normalized`
- `provenance.methodId`: `SOT-IF-057`

`returnedCount`、`totalCount` および `revisions` の件数は一致させ、
`totalRelation` は `exact`、継続トークンはなしとする。`revisions: []` は成功した
空一覧とし、`revisions: null` または欠落は受理しない。同じ
`law_revision_id` を重複して成功へ変換しない。

## 資源予算と契約変更

`SOT-IF-004` の `law-revisions-json` を使用し、transfer body 8 MiB、展開後
16 MiB、JSON value 200000、depth 32、解析時間 3 秒、同時実行 group
`egov-http` の一件を上限とする。UTF-8 不正、重複 JSON key、上限超過、取消、
解析期限超過および構造不一致を部分結果へ変換しない。

公式 OpenAPI `2.1.139` から記録した endpoint、成功 media type、最上位必須値、
主要項目の型および enum 集合を契約 fixture として検証する。記録済み公式契約の
不一致は `source_contract_changed`、個別の runtime 応答不正は
`invalid_source_response` として区別する。

## 組込み採用

`law.revision.list@1` は、`SOT-IF-004` と `SOT-IF-060` が定める現在の
`e-gov-law-api-v2` descriptor と六つの compiled binding に保持する。
組込み既定値には次の route を維持する。

```yaml
providerRoutes:
  law.revision.list@1:
    selection: primary
    defaultProviderId: e-gov-law-api-v2
```

composition root はこの route から `list_law_revisions` service と専門操作を構成し、
stdio と Streamable HTTP の両方へ同じ schema で提供する。公開方式ごとの直接登録と
専門操作 registry の構成は `SOT-IF-077` に従う。既存 route、拡張パックの有効条件、
検索結果、継続位置および provider setting は変更しない。

## 確認

公式例に基づく正常、空一覧、`404`、全項目、`false`、各 enum、未知 enum、
必須値欠落、日付・日時・型不正、重複 ID、順序、URL encoding、media type、
全資源予算、取消、同時実行、error normalization および秘密非露出を fixture と
fake transport で確認する。descriptor、binding inventory、既定 route、
composition root、公開方式ごとのツール集合と専門操作 registry、stdio と
Streamable HTTP の schema 一致、および一回の MCP smoke test を確認する。

## 関連

- [SOT-SCN-012: 法令の改正履歴を取得する](../10-scenarios/12-list-law-revisions.md)
- [SOT-MODEL-032: LawRevision](../20-model/32-law-revision.md)
- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-IF-026: プロバイダールーティング設定](26-provider-routing-configuration.md)
- [SOT-IF-055: `law.revision.list` capability v1](55-law-revision-list-capability.md)
- [SOT-IF-056: MCP `list_law_revisions`](56-mcp-list-law-revisions.md)
- [SOT-IF-060: e-Gov 法令版間比較のマッピングと組込み採用](60-egov-law-version-comparison-mapping-and-adoption.md)
- [SOT-IF-077: MCP ツール公開方式と拡張パック有効化](77-mcp-tool-exposure-and-extension-packs.md)
- [e-Gov 法令 API Version 2 OpenAPI](https://laws.e-gov.go.jp/api/2/swagger-ui/lawapi-v2.yaml)
