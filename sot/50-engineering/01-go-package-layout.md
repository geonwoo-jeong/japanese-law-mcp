# SOT-ENG-001: Go パッケージ構成

- 状態: 有効

## 規定

現在の実装は Go モジュールとして構成し、実行入口、MCP トランスポート、MCP ツール、ユースケース、情報モデルおよび情報源アダプターを独立したパッケージ境界に置く。

## 基準構成

```text
cmd/
└── japanese-law-mcp/
    └── main.go

internal/
├── buildinfo/
├── cli/
├── config/
├── transport/
│   ├── stdio/
│   └── http/
├── mcp/
├── application/
├── model/
└── source/
    └── egov/
```

`cmd` は依存関係の組み立てと起動だけを扱う。再利用する処理を `cmd` に置かない。

`buildinfo` は実行ファイルへ埋め込むビルド情報、`cli` はコマンドの構築と実行制御、`config` は起動時設定の読込みと検証を扱う。

`application` はユースケースとユースケースが所有する情報源ポートを含む。`source/egov` はそのポートを実装する。

`internal` のパッケージは、アーキテクチャ SOT に定義した境界へ対応させる。

## 関連

- [SOT-ARCH-001: リクエスト処理パイプライン](../30-architecture/01-request-pipeline.md)
- [SOT-ARCH-003: ユースケース境界](../30-architecture/03-application-boundary.md)
- [SOT-ARCH-007: 依存方向](../30-architecture/07-dependency-direction.md)
- [SOT-ARCH-015: 起動時設定境界](../30-architecture/15-startup-configuration-boundary.md)
- [SOT-ENG-014: CLI 実装境界](14-cli-implementation-boundary.md)
