# SOT-ARCH-011: 情報源データの正規化

- 状態: 廃止
- 後継: [SOT-ARCH-018: 拡張パック単位の正規化境界](18-pack-scoped-normalization-boundary.md)

## 規定

情報源データは、意味が一致する最小の共通メタデータと能力別の型付きモデルへ変換し、意味が異なる項目を同じ共通キーへ平坦化しない。

## 廃止理由

製品範囲に含めない統計、空間情報および XBRL まで正規化対象として規定しているため、この規定を `SOT-ARCH-018` に置き換える。

## 共通化する範囲

次の項目だけをプロバイダー横断の共通語彙とする。

- `InformationSource`
- `SourceResourceKey`
- `SourceResourceRef`
- `Provenance`
- `ProviderCapability`
- `SourcePage`
- 共通の情報源エラー分類

法令能力の意味と項目は、安定提供される e-Gov 法令 API Version 2 の XML、法令識別子およびリビジョンを基準とする。e-Gov 法令 API Version 1 または別の法令情報源は、同じ意味を確認できる項目だけを法令モデルへ対応させる。

## 共通化しない範囲

次のように名称が同じでも意味が異なる項目は、能力別モデルに残す。

- 法令 ID、発言 ID、議案番号、事件番号、統計表 ID、法人番号、登録番号および開示書類 ID
- 公布日、施行日、開催日、裁決日、観測期間、変更日、有効判定日および提出日
- 法令の状態、審議状況、法人の変更状態、登録状態および開示状態
- 法令 XML、発言本文、HTML、PDF、統計の多次元値、地理形状および XBRL fact

統計では次元、コード、ラベル、単位、観測期間、注記および丸めを保持する。空間情報では geometry、座標系およびタイル文脈を保持する。XBRL では QName、context、unit、period、entity、decimals および taxonomy の版を保持する。

共通モデルへ対応できない値を無制限の `map[string]any`、`raw` 文字列または名前空間のない拡張項目へ格納しない。共通の意味が必要になった場合は能力別モデルを追加し、固有の意味に留まる場合はプロバイダー固有能力として定義する。

## 確認

変換前後で公式識別子、コード、単位、版および意味を失っておらず、確認できない値を補完していないことを契約テストで確認する。

## 関連

- [SOT-MODEL-011: SourceResourceKey](../20-model/11-source-resource-key.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-MODEL-012: Provenance](../20-model/12-provenance.md)
- [SOT-ENG-002: 境界型の分離](../50-engineering/02-boundary-types.md)
