# SOT-IF-004: e-Gov 法令 API Version 2

- 状態: 有効

## 規定

e-Gov 法令 API Version 2 は、Japanese Law MCP が日本の法令を検索および取得するための一次情報源とし、運用時の接続先と外部フィールドの定義はデジタル庁が公開する公式 OpenAPI 仕様に固定する。

## 公式仕様

- 提供主体: デジタル庁
- 仕様: [e-Gov 法令 API Version 2](https://laws.e-gov.go.jp/api/2/redoc/)
- 確認した仕様バージョン: `2.1.139`
- 確認日: `2026-07-25`
- API ベース URL: `https://laws.e-gov.go.jp/api/2`
- 情報源 ID: `e-gov-law-api-v2`
- 位置付け: 公式情報

## 利用範囲

Japanese Law MCP は、次の利用目的に必要な操作だけを使用する。

| e-Gov operation | 利用目的 | マッピング SOT |
|---|---|---|
| `GET /laws` | 法令名検索 | `SOT-IF-009` |
| `GET /keyword` | 法令本文検索 | `SOT-IF-010` |
| `GET /law_data/{law_id_or_num_or_revision_id}` | 法令本文取得 | `SOT-IF-011` |
| `GET /law_data/{law_id_or_num_or_revision_id}` | 条文取得 | `SOT-IF-012` |

外部 API のリクエスト項目とレスポンス項目は、公式 OpenAPI 仕様を定義元とする。プロジェクト内のモデルへの変換だけを各マッピング SOT で定義する。

## 試行提供機能

確認した仕様では、次の機能が試行提供と明記されている。

- 法令本文取得 API が返す JSON 形式の本文
- 法令本文ファイル取得 API が返す JSON ファイル
- キーワード検索 API で名称に `law_num` を含むパラメータを指定した場合のレスポンス

公式機能として使用する場合は、対応するマッピング SOT に試行提供であることと変更検出方法を明記する。現在の法令本文取得と条文取得は XML を使用し、試行提供の JSON 本文には依存しない。

## 関連

- [SOT-PROD-003: 法情報の採用基準](../00-product/03-legal-source-eligibility.md)
- [SOT-ARCH-004: 情報源アダプター境界](../30-architecture/04-source-adapter-boundary.md)
- [SOT-ENG-005: SOT と変更の整合](../50-engineering/05-sot-change-unit.md)
