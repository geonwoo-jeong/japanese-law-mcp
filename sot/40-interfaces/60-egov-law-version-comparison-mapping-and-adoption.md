# SOT-IF-060: e-Gov 法令版間比較のマッピングと組込み採用

- 状態: 有効

## 規定

e-Gov 法令 API Version 2 の
`GET /law_data/{law_id_or_num_or_revision_id}` を二版に対して順に使用し、
`law.version.compare@1` へ対応させ、無設定時の法令コアで
`compare_law_versions` として公開する。

## 外部リクエスト

比較前後の各版は `SOT-IF-011` と同じ XML の法令本文取得 API を使用する。

- `revisionId` は一つの path segment として percent-encoding して設定する。
- `asOf` は入力 `lawId` を path、`asof` を query へ設定する。
- `response_format=xml` と `law_full_text_format=xml` を指定し、`elm`、
  添付内容又は省略条件を追加しない。

接続 origin、base path、redirect 拒否、proxy、再試行及び開始間隔は
`SOT-IF-004` の `law-data-xml` を使用する。一回の比較では `egov-http` の
一枠を前後の取得と比較が完了するまで保持し、二回を並列化せず、二回目の開始を
一回目から一秒以上空ける。一方が失敗した場合は他方だけを返さない。

## 版確定と同一法令

各応答は `SOT-IF-011` の必須構造、法令 ID、法令履歴 ID、法令名及び XML 本文の
検証を満たす。選択した二版の `lawId` は入力 `lawId` と一致させる。

`revisionId` で取得した応答が別の法令 ID を示す場合は、指定した法令と版の組が
存在しないものとして `not_found` とする。`lawId + asOf` の応答が別の法令 ID
を示す場合、又は要求した `revisionId` と返却版 ID が一致しない場合は
`invalid_source_response` とする。比較前後が同じ履歴 ID へ解決されても成功とする。

## 対象条の選択

各版の `Law > LawBody` から次の `Article` だけを対象にする。

- `MainProvision` の下で、`Part`、`Chapter`、`Section`、
  `Subsection` 又は `Division` だけを経由して到達する条
- `AmendLawNum` 属性を持たない `SupplProvision` の下で、同じ構造要素だけを
  経由して到達する条

別の `Article`、`Paragraph`、表、引用構造、`AmendProvision`、
`NewProvision`、別表又は改正法附則の内側にある `Article` は独立した比較対象に
しない。対象の単一条の `Article.Num` は `SOT-MODEL-018` の正規形へ対応させる。
一つの `Article` が `38:84` のような連続する削除条の範囲を示す場合は、両端を
同じ正規形として検証し、`SOT-MODEL-033` の `start:end` へ対応させる。範囲は
元の一つの `Article` に対応する一つの比較対象として保持し、個別条へ展開しない。
空の端点、複数の `:`、同一の端点又は逆順の範囲は `invalid_source_response` とする。
一つの版で同じ `provision + articleNumber` が複数になる場合は
`invalid_source_response` とする。

`Part`、`Chapter`、`Section`、`Subsection` 及び `Division` の
`Num` は存在する場合だけ補助位置へ保持する。条の同一性には含めない。
`ArticleTitle` と `ArticleCaption` はそれぞれ別の項目へ保持し、相互に
代用しない。

## 比較用文字列と構造

条の `text` は、対象 `Article` 内の文字データを文書順に集め、各 Unicode
空白列を一つの U+0020 へ変換し、先頭と末尾の空白を除いて作る。空でない文字列を
別要素から補わない。改行、indentation 又は空白量だけの差は `text` の変更に
しない。Unicode 正規化、表記変換、形態素解析、同義語置換又は誤字補正は行わない。

文字列以外の構造比較では、対象 `Article` 内の開始要素と終了要素の順序、
要素名、及び名前順に整列した属性名と属性値を使用する。文字データ、namespace
宣言、属性順及び XML formatting は構造値に含めない。未知の namespace、
directive、DTD、外部 entity 又は安全に正規化できない構造を成功へ変換しない。

## 対応付けと分類

条は `provision + articleNumber` で対応付ける。

- 比較後版にだけ存在する条は `added`
- 比較前版にだけ存在する条は `removed`
- 両版に存在し、補助位置、比較用文字列又は構造の一つ以上が異なる条は
  `modified`
- すべて同じ条は `unchangedCount` に数え、`items` へ含めない

`provision` 又は `articleNumber` が変わった条を同一条の移動とは推測せず、
それぞれ `removed` と `added` に分類する。

`modified.changeReasons` は差があるものだけを `location`、`text`、
`structure` の順に設定する。比較後に存在する変更は比較後の文書順、その後の
`removed` は比較前の文書順で返す。

条ごとの `Citation.location` は
`{main|supplementary}:article={articleNumber}` とし、URL、法令 ID と履歴 ID は
対応する版に合わせる。版全体の `Citation` も前後それぞれ保持する。

## 出典

結果の `ref` は比較後版を指す。最後の `Provenance` は、比較後版の key を
`resourceKey`、比較前後の版 key をこの順の `inputKeys`、
`transformation: derived`、`methodId: SOT-IF-060`、
`mediaType: application/xml` とする。取得した二つの endpoint と時刻を
それ以前の provenance で保持し、比較結果を e-Gov が直接提供した差分と表示しない。

## 資源予算と契約変更

各本文取得は `SOT-IF-004` の `law-data-xml` 予算に従う。比較処理には追加で、
各版 10000 条、二版の比較用文字列合計 8 MiB、変更 10000 件、比較処理 3 秒及び
JSON 化した成功結果 12 MiB を上限とする。一単位でも超えた場合は部分結果へ
切り詰めず `source_processing_limit` とする。

公式 OpenAPI `2.1.139` の endpoint、XML media type と版選択条件、及び公式の
法令標準 XML の `MainProvision`、`SupplProvision`、`AmendLawNum`、
`Article`、単一条と削除条範囲の `Num`、及び階層要素を契約 fixture で確認する。
保存済み公式契約との不一致は
`source_contract_changed`、個別 runtime 応答又は条同一性の不正は
`invalid_source_response` とする。

## 組込み採用

`e-gov-law-api-v2` の descriptor は `adapterContractVersion: 1.2.0`、
`verifiedAt: 2026-08-23` とし、既存五能力に
`law.version.compare@1` を加えた六つの compiled binding を持つ。

```yaml
providerRoutes:
  law.version.compare@1:
    selection: primary
    defaultProviderId: e-gov-law-api-v2
```

composition root はこの route から `compare_law_versions` service を構成し、
既定の stdio と Streamable HTTP の両方へ同じ schema で登録する。これにより
法令コアは八ツール、`judicial-cases` 有効時は十ツールとなる。統合法情報照会の
profile、辞書、候補、評価 corpus、既存 route、pack 条件及び provider setting は
変更しない。

## 確認

`revisionId` と `asOf`、同版比較、追加、削除、位置だけの変更、文字列変更、
構造だけの変更、本則と原始附則、改正法附則の除外、nested Article の除外、
重複同一性、空文字の条、空白差、順序、全件数、各 Citation、derived provenance、
二回取得の開始間隔、同時実行、単一条と削除条範囲の共存、範囲の一単位保持、
全資源上限、error normalization、descriptor、
binding、既定 route、八・十ツール、両 transport の schema 及び MCP smoke test を
fixture と fake transport で確認する。

## 関連

- [SOT-SCN-013: 法令の二つの版を比較する](../10-scenarios/13-compare-law-versions.md)
- [SOT-MODEL-033: LawVersionComparison](../20-model/33-law-version-comparison.md)
- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-IF-011: e-Gov 法令本文取得マッピング](11-egov-law-document-mapping.md)
- [SOT-IF-026: プロバイダールーティング設定](26-provider-routing-configuration.md)
- [SOT-IF-058: `law.version.compare` capability v1](58-law-version-compare-capability.md)
- [SOT-IF-059: MCP `compare_law_versions`](59-mcp-compare-law-versions.md)
- [法令 API Version 2 OpenAPI](https://laws.e-gov.go.jp/api/2/swagger-ui/lawapi-v2.yaml)
- [法令標準 XML スキーマ](https://laws.e-gov.go.jp/docs/law-data-basic/419a603-xml-schema-for-japanese-law/)
