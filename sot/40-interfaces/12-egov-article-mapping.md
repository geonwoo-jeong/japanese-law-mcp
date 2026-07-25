# SOT-IF-012: e-Gov 条文取得マッピング

- 状態: 有効

## 規定

`get_article` は e-Gov 法令 API Version 2 の法令本文取得 API から XML を取得し、法令標準 XML の構造と `Num` 属性によって対象の `Article` または `Paragraph` を特定する。

## 取得

e-Gov Version 2 には独立した条文取得 operation がないため、`SOT-IF-011` と同じ `GET /law_data/{law_id_or_num_or_revision_id}` のリクエストを使用する。`elm` による部分取得には依存せず、取得した `Law` 要素から対象を選択する。

`law.article.read@1` では、`resource.providerId` と `resource.key.sourceId` がともに `e-gov-law-api-v2`、`resource.key.resourceType` が `law` であることを確認する。`resource.key.versionId` がある場合は `SOT-IF-011` と同じくパスへその値を設定し、ない場合は `resource.key.resourceId` を設定する。`asOf` がある場合だけ `asof` を設定し、`resource.key.versionId` と `asOf` を同時に送信しない。

`asOf` が `2017-04-01` より前の場合は、外部呼出し前に `unsupported_query` とする。

`location.provision`、`location.articleNumber` および `location.paragraphNumber` は `SOT-MODEL-018` の正規形として検証する。e-Gov の外部リクエストにはこれらを送信せず、取得後の XML 要素選択にだけ使用する。

## 位置の選択

- `provision` が `main` の場合は `Law > LawBody > MainProvision` を対象とする。
- `provision` が `supplementary` の場合は、`AmendLawNum` 属性を持たない `Law > LawBody > SupplProvision` を対象とする。
- 対象の規定内で、編、章、節、款または目に相当する構造要素を経由して含まれる `Article` のうち、`Num` が `location.articleNumber` と一致するものを選ぶ。
- 別の `Article`、`Paragraph`、表または引用構造の内側にある `Article` は候補に含めない。
- `location.paragraphNumber` がある場合は、選択した `Article` の直下にある `Paragraph` のうち、`Num` の十進整数値が一致するものを選ぶ。

候補がない場合は `not_found`、候補が複数ある場合は `ambiguous_location` とする。

## 結果

選択した要素を UTF-8 XML として内容を変更せずにシリアライズし、`content` に設定する。`format` は `xml` とする。

`Citation.location` は次の形式とする。

```text
{main|supplementary}:article={articleNumber}[;paragraph={paragraphNumber}]
```

`Citation` の法令 ID、リビジョン ID および URL は `SOT-IF-011` と同じ規則で生成する。

`law.article.read@1` の結果は `SourcedResource<LawArticleFragment>` とし、`LawArticleFragment.location` は正規化済みの入力 `location` と同じ値にする。`ref.providerId` と `ref.key` は取得した法令本文と同じ provider と `law` 資源を指す。`provenance.url` は `Citation.url`、`provenance.location` は `Citation.location`、`mediaType` は `application/xml`、`transformation` は `extracted`、`methodId` は `SOT-IF-012` とする。

入力に `resource.key.versionId` がある場合は返されたリビジョン ID、入力の `resource.key.resourceId` は返された法令 ID と完全一致することを確認する。既存 `get_article` facade は、内部結果の `data` から `format`、`content` および `citation` だけを公開する。

## 関連

- [SOT-IF-032: MCP `get_article`](32-mcp-get-article.md)
- [SOT-IF-011: e-Gov 法令本文取得マッピング](11-egov-law-document-mapping.md)
- [SOT-IF-025: law.article.read capability v1](25-law-article-read-capability.md)
- [SOT-MODEL-015: LawArticleFragment](../20-model/15-law-article-fragment.md)
- [SOT-MODEL-018: LawArticleLocation](../20-model/18-law-article-location.md)
- [法令の条文構造と法令 XML](https://laws.e-gov.go.jp/docs/law-data-basic/8ebd8bc-law-structure-and-xml/)
