# SOT-IF-059: MCP `compare_law_versions`

- 状態: 有効

## 規定

`compare_law_versions` は、一つの法令 ID と比較前後の版指定を受け取り、本則及び
原始附則の条に限定した二版比較を返す MCP ツールとする。

## 入力

入力は次の JSON object とする。

| 名前 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `lawId` | string | はい | 比較対象の法令 ID |
| `before` | object | はい | 比較前版の指定 |
| `after` | object | はい | 比較後版の指定 |

`before` と `after` は同じ構造とし、次の項目だけを許可する。

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `revisionId` | string | 条件付き | 1 文字以上、UTF-8 で 512 byte 以下、端の U+0020 と ASCII 制御文字禁止 | 正確な履歴 ID |
| `asOf` | string | 条件付き | 実在する `YYYY-MM-DD` | 基準日 |

各側は `revisionId` 又は `asOf` のどちらか一方だけを持つ。`lawId` は端の
U+0020 を除いた後に 1 文字以上、UTF-8 で 256 byte 以下、ASCII 制御文字なしと
する。検証後の `lawId` を元の未処理文字列へ戻さない。

欠落、`null`、型不正、未知の root 又は selector 項目、及び selector の欠落又は
相互排他違反は、外部情報源を呼び出す前に `invalid_argument` とする。

## 出力

成功時の `structuredContent` は `SOT-MODEL-033` の JSON 表現を使用する。
`scope` は固定値 `main_and_original_supplementary_articles` とし、対象外の
改正法附則、前文、条外の項、別表、様式及び添付資料を含むと解釈させない。

省略可能な値がない項目は省略し、`null` 又は推測値で補わない。各
`LawVersionArticle.text` は空文字でも省略しない。

## 共通 capability からの投影

primary route から `law.version.compare@1` を選び、`lawId` を
`resource.key.resourceId`、選択した provider の source と `law` を固定値として
`SourceResourceRef` を組み立てる。`before` と `after` は対応する共通 selector
へ渡す。

返された `SourcedResource<LawVersionComparison>` の `ref`、`provenance` と
`data` を検証した後、`data` だけを公開する。provider 固有 DTO、比較途中の
作業状態、原文全体又は内部診断情報を公開結果へ含めない。

## 空結果

同じ版へ解決された場合又は変更がない場合は、`totalCount: 0` と
`items: []` を持つ成功結果とする。`unchangedCount` と前後の対象条数は
実際に比較した件数を保持する。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 対象法令又は指定版が存在しない場合は `not_found` を返す。
- route、問い合わせ範囲、設定又は認証の失敗は、原因に応じて
  `unsupported_capability`、`unsupported_query`、`configuration_required`
  又は `source_auth_failed` を返す。
- 外部情報源の制限、一時障害又はローカル同時実行上限は、原因に応じて
  `rate_limited`、`source_timeout`、`source_unavailable` 又は
  `source_busy` を返す。
- 外部契約、応答又は安全上限の問題は、原因に応じて
  `source_contract_changed`、`invalid_source_response`、
  `source_response_too_large`、`source_processing_limit` 又は
  `unsafe_source_content` を返す。
- 上記へ分類できない内部処理の失敗は `internal_error` を返す。

## 採用境界

この規定だけから組込みツール集合への登録、provider binding、primary route 又は
無設定時の公開を決定しない。それらは provider mapping と組込み採用 SOT で
一つの変更として採用する。

## 確認

`lawId` と各 selector の全境界違反が外部呼出し前に失敗すること、成功結果の
全項目、件数、空文字の条、前後の出典、空比較、provider 固有内部情報の非公開、
及び stdio と Streamable HTTP の schema 一致を契約テストで確認する。

## 関連

- [SOT-PROD-013: 法令版間比較](../00-product/13-law-version-comparison.md)
- [SOT-SCN-013: 法令の二つの版を比較する](../10-scenarios/13-compare-law-versions.md)
- [SOT-MODEL-033: LawVersionComparison](../20-model/33-law-version-comparison.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-058: `law.version.compare` capability v1](58-law-version-compare-capability.md)
