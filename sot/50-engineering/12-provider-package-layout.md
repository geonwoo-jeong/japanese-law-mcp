# SOT-ENG-012: プロバイダーパッケージ構成

- 状態: 有効

## 規定

プロバイダー固有の実装は、外部サービスと互換性を持たない版ごとの独立した Go パッケージに置き、各パッケージ内でも責務ごとに小さなファイルへ分ける。

## 基準構成

```text
internal/source/
├── shared/
├── egov/
│   ├── lawv2/
│   └── lawv1/
├── courts/
│   └── hanrei/
├── ndl/
│   └── kokkai/
├── shugiin/
│   └── bills/
├── sangiin/
│   └── bills/
├── nta/
│   └── circulars/
├── kfs/
│   └── decisions/
├── mhlw/
│   └── guidance/
├── jaish/
│   └── guidance/
```

この構成は、`SOT-PROD-008` と `SOT-PROD-009` が採用を許可する法情報源の接続境界だけを示し、各情報源を公開機能として採用したことを意味しない。

## パッケージの責務

各プロバイダーパッケージは、必要に応じて次のファイルを独立して持つ。

- `adapter.go`: 能力別ポートの実装と外部公開しない組立て
- `client.go`: リクエスト生成、認証および HTTP 呼出し
- `request.go` と `response.go`: 外部仕様の DTO
- `parser.go`: XML、HTML、CSV、PDF、GeoJSON、PBF、ZIP または XBRL の解析
- `mapper.go`: 能力別モデルへの変換
- `errors.go`: 外部エラーの正規化
- `fixtures/` と `_test.go`: 公式応答 fixture、golden test および構造変更の検出

一つのファイルへプロバイダー全体を集約することを要求しない。プロバイダーごとの独立パッケージを変更影響の境界とし、ファイルは 200 行から 400 行を目安に単一の責務へ分け、800 行を超えない。

`shared` には情報の意味を持たない HTTP、安全な展開、文字コード変換および上限検査だけを置く。プロバイダー固有の DTO、selector、列挙値、ページ規則または mapping を置かない。

各パッケージの実装を開始する前に、その情報源の公式仕様または取得方法、採用範囲、利用条件および実装する能力ごとの mapping を定義する独立した SOT を用意する。`jaish` のような補助情報源は、公式情報源とは別の権威区分と採用理由を個別 SOT に定義しなければならない。

## 関連

- [SOT-ENG-001: Go パッケージ構成](01-go-package-layout.md)
- [SOT-ARCH-010: プロバイダーの分離](../30-architecture/10-provider-isolation.md)
- [SOT-ARCH-017: 採用可能な能力群](../30-architecture/17-approved-capability-families.md)
- [SOT-PROD-009: 選択型法情報拡張パックの境界](../00-product/09-selectable-legal-information-extension-packs.md)
