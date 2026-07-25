# SOT-DEL-012: ローカル実行経路

- 状態: 有効

## 規定

すべての公式提供形態は Japanese Law MCP の process を利用者のローカル環境で実行し、法情報リクエストをその process から公式情報源へ直接送る。

## 経路

| 提供形態 | MCP の実行場所 | 法情報リクエストの経路 |
|---|---|---|
| 実行ファイルまたはパッケージ | 利用者の環境 | MCP クライアント → ローカル process → 公式情報源 |
| MCP クライアント向け設定またはプラグイン | 利用者の環境 | MCP クライアント → 設定が起動したローカル process → 公式情報源 |
| ローカル Streamable HTTP | 利用者の環境 | 同じ host の MCP クライアント → loopback → ローカル process → 公式情報源 |

既定の実行方式は、MCP クライアントが子 process として起動する stdio とする。Streamable HTTP は同じ host の複数クライアントが必要な場合の loopback 補助方式に限る。

非 loopback の待受け、別 host のクライアント、自己運用コンテナ、プロジェクト管理の中継先および管理された MCP エンドポイントを公式実行経路に含めない。

利用者の request、response、検索語、認証情報および診断情報は `SOT-ARCH-005` に従い、現在のローカル request の処理中だけ扱う。

## 確認

公式のインストール手順と接続例に、ローカル process、transport および接続先を記載する。stdio と loopback HTTP について、プロジェクト管理の中継先へ接続せず、設定した公式情報源だけへ outbound 接続することを検証する。

## 関連

- [SOT-ARCH-005: リクエスト情報の一時性](../30-architecture/05-ephemeral-request-lifecycle.md)
- [SOT-DEL-001: stdio](01-stdio.md)
- [SOT-DEL-011: ローカル公式配布物](11-local-distributions.md)
- [SOT-DEL-013: ローカル Streamable HTTP](13-local-streamable-http.md)
