# SOT-IF-071: 裁判所 PDF の判例引用抽出マッピング

- 状態: 有効

## 規定

`courts-hanrei-pdf` は、詳細 HTML が直接示した `full_text` PDF の text layer だけを使い、判例参照とその根拠を `SOT-IF-068` の共通出力へ対応させる。

## 入力対応

- `ref` はルート裁判例の `SourceResourceRef` をそのまま使用する。
- `documentLink` は同じルート裁判例詳細が示した `JudicialDocumentLink` をそのまま使用する。
- PDF URL、判例番号パターン、正規化辞書または fuzzy 候補を入力へ追加しない。

## 抽出規則

- 判例参照として扱うのは、事件番号、裁判所種別、裁判年月日、判例集表記その他の組合せから、公表裁判例詳細 URL を一意に構成できる明示的記載だけとする。
- 一意に構成できない場合、または複数の公表裁判例候補へ一致する場合は `unresolvedMentions` とする。
- 同一 PDF 内の重複言及は occurrence として数えられるが、最終 graph edge では統合できるよう根拠を保持する。
- 言及位置は PDF ページ番号と、そのページ内の短い周辺文字列で表せる範囲に限る。

## 法的意味の禁止

抽出時に次を推論しない。

- 先例性、拘束力、判例変更、破棄、上訴帰結または treatment
- 法令 revision、施行時点または法的評価
- OCR 補正や画像推定に基づく本文補完

## 確認

明示的参照の抽出、一意に解決できない言及の未解決化、重複 occurrence、自己参照の扱い、抜粋長上限、text layer 不在の成功縮退および法的意味の非推論を mapping テストで確認する。

## 関連

- [SOT-IF-068: `judicial-decision.case-citation.extract` capability v1](68-judicial-case-citation-extract-capability.md)
- [SOT-IF-070: 裁判所「裁判例検索」PDF 情報源](70-source-courts-hanrei-pdf.md)
