# SOT-ARCH-012: プロバイダーの登録

- 状態: 有効

## 規定

実行入口は、起動時に構成したプロバイダーを、プロバイダー ID、能力 ID および能力のメジャーバージョンで検証して登録し、アプリケーションへ型付きの能力別ポートを提供する。

## 登録

プロバイダーはコンパイル時に含まれる実装から構成し、初期段階では動的なコード読み込みを行わない。

一つの `ProviderDescriptor` を `providerId` で一意に登録し、その記述子が宣言する複数の能力を `(providerId, capabilityId, majorVersion)` の binding として登録する。同じ `providerId` の異なる記述子、同じ binding の重複、宣言した能力と実装の不一致、および能力の互換性不一致は起動エラーとする。一つのプロバイダーが複数の異なる binding を持つことは許可する。

型付き入出力と検証方法を定義した能力別 SOT が存在しない能力は、binding として登録しない。

`ProviderDescriptor` は登録後に変更しない。外部情報源の一時的な障害を、能力の追加または削除として扱わない。

アプリケーションはレジストリから具象アダプターを取得せず、必要な能力別ポートだけを受け取る。レジストリの実行ハンドラーを無型の関数または値として保持しない。

composition root は、能力 ID とメジャーバージョンごとに、選択方法、既定の `providerId` および集約時の順序付き `providerId` を route として構成する。route が参照する binding の欠落、重複または版の不一致は起動エラーとする。ユースケースは既定プロバイダー ID を直書きせず、この route から能力別ポートを受け取る。

入力に `SourceResourceRef` を持つ能力は、route の既定値ではなく `ref.providerId` の binding を選択し、記述子の `source.id` と `ref.key.sourceId` の一致を外部呼出し前に確認する。該当 binding がない場合は別の provider へ fallback しない。既存の公開 facade が情報源固有の識別子しか持たない場合は、その facade の mapping SOT が primary route から `SourceResourceRef` を組み立てる。

## 構成状態

任意のプロバイダーに必要な認証情報がない場合は、そのプロバイダーを `misconfigured` と判別できる状態にし、ネットワーク呼び出し前に失敗させる。既存の公開機能に必要な既定プロバイダーが構成できない場合は起動を失敗させる。

構成状態は起動時または現在のリクエスト内で評価し、状態履歴を保存しない。

## 関連

- [SOT-IF-014: ProviderDescriptor](../40-interfaces/14-provider-descriptor.md)
- [SOT-IF-018: プロバイダー設定境界](../40-interfaces/18-provider-configuration.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-ARCH-007: 依存方向](07-dependency-direction.md)
