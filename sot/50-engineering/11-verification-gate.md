# SOT-ENG-011: 変更の検証ゲート

- 状態: 廃止
- 後継: [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)

## 規定

Go コードまたは SOT を変更した成果物は、対象に応じた共通検証ゲートをすべて通過した場合にだけ変更完了として扱う。

## ゲート

- Go ファイルがある場合は `gofmt` の差分がない。
- Go ファイルがある場合は `go vet ./...` が成功する。
- Go ファイルがある場合は `go test ./...` が成功する。
- Go のテスト対象コードについて、全体のステートメントカバレッジが 80% 以上である。
- SOT の ID、状態、番号、索引および相対リンクの静的検査が成功する。
- 有効なインターフェース SOT に対応する契約テストが成功する。
- 新しい provider または capability binding を含む場合は、ローカルと CI の両方で `SOT-ENG-018` の `go run ./cmd/provider-onboarding-fit --base-ref <git-revision>` が成功する。

## 確認

ローカルと CI が同じ検証コマンドを実行し、いずれかのゲートが失敗した変更を成功として扱わないことを確認する。

## 廃止理由

静的解析、脆弱性検査および秘密情報検査を共通ゲートに含めて適用範囲を拡張したため、`SOT-ENG-020` へ置き換えた。

## 関連

- [SOT-ENG-004: SOT に結び付く検証](04-sot-linked-verification.md)
- [SOT-ENG-018: プロバイダー追加 fitness gate](18-provider-onboarding-fitness-gate.md)
- [SOT-DEL-004: リリース整合性](../60-delivery/04-release-consistency.md)
