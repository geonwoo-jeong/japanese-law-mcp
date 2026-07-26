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

## 初回導入

初回導入の比較開始点は、この節、`SOT-ENG-020` の実行規定および `SOT-ENG-021` の CI 規定を新設した SOT-only commit そのものとする。この commit は親を一つだけ持ち、その親との差分がこのファイル、`20-verification-gate.md` および `21-git-hook-staged-verification.md` だけであり、その tree にこの節が存在することを機械的に確認する。さらに、その tree で `SOT-ENG-017` の canonical schema、`internal/providerconformance` の matrix loader および `cmd/provider-onboarding-fit` の command がいずれも存在しない場合に限り、これらを同時に有効化する変更を初回導入として扱う。

canonical schema が存在するとは、`SOT-ENG-017` が定める正規 path に Git 管理下の通常ファイルが一つ存在することをいう。matrix loader と command が存在するとは、それぞれの正規 directory に Git 管理下の通常ファイルである `_test.go` 以外の Go source が一つ以上あり、通常の build context で package または command として build 対象になることをいう。symbolic link、空 directory、test-only package、三要素の一部だけがある状態は不在とはみなさず、初回導入を開始できない gate failure とする。その状態の回復条件は、実装を変更する前の別の SOT-only commit で定義する。

初回導入の差分に含めてよいのは、canonical schema、全 row が `status: planned` である最初の provider matrix 一ファイル、共通 matrix loader とその test、`provider-onboarding-fit` command とその test、同じ command を実行する `.githooks/`、`.github/workflows/quality.yml`、`internal/qualitygate` または `cmd/quality-gate` の接続、loader または command から到達する固定依存だけを追加する root の `go.mod` と `go.sum`、および導入状況を示す `wiki/10-implementation-status.md` だけとする。loader または command から到達しない依存、provider package、fixture、`ProviderDescriptor`、capability binding、provider 設定 schema、provider route、組込み既定値、共通 model または共通 capability の変更を同じ差分へ含めない。

module の変更では、loader または command の `_test.go` 以外の通常 source から空 import ではない import graph で到達し、実行時の読込または検証に必要な直接依存と、`go mod tidy` が導く推移依存の checksum だけを追加する。test-only import、空 import または到達しない package を依存追加の根拠にしない。既存依存の版更新または削除、`go` directive または `toolchain` directive の変更、`replace` directive、および `tools/` 配下の module 変更を初回導入へ含めない。

初回導入で追加する `provider-onboarding-fit` 自身を、同じ差分に対して実行する。command は通常時と同じく base ref の解決、merge base、`HEAD`、index、working tree、未追跡ファイル、canonical matrix loader および通常の Go test を検査し、初回導入で許可していない差分が一つでもあれば失敗する。解決した base ref と merge base は初回導入の比較開始点に一致しなければならない。commit 前のローカル検証では `HEAD` が比較開始点そのものであることを、commit 後の CI 検証では比較開始点から `HEAD` までが初回導入の実装 commit 一つだけであることを確認する。

比較開始点に canonical schema、matrix loader または command のいずれかが存在する場合、比較開始点が前記の SOT-only commit ではない場合、または比較開始点から `HEAD` までに二つ以上の commit がある場合は初回導入として扱わない。任意の過去 commit、固定 branch、暗黙の既定値、環境変数、設定ファイル、追加 flag または失敗時の fallback で初回導入判定を強制せず、導入後は常に通常の八条件を適用する。初回導入も `SOT-ENG-020` の適用変更とし、ローカルと CI の両方でこの command の成功を必須とする。

初回導入の条件または許可範囲を採用、変更または廃止する SOT 変更は、初回導入の実装差分へ含めず、SOT だけの先行変更として検証とレビューを完了する。同じ比較範囲内の先行 commit へ置くだけでは分離したと扱わない。

## 成功条件

八条件の一つでも確認できない場合は warning ではなく gate failure とする。プロバイダーをまだ実装しない SOT だけの変更では、SOT 静的検査と設計レビューを行い、実装開始後の最初の provider 変更からこの gate を必須にする。

## 関連

- [SOT-ARCH-016: プロバイダーの段階的な追加](../30-architecture/16-incremental-provider-onboarding.md)
- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-013: プロバイダー契約の検証](13-provider-contract-verification.md)
- [SOT-ENG-017: プロバイダー適合性 matrix](17-provider-conformance-matrix.md)
