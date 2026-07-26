# 実装状況

この文書は現在の実装状況を示す。理想状態の定義元は [SOT](../sot/00-index.md) とし、この文書を仕様の定義元にはしない。

## 実装済み

- CLI の起動、ヘルプおよびバージョン表示
- フラグ、環境変数および設定ファイルを統合する起動時設定
- 設定値と組合せの検証
- txtar による CLI 契約テスト
- [SOT-IF-013](../sot/40-interfaces/13-mcp-protocol-version.md) に従う MCP `2025-11-25` の初期化と `tools` capability
- [SOT-DEL-001](../sot/60-delivery/01-stdio.md) に従う、ローカル子プロセスとして動作する stdio トランスポート
- MCP クライアントによる初期化とツール一覧取得の契約テスト
- [SOT-ENG-019](../sot/50-engineering/19-static-analysis-and-coding-style.md) と [SOT-ENG-020](../sot/50-engineering/20-verification-gate.md) に従う、バージョン固定した Go リンター、SOT 固有解析器、カバレッジ下限、脆弱性・秘密情報検査および GitHub Actions の共通品質ゲート
- [SOT-ENG-021](../sot/50-engineering/21-git-hook-staged-verification.md) に従う、Git index の `pre-commit` 検査、送信 tip と ref 範囲の `pre-push` 検査、ならびにリポジトリローカルな Git フックの導入・確認・解除
- [SOT-ENG-017](../sot/50-engineering/17-provider-conformance-matrix.md) と [SOT-ENG-018](../sot/50-engineering/18-provider-onboarding-fitness-gate.md) に従う、プロバイダー適合性 matrix の schema・共通 loader、`provider-onboarding-fit`、ならびに `pre-push` と GitHub Actions への先行 gate 接続
- [SOT-MODEL-009](../sot/20-model/09-json-serialization.md)、[SOT-MODEL-010](../sot/20-model/10-information-source.md)、[SOT-MODEL-013](../sot/20-model/13-provider-capability.md) および [SOT-IF-014](../sot/40-interfaces/14-provider-descriptor.md) に従う、不変なプロバイダーメタデータ型、検証および JSON 表現
- [SOT-MODEL-001](../sot/20-model/01-law-summary.md) と [SOT-MODEL-003](../sot/20-model/03-legal-source.md) に従う、`InformationSource` から決定的に投影する `LegalSource` と、不変な `LawSummary`
- [SOT-MODEL-011](../sot/20-model/11-source-resource-key.md) および [SOT-MODEL-016](../sot/20-model/16-source-resource-ref.md) に従う、不変な情報源資源キーとプロバイダー参照の構造検証および JSON 表現
- [SOT-MODEL-012](../sot/20-model/12-provenance.md) に従う、不変な出典、変換種別ごとの従属制約、日時・MIME type・ダイジェストの検証および JSON 表現
- [SOT-MODEL-014](../sot/20-model/14-source-page.md) に従う、不変なページ情報、`totalCount` と `totalRelation` の従属制約、継続トークン省略規則および JSON 表現
- [SOT-IF-015](../sot/40-interfaces/15-source-operation-contract.md) に従う、`SourceResourceRef`、一件以上の `Provenance` および検証可能な型付き data を結び付ける不変な `SourcedResource<T>` と出典経路の整合検証
- [SOT-IF-017](../sot/40-interfaces/17-source-error-normalization.md) に従う、十三分類、固定された安全な日本語メッセージおよび再試行可否を持ち、[SOT-IF-027](../sot/40-interfaces/27-public-source-error-contract.md) が許可する二分類に限って明示された `retryAfter` を保持する不変な `SourceError`
- [SOT-ARCH-012](../sot/30-architecture/12-provider-registry.md) のうち、`providerId` と能力 ID・メジャーバージョンの宣言を起動時に検証して保持し、`SourceResourceRef` の provider と情報源の一致を照合する不変な descriptor registry
- [SOT-IF-016](../sot/40-interfaces/16-source-continuation-contract.md) および [SOT-IF-026](../sot/40-interfaces/26-provider-routing-configuration.md) の構成状態 fingerprint 規定に従う、プロセスローカル鍵、RFC 8785 正規化、条件・構成状態の結合、期限・長さ検証および再起動時無効化を備えた共通 continuation token kernel
- [SOT-IF-022](../sot/40-interfaces/22-law-search-capability.md) に従う、正規化済みの型付き `law.search@1` 入力、継続条件、検索ページおよび能力別ポート

## 未実装

- MCP ツールとユースケース
- e-Gov 法令 API Version 2 アダプター
- e-Gov 法令 API Version 2 の組込み `ProviderDescriptor`、型付き capability binding、route、継続位置・snapshot・sort の provider 固有 mapping と呼出経路への組込み、`SourceResourceRef` と能力・構成状態の照合、および [SOT-IF-015](../sot/40-interfaces/15-source-operation-contract.md) から [SOT-IF-029](../sot/40-interfaces/29-local-runtime-configuration.md) に定義した残りの provider 設定と registry 処理
- loopback 限定の Streamable HTTP トランスポートとリソース制限
- ローカル公式配布物を生成するリリース処理

引数を指定しないルートコマンドは stdio MCP サーバーを起動する。公開ツールはまだ登録されていないため、初期ツール一覧は空配列となる。Streamable HTTP を指定した場合は、未実装であることを示す終了コード `1` を返す。

現在の CLI 設定実装は、provider routing と credential environment reference をまだ扱わない。設計上の現在の定義元は [SOT-IF-026](../sot/40-interfaces/26-provider-routing-configuration.md) と [SOT-IF-029](../sot/40-interfaces/29-local-runtime-configuration.md) であり、実装開始後にこの差分を解消する。
