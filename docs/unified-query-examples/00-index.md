# 統合照会の検索例

このディレクトリは、現行の `query_legal_information` を固定評価 fixture で
確認できる代表例だけを案内する。定義元は SOT と評価 fixture であり、ここは
人が読みやすく確認するための派生成果物である。

- `catalogVersion`: `unified-query-examples-v1`
- `corpusVersion`: `corpus-v9`
- `baselineVersion`: `default-1`

採用済みでも未実装の挙動、`corpus-v10`、`default-2` および将来の test ID は
掲載しない。実装との差分は [実装状況](../../wiki/10-implementation-status.md)、
カタログの契約は
[SOT-ENG-029](../../sot/50-engineering/29-unified-query-example-catalog.md) を参照する。

現行カタログの機械的な照合元は、
[corpus-v9 manifest](../../testdata/legalquery/corpus-v9/manifest.json) と
[default-1 baseline](../../testdata/legalquery/baselines/default.json) である。

## 読み方

- `example_id`: カタログ内で一意な安定 ID
- `example_kind`: 意味計画を確認する `semantic`、または公開 status まで確認する
  `execution`
- `query`: 対応する semantic fixture の `request.query` と完全に同じ照会
- `request_context`: fixture の pack 状態、`ref` および `limitPerAttempt`
- `verification_artifact`: query、context および期待値を確認する exact fixture
- `expected_plan_decision`: semantic fixture の decision
- `expected_public_status`: execution fixture または非実行 plan で確認できる status。
  実行可能な semantic 例だけの場合は `—`
- `expected_summary`: interpretation、step または非実行理由の要点
- `related_sots`: 挙動の定義元

## `verification_artifact` の解決

`verification_artifact` は
`{corpusVersion}:{artifactKind}:{caseId}` の三部分で表す。任意 path として
解釈せず、次の規則で現行 corpus manifest から解決する。

- `semantic` は manifest の `development` と `holdout` から `caseId` が一致する
  一件だけを探し、`{set}/{caseId}.json` を照合する。零件または複数件なら
  カタログ不整合とする
- `execution` は manifest の `execution` から `caseId` が一致する一件を探し、
  `execution/{caseId}.json` を照合する。query、request context および plan
  decision は、その fixture の `semanticCaseId` が指す development fixture から、
  公開 status は execution fixture の期待値から確認する
- prefix の `corpusVersion` は、この index の `corpusVersion` および manifest の
  `corpusVersion` と一致させる。別 corpus の同名 case へ fallback しない
- baseline の `corpusVersion` と `baselineVersion` はこの index の宣言、
  `holdoutDigest` は corpus manifest と一致させる。採用 manifest の初回導入後は、
  baseline の profile set と version も current adoption tuple と一致させる

## 章

- [実行される代表例](10-execution.md): 実行可能な意味計画と
  `completed`、`empty`、`partial`
- [明確化と対象外の代表例](20-clarification-and-unsupported.md):
  `needs_clarification`、`capability_unavailable`、`unsupported`
