# SOT-PROD-008: e-Gov 法令コアの製品範囲

- 状態: 有効

## 規定

Japanese Law MCP の既定の製品範囲は、e-Gov の公式法令情報を根拠として、法令名検索、法令本文検索、法令本文取得、条文取得、法令更新一覧、法令改正履歴一覧および各結果の出典を提供する法令コアとする。

## コアの境界

法令コアは、次の採用済み能力と、それらを公開する MCP ツールで構成する。

| 能力 | 利用目的 |
|---|---|
| `law.search@1` | 法令名と条件から法令を探す |
| `law.revision.list@1` | 一つの法令の改正履歴を確認する |
| `law.content.search@1` | 法令本文の一致位置を探す |
| `law.document.read@1` | 一つの法令本文を取得する |
| `law.article.read@1` | 一つの法令内の条文を取得する |
| `law.update.list@1` | 指定日の法令更新一覧を取得する |

個別能力の入力、出力、情報源および外部仕様との対応は、それぞれの有効なシナリオ、モデルおよびインターフェース SOT を定義元とする。この規定だけから、差分、添付資料その他の未採用能力を公開しない。

法的判断、法的助言、情報源で確認できない内容の補完、および `SOT-PROD-009` に定める選択型拡張パックは、既定の法令コアに含めない。

## 確認

既定設定で公開する各 MCP ツールが、表に示した能力の有効なシナリオ、モデル、情報源 mapping および検証方法へ到達できることを確認する。

## 関連

- [SOT-PROD-001: 製品定義](01-product-definition.md)
- [SOT-PROD-004: 機能の採用条件](04-feature-adoption.md)
- [SOT-PROD-009: 選択型法情報拡張パックの境界](09-selectable-legal-information-extension-packs.md)
- [SOT-ARCH-017: 採用可能な能力群](../30-architecture/17-approved-capability-families.md)
- [SOT-IF-004: e-Gov 法令 API Version 2](../40-interfaces/04-source-egov-law-api-v2.md)
- [SOT-IF-035: e-Gov 法令 API Version 1](../40-interfaces/35-source-egov-law-api-v1.md)
- [利用シナリオ SOT](../10-scenarios/00-index.md)
