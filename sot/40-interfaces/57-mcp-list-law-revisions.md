# SOT-IF-057: MCP `list_law_revisions`

- 状態: 有効

## 規定

`list_law_revisions` は、一つの法令 ID または法令番号を受け取り、その法令の完全な改正履歴を共通化した項目と正確な総件数で返す MCP ツールとする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `lawIdOrNumber` | string | はい | 端の U+0020 を除いた後に一文字以上、UTF-8 で 256 byte 以下、ASCII 制御文字禁止 | 法令 ID または法令番号 |

欠落、`null`、形式不正および定義していない入力項目は受け付けず、外部情報源を呼び出す前に `invalid_argument` とする。

## 出力

成功時の `structuredContent` は次の JSON object とする。

| 名前 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `lawId` | string | はい | 返した履歴が属する法令 ID |
| `totalCount` | integer | はい | `items` の正確な件数 |
| `items` | `LawRevision[]` | はい | 新しい履歴から順に並べた完全な改正履歴 |

`items` は `SOT-MODEL-032` の JSON 表現を変更せず使用する。省略可能な値がない項目は省略し、`null` 又は推測値で補わない。明示された boolean の `false` は保持する。

## 共通 capability からの投影

入力の `lawIdOrNumber` を `law.revision.list@1` の型付き入力へ渡す。primary route から返された `SourcedResource<LawRevision>` を共通契約に従って検証した後、`data` だけを `items` へ投影する。

内部の `ref`、`provenance`、`SourcePage`、provider 固有 DTO および継続位置は公開結果へ含めない。

## 件数、順序および該当なし

- 出力の `lawId` はすべての `items[].lawId` と一致させる。
- `totalCount` は `items` の配列長と一致する 0 以上の正確な件数とする。
- `items` は情報源が返した新しい履歴からの順序を保持する。
- 対象法令が存在し履歴がない場合は、`totalCount: 0` と `items: []` を持つ成功結果とする。
- 対象法令が存在しない場合は `not_found` とする。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 該当する法令がない場合は `not_found` を返す。
- route、設定又は認証の失敗は、原因に応じて `unsupported_capability`、`configuration_required` 又は `source_auth_failed` を返す。
- 外部情報源の制限、一時障害又はローカル同時実行上限は、原因に応じて `rate_limited`、`source_timeout`、`source_unavailable` 又は `source_busy` を返す。
- 外部契約、応答又は安全上限の問題は、原因に応じて `source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` 又は `unsafe_source_content` を返す。
- 上記へ分類できない内部処理の失敗は `internal_error` を返す。

情報源エラーの公開値は `SOT-IF-027` に従い、別のコードへ縮約しない。

## 確認

少なくとも次を契約テストで確認する。

- 有効な `lawIdOrNumber` だけを受理し、不正な入力では情報源を呼び出さないこと
- `lawId`、`totalCount`、`items` の順序および全公開項目が内部能力結果と一致すること
- 空結果を空でない配列や `not_found` に変換しないこと
- `ref`、`provenance`、内部ページおよび provider 固有値を公開しないこと
- `not_found` と各情報源エラーを保持すること

## 関連

- [SOT-SCN-012: 法令の改正履歴を取得する](../10-scenarios/12-list-law-revisions.md)
- [SOT-MODEL-032: LawRevision](../20-model/32-law-revision.md)
- [SOT-IF-007: MCP ツール結果](07-mcp-tool-result.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-055: `law.revision.list` capability v1](55-law-revision-list-capability.md)
