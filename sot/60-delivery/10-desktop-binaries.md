# SOT-DEL-010: デスクトップ向け実行ファイル

- 状態: 有効

## 規定

公式リリースは、対象 OS と CPU アーキテクチャの組み合わせごとに、単体で起動できる個別の実行ファイルを提供する。

## 対象

| オペレーティングシステム | アーキテクチャ | 配布形式 |
|---|---|---|
| macOS | Apple Silicon (`arm64`) | `.tar.gz` |
| macOS | Intel (`amd64`) | `.tar.gz` |
| Windows | `amd64` | `.zip` |
| Windows | `arm64` | `.zip` |

Windows 用のアーカイブには `japanese-law-mcp.exe`、macOS 用のアーカイブには `japanese-law-mcp` を含める。各アーカイブに含める実行ファイルは一つだけとする。

アーカイブ名は `japanese-law-mcp_{version}_{os}_{arch}` を基準とし、対象のバージョン、OS およびアーキテクチャをファイル名だけで識別できるようにする。

実行に Go のツールチェーン、パッケージマネージャーまたは別の言語ランタイムを要求しない。

## 成果物

各アーカイブに SHA-256 チェックサムを付与し、リリースバージョン、対象 OS、対象アーキテクチャおよび生成元コミットを対応付ける。

すべての対象環境で、同じ MCP プロトコルバージョン、ツール一覧および結果契約を提供する。

## 確認

各対象環境の新しい利用者プロファイルで、アーカイブの展開、バージョン表示、stdio 起動、MCP 初期化および公式ツールの呼び出しを確認する。

## 関連

- [SOT-DEL-001: stdio](01-stdio.md)
- [SOT-DEL-011: ローカル公式配布物](11-local-distributions.md)
- [SOT-DEL-012: ローカル実行経路](12-local-execution-paths.md)
- [SOT-DEL-004: リリース整合性](04-release-consistency.md)
