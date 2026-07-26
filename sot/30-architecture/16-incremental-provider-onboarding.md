# SOT-ARCH-016: プロバイダーの段階的な追加

- 状態: 有効

## 規定

新しいプロバイダーは、既存の共通能力、情報モデルおよび他のプロバイダーアダプターを変更せずに追加できる独立した縦の変更単位とし、同じ意味を確認できた能力だけを既存の共通 capability へ binding する。

## 追加単位

一つのプロバイダーを実装対象へ加える変更は、少なくとも次をそのプロバイダーの単位で所有する。

1. 公式仕様、採用範囲、接続条件および確認日を定義するプロバイダー固有のインターフェース SOT
2. 固定した `ProviderDescriptor`、credential slot および provider configuration scope
3. 実装する各 capability と外部 operation の mapping SOT
4. 独立したプロバイダーパッケージ、外部 DTO、client、parser、mapper および error normalization
5. 公式 fixture、golden test、資源予算および conformance matrix の row
6. composition root の登録と、採用した capability に限る明示的な route

既存の共通 capability を実装するだけの追加では、共通 capability の入力、出力、欠落、エラー、継続取得および出典の意味を変更しない。別のプロバイダーのパッケージ、fixture、parser または mapping を変更しない。

## 共通化の判断

次のすべてが一致する場合だけ、既存の共通 capability へ binding する。

- 利用者の目的
- 入力項目の意味と検証条件
- 出力項目の意味と必須性
- 空結果、欠落、曖昧性および `not_found` の区別
- 時点、版および識別子の意味
- 継続取得と並び順の保証
- 到達し得る失敗と公開エラーへの対応
- `SourceResourceKey` と `Provenance` を作れる根拠

外部フィールド名が似ていること、同じ文字列またはファイル形式を返すこと、同じ提供主体であることだけを意味の一致としない。

既存の共通 capability より狭い機能しか持たない場合は、空値、推測値または暗黙の追加取得で不足を隠さない。より小さい別の共通 capability または `provider.{providerId}.{operation}` の provider-specific capability として定義する。

一つのプロバイダーだけが実装する capability も、外部 DTO や検索文法を漏らさず独立した利用目的と型付き契約を定義できる場合は、将来の共通 capability として採用できる。後から追加するプロバイダーは、その既存契約へ完全に適合できる場合だけ同じ binding を使用する。

## 変更境界

新しいプロバイダーの追加によって変更してよい共通箇所は、composition root への登録、利用者が明示した場合だけ使用する route の受付、および conformance matrix の列挙に限る。既存 capability の組込み既定 provider、組込み route または無設定時の有効 provider 集合を、プロバイダー追加と同じ変更で変えない。

既存 capability の組込み既定 provider、組込み route または無設定時の有効 provider 集合を変える場合は、プロバイダー追加から分離した SOT 変更として、既存利用者の結果、設定、継続トークンおよび rollback への影響を先に採用する。

既存の共通モデルまたは capability を拡張する必要がある場合は、プロバイダー追加と同じ変更へ暗黙に混ぜず、互換性と全既存 binding への影響を独立した SOT 変更として先に判断する。

optional field の追加であっても、既存フィールドと重複する別名、特定プロバイダーだけの生 field、無型の `extensions` object または `map[string]any` を共通モデルへ追加しない。

## 確認

共通 capability ごとに、`SOT-ENG-018` の `provider-onboarding-fit` で最小の test provider を同じ conformance suite へ binding し、共通モデル、能力別 port および既存プロバイダーパッケージを変更せず登録、route、成功、空結果、失敗および出典を検証できることを確認する。

新しいプロバイダーの変更では、該当パッケージと composition root 以外の既存プロバイダーパッケージに差分がないこと、および既存 provider の conformance suite がそのまま成功することを検証する。

## 関連

- [SOT-ARCH-017: 採用可能な能力群](17-approved-capability-families.md)
- [SOT-ARCH-010: プロバイダーの分離](10-provider-isolation.md)
- [SOT-ARCH-012: プロバイダーの登録](12-provider-registry.md)
- [SOT-MODEL-013: ProviderCapability](../20-model/13-provider-capability.md)
- [SOT-ENG-012: プロバイダーパッケージ構成](../50-engineering/12-provider-package-layout.md)
- [SOT-ENG-017: プロバイダー適合性 matrix](../50-engineering/17-provider-conformance-matrix.md)
- [SOT-ENG-018: プロバイダー追加 fitness gate](../50-engineering/18-provider-onboarding-fitness-gate.md)
