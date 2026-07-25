# SOT-ENG-013: プロバイダー契約の検証

- 状態: 有効

## 規定

各プロバイダーは、宣言した能力、公式仕様、外部構造、安全上の境界および能力別モデルへの変換を、共通の適合性テストとプロバイダー固有の契約テストで検証する。

## 共通適合性テスト

- `ProviderDescriptor` の識別子、能力 ID、版および宣言順
- 宣言した能力と実装した型付きポートの一致
- `SourceResourceRef` の provider、情報源および外部識別子の安定性と無変換
- 検索結果の `SourceResourceRef` を読み取り能力へ渡したときの同一 provider・資源・版の roundtrip
- 必須の `Provenance` と変換種別
- 欠落値の推測禁止、空の検索結果と `not_found` の区別
- 継続トークンの往復、条件不一致、改変および期限切れ
- エラー分類、`retryable`、秘密情報と外部本文の非露出
- キャンセル、タイムアウト、応答サイズおよび解析上限
- プロバイダー間 import の禁止

## プロバイダー固有の契約テスト

公式仕様の版または確認日を固定し、公式例または権利上テストに使用できる代表応答を fixture として、リクエスト、解析、mapping およびエラーを golden test で検証する。

HTML のプロバイダーは、文字コード、一覧、詳細、結果なし、欠落項目および代表的な構造変更の fixture を持つ。必須 selector または文書構造が一致しない場合は `source_contract_changed` とする。

宣言された MIME type と実体の不一致を検証する。XML は外部エンティティ、ZIP はパストラバーサル、ファイル数と展開量、gzip は展開量、HTML は実行可能内容、GeoJSON と PBF は座標とサイズ、XBRL は taxonomy、QName、context および unit の境界を検証する。

PDF は、暗号化、埋込みファイル、壊れた xref または object stream、ページ数、object 数、抽出サイズおよび解析時間の上限を検証する。HTML、PDF、XML および XBRL の解析中に外部参照によるネットワーク呼出しが発生しないことを確認する。

外部ネットワークを使う確認は、再現可能な fixture テストを置き換えず、任意の定期確認として分離する。外部サービスの一時障害だけで通常の単体テストを失敗させない。

## 変更検出

外部仕様、公式掲載ページ、利用条件、認証または代表応答の変更を確認した場合は、該当プロバイダーのインターフェース SOT、fixture、parser および mapping だけを更新し、共通能力の意味が変わる場合に限り能力契約の版を変更する。

## 関連

- [SOT-ENG-004: SOT に結び付く検証](04-sot-linked-verification.md)
- [SOT-ENG-011: 変更の検証ゲート](11-verification-gate.md)
- [SOT-IF-017: 情報源エラーの正規化](../40-interfaces/17-source-error-normalization.md)
