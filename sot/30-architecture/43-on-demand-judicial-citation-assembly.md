# SOT-ARCH-043: 判例引用追跡のオンデマンド一時組立て

- 状態: 有効

## 規定

判例引用追跡は、各リクエスト内でルート詳細 HTML、必要な PDF および必要な公式検索結果だけを取得して graph を一時組立てし、リクエスト間で原文、抽出 text、引用索引または解析結果を再利用しない。

## 取得順序

1. ルート裁判例の詳細 HTML を一度取得する。
2. 詳細 HTML に `full_text` PDF があり、`direction` に `outgoing` が含まれる場合だけ、その PDF を一度取得して解析する。
3. `direction` に `incoming` が含まれる場合だけ、事件番号と存在する場合の判例集表記で公式検索を最大二回実行する。
4. 取得した原文から request scope の graph を組み立て、応答後に破棄する。

## 永続化の禁止

- HTML、PDF、抽出 text、引用候補、解析済み node/edge、法条正規化結果または検索結果を SQLite、ファイル、埋込み資源またはリクエスト間キャッシュへ保存しない。
- 裁判所検索全件を前もって収集した被引用索引を作成しない。
- 同一プロセス内であっても、別リクエストの PDF 抽出結果を再利用しない。

## 資源上限

判例引用追跡は、少なくとも次の request scope 上限を持つ。

- PDF 応答 16 MiB
- 展開後データ 24 MiB
- PDF 300 ページ
- PDF object 50,000
- 参照深度 32
- parser 実行 4 秒
- 抽出 text 2 MiB
- citation occurrence 256 件
- 判例 edge 64 件
- 法条 edge 32 件
- PDF 同時処理 1 件
- evidence excerpt 256 UTF-8 byte

## 失敗単位

- ルート詳細取得失敗、全要求方向失敗または全体キャンセルはツールエラーとする。
- 片方向だけ失敗した場合は、成功方向の結果を保持して `issues` に失敗を記録する。
- PDF の text layer 不在は、確認済み引用ゼロではなく coverage/issue の縮退理由として扱う。

## 確認

成功、片方向失敗、両方向失敗、キャンセル、タイムアウトおよび上限超過で、一時ファイル、worker、抽出 text および同時実行枠がリクエスト終了後に残らないことを確認する。

## 関連

- [SOT-ARCH-014: 外部原文の一時処理](14-ephemeral-source-artifacts.md)
- [SOT-PROD-016: 判例引用追跡拡張パック](../00-product/16-judicial-citations-extension-pack.md)
