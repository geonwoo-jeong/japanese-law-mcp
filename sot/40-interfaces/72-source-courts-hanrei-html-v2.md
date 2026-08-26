# SOT-IF-072: 裁判所「裁判例検索」HTML 情報源 v2

- 状態: 有効

## 規定

`courts-hanrei-html` は、最高裁判所が `https://www.courts.go.jp/hanrei/` 配下で公開する HTML を安全に読み取り、`judicial-decision.search@1`、`judicial-decision.read@1` および `judicial-decision.citing-candidate.search@1` を提供する組込み provider とする。

## 識別

`providerId`、`source`、`serviceUrl`、authority、credential 要件、固定 origin および `judicial-decision.search@1` と `judicial-decision.read@1` の既存契約は、`SOT-IF-043` の後継として維持する。変更点は、descriptor が `judicial-decision.citing-candidate.search@1` を追加で宣言できること、および citation pack が無効なときは同 capability の binding を実効構成へ参加させないことだけとする。

## 公式公開面

- 統合検索: `https://www.courts.go.jp/hanrei/search1/index.html`
- 詳細: `https://www.courts.go.jp/hanrei/{id}/detail{2..8}/index.html`
- 検索方法の案内: `https://www.courts.go.jp/hanrei/tukaikata/index.html`
- 掲載判例の説明: `https://www.courts.go.jp/hanrei/setumei/index.html`
- サイト利用条件: `https://www.courts.go.jp/outline/index.html`

公式に文書化された機械 API は採用せず、検索結果と詳細の HTML だけを取得する。HTML が直接示す PDF は URL と metadata だけを返し、この provider は PDF を取得または解析しない。

## citation 候補検索の追加境界

- 被引用候補検索は、既存検索と同じ統合検索 HTML を使用する。
- 追加 capability は、事件番号と存在する場合の判例集表記だけを検索語として使う。
- 検索語、HTML 本文、URL query または結果本文をエラー、診断またはログへ含めない。
- citation pack が無効な場合は、この capability の route を登録しない。

## 確認

既存の二能力契約を維持したまま descriptor に第三能力を追加できること、pack 無効時に citation capability route を登録しないこと、既存 search/read の資源予算と動作を変えないこと、および公式 HTML fixture で第三能力を追加確認することを契約テストで確認する。

## 関連

- [SOT-IF-069: `judicial-decision.citing-candidate.search` capability v1](69-judicial-citing-candidate-search-capability.md)
- [SOT-IF-073: 裁判所検索の被引用候補マッピング](73-courts-hanrei-citing-candidate-search-mapping.md)
