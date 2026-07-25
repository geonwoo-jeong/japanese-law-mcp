# SOT-ARCH-001: リクエスト処理パイプライン

- 状態: 有効

## 規定

すべてのリクエストは、MCP トランスポート、MCP ツール、ユースケース、情報源ポートおよび情報源アダプターの順に処理し、逆方向に結果を返す単方向のパイプラインとして構成する。

## 構造

```mermaid
flowchart LR
    Client["MCP クライアント"]
    Transport["MCP トランスポート"]
    Tool["MCP ツール"]
    Application["ユースケース"]
    Port["情報源ポート"]
    Source["情報源アダプター"]
    Provider["公式情報源"]

    Client --> Transport
    Transport --> Tool
    Tool --> Application
    Application --> Port
    Port --> Source
    Source --> Provider
```

各境界は、内側の処理を呼び出すために必要な値だけを渡す。外部形式を複数の境界へ伝播させない。

## 確認

一つのリクエストについて、受信、ユースケースの実行、情報源の呼び出し、結果返却の経路をコード上で追跡できることを確認する。

## 関連

- [SOT-ARCH-002: MCP トランスポート境界](02-transport-boundary.md)
- [SOT-ARCH-003: ユースケース境界](03-application-boundary.md)
- [SOT-ARCH-004: 情報源アダプター境界](04-source-adapter-boundary.md)
- [SOT-ARCH-006: MCP ツール境界](06-mcp-tool-boundary.md)
