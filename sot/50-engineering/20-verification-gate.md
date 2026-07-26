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
- `SOT-ENG-018` が定義する適用変更または初回導入を含む場合は、ローカルと CI の両方で `go run ./cmd/provider-onboarding-fit --base-ref <git-revision>` と中央の品質ゲートが順に成功する。

## 実行

検証ツールはリポジトリ内で固定したバージョンを使用し、中央の品質ゲート実行処理から同じ設定と引数で呼び出す。段階ごとの実行時点、検査スナップショットおよび外部状態の扱いは `SOT-ENG-021` に従う。

変更完了を判定する中央の標準コマンドは、clean checkout したリポジトリのルートから次のとおり実行する。

```text
go run ./cmd/quality-gate --profile=ci --repository=. --git-repository=.
```

`SOT-ENG-018` の適用変更または初回導入では、clean checkout したリポジトリのルートから provider 固有の比較を行う command と中央の標準コマンドを次の順に実行し、両方の成功を変更完了の条件とする。

```text
go run ./cmd/provider-onboarding-fit --base-ref <git-revision>
go run ./cmd/quality-gate --profile=ci --repository=. --git-repository=.
```

通常の pull request の CI は信頼できる event metadata が示す target commit を、push の CI は同じ metadata が示す変更前 commit を `<git-revision>` に渡し、変更側が指定した値で上書きしない。ローカル検証は統合先として意図する commit を明示する。初回導入のローカル検証と CI は、`SOT-ENG-018` が定める SOT-only commit の同じ不変 object ID を渡す。任意の古い commit、固定 branch、暗黙の既定 ref または一方の command だけの成功を完了判定に使用しない。

品質ゲートは最初の失敗で非ゼロ終了し、脆弱性データベース、Git 全履歴または検査ツールへ到達できない状態を成功として扱わない。一つでも失敗した検査を警告へ緩和せず、原因を解消してすべての適用可能なゲートを再実行する。

## 確認

同じソース状態に対する決定的な検査がローカルと CI で一致し、現在の外部データを用いる検査を含む CI の全ゲートが成功することを確認する。各検査の結果からこの SOT または検査対象の SOT ID へ到達できることを確認する。

## 関連

- [SOT-ENG-004: SOT に結び付く検証](04-sot-linked-verification.md)
- [SOT-ENG-018: プロバイダー追加 fitness gate](18-provider-onboarding-fitness-gate.md)
- [SOT-ENG-019: 静的解析とコーディングスタイル](19-static-analysis-and-coding-style.md)
- [SOT-ENG-021: Git フックによる段階的検証](21-git-hook-staged-verification.md)
- [SOT-DEL-004: リリース整合性](../60-delivery/04-release-consistency.md)
