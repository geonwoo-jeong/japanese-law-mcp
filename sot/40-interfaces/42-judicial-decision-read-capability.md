# SOT-IF-042: `judicial-decision.read` capability v1

- 状態: 有効

## 規定

`judicial-decision.read@1` は、検索結果から受け取った同一 provider の `SourceResourceRef` を使い、公式詳細ページの裁判例情報を取得する読取り専用の型付き capability とする。

## 能力識別子

| 項目 | 値 |
|---|---|
| `ProviderCapability.id` | `judicial-decision.read` |
| `ProviderCapability.majorVersion` | `1` |
| `ProviderCapability.level` | `extended` |
| `ProviderCapability.stability` | `stable` |

## 型付き入力

`JudicialDecisionReadRequestV1` は、必須の `ref: SourceResourceRef` だけを持つ。

- `ref.key.resourceType` は `judicial-decision` とする。
- `ref.key.versionId` は使用しない。
- provider、source および resource ID は検索結果から変更しない。
- 未知の provider、provider と source の不一致、空の識別子、異なる resource type または version は、外部呼出し前に `invalid_argument` とする。
- 登録済みだが無効な provider は `configuration_required`、対象 capability を持たない provider は `unsupported_capability` とし、別 provider へ fallback しない。

## 型付き出力

成功時は `SourcedResource<JudicialDecisionDetails>` を一件返す。

- 出力 `ref` は入力 `ref` と同じ値とする。
- `data.summary.source.id`、`ref.key.sourceId` および全 `provenance[].source.id` は一致させる。
- 最後の provenance の `resourceKey` は入力 `ref.key` と一致させる。
- 公式詳細ページに存在しない省略可能項目を補完しない。

正確な参照に対応する公式詳細ページが存在しない場合は `not_found` とする。PDF のリンクだけがある本文または要旨を文字列本文へ変換しない。

## ポートと失敗

能力別ポートは `Read(context.Context, Request) (SourcedResource<JudicialDecisionDetails>, error)` とし、外部 HTML または provider 固有 DTO を公開しない。

到達し得る失敗は `invalid_argument`、`not_found`、`unsupported_capability`、`configuration_required`、`source_auth_failed`、`rate_limited`、`source_timeout`、`source_unavailable`、`source_busy`、`source_contract_changed`、`invalid_source_response`、`source_response_too_large`、`source_processing_limit` および `unsafe_source_content` とする。

## 確認

検索から詳細への参照往復、参照の各不一致、無効 provider、別 provider への fallback 禁止、欠落可能項目、404 および情報源エラーを契約テストで確認する。

## 関連

- [SOT-SCN-007: 公表裁判例の詳細を取得する](../10-scenarios/07-get-judicial-case.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-MODEL-021: JudicialDecisionDetails](../20-model/21-judicial-decision-details.md)
- [SOT-IF-048: MCP `get_judicial_case`](48-mcp-get-judicial-case.md)
