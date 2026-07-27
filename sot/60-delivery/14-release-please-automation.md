# SOT-DEL-014: Release Please による公式リリース

- 状態: 有効

## 規定

公式リリースは、`main` の Conventional Commits を Release Please がまとめる Release PR を入口とし、その PR を merge したソース状態から tag、変更履歴、GitHub release および公式配布物を一つの検証付き workflow で作成する。

## Release PR

Release Please の版は完全な commit SHA で固定し、設定と直前の公式版をリポジトリ内で管理する。Release PR の件名には、merge 後に版を一意に再取得できるように Release Please の版 token を必ず含める。通常の `main` への push は Release PR を作成または更新するだけで、公式配布物を作成しない。

Release PR は次を同じ変更単位で更新する。

- `v` で始まる SemVer
- 日本語の変更区分を持つ `CHANGELOG.md`
- その版で提供する SOT、未実装差分および互換性のない変更を記載するリリース契約

Release PR の merge 後、Release Please は merge commit を指す tag と draft の GitHub release を作成する。draft release に対して tag を遅延作成しない。後続の検証は Release Please の manifest、現在の `main` commit、tag および draft release を相互に照合して確定した版、tag、commit SHA および release 識別子だけを使用する。

## 配布と公開

Release Please と配布処理は、`GITHUB_TOKEN` が作成した event による別 workflow の再起動へ依存せず、同じ `main` push workflow の job として接続する。

Release Please が新しい release を作成した場合、または workflow の再実行時に同じ `main` commit を指す未公開の draft release を確認できた場合だけ、次を順に実行する。通常の push、別 commit の draft および公開済み release は配布処理へ進めない。

1. checkout した commit、tag および Release Please の出力が一致することを確認する。
2. リリース契約と draft release の版、tag および生成元を確認する。
3. `SOT-ENG-020` の品質ゲートを実行する。
4. 同じ commit から四つのデスクトップ向け archive と checksum を作成し、既存の draft release へ追加する。
5. 生成物の版、生成元、内容および checksum を検証する。
6. macOS と Windows の各対象環境で archive の展開、版表示、stdio 初期化および公開ツールを smoke test する。
7. checkout 済みの `CHANGELOG.md` から当該版だけを抽出し、同じ commit のリリース契約と結合して GitHub release 情報とし、draft を公開する。

いずれかの検証に失敗した場合は release を draft のまま保持し、公式配布物として公開しない。同じ workflow を再実行した場合は、同じ生成元 commit の一時 artifact を明示的に置き換えて同じ draft から再開できるようにする。公開済みの同じ tag、別の commit を指す release または GitHub 上で変更された release 本文を検証済みの成果物として上書きまたは転用しない。

## 権限

workflow 全体の既定権限は `contents: read` とし、Release PR、draft release、配布物および公開状態を扱う job だけに必要な書込み権限を付与する。第三者 Action は完全な commit SHA で固定する。

リポジトリでは GitHub Actions による Pull Request の作成を許可する。Release Please には既定の `GITHUB_TOKEN` を使用し、この token が作成した event から別 workflow が起動することは前提にしない。Release PR の merge 後に同じ workflow で権威ある品質ゲートを再実行するため、追加の Personal Access Token はリリースの必須条件にしない。

## 確認

契約テストで、`main` push 以外に起動条件を広げていないこと、Release Please の設定、版を含む Release PR 件名および Action が固定されていること、通常の push では配布 job が実行されないこと、同じ commit の draft だけを再開できること、公開情報を checkout 済みの変更履歴から生成すること、および品質ゲートと四対象の smoke test より前に draft が公開されないことを確認する。

## 関連

- [SOT-DEL-004: リリース整合性](04-release-consistency.md)
- [SOT-DEL-007: インターフェース変更の告知](07-interface-change-disclosure.md)
- [SOT-DEL-010: デスクトップ向け実行ファイル](10-desktop-binaries.md)
- [SOT-ENG-020: 変更の検証ゲート](../50-engineering/20-verification-gate.md)
