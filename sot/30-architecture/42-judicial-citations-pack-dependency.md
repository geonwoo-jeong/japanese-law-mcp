# SOT-ARCH-042: 判例引用追跡拡張パックの従属有効化

- 状態: 有効

## 規定

`judicial-citations` は `judicial-cases` に従属する機能拡張パックとし、`judicial-cases` の専門公開面、provider route および裁判例 `ref` の往復契約が有効な場合に限り、同じ起動単位で追加公開できる。これは `SOT-ARCH-019` の法情報パックとは別の情報領域を増やさず、裁判例領域内の明示的な依存関係だけを追加する。

## 構成境界

- `judicial-citations` は `judicial-cases` の代替ではなく追加 pack とする。
- `judicial-citations` が有効なときは、`judicial-cases` も同時に有効でなければならない。
- `judicial-citations` だけを有効にした設定は、transport 開始前の設定エラーとして拒否する。
- 依存違反を `judicial-cases` の自動有効化、`judicial-citations` の黙示的無効化または実行時 fallback に読み替えない。
- `judicial-cases` が無効な間は、`judicial-citations` の provider factory、binding、route、MCP tool および依存関係を実効構成へ加えない。

## 原子的に有効化する集合

`judicial-citations` が有効な場合に限り、次を一つの集合として追加する。

- 利用シナリオ: `SOT-SCN-015`
- 共通モデル: `SOT-MODEL-035`
- capability: `judicial-decision.case-citation.extract@1` および `judicial-decision.citing-candidate.search@1`
- provider と route: `SOT-IF-074`
- MCP ツール: `trace_judicial_citations`
- application service: 詳細取得、法条・原審正規化、PDF 抽出、候補検索、graph 合成および coverage 集計

どれか一つでも構成できない場合は transport を開始しない。片方の capability だけ、片方の provider だけ、route だけ、または tool だけを公開しない。`judicial-citations` を無効に戻した場合はこの集合だけを除き、`judicial-cases` の検索・詳細公開面を維持する。

## 統合照会との境界

本規定は `query_legal_information` の profile、cue、候補、実行 contribution または公開結果を変更しない。判例関係分析の自然文受理は引き続き未採用とし、専門ツールだけを追加する。

## 確認

`judicial-cases` のみ有効、両方有効、両方無効、および `judicial-citations` のみ有効の各設定で、tool 数、route、provider factory および `query_legal_information` の不変性を構成テストで確認する。

## 関連

- [SOT-ARCH-019: 拡張パックの有効化境界](19-extension-pack-activation-boundary.md)
- [SOT-ARCH-041: 拡張パックの専門公開面の段階採用](41-staged-specialist-extension-surface.md)
- [SOT-IF-067: `judicial-cases` と `judicial-citations` の有効化](../40-interfaces/67-judicial-citations-pack-activation.md)
