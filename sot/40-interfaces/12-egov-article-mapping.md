# SOT-IF-012: e-Gov 条文取得マッピング

- 状態: 有効

## 規定

`get_article` は e-Gov 法令 API Version 2 の法令本文取得 API から XML を取得し、法令標準 XML の構造と `Num` 属性によって対象の `Article` または `Paragraph` を特定する。

## 取得

e-Gov Version 2 には独立した条文取得 operation がないため、`SOT-IF-011` と同じ `GET /law_data/{law_id_or_num_or_revision_id}` のリクエストを使用する。`elm` による部分取得には依存せず、取得した `Law` 要素から対象を選択する。

## 位置の選択

- `provision` が `main` の場合は `Law > LawBody > MainProvision` を対象とする。
- `provision` が `supplementary` の場合は、`AmendLawNum` 属性を持たない `Law > LawBody > SupplProvision` を対象とする。
- 対象の規定内で、編、章、節、款または目に相当する構造要素を経由して含まれる `Article` のうち、`Num` が `article` と一致するものを選ぶ。
- 別の `Article`、`Paragraph`、表または引用構造の内側にある `Article` は候補に含めない。
- `paragraph` がある場合は、選択した `Article` の直下にある `Paragraph` のうち、`Num` が一致するものを選ぶ。

候補がない場合は `not_found`、候補が複数ある場合は `ambiguous_location` とする。

## 結果

選択した要素を UTF-8 XML として内容を変更せずにシリアライズし、`content` に設定する。`format` は `xml` とする。

`Citation.location` は次の形式とする。

```text
{main|supplementary}:article={article}[;paragraph={paragraph}]
```

`Citation` の法令 ID、リビジョン ID および URL は `SOT-IF-011` と同じ規則で生成する。

## 関連

- [SOT-IF-003: MCP `get_article`](03-mcp-get-article.md)
- [SOT-IF-011: e-Gov 法令本文取得マッピング](11-egov-law-document-mapping.md)
- [法令の条文構造と法令 XML](https://laws.e-gov.go.jp/docs/law-data-basic/8ebd8bc-law-structure-and-xml/)
