# SOT-ENG-018: プロバイダー追加 fitness gate

- 状態: 有効

## 規定

新しいプロバイダーまたは既存プロバイダーへの capability binding を追加する変更は、`provider-onboarding-fit` の機械検証により、既存の共通契約と他のプロバイダーを変更せずに追加できたことを確認する。

## 適用変更

merge base との差分に、次のいずれかが含まれる場合にこの gate を適用する。

- 新しい provider-specific SOT、provider package または `ProviderDescriptor`
- 既存 provider の新しい capability binding
- provider route の受付、provider 設定 schema または conformance matrix row の追加

共通 capability または共通モデル自体を変更する変更は、プロバイダー追加と分離する。両方を同じ差分に含む場合は `provider-onboarding-fit` を失敗とする。

## 機械検証

`provider-onboarding-fit` は、次の共通コマンド契約で実行する。

```text
go run ./cmd/provider-onboarding-fit --base-ref <git-revision>
```

`--base-ref` は一回だけ必須とし、空値、`-` で始まる値および commit として解決できない値を拒否する。実行時に `git rev-parse --verify <git-revision>^{commit}` で commit へ解決し、その commit と `HEAD` の `git merge-base` を比較開始点とする。比較開始点から `HEAD`、index および working tree までの差分を対象に含め、未追跡の provider、fixture、SOT および matrix artifact も検査する。

ローカルと CI は同じ `go run ./cmd/provider-onboarding-fit` と `--base-ref` を使用する。この command は `SOT-ENG-017` の canonical matrix loader と通常の Go test を呼び出し、別形式の matrix または別の検証入口を受け付けない。VCS 情報、`HEAD`、比較 commit または merge base を取得できない場合は、比較を省略して成功にせず gate failure とする。固定ブランチ名、環境変数、設定ファイルまたは暗黙の既定 ref を別の入力経路として使用しない。

この比較対象と通常の Go test を使って次を検証する。

1. 追加対象以外の既存 provider package と provider-specific fixture に差分がない。
2. 共通モデル、既存 capability SOT、既存の能力別 port および既存 provider mapping に差分がない。
3. 共通箇所の差分は、composition root の登録、provider 固有の設定 schema 登録、利用者が明示した場合だけ使う route の受付、および conformance matrix row に限られる。
4. 組込み既定 provider、組込み route および無設定時の enabled set に差分がない。
5. 新 provider package が他の provider package を import しない。
6. 新 binding が同じ capability conformance suite に合格する。
7. capability ごとの最小 test provider が、外部 DTO と provider 固有 package を使わず、同じ typed port、成功、空結果、失敗、`SourceResourceRef`、`Provenance` および continuation の適用可能な case に合格する。
8. 変更前から存在するすべての provider conformance suite が変更なしで合格する。

差分判定はファイル名だけで成功とせず、禁止された package import と conformance case をテストで確認する。

## 成功条件

八条件の一つでも確認できない場合は warning ではなく gate failure とする。プロバイダーをまだ実装しない SOT だけの変更では、SOT 静的検査と設計レビューを行い、実装開始後の最初の provider 変更からこの gate を必須にする。

## 関連

- [SOT-ARCH-016: プロバイダーの段階的な追加](../30-architecture/16-incremental-provider-onboarding.md)
- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-013: プロバイダー契約の検証](13-provider-contract-verification.md)
- [SOT-ENG-017: プロバイダー適合性 matrix](17-provider-conformance-matrix.md)
