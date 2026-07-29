# SOT-ENG-020: 変更の検証ゲート

- 状態: 有効

## 規定

コード、SOT、検証設定または提供物を変更した成果物は、変更対象に適用できる共通検証ゲートをすべて通過した場合にだけ変更完了として扱う。

## ゲート

- 開発原則のチェックサムが一致する。
- Go ファイルがある場合は、`gofmt`、`go vet`、`SOT-ENG-019` に基づく汎用リンターの設定検証と解析、およびプロジェクト固有の SOT 解析器が成功する。
- Go ファイルがある場合はテストがキャッシュに依存せず成功し、全体のステートメントカバレッジが 80% 以上である。
- 製品モジュールと検証ツール用モジュールの定義に差分がなく、取得済み依存物のチェックサムが一致する。
- SOT の ID、状態、番号、索引および相対リンクの静的検査が成功する。
- 有効なインターフェース SOT に対応する契約テストが成功する。
- GitHub Actions の定義がある場合は、その静的解析が成功する。
- Go の依存関係を変更した場合、または Go コードがある場合は、テスト用パッケージを含む製品コード、および固定済み検証ツールから到達可能な既知の脆弱性検査が、現在の脆弱性データベースに対して成功する。
- 検査対象のソース状態と取得した Git 全履歴に対する秘密情報検査が成功する。
- `SOT-ENG-018` が定義する適用変更または初回導入を含む場合は、CI で `go run ./cmd/provider-onboarding-fit --base-ref <git-revision>` と中央の品質ゲートが順に成功する。
- 統合照会の application、profile、辞書、planner model、公開 interface、評価 corpus、baseline または evaluator を変更した場合は、`SOT-ENG-024` の固定 corpus、最小件数、baseline および全受入基準を標準 command で検証する。

## 実行

検証ツールはリポジトリ内で固定したバージョンを使用し、中央の品質ゲート実行処理から同じ設定と引数で呼び出す。段階ごとの実行時点、検査スナップショット、利用者端末の資源上限および外部状態の扱いは `SOT-ENG-027` に従う。

変更完了を判定する中央の標準コマンドは、clean checkout したリポジトリのルートから次のとおり実行する。

```text
go run ./cmd/quality-gate --profile=ci --repository=. --git-repository=.
```

統合照会の標準評価 command を導入した clean checkout では、中央の品質ゲートが `SOT-ENG-024` の固定引数で定義された command を同じ検査スナップショット内で呼び出す。利用者の実照会または外部ネットワークを評価入力にしない。初回導入までは command が存在するものとして成功扱いせず、初回導入では command、baseline、中央品質ゲートへの接続および全受入基準の成功を同じ変更で完了する。

`SOT-ENG-018` の適用変更または初回導入では、CI の clean checkout した対象
commit に対して provider 固有の比較を行う command と中央の標準コマンドを
次の順に実行し、両方の成功を変更完了の条件とする。ローカルの Git hook では
この provider 固有 command と、それが起動する conformance test を
繰り返さない。

```text
go run ./cmd/provider-onboarding-fit --base-ref <git-revision>
go run ./cmd/quality-gate --profile=ci --repository=. --git-repository=.
```

通常の pull request の CI は信頼できる event metadata が示す target commit を、
push の CI は同じ metadata が示す変更前 commit を `<git-revision>` に渡し、
変更側が指定した値で上書きしない。新規 ref の push で変更前 commit が
all-zero の場合は、同じ event metadata が示す repository の既定 branch を
完全履歴の remote-tracking ref として検証し、その commit を比較元にする。
既定 branch を検証または取得できない場合は失敗し、`HEAD` の第一親だけを
比較元にしない。初回導入では、schema、loader および command を追加する変更の
直前に review 済みの commit を渡す。任意の古い commit、固定 branch、暗黙の
既定 ref または一方の command だけの成功を完了判定に使用しない。

品質ゲートは最初の失敗で非ゼロ終了し、脆弱性データベース、Git 全履歴または検査ツールへ到達できない状態を成功として扱わない。一つでも失敗した検査を警告へ緩和せず、原因を解消してすべての適用可能なゲートを再実行する。

## 確認

ローカルで任意に実行した限定的な決定的検査は、同じソース状態に対する CI の
対応する検査と結果が一致することを確認する。現在の外部データを用いる検査を
含む CI の全ゲートが成功し、各検査の結果からこの SOT または検査対象の
SOT ID へ到達できることを確認する。

## 関連

- [SOT-ENG-004: SOT に結び付く検証](04-sot-linked-verification.md)
- [SOT-ENG-018: プロバイダー追加 fitness gate](18-provider-onboarding-fitness-gate.md)
- [SOT-ENG-019: 静的解析とコーディングスタイル](19-static-analysis-and-coding-style.md)
- [SOT-ENG-027: 省資源の段階的検証](27-resource-aware-verification-stages.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](24-unified-query-evaluation-gate.md)
- [SOT-DEL-004: リリース整合性](../60-delivery/04-release-consistency.md)
