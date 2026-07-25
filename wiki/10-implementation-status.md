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

## 未実装

- MCP ツールとユースケース
- e-Gov 法令 API Version 2 アダプター
- [SOT-IF-014](../sot/40-interfaces/14-provider-descriptor.md) から [SOT-IF-029](../sot/40-interfaces/29-local-runtime-configuration.md) に定義した provider registry、capability、route、継続取得および provider 設定
- [SOT-ENG-017](../sot/50-engineering/17-provider-conformance-matrix.md) と [SOT-ENG-018](../sot/50-engineering/18-provider-onboarding-fitness-gate.md) の機械検証
- loopback 限定の Streamable HTTP トランスポートとリソース制限
- ローカル公式配布物を生成するリリース処理

引数を指定しないルートコマンドは stdio MCP サーバーを起動する。公開ツールはまだ登録されていないため、初期ツール一覧は空配列となる。Streamable HTTP を指定した場合は、未実装であることを示す終了コード `1` を返す。

現在の CLI 設定実装は、provider routing と credential environment reference をまだ扱わない。設計上の現在の定義元は [SOT-IF-026](../sot/40-interfaces/26-provider-routing-configuration.md) と [SOT-IF-029](../sot/40-interfaces/29-local-runtime-configuration.md) であり、実装開始後にこの差分を解消する。
