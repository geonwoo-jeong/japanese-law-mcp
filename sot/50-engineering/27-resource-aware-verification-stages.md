# SOT-ENG-027: 省資源の段階的検証

- 状態: 有効

## 規定

コミット前、プッシュ前および CI の検証では、利用者の端末で同じ全体検査を
重複実行しない。ローカルの Git hook は変更を安全に送信するための
上限付き検査とし、変更完了を判定する `SOT-ENG-020` の全ゲートは
GitHub Actions の clean checkout で一回だけ実行する。

## 有効化

- Git hook はリポジトリで管理する `.githooks` を使用する。
- clone 後に `.githooks/manage install` を実行し、
  `.githooks/manage check` で当該リポジトリの設定、hook 本文、実行権限および
  private cache を確認する。
- 管理処理は大域 Git 設定を変更しない。当該リポジトリに別の local
  `core.hooksPath` が設定済みの場合は上書きしない。大域設定から継承した
  `core.hooksPath` は、当該リポジトリだけの local 設定で `.githooks` へ
  置き換えることができる。
- Git common directory、専用 `GOCACHE` および
  `GOLANGCI_LINT_CACHE` は symlink ではない private directory とし、
  group または other の書込みを許可しない。
- Go module cache は標準の content-addressed cache を再利用する。
  hook は `GOPROXY=off` で実行し、不足する固定依存物をネットワークから
  暗黙に取得しない。
- hook から起動する Go command は `-p=1` と `GOMAXPROCS=2` で、package の
  build 並列度と process 内並列度を制限する。

## コミット前

`pre-commit` はコミット予定の Git index だけを一時 snapshot へ展開し、
次の変更依存検査だけを行う。

- 開発原則 checksum と snapshot cache policy
- 変更した Go file の `gofmt`
- staged diff の空白 error
- staged 内容の秘密情報
- SOT、README、Wiki または SOT checker を変更した場合の SOT 構造と link
- workflow を変更した場合の `actionlint`
- lint 設定または検証 tool module を変更した場合の lint 設定検証

作業 tree、index または利用者の cache を書き換えず、対話入力、外部
database またはネットワークを要求しない。失敗した場合は commit を中止する。

## プッシュ前

`pre-push` は削除以外の ref 更新が指す重複のない commit tip ごとに snapshot
を作り、次だけを行う。

- `SOT-ENG-018` の適用変更に対する provider onboarding fitness
- 開発原則 checksum と snapshot cache policy
- remote tip を取得できる場合はその tip から local tip まで、取得できない
  新規 ref では local tip から到達可能な履歴に対する秘密情報検査

全 package の test、coverage、`go vet`、汎用 lint、module 整合性、
workflow 解析、snapshot 全体の秘密情報検査および脆弱性検査を
`pre-push` で繰り返さない。これらは CI が同じ commit に対して行う。
削除だけを含む更新は検査対象の tip を持たないため snapshot 検査を行わない。

## CI

CI は clean checkout した対象 commit で `SOT-ENG-020` の `ci` profile を
権威ある完了 gate として実行する。

- 全 Go test と 80% statement coverage は一回だけ測定する。
- test package の並列度は `-p=1` に制限する。
- coverage は各 package が直接 test する code だけを計測し、全 test binary
  へ `./...` を繰り返し計測する `-coverpkg=./...` は使用しない。
- 外部 vulnerability database、Git 全履歴および固定 tool の検査は CI だけで
  実行する。
- `minimum-go` job は最小 Go version で build できることだけを確認し、
  権威 gate と同じ全 test または `go vet` を重複実行しない。
- `SOT-ENG-018` の適用変更では、event metadata が提供する base ref で
  provider onboarding fitness を先に実行する。

CI が外部 database、全履歴または固定 tool へ到達できない場合は、成功へ
緩和しない。

## ローカル開発中の確認

Git hook 以外のローカル確認は、変更した package と直接結び付く契約だけを
選ぶ。

- `go test -p=1 -count=1 ./path/to/changed/package`
- 必要な場合は一つの `-run` pattern で新しい regression を先に確認する。
- 公開動作を変更した場合は最終 binary を一回 build し、該当する MCP request
  を一回以上 smoke test する。

ローカル完了のために `quality-gate --profile=ci`、全 package race、
全 package coverage または同じ test の反復実行を要求しない。
CI 結果が出る前に `SOT-ENG-020` の変更完了を主張しない。

## 確認

quality gate の plan test で `pre-push` に全 package test、vet または lint が
なく、checksum、cache policy および push 範囲の秘密情報検査だけがあることを
確認する。Git hook の pre-push test では、適用する provider onboarding
fitness が quality gate より前に成功し、失敗時は送信を中止することを
別に確認する。

CI plan test で test command が `-p=1`、`-covermode=set` および package 自身の
coverage を使用し、全検証と外部検査に欠落がないことを確認する。Git hook
test で index と作業 tree の分離、tip の重複除去、新規 ref、削除 ref、
private cache および改変した hook の拒否を確認する。

## 関連

- [SOT-ENG-018: プロバイダー追加 fitness gate](18-provider-onboarding-fitness-gate.md)
- [SOT-ENG-019: 静的解析とコーディングスタイル](19-static-analysis-and-coding-style.md)
- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-DEL-004: リリース整合性](../60-delivery/04-release-consistency.md)
