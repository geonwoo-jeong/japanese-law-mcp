# Japanese Law MCP v0.0.0 <!-- x-release-please-version -->

## 提供する SOT

- `SOT-PROD-008`: e-Gov 法令コア
- `SOT-PROD-010`: 裁判例拡張パック
- `SOT-PROD-017`: 段階的に利用する簡潔な MCP 公開面
- `SOT-SCN-017`: 専門操作を発見して実行する利用シナリオ
- `SOT-ARCH-021`: プロバイダー非依存の法令名検索語前処理
- `SOT-ARCH-044`: MCP 公開ツールと専門操作 registry の境界
- `SOT-SCN-016`: 上限と省略件数を伴う法令更新一覧
- `SOT-MODEL-033`: 公式の削除条範囲を保持する法令版間比較
- `SOT-IF-035`: 多数項目を安全に処理する e-Gov Version 1 更新一覧
- `SOT-IF-060`: e-Gov 法令版間比較のマッピング
- `SOT-IF-076`: 返却件数と省略件数を明示する `list_law_updates` v2
- `SOT-IF-077`: `compact` と `full` の MCP ツール公開方式および拡張パック有効化
- `SOT-DEL-010`: macOS と Windows のデスクトップ向け実行ファイル
- `SOT-DEL-013`: loopback 限定の Streamable HTTP
- `SOT-DEL-014`: Release Please による検証付き公式リリース

## 未実装の SOT 差分

- [SOT-ENG-024](../sot/50-engineering/24-unified-query-evaluation-gate.md) と
  [SOT-ENG-025](../sot/50-engineering/25-unified-query-package-layout.md) が定める、
  統一評価 command、baseline および中央品質ゲートへの評価接続は未実装。

## 互換性のない変更

- `SOT-IF-077` に従い、無設定時に直接公開する MCP ツールを法令コアの八つから
  `discover_legal_tools`、`execute_legal_tool` および `query_legal_information` の
  三つへ変更する。全拡張パック有効時も十二ツールから三ツールへ減る。従来の専門名を
  直接呼び出すクライアントは、発見した操作を `execute_legal_tool` で実行するか、
  設定ファイルへ `toolExposure: full` を明示して移行する。
- `SOT-IF-076` に従い、`list_law_updates` は完全一覧の返却をやめ、既定で先頭 50 件までを返す。成功時の
  出力には `returnedCount`、`omittedCount` および `truncated` を追加し、
  必要な場合だけ `limit` で 1 以上 512 以下の返却上限を指定する。
